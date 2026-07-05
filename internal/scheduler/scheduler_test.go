package scheduler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/niklucky/signal/internal/models"
)

type fakeEventStore struct {
	lastBodySnippet string
	lastSuccess     bool
}

func (f *fakeEventStore) CreateEvent(ctx context.Context, hostID, beaconID int64, responseTimeMs int, responseStatus int, success bool, errorMessage, bodySnippet string) error {
	f.lastSuccess = success
	if success {
		bodySnippet = ""
	}
	f.lastBodySnippet = bodySnippet
	return nil
}

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

	first := hosts[0].Host
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

	second := hosts[1].Host
	if second.Method != "GET" {
		t.Errorf("expected default method GET, got %s", second.Method)
	}
	if second.Timeout != 10 {
		t.Errorf("expected default timeout 10, got %d", second.Timeout)
	}
}

func TestRecordEventSkipsBodyOnSuccess(t *testing.T) {
	store := &fakeEventStore{}
	s := &Scheduler{
		store:  store,
		states: make(map[string]*hostState),
	}

	host := ScheduledHost{
		ID: 1,
		Host: models.Host{
			Name: "success-host",
			URL:  "https://example.com",
		},
	}

	s.recordEvent(host, http.StatusOK, "this should not be stored", nil, 10*time.Millisecond)

	if !store.lastSuccess {
		t.Errorf("expected success=true")
	}
	if store.lastBodySnippet != "" {
		t.Errorf("expected empty body snippet for successful event, got %q", store.lastBodySnippet)
	}
}

func TestRecordEventKeepsBodyOnFailure(t *testing.T) {
	store := &fakeEventStore{}
	s := &Scheduler{
		store:  store,
		states: make(map[string]*hostState),
	}

	host := ScheduledHost{
		ID: 2,
		Host: models.Host{
			Name: "failure-host",
			URL:  "https://example.com",
		},
	}

	s.recordEvent(host, http.StatusInternalServerError, "error body", nil, 10*time.Millisecond)

	if store.lastSuccess {
		t.Errorf("expected success=false")
	}
	if store.lastBodySnippet != "error body" {
		t.Errorf("expected body snippet to be kept for failure, got %q", store.lastBodySnippet)
	}
}

func TestDoRequestLimitsResponseBody(t *testing.T) {
	largeBody := strings.Repeat("x", maxResponseBytes+100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	s := &Scheduler{
		client: &http.Client{},
		states: make(map[string]*hostState),
	}

	status, body, err := s.doRequest(models.Host{
		Name:    "huge-host",
		Method:  http.MethodGet,
		URL:     server.URL,
		Timeout: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if len(body) <= maxResponseBytes {
		t.Errorf("expected body to be truncated to at most %d bytes, got %d", maxResponseBytes, len(body))
	}
	if !strings.HasSuffix(body, "(response truncated: exceeded 1 MiB limit)") {
		t.Errorf("expected truncation marker in body, got suffix %q", body[len(body)-50:])
	}
}
