package templates

import (
	"fmt"
	"strings"

	"github.com/niklucky/signal/internal/models"
)

// Matrix renders a Grafana webhook payload as a Markdown message for Matrix.
func Matrix(payload models.GrafanaWebhook) string {
	if len(payload.Alerts) == 0 {
		return escapeMarkdown(fmt.Sprintf("Grafana alert: %s", payload.Title))
	}

	alert := payload.Alerts[0]
	emoji := statusEmoji(payload.Status)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s **%s**\n\n", emoji, escapeMarkdown(payload.Title)))
	b.WriteString(fmt.Sprintf("**Status:** %s\n", escapeMarkdown(payload.Status)))

	if instance := alert.Labels["instance"]; instance != "" {
		b.WriteString(fmt.Sprintf("**Instance:** %s\n", escapeMarkdown(instance)))
	}

	if len(alert.Values) > 0 {
		b.WriteString("**Values:**\n")
		for k, v := range alert.Values {
			b.WriteString(fmt.Sprintf("  %s = %v\n", escapeMarkdown(k), v))
		}
	}

	if summary := firstNonEmpty(alert.Annotations["summary"], payload.CommonAnnotations["summary"]); summary != "" {
		b.WriteString(fmt.Sprintf("\n**Summary:**\n%s\n", escapeMarkdown(summary)))
	}

	if alert.StartsAt != "" {
		b.WriteString(fmt.Sprintf("\n**Since:** %s\n", escapeMarkdown(alert.StartsAt)))
	}

	if alert.GeneratorURL != "" {
		b.WriteString(fmt.Sprintf("\n[View in Grafana](%s)\n", alert.GeneratorURL))
	}
	if alert.SilenceURL != "" {
		b.WriteString(fmt.Sprintf("[Silence alert](%s)\n", alert.SilenceURL))
	}

	return b.String()
}

func escapeMarkdown(v string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"[", "\\[",
		"]", "\\]",
	).Replace(v)
}
