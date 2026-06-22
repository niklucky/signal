package scheduler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.yml")

	data := `
hosts:
  - name: "test-api"
    method: "POST"
    url: "https://example.com/health"
    headers:
      Authorization: "Bearer x"
    body: '{"ok":true}'
    timeout: 5
    interval: 10
    resend_interval: 30
  - name: "test-defaults"
    url: "https://example.com/"
    interval: 5
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write hosts file: %v", err)
	}

	hosts, err := LoadHosts(path)
	if err != nil {
		t.Fatalf("load hosts: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}

	first := hosts[0]
	if first.Name != "test-api" {
		t.Errorf("expected name test-api, got %s", first.Name)
	}
	if first.Method != "POST" {
		t.Errorf("expected method POST, got %s", first.Method)
	}
	if first.URL != "https://example.com/health" {
		t.Errorf("unexpected url: %s", first.URL)
	}
	if first.Headers["Authorization"] != "Bearer x" {
		t.Errorf("unexpected authorization header: %s", first.Headers["Authorization"])
	}
	if first.Body != `{"ok":true}` {
		t.Errorf("unexpected body: %s", first.Body)
	}
	if first.Timeout != 5 {
		t.Errorf("expected timeout 5, got %d", first.Timeout)
	}
	if first.Interval != 10 {
		t.Errorf("expected interval 10, got %d", first.Interval)
	}
	if first.ResendInterval != 30 {
		t.Errorf("expected resend_interval 30, got %d", first.ResendInterval)
	}

	second := hosts[1]
	if second.Method != "GET" {
		t.Errorf("expected default method GET, got %s", second.Method)
	}
	if second.Timeout != 10 {
		t.Errorf("expected default timeout 10, got %d", second.Timeout)
	}
}
