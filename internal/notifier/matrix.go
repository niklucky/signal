package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/niklucky/signal/internal/config"
)

// Matrix sends messages to a Matrix room using the Client-Server API.
type Matrix struct {
	cfg    config.MatrixConfig
	client *http.Client
}

// NewMatrix creates a new Matrix notifier.
func NewMatrix(cfg config.MatrixConfig) *Matrix {
	return &Matrix{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Send delivers the provided text to the configured Matrix room.
func (m *Matrix) Send(text string) error {
	if !m.cfg.Enabled {
		return nil
	}

	txnID := fmt.Sprintf("signal-%d", time.Now().UnixNano())
	apiURL := fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s?access_token=%s",
		m.cfg.Homeserver,
		url.PathEscape(m.cfg.RoomID),
		url.PathEscape(txnID),
		url.QueryEscape(m.cfg.AccessToken),
	)

	payload := map[string]string{
		"msgtype": "m.text",
		"body":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal matrix payload: %w", err)
	}

	resp, err := m.client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("matrix request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("matrix returned status %d", resp.StatusCode)
	}

	return nil
}
