package leave_request

import (
	"github.com/go-chi/chi/v5"

	"leavemang/internal/slices/authentication"
)

// RegisterRoutes registers leave request endpoints on the chi Router.
func RegisterRoutes(r chi.Router, handler *Handler, middleware *authentication.Middleware) {
	// Employee routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate)

		r.Get("/my/leave-requests", handler.HandleListEmployeeRequests)
		r.Get("/my/leave-requests/new", handler.HandleNewRequestForm)
		r.Post("/my/leave-requests", handler.HandleCreateRequest)
		r.Get("/my/leave-requests/{id}", handler.HandleRequestDetails)
	})

	// Manager routes (AR-01, BR-01)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate)
		r.Use(middleware.RequireRole(authentication.RoleManager, authentication.RoleAdmin))

		r.Get("/manager/leave-requests", handler.HandleListPendingRequests)
		r.Get("/manager/leave-requests/{id}", handler.HandleManagerRequestDetails)
		r.Post("/manager/leave-requests/{id}/approve", handler.HandleApproveRequest)
		r.Post("/manager/leave-requests/{id}/reject", handler.HandleRejectRequest)
	})
}
