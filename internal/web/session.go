package web

import (
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

const sessionName = "signal-session"

var store *sessions.CookieStore

// InitSessionStore configures the cookie session store.
func InitSessionStore() {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "signal-dev-secret-change-in-production"
	}
	store = sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// GetSession returns the current session.
func GetSession(r *http.Request) *sessions.Session {
	s, _ := store.Get(r, sessionName)
	return s
}

// SaveSession persists the session.
func SaveSession(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	_ = s.Save(r, w)
}

// Flash adds a flash message.
func Flash(w http.ResponseWriter, r *http.Request, kind, message string) {
	s := GetSession(r)
	s.AddFlash(message, kind)
	SaveSession(w, r, s)
}

// GetFlashes returns and clears flash messages for a given kind.
func GetFlashes(w http.ResponseWriter, r *http.Request, kind string) []interface{} {
	s := GetSession(r)
	flashes := s.Flashes(kind)
	SaveSession(w, r, s)
	return flashes
}

// GetFlashString returns the first flash message of a kind as a string.
func GetFlashString(w http.ResponseWriter, r *http.Request, kind string) string {
	flashes := GetFlashes(w, r, kind)
	if len(flashes) > 0 {
		if s, ok := flashes[0].(string); ok {
			return s
		}
	}
	return ""
}

// SetUserID stores the authenticated user ID in the session.
func SetUserID(w http.ResponseWriter, r *http.Request, userID int64) {
	s := GetSession(r)
	s.Values["user_id"] = userID
	SaveSession(w, r, s)
}

// GetUserID returns the authenticated user ID, or 0 if not authenticated.
func GetUserID(r *http.Request) int64 {
	s := GetSession(r)
	if v, ok := s.Values["user_id"].(int64); ok {
		return v
	}
	return 0
}

// ClearSession removes the session.
func ClearSession(w http.ResponseWriter, r *http.Request) {
	s := GetSession(r)
	s.Options.MaxAge = -1
	SaveSession(w, r, s)
}
