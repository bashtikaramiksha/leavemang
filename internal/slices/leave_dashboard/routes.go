package leave_dashboard

import (
	"github.com/go-chi/chi/v5"

	"leavemang/internal/slices/authentication"
)

// RegisterRoutes registers leave dashboard endpoints on the chi Router.
func RegisterRoutes(r chi.Router, handler *Handler, middleware *authentication.Middleware) {
	// Employee routes (AR-01, AR-02, AR-03, BR-01)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate)

		r.Get("/my/dashboard", handler.HandleEmployeeDashboard)
		r.Get("/my/leave-balances", handler.HandleEmployeeBalances)
	})

	// Manager routes (AR-07 to AR-10, BR-02)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Use(middleware.RequireRole(authentication.RoleManager, authentication.RoleAdmin))

		r.Get("/manager/dashboard", handler.HandleManagerDashboard)
	})
}
