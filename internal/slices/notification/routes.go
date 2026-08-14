package notification

import (
	"github.com/go-chi/chi/v5"

	"leavemang/internal/slices/authentication"
)

// RegisterRoutes registers notification endpoints on the chi Router.
func RegisterRoutes(r chi.Router, handler *Handler, middleware *authentication.Middleware) {
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate)

		r.Get("/notifications", handler.HandleList)
		r.Post("/notifications/{id}/read", handler.HandleMarkAsRead)
		r.Post("/notifications/read-all", handler.HandleMarkAllAsRead)
		r.Get("/notifications/unread-count", handler.HandleUnreadCount)
	})
}
