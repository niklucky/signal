package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/niklucky/signal/internal/config"
	"github.com/niklucky/signal/internal/notifier"
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

// ServeHTTP accepts a Grafana webhook, dumps the full request, and relays it.
func (w *Webhook) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read body", "error", err)
		http.Error(rw, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	message := buildMessage(r, body)

	slog.Info("received grafana webhook",
		"method", r.Method,
		"path", r.URL.Path,
		"remote", r.RemoteAddr,
	)

	if w.telegram != nil {
		if err := w.telegram.Send(message); err != nil {
			slog.Error("failed to send telegram message", "error", err)
			http.Error(rw, "failed to relay to telegram", http.StatusInternalServerError)
			return
		}
	}

	if w.matrix != nil {
		if err := w.matrix.Send(message); err != nil {
			slog.Error("failed to send matrix message", "error", err)
			http.Error(rw, "failed to relay to matrix", http.StatusInternalServerError)
			return
		}
	}

	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("OK"))
}

func buildMessage(r *http.Request, body []byte) string {
	var msg string
	msg += fmt.Sprintf("<b>Grafana Webhook</b>\n\n")
	msg += fmt.Sprintf("<b>Method:</b> %s\n", notifier.Escape(r.Method))
	msg += fmt.Sprintf("<b>Path:</b> %s\n", notifier.Escape(r.URL.Path))
	msg += fmt.Sprintf("<b>Remote:</b> %s\n\n", notifier.Escape(r.RemoteAddr))

	msg += "<b>Headers:</b>\n"
	for name, values := range r.Header {
		for _, v := range values {
			msg += fmt.Sprintf("  %s: %s\n", notifier.Escape(name), notifier.Escape(v))
		}
	}

	msg += "\n<b>Body:</b>\n<pre>"
	pretty, err := json.MarshalIndent(stripJSON(body), "", "  ")
	if err != nil {
		msg += notifier.Escape(string(body))
	} else {
		msg += notifier.Escape(string(pretty))
	}
	msg += "</pre>"

	return msg
}

func stripJSON(raw []byte) any {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}
