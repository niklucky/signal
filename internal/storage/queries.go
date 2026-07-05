package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/niklucky/signal/internal/models"
)

// EventStore defines operations for persisting check events.
type EventStore interface {
	CreateEvent(ctx context.Context, hostID, beaconID int64, responseTimeMs int, responseStatus int, success bool, errorMessage, bodySnippet string) error
}

// User represents a stored user row.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    string
}

// Host represents a stored host row.
type Host struct {
	ID                int64
	UserID            int64
	Name              string
	Method            string
	URL               string
	Headers           map[string]string
	Body              string
	TimeoutSec        int
	IntervalSec       int
	ResendIntervalSec int
	ExpectedStatus    int
	Active            bool
	CreatedAt         string
	UpdatedAt         string
}

// CreateEvent inserts a new check event into the database. Response bodies are
// only kept for unsuccessful checks; success events always store an empty body
// snippet to avoid bloating the database.
func (s *Storage) CreateEvent(ctx context.Context, hostID, beaconID int64, responseTimeMs int, responseStatus int, success bool, errorMessage, bodySnippet string) error {
	if success {
		bodySnippet = ""
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO events (host_id, beacon_id, ts, response_time_ms, response_status, success, error_message, body_snippet)
		VALUES ($1, $2, NOW(), $3, $4, $5, $6, $7)
	`, hostID, beaconID, responseTimeMs, responseStatus, success, errorMessage, bodySnippet)
	return err
}

// CreateUser inserts a new user and returns the generated ID.
func (s *Storage) CreateUser(ctx context.Context, email, passwordHash, role string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id
	`, email, passwordHash, role).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetUserByEmail returns a user by email address.
func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpsertHost inserts a new host or updates an existing one matched by user_id and name.
// It returns the host's database ID.
func (s *Storage) UpsertHost(ctx context.Context, userID int64, host models.Host, expectedStatus int, active bool) (int64, error) {
	headersJSON, err := json.Marshal(host.Headers)
	if err != nil {
		return 0, fmt.Errorf("marshal headers: %w", err)
	}

	var resendInterval sql.NullInt64
	if host.ResendInterval > 0 {
		resendInterval = sql.NullInt64{Int64: int64(host.ResendInterval), Valid: true}
	}

	var id int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO hosts (user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, name) DO UPDATE SET
			method = EXCLUDED.method,
			url = EXCLUDED.url,
			headers = EXCLUDED.headers,
			body = EXCLUDED.body,
			timeout_sec = EXCLUDED.timeout_sec,
			interval_sec = EXCLUDED.interval_sec,
			resend_interval_sec = EXCLUDED.resend_interval_sec,
			expected_status = EXCLUDED.expected_status,
			active = EXCLUDED.active,
			updated_at = NOW()
		RETURNING id
	`, userID, host.Name, host.Method, host.URL, headersJSON, host.Body, host.Timeout, host.Interval, resendInterval, expectedStatus, active).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// CreateHost inserts a new host and returns the generated ID.
func (s *Storage) CreateHost(ctx context.Context, userID int64, host models.Host, expectedStatus int, active bool) (int64, error) {
	headersJSON, err := json.Marshal(host.Headers)
	if err != nil {
		return 0, fmt.Errorf("marshal headers: %w", err)
	}

	var id int64
	var resendInterval sql.NullInt64
	if host.ResendInterval > 0 {
		resendInterval = sql.NullInt64{Int64: int64(host.ResendInterval), Valid: true}
	}

	err = s.db.QueryRowContext(ctx, `
		INSERT INTO hosts (user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, userID, host.Name, host.Method, host.URL, headersJSON, host.Body, host.Timeout, host.Interval, resendInterval, expectedStatus, active).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetHostsByUser returns all hosts belonging to a user.
func (s *Storage) GetHostsByUser(ctx context.Context, userID int64) ([]Host, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active, created_at, updated_at
		FROM hosts
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		var h Host
		var headersBytes []byte
		var resendInterval sql.NullInt64
		if err := rows.Scan(
			&h.ID, &h.UserID, &h.Name, &h.Method, &h.URL, &headersBytes, &h.Body,
			&h.TimeoutSec, &h.IntervalSec, &resendInterval, &h.ExpectedStatus, &h.Active,
			&h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(headersBytes) > 0 {
			if err := json.Unmarshal(headersBytes, &h.Headers); err != nil {
				return nil, fmt.Errorf("unmarshal headers: %w", err)
			}
		}
		if resendInterval.Valid {
			h.ResendIntervalSec = int(resendInterval.Int64)
		}
		hosts = append(hosts, h)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hosts, nil
}

// GetHostByName returns a host by user and name.
func (s *Storage) GetHostByName(ctx context.Context, userID int64, name string) (*Host, error) {
	var h Host
	var headersBytes []byte
	var resendInterval sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active, created_at, updated_at
		FROM hosts
		WHERE user_id = $1 AND name = $2
	`, userID, name).Scan(
		&h.ID, &h.UserID, &h.Name, &h.Method, &h.URL, &headersBytes, &h.Body,
		&h.TimeoutSec, &h.IntervalSec, &resendInterval, &h.ExpectedStatus, &h.Active,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(headersBytes) > 0 {
		if err := json.Unmarshal(headersBytes, &h.Headers); err != nil {
			return nil, fmt.Errorf("unmarshal headers: %w", err)
		}
	}
	if resendInterval.Valid {
		h.ResendIntervalSec = int(resendInterval.Int64)
	}
	return &h, nil
}
