package employee

import (
	"github.com/go-chi/chi/v5"

	"leavemang/internal/slices/authentication"
)

// RegisterRoutes registers all HTTP endpoints for the Employee Management slice.
func RegisterRoutes(r chi.Router, h *Handler, m *authentication.Middleware) {
	r.Group(func(r chi.Router) {
		r.Use(m.Authenticate)

		// Self-profile route (All authenticated users)
		r.Get("/profile", h.HandleViewProfile)

		// Directory and details routes
		r.Get("/employees", h.HandleListEmployees)
		r.Get("/employees/{id}", h.HandleViewEmployee)

		// Admin-only employee management routes
		r.With(m.RequireRole(authentication.RoleAdmin)).Group(func(r chi.Router) {
			r.Get("/employees/new", h.HandleNewEmployeeForm)
			r.Post("/employees", h.HandleCreateEmployee)
			r.Get("/employees/{id}/edit", h.HandleEditEmployeeForm)
			r.Post("/employees/{id}", h.HandleUpdateEmployee)
			r.Post("/employees/{id}/activate", h.HandleActivateEmployee)
			r.Post("/employees/{id}/deactivate", h.HandleDeactivateEmployee)
		})
	})
}
