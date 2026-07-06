package web

import (
	"context"
	"net/http"

	"github.com/niklucky/signal/internal/storage"
)

type contextKey string

const userContextKey contextKey = "user"

// RequireAuth redirects anonymous users to /login and loads the user for authenticated requests.
func RequireAuth(store Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r)
		if userID == 0 {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if store == nil {
			http.Error(w, "database not configured", http.StatusInternalServerError)
			return
		}
		user, err := store.GetUserByID(r.Context(), userID)
		if err != nil {
			ClearSession(w, r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentUser returns the user stored in the request context.
func CurrentUser(r *http.Request) *storage.User {
	if u, ok := r.Context().Value(userContextKey).(*storage.User); ok {
		return u
	}
	return nil
}
