package authentication

import (
	"fmt"
	"net/http"

	sharedHttp "leavemang/internal/shared/http"
)

type Handler struct {
	service     *Service
	templateDir string
}

func NewHandler(service *Service, templateDir string) *Handler {
	return &Handler{
		service:     service,
		templateDir: templateDir,
	}
}

// HandleLoginGet renders the login page. If already authenticated, redirects home.
func (h *Handler) HandleLoginGet(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "" {
		if _, user, err := h.service.ValidateSession(cookie.Value); err == nil && user != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	layoutPath := fmt.Sprintf("%s/layout.html", h.templateDir)
	loginPath := fmt.Sprintf("%s/login.html", h.templateDir)

	data := map[string]interface{}{
		"Title": "Login - Leave Management System",
		"Error": "",
	}
	sharedHttp.RenderHTML(w, r, http.StatusOK, data, layoutPath, loginPath)
}

// HandleLoginPost processes user login credentials.
func (h *Handler) HandleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderLoginError(w, r, "Invalid form data submitted")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.service.AuthenticateUser(username, password)
	if err != nil {
		h.renderLoginError(w, r, err.Error())
		return
	}

	session, err := h.service.CreateSession(user.ID)
	if err != nil {
		h.renderLoginError(w, r, "Server error creating session")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// renderLoginError helper handles HTMX and standard form error responses.
func (h *Handler) renderLoginError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	if sharedHttp.IsHTMX(r) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `<div id="error-message" class="error-badge alert alert-danger">%s</div>`, errorMsg)
		return
	}

	layoutPath := fmt.Sprintf("%s/layout.html", h.templateDir)
	loginPath := fmt.Sprintf("%s/login.html", h.templateDir)

	data := map[string]interface{}{
		"Title": "Login - Leave Management System",
		"Error": errorMsg,
	}
	sharedHttp.RenderHTML(w, r, http.StatusUnauthorized, data, layoutPath, loginPath)
}

// HandleLogoutPost terminates the current user session.
func (h *Handler) HandleLogoutPost(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleHomeGet displays the authenticated home page with user role information.
func (h *Handler) HandleHomeGet(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)

	layoutPath := fmt.Sprintf("%s/layout.html", h.templateDir)
	homePath := fmt.Sprintf("%s/home.html", h.templateDir)

	data := map[string]interface{}{
		"Title": "Home - Leave Management System",
		"User":  user,
	}
	sharedHttp.RenderHTML(w, r, http.StatusOK, data, layoutPath, homePath)
}

// HandleEmployeeGet displays the protected employee area stub.
func (h *Handler) HandleEmployeeGet(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)

	layoutPath := fmt.Sprintf("%s/layout.html", h.templateDir)
	empPath := fmt.Sprintf("%s/employee.html", h.templateDir)

	data := map[string]interface{}{
		"Title": "Employee Portal",
		"User":  user,
	}
	sharedHttp.RenderHTML(w, r, http.StatusOK, data, layoutPath, empPath)
}

// HandleManagerGet displays the protected manager area stub.
func (h *Handler) HandleManagerGet(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)

	layoutPath := fmt.Sprintf("%s/layout.html", h.templateDir)
	mgrPath := fmt.Sprintf("%s/manager.html", h.templateDir)

	data := map[string]interface{}{
		"Title": "Manager Portal",
		"User":  user,
	}
	sharedHttp.RenderHTML(w, r, http.StatusOK, data, layoutPath, mgrPath)
}

// HandleAdminGet displays the protected admin area stub.
func (h *Handler) HandleAdminGet(w http.ResponseWriter, r *http.Request) {
	user := CurrentUser(r)

	layoutPath := fmt.Sprintf("%s/layout.html", h.templateDir)
	adminPath := fmt.Sprintf("%s/admin.html", h.templateDir)

	data := map[string]interface{}{
		"Title": "Admin Portal",
		"User":  user,
	}
	sharedHttp.RenderHTML(w, r, http.StatusOK, data, layoutPath, adminPath)
}
