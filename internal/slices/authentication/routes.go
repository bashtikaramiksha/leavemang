package authentication

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes sets up all authentication and role-protected routes.
func RegisterRoutes(r chi.Router, h *Handler, m *Middleware) {
	// Public routes
	r.Get("/login", h.HandleLoginGet)
	r.Post("/login", h.HandleLoginPost)

	// Protected routes (Requires valid session)
	r.Group(func(r chi.Router) {
		r.Use(m.Authenticate)

		r.Post("/logout", h.HandleLogoutPost)
		r.Get("/", h.HandleHomeGet)

		// Role-restricted routes
		r.With(m.RequireRole(RoleEmployee)).Get("/employee", h.HandleEmployeeGet)
		r.With(m.RequireRole(RoleManager)).Get("/manager", h.HandleManagerGet)
		r.With(m.RequireRole(RoleAdmin)).Get("/admin", h.HandleAdminGet)
	})
}
