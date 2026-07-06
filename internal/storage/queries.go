package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/niklucky/signal/internal/models"
)

// EventStore defines operations for persisting check events.
type EventStore interface {
	CreateEvent(ctx context.Context, hostID, beaconID int64, responseTimeMs int, responseStatus int, success bool, errorMessage, bodySnippet string) error
}

// User represents a stored user row.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    string
}

// Host represents a stored host row.
type Host struct {
	ID                int64
	UserID            int64
	Name              string
	Method            string
	URL               string
	Headers           map[string]string
	Body              string
	TimeoutSec        int
	IntervalSec       int
	ResendIntervalSec int
	ExpectedStatus    int
	Active            bool
	CreatedAt         string
	UpdatedAt         string
}

// CreateEvent inserts a new check event into the database. Response bodies are
// only kept for unsuccessful checks; success events always store an empty body
// snippet to avoid bloating the database.
func (s *Storage) CreateEvent(ctx context.Context, hostID, beaconID int64, responseTimeMs int, responseStatus int, success bool, errorMessage, bodySnippet string) error {
	if success {
		bodySnippet = ""
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO events (host_id, beacon_id, ts, response_time_ms, response_status, success, error_message, body_snippet)
		VALUES ($1, $2, NOW(), $3, $4, $5, $6, $7)
	`, hostID, beaconID, responseTimeMs, responseStatus, success, errorMessage, bodySnippet)
	return err
}

// CreateUser inserts a new user and returns the generated ID.
func (s *Storage) CreateUser(ctx context.Context, email, passwordHash, role string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id
	`, email, passwordHash, role).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetUserByEmail returns a user by email address.
func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpsertHost inserts a new host or updates an existing one matched by user_id and name.
// It returns the host's database ID.
func (s *Storage) UpsertHost(ctx context.Context, userID int64, host models.Host, expectedStatus int, active bool) (int64, error) {
	headersJSON, err := json.Marshal(host.Headers)
	if err != nil {
		return 0, fmt.Errorf("marshal headers: %w", err)
	}

	var resendInterval sql.NullInt64
	if host.ResendInterval > 0 {
		resendInterval = sql.NullInt64{Int64: int64(host.ResendInterval), Valid: true}
	}

	var id int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO hosts (user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id, name) DO UPDATE SET
			method = EXCLUDED.method,
			url = EXCLUDED.url,
			headers = EXCLUDED.headers,
			body = EXCLUDED.body,
			timeout_sec = EXCLUDED.timeout_sec,
			interval_sec = EXCLUDED.interval_sec,
			resend_interval_sec = EXCLUDED.resend_interval_sec,
			expected_status = EXCLUDED.expected_status,
			active = EXCLUDED.active,
			updated_at = NOW()
		RETURNING id
	`, userID, host.Name, host.Method, host.URL, headersJSON, host.Body, host.Timeout, host.Interval, resendInterval, expectedStatus, active).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// CreateHost inserts a new host and returns the generated ID.
func (s *Storage) CreateHost(ctx context.Context, userID int64, host models.Host, expectedStatus int, active bool) (int64, error) {
	headersJSON, err := json.Marshal(host.Headers)
	if err != nil {
		return 0, fmt.Errorf("marshal headers: %w", err)
	}

	var id int64
	var resendInterval sql.NullInt64
	if host.ResendInterval > 0 {
		resendInterval = sql.NullInt64{Int64: int64(host.ResendInterval), Valid: true}
	}

	err = s.db.QueryRowContext(ctx, `
		INSERT INTO hosts (user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, userID, host.Name, host.Method, host.URL, headersJSON, host.Body, host.Timeout, host.Interval, resendInterval, expectedStatus, active).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetHostsByUser returns all hosts belonging to a user.
func (s *Storage) GetHostsByUser(ctx context.Context, userID int64) ([]Host, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active, created_at, updated_at
		FROM hosts
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		var h Host
		var headersBytes []byte
		var resendInterval sql.NullInt64
		if err := rows.Scan(
			&h.ID, &h.UserID, &h.Name, &h.Method, &h.URL, &headersBytes, &h.Body,
			&h.TimeoutSec, &h.IntervalSec, &resendInterval, &h.ExpectedStatus, &h.Active,
			&h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(headersBytes) > 0 {
			if err := json.Unmarshal(headersBytes, &h.Headers); err != nil {
				return nil, fmt.Errorf("unmarshal headers: %w", err)
			}
		}
		if resendInterval.Valid {
			h.ResendIntervalSec = int(resendInterval.Int64)
		}
		hosts = append(hosts, h)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hosts, nil
}

// GetUserByID returns a user by ID.
func (s *Storage) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserPassword updates the password hash for a user.
func (s *Storage) UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $1 WHERE id = $2
	`, passwordHash, id)
	return err
}

// UpdateUserProfile updates a user's email.
func (s *Storage) UpdateUserProfile(ctx context.Context, id int64, email string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET email = $1 WHERE id = $2
	`, email, id)
	return err
}

// GetHostByID returns a host by ID.
func (s *Storage) GetHostByID(ctx context.Context, id int64) (*Host, error) {
	var h Host
	var headersBytes []byte
	var resendInterval sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active, created_at, updated_at
		FROM hosts
		WHERE id = $1
	`, id).Scan(
		&h.ID, &h.UserID, &h.Name, &h.Method, &h.URL, &headersBytes, &h.Body,
		&h.TimeoutSec, &h.IntervalSec, &resendInterval, &h.ExpectedStatus, &h.Active,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(headersBytes) > 0 {
		if err := json.Unmarshal(headersBytes, &h.Headers); err != nil {
			return nil, fmt.Errorf("unmarshal headers: %w", err)
		}
	}
	if resendInterval.Valid {
		h.ResendIntervalSec = int(resendInterval.Int64)
	}
	return &h, nil
}

// UpdateHost updates an existing host.
func (s *Storage) UpdateHost(ctx context.Context, id int64, host models.Host, expectedStatus int, active bool) error {
	headersJSON, err := json.Marshal(host.Headers)
	if err != nil {
		return fmt.Errorf("marshal headers: %w", err)
	}
	var resendInterval sql.NullInt64
	if host.ResendInterval > 0 {
		resendInterval = sql.NullInt64{Int64: int64(host.ResendInterval), Valid: true}
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE hosts SET
			name = $1,
			method = $2,
			url = $3,
			headers = $4,
			body = $5,
			timeout_sec = $6,
			interval_sec = $7,
			resend_interval_sec = $8,
			expected_status = $9,
			active = $10,
			updated_at = NOW()
		WHERE id = $11
	`, host.Name, host.Method, host.URL, headersJSON, host.Body, host.Timeout, host.Interval, resendInterval, expectedStatus, active, id)
	return err
}

// DeleteHost removes a host by ID.
func (s *Storage) DeleteHost(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM hosts WHERE id = $1`, id)
	return err
}

// GetRecentEvents returns the most recent events for a host.
func (s *Storage) GetRecentEvents(ctx context.Context, hostID int64, limit int) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, host_id, beacon_id, ts, response_time_ms, response_status, success, error_message, body_snippet
		FROM events
		WHERE host_id = $1
		ORDER BY ts DESC
		LIMIT $2
	`, hostID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.HostID, &e.BeaconID, &e.Timestamp, &e.ResponseTimeMs, &e.ResponseStatus, &e.Success, &e.ErrorMessage, &e.BodySnippet); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// Event represents a stored check event row.
type Event struct {
	ID             int64
	HostID         int64
	BeaconID       int64
	Timestamp      string
	ResponseTimeMs int
	ResponseStatus int
	Success        bool
	ErrorMessage   string
	BodySnippet    string
}

// DashboardData returns aggregated recent metrics for all hosts of a user.
func (s *Storage) DashboardData(ctx context.Context, userID int64) ([]DashboardHost, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT h.id, h.name, h.url, h.active,
			(SELECT COUNT(*) FROM events e WHERE e.host_id = h.id AND e.ts > NOW() - INTERVAL '24 hours' AND e.success = true) as successes,
			(SELECT COUNT(*) FROM events e WHERE e.host_id = h.id AND e.ts > NOW() - INTERVAL '24 hours' AND e.success = false) as failures
		FROM hosts h
		WHERE h.user_id = $1
		ORDER BY h.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []DashboardHost
	for rows.Next() {
		var h DashboardHost
		if err := rows.Scan(&h.ID, &h.Name, &h.URL, &h.Active, &h.Successes24h, &h.Failures24h); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hosts, nil
}

// DashboardHost is a summary row for the dashboard.
type DashboardHost struct {
	ID           int64
	Name         string
	URL          string
	Active       bool
	Successes24h int
	Failures24h  int
}

// HostTimeSeries returns response time and success data for a host over a window.
func (s *Storage) HostTimeSeries(ctx context.Context, hostID int64, window string) ([]TimePoint, error) {
	query := `
		SELECT EXTRACT(EPOCH FROM ts)::bigint as ts, response_time_ms, success
		FROM events
		WHERE host_id = $1 AND ts > NOW() - $2::interval
		ORDER BY ts ASC
	`
	rows, err := s.db.QueryContext(ctx, query, hostID, window)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TimePoint
	for rows.Next() {
		var p TimePoint
		if err := rows.Scan(&p.Timestamp, &p.ResponseTimeMs, &p.Success); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// TimePoint is a single dashboard metric point.
type TimePoint struct {
	Timestamp      int64 `json:"ts"`
	ResponseTimeMs int   `json:"response_time_ms"`
	Success        bool  `json:"success"`
}

// GetUsers returns all users.
func (s *Storage) GetUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}


func (s *Storage) GetHostByName(ctx context.Context, userID int64, name string) (*Host, error) {
	var h Host
	var headersBytes []byte
	var resendInterval sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, method, url, headers, body, timeout_sec, interval_sec, resend_interval_sec, expected_status, active, created_at, updated_at
		FROM hosts
		WHERE user_id = $1 AND name = $2
	`, userID, name).Scan(
		&h.ID, &h.UserID, &h.Name, &h.Method, &h.URL, &headersBytes, &h.Body,
		&h.TimeoutSec, &h.IntervalSec, &resendInterval, &h.ExpectedStatus, &h.Active,
		&h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(headersBytes) > 0 {
		if err := json.Unmarshal(headersBytes, &h.Headers); err != nil {
			return nil, fmt.Errorf("unmarshal headers: %w", err)
		}
	}
	if resendInterval.Valid {
		h.ResendIntervalSec = int(resendInterval.Int64)
	}
	return &h, nil
}
