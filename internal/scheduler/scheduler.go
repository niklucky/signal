package scheduler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/niklucky/signal/internal/models"
	"github.com/niklucky/signal/internal/notifier"
	"gopkg.in/yaml.v3"
)

// Scheduler periodically pings configured hosts and sends notifications on failure.
type Scheduler struct {
	hosts    []models.Host
	telegram *notifier.Telegram
	matrix   *notifier.Matrix
	client   *http.Client
	states   map[string]*hostState
	mu       sync.Mutex
}

type hostState struct {
	failing   bool
	lastAlert time.Time
}

// LoadHosts reads the hosts configuration from the given YAML path.
func LoadHosts(path string) ([]models.Host, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read hosts file: %w", err)
	}

	var file models.HostsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse hosts file: %w", err)
	}

	for i := range file.Hosts {
		if file.Hosts[i].Method == "" {
			file.Hosts[i].Method = http.MethodGet
		}
		if file.Hosts[i].Timeout <= 0 {
			file.Hosts[i].Timeout = 10
		}
	}

	return file.Hosts, nil
}

// New creates a scheduler from the loaded host list.
func New(hosts []models.Host, telegram *notifier.Telegram, matrix *notifier.Matrix) *Scheduler {
	return &Scheduler{
		hosts:    hosts,
		telegram: telegram,
		matrix:   matrix,
		client:   &http.Client{},
		states:   make(map[string]*hostState),
	}
}

// Start begins the background checks. It returns immediately.
func (s *Scheduler) Start() {
	for _, host := range s.hosts {
		if host.Interval <= 0 {
			slog.Warn("skipping host with invalid interval", "host", host.Name, "interval", host.Interval)
			continue
		}

		h := host
		go s.run(h)
	}
}

func (s *Scheduler) run(host models.Host) {
	ticker := time.NewTicker(time.Duration(host.Interval) * time.Second)
	defer ticker.Stop()

	// Run the first check immediately.
	s.check(host)

	for range ticker.C {
		s.check(host)
	}
}

func (s *Scheduler) check(host models.Host) {
	status, err := s.doRequest(host)
	if err != nil {
		slog.Error("host check request failed",
			"host", host.Name,
			"url", host.URL,
			"error", err,
		)
	}

	s.handleResult(host, status)
}

func (s *Scheduler) handleResult(host models.Host, status int) {
	s.mu.Lock()
	state := s.states[host.Name]
	if state == nil {
		state = &hostState{}
		s.states[host.Name] = state
	}

	if status == http.StatusOK {
		if state.failing {
			state.failing = false
			state.lastAlert = time.Time{}
			s.mu.Unlock()
			s.sendResolved(host)
			return
		}
		s.mu.Unlock()
		return
	}

	shouldSend := !state.failing || state.lastAlert.IsZero()
	if host.ResendInterval > 0 && !state.lastAlert.IsZero() {
		if time.Since(state.lastAlert) >= time.Duration(host.ResendInterval)*time.Second {
			shouldSend = true
		}
	}

	state.failing = true
	if shouldSend {
		state.lastAlert = time.Now()
	}
	s.mu.Unlock()

	if shouldSend {
		s.sendAlert(host, status)
	}
}

func (s *Scheduler) sendResolved(host models.Host) {
	message := fmt.Sprintf("✅ Host recovered: %s\n\n%s %s is back to OK", host.Name, host.Method, host.URL)

	slog.Info("host recovered",
		"host", host.Name,
		"url", host.URL,
	)

	s.send(host, message)
}

func (s *Scheduler) sendAlert(host models.Host, status int) {
	message := fmt.Sprintf("🔥 Host check failed: %s\n\n%s %s returned status %d", host.Name, host.Method, host.URL, status)
	if status == 0 {
		message = fmt.Sprintf("🔥 Host check failed: %s\n\n%s %s is unreachable", host.Name, host.Method, host.URL)
	}

	slog.Warn("host check failed",
		"host", host.Name,
		"url", host.URL,
		"status", status,
	)

	s.send(host, message)
}

func (s *Scheduler) send(host models.Host, message string) {
	if s.telegram != nil {
		if err := s.telegram.Send(message); err != nil {
			slog.Error("failed to send telegram message", "host", host.Name, "error", err)
		}
	}

	if s.matrix != nil {
		if err := s.matrix.Send(message); err != nil {
			slog.Error("failed to send matrix message", "host", host.Name, "error", err)
		}
	}
}

func (s *Scheduler) doRequest(host models.Host) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(host.Timeout)*time.Second)
	defer cancel()

	var body io.Reader
	if host.Body != "" {
		body = bytes.NewReader([]byte(host.Body))
	}

	req, err := http.NewRequestWithContext(ctx, host.Method, host.URL, body)
	if err != nil {
		return 0, err
	}

	for key, value := range host.Headers {
		req.Header.Set(key, value)
	}

	if host.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
