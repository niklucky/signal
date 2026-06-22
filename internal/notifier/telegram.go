package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/niklucky/signal/internal/config"
)

// Telegram sends messages via the Telegram Bot API.
type Telegram struct {
	cfg config.TelegramConfig
}

// NewTelegram creates a new Telegram notifier.
func NewTelegram(cfg config.TelegramConfig) *Telegram {
	return &Telegram{cfg: cfg}
}

// Send delivers the provided text to the configured Telegram chat.
func (t *Telegram) Send(text string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.cfg.BotToken)

	payload := map[string]string{
		"chat_id":    t.cfg.ChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}

	return nil
}

// Escape escapes characters that have special meaning in Telegram HTML mode.
func Escape(v string) string {
	replacer := map[string]string{
		"&": "&amp;",
		"<": "&lt;",
		">": "&gt;",
	}
	out := []rune(v)
	var result []rune
	for _, r := range out {
		s := string(r)
		if rep, ok := replacer[s]; ok {
			result = append(result, []rune(rep)...)
			continue
		}
		result = append(result, r)
	}
	return string(result)
}

