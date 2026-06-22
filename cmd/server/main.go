package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/niklucky/signal/internal/config"
	"github.com/niklucky/signal/internal/handlers"
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

	http.Handle("/webhooks/grafana", handlers.NewWebhook(cfg))
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
