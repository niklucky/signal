package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/niklucky/signal/internal/config"
	"github.com/niklucky/signal/internal/handlers"
	"github.com/niklucky/signal/internal/models"
	"github.com/niklucky/signal/internal/notifier"
	"github.com/niklucky/signal/internal/scheduler"
	"github.com/niklucky/signal/internal/storage"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config")
	flag.Parse()

	setupLogging()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	var telegram *notifier.Telegram
	if cfg.Telegram.Enabled {
		telegram = notifier.NewTelegram(cfg.Telegram)
	}

	var matrix *notifier.Matrix
	if cfg.Matrix.Enabled {
		matrix = notifier.NewMatrix(cfg.Matrix)
	}

	var store *storage.Storage
	if cfg.Database.URL != "" {
		var err error
		store, err = storage.New(cfg.Database.URL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer store.Close()
		slog.Info("database connected")
	}

	http.Handle("/webhooks/grafana", handlers.NewWebhook(cfg, telegram, matrix))

	hosts, err := loadSchedulerHosts(cfg, store)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("hosts file not found, scheduler disabled", "file", cfg.Scheduler.HostsFile)
		} else {
			slog.Error("failed to load hosts", "error", err)
			os.Exit(1)
		}
	}

	if len(hosts) > 0 {
		source := "database"
		var eventStore storage.EventStore
		if store == nil {
			slog.Info("store is not configured, hosts will be loaded from file", "file", cfg.Scheduler.HostsFile)
			source = cfg.Scheduler.HostsFile
		} else {
			eventStore = store
		}
		slog.Info("starting scheduler", "hosts", len(hosts), "source", source)
		scheduler.New(hosts, telegram, matrix, eventStore).Start()
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	slog.Info("starting signal server", "address", cfg.Server.Address)
	if err := http.ListenAndServe(cfg.Server.Address, nil); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// loadSchedulerHosts loads hosts from the configured file.
// If a database is configured, it ensures a default user exists, syncs the
// file-based hosts into the database, and returns the hosts from the database
// so that events are recorded with real host IDs.
func loadSchedulerHosts(cfg *config.Config, store *storage.Storage) ([]scheduler.ScheduledHost, error) {
	hosts, err := scheduler.LoadHosts(cfg.Scheduler.HostsFile)
	if err != nil {
		return nil, err
	}

	if store == nil {
		return hosts, nil
	}

	ctx := context.Background()

	// Ensure a default user exists for file-based hosts until auth is implemented.
	const defaultEmail = "default@signal.local"
	user, err := store.GetUserByEmail(ctx, defaultEmail)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("lookup default user: %w", err)
		}
		id, createErr := store.CreateUser(ctx, defaultEmail, "", "admin")
		if createErr != nil {
			return nil, fmt.Errorf("create default user: %w", createErr)
		}
		user = &storage.User{ID: id, Email: defaultEmail, Role: "admin"}
	}

	for i := range hosts {
		id, err := store.UpsertHost(ctx, user.ID, hosts[i].Host, http.StatusOK, true)
		if err != nil {
			return nil, fmt.Errorf("sync host %s: %w", hosts[i].Host.Name, err)
		}
		hosts[i].ID = id
	}

	dbHosts, err := store.GetHostsByUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("load hosts from database: %w", err)
	}

	scheduled := make([]scheduler.ScheduledHost, len(dbHosts))
	for i, h := range dbHosts {
		scheduled[i] = scheduler.ScheduledHost{
			ID: h.ID,
			Host: models.Host{
				Name:           h.Name,
				Method:         h.Method,
				URL:            h.URL,
				Headers:        h.Headers,
				Body:           h.Body,
				Timeout:        h.TimeoutSec,
				Interval:       h.IntervalSec,
				ResendInterval: h.ResendIntervalSec,
			},
		}
	}
	return scheduled, nil
}

func setupLogging() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}
