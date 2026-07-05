package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Storage wraps a sql.DB and provides data access methods.
type Storage struct {
	db *sql.DB
}

// New opens a connection to the Postgres database at dsn and runs migrations.
func New(dsn string) (*Storage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &Storage{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return s, nil
}

// Close closes the database connection pool.
func (s *Storage) Close() error {
	return s.db.Close()
}

// DB exposes the underlying connection pool for advanced use.
func (s *Storage) DB() *sql.DB {
	return s.db
}

// migrate applies SQL migrations from the embedded migrations directory.
func (s *Storage) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	var versions []int64
	migrationSQL := make(map[int64]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, err := strconv.ParseInt(strings.SplitN(entry.Name(), "_", 2)[0], 10, 64)
		if err != nil {
			return fmt.Errorf("parse migration version from %s: %w", entry.Name(), err)
		}

		data, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		versions = append(versions, version)
		migrationSQL[version] = string(data)
	}

	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })

	for _, version := range versions {
		var applied bool
		if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if applied {
			continue
		}

		tx, err := s.db.BeginTx(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}

		if _, err := tx.Exec(migrationSQL[version]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", version, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}

	return nil
}
