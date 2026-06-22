package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/niklucky/signal/internal/config"
	"github.com/niklucky/signal/internal/models"
	"github.com/niklucky/signal/internal/notifier"
	"github.com/niklucky/signal/internal/templates"
)

// Webhook handles incoming Grafana webhook payloads.
type Webhook struct {
	cfg      *config.Config
	telegram *notifier.Telegram
	matrix   *notifier.Matrix
}

// NewWebhook creates a webhook handler.
func NewWebhook(cfg *config.Config) *Webhook {
	var tg *notifier.Telegram
	if cfg.Telegram.Enabled {
		tg = notifier.NewTelegram(cfg.Telegram)
	}

	var mx *notifier.Matrix
	if cfg.Matrix.Enabled {
		mx = notifier.NewMatrix(cfg.Matrix)
	}

	return &Webhook{
		cfg:      cfg,
		telegram: tg,
		matrix:   mx,
	}
}

// ServeHTTP accepts a Grafana webhook, formats it, and relays it.
func (w *Webhook) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read body", "error", err)
		http.Error(rw, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload models.GrafanaWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Error("failed to parse grafana payload", "error", err)
		http.Error(rw, "invalid json", http.StatusBadRequest)
		return
	}

	if len(payload.Alerts) == 0 {
		slog.Warn("grafana payload contains no alerts")
		http.Error(rw, "no alerts in payload", http.StatusBadRequest)
		return
	}

	slog.Info("received grafana webhook",
		"status", payload.Status,
		"title", payload.Title,
		"alerts", len(payload.Alerts),
		"remote", r.RemoteAddr,
	)
	slog.Debug("grafana payload", "body", string(body))

	if w.telegram != nil {
		if err := w.telegram.Send(templates.Telegram(payload)); err != nil {
			slog.Error("failed to send telegram message", "error", err)
			http.Error(rw, "failed to relay to telegram", http.StatusInternalServerError)
			return
		}
	}

	if w.matrix != nil {
		if err := w.matrix.Send(templates.Matrix(payload)); err != nil {
			slog.Error("failed to send matrix message", "error", err)
			http.Error(rw, "failed to relay to matrix", http.StatusInternalServerError)
			return
		}
	}

	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("OK"))
}
