package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/niklucky/signal/internal/config"
	"github.com/niklucky/signal/internal/models"
	"github.com/niklucky/signal/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type mockStore struct {
	users  map[int64]*storage.User
	hosts  map[int64]*storage.Host
	nextID int64
}

func newMockStore() *mockStore {
	return &mockStore{
		users:  make(map[int64]*storage.User),
		hosts:  make(map[int64]*storage.Host),
		nextID: 1,
	}
}

func (m *mockStore) GetUserByEmail(_ context.Context, email string) (*storage.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, sqlErrNoRows()
}

func (m *mockStore) GetUserByID(_ context.Context, id int64) (*storage.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, sqlErrNoRows()
}

func (m *mockStore) UpdateUserPassword(_ context.Context, id int64, hash string) error {
	if u, ok := m.users[id]; ok {
		u.PasswordHash = hash
		return nil
	}
	return sqlErrNoRows()
}

func (m *mockStore) UpdateUserProfile(_ context.Context, id int64, email string) error {
	if u, ok := m.users[id]; ok {
		u.Email = email
		return nil
	}
	return sqlErrNoRows()
}

func (m *mockStore) GetHostsByUser(_ context.Context, userID int64) ([]storage.Host, error) {
	var hosts []storage.Host
	for _, h := range m.hosts {
		if h.UserID == userID {
			hosts = append(hosts, *h)
		}
	}
	return hosts, nil
}

func (m *mockStore) GetHostByID(_ context.Context, id int64) (*storage.Host, error) {
	if h, ok := m.hosts[id]; ok {
		return h, nil
	}
	return nil, sqlErrNoRows()
}

func (m *mockStore) CreateHost(_ context.Context, userID int64, host models.Host, _ int, _ bool) (int64, error) {
	id := m.nextID
	m.nextID++
	m.hosts[id] = &storage.Host{
		ID:          id,
		UserID:      userID,
		Name:        host.Name,
		Method:      host.Method,
		URL:         host.URL,
		Body:        host.Body,
		TimeoutSec:  host.Timeout,
		IntervalSec: host.Interval,
	}
	return id, nil
}

func (m *mockStore) UpdateHost(_ context.Context, id int64, host models.Host, _ int, _ bool) error {
	if h, ok := m.hosts[id]; ok {
		h.Name = host.Name
		h.Method = host.Method
		h.URL = host.URL
		h.Body = host.Body
		h.TimeoutSec = host.Timeout
		h.IntervalSec = host.Interval
		return nil
	}
	return sqlErrNoRows()
}

func (m *mockStore) DeleteHost(_ context.Context, id int64) error {
	delete(m.hosts, id)
	return nil
}

func (m *mockStore) DashboardData(_ context.Context, _ int64) ([]storage.DashboardHost, error) {
	return nil, nil
}

func (m *mockStore) HostTimeSeries(_ context.Context, _ int64, _ string) ([]storage.TimePoint, error) {
	return nil, nil
}

func (m *mockStore) HostHourlySeries(_ context.Context, _ int64, _ string) ([]storage.HostHourlyAvg, error) {
	return nil, nil
}

func sqlErrNoRows() error {
	return &noRowsError{}
}

type noRowsError struct{}

func (e *noRowsError) Error() string { return "sql: no rows in result set" }

func setupTestHandler(t *testing.T) (*Handler, *mockStore) {
	t.Helper()
	InitSessionStore()
	store := newMockStore()
	cfg := &config.Config{
		Server: config.ServerConfig{Address: ":8080"},
	}
	h := New(cfg, "", store)
	return h, store
}

func loginRequest(userID int64) (*http.Request, *http.Cookie) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	SetUserID(rec, req, userID)
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionName {
			cookie = c
			break
		}
	}
	return req, cookie
}

func TestHome(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.Home(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Sign in") {
		t.Fatalf("expected sign-in link in body")
	}
}

func TestLoginForm(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	h.LoginForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Sign in") {
		t.Fatalf("expected login form")
	}
}

func TestLoginDefaultUser(t *testing.T) {
	h, store := setupTestHandler(t)
	store.users[1] = &storage.User{ID: 1, Email: "default@signal.local", Role: "admin"}

	form := url.Values{}
	form.Set("email", "default@signal.local")
	form.Set("password", "anything")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/profile" {
		t.Fatalf("expected redirect to /profile, got %s", loc)
	}
}

func TestLoginWithPassword(t *testing.T) {
	h, store := setupTestHandler(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	store.users[1] = &storage.User{ID: 1, Email: "user@example.com", PasswordHash: string(hash), Role: "admin"}

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "secret")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %s", loc)
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	h, store := setupTestHandler(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	store.users[1] = &storage.User{ID: 1, Email: "user@example.com", PasswordHash: string(hash), Role: "admin"}

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "wrong")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/login" {
		t.Fatalf("expected redirect to /login, got %s", loc)
	}
}

func TestRequireAuth(t *testing.T) {
	h, store := setupTestHandler(t)
	store.users[1] = &storage.User{ID: 1, Email: "user@example.com", Role: "admin"}

	handler := RequireAuth(store, http.HandlerFunc(h.Dashboard))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect for anonymous user, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	_, cookie := loginRequest(1)
	if cookie != nil {
		req2.AddCookie(cookie)
	}
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 for authenticated user, got %d", rec2.Code)
	}
}

func TestLogout(t *testing.T) {
	h, _ := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/" {
		t.Fatalf("expected redirect to /, got %s", loc)
	}
}

func TestSaveProfile(t *testing.T) {
	h, store := setupTestHandler(t)
	store.users[1] = &storage.User{ID: 1, Email: "user@example.com", Role: "admin"}

	form := url.Values{}
	form.Set("email", "new@example.com")
	form.Set("new_password", "secret")
	form.Set("confirm_password", "secret")

	req := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(context.WithValue(req.Context(), userContextKey, store.users[1]))
	rec := httptest.NewRecorder()
	h.SaveProfile(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if store.users[1].Email != "new@example.com" {
		t.Fatalf("expected email to be updated")
	}
	if store.users[1].PasswordHash == "" {
		t.Fatalf("expected password to be set")
	}
}
