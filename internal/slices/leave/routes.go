package leave

import (
	"github.com/go-chi/chi/v5"

	"leavemang/internal/slices/authentication"
)

// RegisterRoutes registers HTTP routes for Leave Type management and Leave Balances.
func RegisterRoutes(r chi.Router, h *Handler, m *authentication.Middleware) {
	r.Group(func(r chi.Router) {
		r.Use(m.Authenticate)

		// Employee route - View own leave balances
		r.Get("/my/leave-balances", h.HandleMyLeaveBalances)

		// Admin-only leave types and balance allocation routes
		r.With(m.RequireRole(authentication.RoleAdmin)).Group(func(r chi.Router) {
			// Leave Types Management
			r.Get("/leave-types", h.HandleListLeaveTypes)
			r.Get("/leave-types/new", h.HandleNewLeaveTypeForm)
			r.Post("/leave-types", h.HandleCreateLeaveType)
			r.Get("/leave-types/{id}/edit", h.HandleEditLeaveTypeForm)
			r.Post("/leave-types/{id}", h.HandleUpdateLeaveType)
			r.Post("/leave-types/{id}/activate", h.HandleActivateLeaveType)
			r.Post("/leave-types/{id}/deactivate", h.HandleDeactivateLeaveType)

			// Admin Leave Balances Overview & Allocation
			r.Get("/leave-balances", h.HandleListBalances)
			r.Get("/leave-balances/new", h.HandleNewAllocationForm)
			r.Post("/leave-balances", h.HandleCreateAllocation)
		})
	})
}
