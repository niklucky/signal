package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data := `
server:
  address: ":9090"
telegram:
  enabled: true
  bot_token: "token"
  chat_id: "123"
matrix:
  enabled: false
  homeserver: "https://matrix.org"
  user_id: "@bot:matrix.org"
  access_token: "tok"
  room_id: "!room:matrix.org"
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.Address != ":9090" {
		t.Errorf("expected server address :9090, got %q", cfg.Server.Address)
	}
	if !cfg.Telegram.Enabled {
		t.Error("expected telegram enabled")
	}
	if cfg.Telegram.BotToken != "token" {
		t.Errorf("expected bot token 'token', got %q", cfg.Telegram.BotToken)
	}
	if cfg.Matrix.Enabled {
		t.Error("expected matrix disabled")
	}
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.Address != ":8080" {
		t.Errorf("expected default address :8080, got %q", cfg.Server.Address)
	}
}
