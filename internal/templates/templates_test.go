package templates

import (
	"strings"
	"testing"

	"github.com/niklucky/signal/internal/models"
)

func samplePayload() models.GrafanaWebhook {
	return models.GrafanaWebhook{
		Status: "firing",
		Title:  "[FIRING:1] CPU is high load",
		Alerts: []models.Alert{
			{
				Status:       "firing",
				Labels:       map[string]string{"instance": "node_exporter:9100"},
				Annotations:  map[string]string{"summary": "High CPU load"},
				StartsAt:     "2026-06-22T15:13:30Z",
				GeneratorURL: "http://example.com/grafana/alerting/view",
				SilenceURL:   "http://example.com/grafana/alerting/silence",
				Values:       map[string]float64{"A": 4.16, "C": 1},
			},
		},
	}
}

func TestTelegram(t *testing.T) {
	msg := Telegram(samplePayload())

	if !strings.Contains(msg, "🔥") {
		t.Error("expected fire emoji")
	}
	if !strings.Contains(msg, "CPU is high load") {
		t.Error("expected title")
	}
	if !strings.Contains(msg, "node_exporter:9100") {
		t.Error("expected instance")
	}
	if !strings.Contains(msg, "View in Grafana") {
		t.Error("expected generator link")
	}
}

func TestMatrix(t *testing.T) {
	msg := Matrix(samplePayload())

	if !strings.Contains(msg, "🔥") {
		t.Error("expected fire emoji")
	}
	if !strings.Contains(msg, "CPU is high load") {
		t.Error("expected title")
	}
	if !strings.Contains(msg, "9100") {
		t.Error("expected instance")
	}
	if !strings.Contains(msg, "[View in Grafana]") {
		t.Error("expected generator link")
	}
}

func TestStatusEmoji(t *testing.T) {
	if statusEmoji("resolved") != "✅" {
		t.Error("expected checkmark for resolved")
	}
	if statusEmoji("firing") != "🔥" {
		t.Error("expected fire for firing")
	}
}
