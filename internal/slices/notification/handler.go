package notification

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	sharedHttp "leavemang/internal/shared/http"
	"leavemang/internal/slices/authentication"
)

type Handler struct {
	service     *Service
	layoutPath  string
	templateDir string
}

func NewHandler(service *Service, layoutPath, templateDir string) *Handler {
	return &Handler{
		service:     service,
		layoutPath:  layoutPath,
		templateDir: templateDir,
	}
}

// HandleList renders the notification history list page (AR-05, AR-07, AR-08, BR-02).
func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	notifications, err := h.service.GetUserNotifications(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load notifications: "+err.Error(), http.StatusInternalServerError)
		return
	}

	unreadCount, err := h.service.GetUnreadCount(r.Context(), user.ID)
	if err != nil {
		unreadCount = 0
	}

	pageData := map[string]interface{}{
		"Title":         "Notifications - Leave Management System",
		"User":          user,
		"Notifications": notifications,
		"UnreadCount":   unreadCount,
	}

	templatePath := fmt.Sprintf("%s/list.html", h.templateDir)
	sharedHttp.RenderHTML(w, r, http.StatusOK, pageData, h.layoutPath, templatePath)
}

// HandleMarkAsRead marks a single notification as read (AR-06, BR-03, BR-04).
func (h *Handler) HandleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}

	if err := h.service.MarkAsRead(r.Context(), id, user.ID); err != nil {
		http.Error(w, "Failed to mark notification as read: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/notifications")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

// HandleMarkAllAsRead marks all notifications as read for current user (AR-06, BR-03, BR-04).
func (h *Handler) HandleMarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := h.service.MarkAllAsRead(r.Context(), user.ID); err != nil {
		http.Error(w, "Failed to mark all notifications as read: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/notifications")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

// HandleUnreadCount returns an HTMX badge fragment with current unread count (AR-07).
func (h *Handler) HandleUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<span></span>`))
		return
	}

	count, err := h.service.GetUnreadCount(r.Context(), user.ID)
	if err != nil {
		count = 0
	}

	data := map[string]interface{}{
		"UnreadCount": count,
	}

	templatePath := fmt.Sprintf("%s/partials/notification_badge.html", h.templateDir)
	sharedHttp.RenderHTML(w, r, http.StatusOK, data, templatePath)
}
