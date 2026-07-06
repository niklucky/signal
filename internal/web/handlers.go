package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/niklucky/signal/internal/config"
	"github.com/niklucky/signal/internal/models"
	"github.com/niklucky/signal/internal/storage"
	"github.com/niklucky/signal/web/templates/pages"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// Handler holds dependencies for UI routes.
type Handler struct {
	cfg     *config.Config
	cfgPath string
	store   Store
}

// New creates a new UI handler.
func New(cfg *config.Config, cfgPath string, store Store) *Handler {
	return &Handler{
		cfg:     cfg,
		cfgPath: cfgPath,
		store:   store,
	}
}

func render(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.WriteHeader(status)
	_ = c.Render(r.Context(), w)
}

// Home shows the public landing page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, pages.Home())
}

// LoginForm shows the sign-in page.
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	errMsg := GetFlashString(w, r, "error")
	render(w, r, http.StatusOK, pages.Login(errMsg))
}

// Login handles authentication.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		Flash(w, r, "error", "invalid form")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	if h.store == nil {
		Flash(w, r, "error", "database not configured")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		Flash(w, r, "error", "invalid email or password")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// First login for the default user with empty password hash: allow any password.
	if user.PasswordHash == "" {
		SetUserID(w, r, user.ID)
		Flash(w, r, "info", "Please set a password for your account.")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		Flash(w, r, "error", "invalid email or password")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	SetUserID(w, r, user.ID)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Logout clears the session.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ClearSession(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Dashboard shows the main dashboard page.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	info := GetFlashString(w, r, "info")
	success := GetFlashString(w, r, "success")

	var hosts []storage.Host
	if h.store != nil {
		var err error
		hosts, err = h.store.GetHostsByUser(r.Context(), user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	render(w, r, http.StatusOK, pages.Dashboard(user.Email, info, success, hosts))
}

// DashboardData returns JSON data for the dashboard table.
func (h *Handler) DashboardData(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	if h.store == nil {
		http.Error(w, "database not configured", http.StatusInternalServerError)
		return
	}

	hosts, err := h.store.GetHostsByUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type series struct {
		HostID      int64                  `json:"host_id"`
		IntervalSec int                    `json:"interval_sec"`
		Hours       []storage.HostHourlyAvg `json:"hours"`
	}

	result := make([]series, 0, len(hosts))
	for _, host := range hosts {
		hours, err := h.store.HostHourlySeries(r.Context(), host.ID, "30 days")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result = append(result, series{
			HostID:      host.ID,
			IntervalSec: host.IntervalSec,
			Hours:       hours,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HostChartData returns JSON time-series data for a single host chart.
func (h *Handler) HostChartData(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	if h.store == nil {
		http.Error(w, "database not configured", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	host, err := h.store.GetHostByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if host.UserID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	points, err := h.store.HostTimeSeries(r.Context(), id, "24 hours")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"host_id": id,
		"points":  points,
	})
}

// Hosts shows the host list.
func (h *Handler) Hosts(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	if h.store == nil {
		http.Error(w, "database not configured", http.StatusInternalServerError)
		return
	}
	hosts, err := h.store.GetHostsByUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	msg := GetFlashString(w, r, "success")
	render(w, r, http.StatusOK, pages.Hosts(user.Email, hosts, msg))
}

// HostForm shows the new/edit host form.
func (h *Handler) HostForm(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	idStr := r.PathValue("id")
	var host *storage.Host
	var err error
	if idStr != "" && idStr != "new" {
		id, parseErr := strconv.ParseInt(idStr, 10, 64)
		if parseErr != nil {
			http.NotFound(w, r)
			return
		}
		host, err = h.store.GetHostByID(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if host.UserID != user.ID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	errMsg := GetFlashString(w, r, "error")
	render(w, r, http.StatusOK, pages.HostForm(user.Email, host, errMsg))
}

// SaveHost creates or updates a host.
func (h *Handler) SaveHost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		Flash(w, r, "error", "invalid form")
		http.Redirect(w, r, "/hosts/new", http.StatusSeeOther)
		return
	}
	user := CurrentUser(r)

	host := models.Host{
		Name:           r.FormValue("name"),
		Method:         r.FormValue("method"),
		URL:            r.FormValue("url"),
		Body:           r.FormValue("body"),
		Timeout:        parseInt(r.FormValue("timeout"), 10),
		Interval:       parseInt(r.FormValue("interval"), 60),
		ResendInterval: parseInt(r.FormValue("resend_interval"), 0),
	}
	if host.Method == "" {
		host.Method = "GET"
	}

	idStr := r.PathValue("id")
	if idStr == "new" || idStr == "" {
		_, err := h.store.CreateHost(r.Context(), user.ID, host, http.StatusOK, r.FormValue("active") == "on")
		if err != nil {
			Flash(w, r, "error", "failed to create host: "+err.Error())
			http.Redirect(w, r, "/hosts/new", http.StatusSeeOther)
			return
		}
		Flash(w, r, "success", "Host created")
		http.Redirect(w, r, "/hosts", http.StatusSeeOther)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	existing, err := h.store.GetHostByID(r.Context(), id)
	if err != nil || existing.UserID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.store.UpdateHost(r.Context(), id, host, http.StatusOK, r.FormValue("active") == "on"); err != nil {
		Flash(w, r, "error", "failed to update host: "+err.Error())
		http.Redirect(w, r, fmt.Sprintf("/hosts/%d", id), http.StatusSeeOther)
		return
	}
	Flash(w, r, "success", "Host updated")
	http.Redirect(w, r, "/hosts", http.StatusSeeOther)
}

// DeleteHost removes a host.
func (h *Handler) DeleteHost(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	existing, err := h.store.GetHostByID(r.Context(), id)
	if err != nil || existing.UserID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.store.DeleteHost(r.Context(), id); err != nil {
		Flash(w, r, "error", "failed to delete host: "+err.Error())
	} else {
		Flash(w, r, "success", "Host deleted")
	}
	http.Redirect(w, r, "/hosts", http.StatusSeeOther)
}

// MessagingForm shows the messaging config page.
func (h *Handler) MessagingForm(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	success := GetFlashString(w, r, "success")
	info := GetFlashString(w, r, "info")
	render(w, r, http.StatusOK, pages.Messaging(user.Email, h.cfg, success, info))
}

// SaveMessaging writes messaging config back to the config file.
func (h *Handler) SaveMessaging(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		Flash(w, r, "error", "invalid form")
		http.Redirect(w, r, "/messaging", http.StatusSeeOther)
		return
	}
	h.cfg.Telegram.Enabled = r.FormValue("telegram_enabled") == "on"
	h.cfg.Telegram.BotToken = r.FormValue("telegram_bot_token")
	h.cfg.Telegram.ChatID = r.FormValue("telegram_chat_id")
	h.cfg.Telegram.ProxyURL = r.FormValue("telegram_proxy_url")
	h.cfg.Matrix.Enabled = r.FormValue("matrix_enabled") == "on"
	h.cfg.Matrix.Homeserver = r.FormValue("matrix_homeserver")
	h.cfg.Matrix.UserID = r.FormValue("matrix_user_id")
	h.cfg.Matrix.AccessToken = r.FormValue("matrix_access_token")
	h.cfg.Matrix.RoomID = r.FormValue("matrix_room_id")

	data, err := yaml.Marshal(h.cfg)
	if err != nil {
		Flash(w, r, "error", "failed to marshal config: "+err.Error())
		http.Redirect(w, r, "/messaging", http.StatusSeeOther)
		return
	}
	if err := os.WriteFile(h.cfgPath, data, 0644); err != nil {
		Flash(w, r, "error", "failed to write config: "+err.Error())
		http.Redirect(w, r, "/messaging", http.StatusSeeOther)
		return
	}
	Flash(w, r, "success", "Config saved. Restart the server to apply changes.")
	http.Redirect(w, r, "/messaging", http.StatusSeeOther)
}

// ProfileForm shows the user profile page.
func (h *Handler) ProfileForm(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)
	success := GetFlashString(w, r, "success")
	info := GetFlashString(w, r, "info")
	errMsg := GetFlashString(w, r, "error")
	render(w, r, http.StatusOK, pages.Profile(user, success, info, errMsg))
}

// SaveProfile updates the user profile and password.
func (h *Handler) SaveProfile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		Flash(w, r, "error", "invalid form")
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	user := CurrentUser(r)
	email := r.FormValue("email")
	current := r.FormValue("current_password")
	newPass := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if email != "" && email != user.Email {
		if err := h.store.UpdateUserProfile(r.Context(), user.ID, email); err != nil {
			Flash(w, r, "error", "failed to update email: "+err.Error())
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
	}

	if newPass != "" {
		if user.PasswordHash != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)); err != nil {
				Flash(w, r, "error", "current password is incorrect")
				http.Redirect(w, r, "/profile", http.StatusSeeOther)
				return
			}
		}
		if newPass != confirm {
			Flash(w, r, "error", "new passwords do not match")
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
		if err != nil {
			Flash(w, r, "error", "failed to hash password: "+err.Error())
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
		if err := h.store.UpdateUserPassword(r.Context(), user.ID, string(hash)); err != nil {
			Flash(w, r, "error", "failed to update password: "+err.Error())
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
		Flash(w, r, "success", "Password updated")
	} else {
		Flash(w, r, "success", "Profile updated")
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// StaticFileServer returns a handler for static assets.
func StaticFileServer() http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.Dir("web/static")))
}

// init sets the default time zone for display.
func init() {
	time.LoadLocation("UTC")
}
