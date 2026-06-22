package templates

import (
	"fmt"
	"strings"

	"github.com/niklucky/signal/internal/models"
)

// Telegram renders a Grafana webhook payload as a Telegram HTML message.
func Telegram(payload models.GrafanaWebhook) string {
	if len(payload.Alerts) == 0 {
		return escapeTelegram(fmt.Sprintf("Grafana alert: %s", payload.Title))
	}

	alert := payload.Alerts[0]
	emoji := statusEmoji(payload.Status)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s <b>%s</b>\n\n", emoji, escapeTelegram(payload.Title)))
	b.WriteString(fmt.Sprintf("<b>Status:</b> %s\n", escapeTelegram(payload.Status)))

	if instance := alert.Labels["instance"]; instance != "" {
		b.WriteString(fmt.Sprintf("<b>Instance:</b> %s\n", escapeTelegram(instance)))
	}

	if len(alert.Values) > 0 {
		b.WriteString("<b>Values:</b>\n")
		for k, v := range alert.Values {
			b.WriteString(fmt.Sprintf("  %s = %v\n", escapeTelegram(k), v))
		}
	}

	if summary := firstNonEmpty(alert.Annotations["summary"], payload.CommonAnnotations["summary"]); summary != "" {
		b.WriteString(fmt.Sprintf("\n<b>Summary:</b>\n%s\n", escapeTelegram(summary)))
	}

	if alert.StartsAt != "" {
		b.WriteString(fmt.Sprintf("\n<b>Since:</b> %s\n", escapeTelegram(alert.StartsAt)))
	}

	if alert.GeneratorURL != "" {
		b.WriteString(fmt.Sprintf("\n<a href=\"%s\">View in Grafana</a>\n", escapeTelegram(alert.GeneratorURL)))
	}
	if alert.SilenceURL != "" {
		b.WriteString(fmt.Sprintf("<a href=\"%s\">Silence alert</a>\n", escapeTelegram(alert.SilenceURL)))
	}

	return b.String()
}

func escapeTelegram(v string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(v)
}
