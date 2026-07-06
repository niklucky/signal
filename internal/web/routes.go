package web

import (
	"net/http"

	"github.com/niklucky/signal/internal/config"
)

// RegisterRoutes wires all UI routes into the default mux.
func RegisterRoutes(cfg *config.Config, cfgPath string, store Store) {
	InitSessionStore()
	h := New(cfg, cfgPath, store)

	http.Handle("/static/", StaticFileServer())

	http.HandleFunc("/", h.Home)
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.Login(w, r)
			return
		}
		h.LoginForm(w, r)
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.Logout(w, r)
	})

	auth := func(next http.HandlerFunc) http.Handler {
		return RequireAuth(store, http.HandlerFunc(next))
	}

	http.Handle("/dashboard", auth(h.Dashboard))
	http.Handle("/api/dashboard", auth(h.DashboardData))
	http.Handle("/api/hosts/{id}/data", auth(h.HostChartData))
	http.Handle("/hosts", auth(h.Hosts))
	http.Handle("/hosts/new", auth(h.HostForm))
	http.Handle("/hosts/{id}", auth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.SaveHost(w, r)
		case http.MethodGet:
			h.HostForm(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	http.Handle("/hosts/{id}/delete", auth(h.DeleteHost))
	http.Handle("/messaging", auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.SaveMessaging(w, r)
			return
		}
		h.MessagingForm(w, r)
	}))
	http.Handle("/profile", auth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.SaveProfile(w, r)
			return
		}
		h.ProfileForm(w, r)
	}))
}
