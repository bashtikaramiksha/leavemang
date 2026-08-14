package authentication

import (
	"context"
	"net/http"
)

type contextKey string

const userContextKey contextKey = "authenticated_user"

// CurrentUser retrieves the authenticated user from the HTTP request context.
// This function satisfies the VS-01 to VS-02 vertical slice handoff requirement.
func CurrentUser(r *http.Request) *User {
	if user, ok := r.Context().Value(userContextKey).(*User); ok {
		return user
	}
	return nil
}

// Middleware handles authentication & authorization middleware logic.
type Middleware struct {
	service *Service
}

func NewMiddleware(service *Service) *Middleware {
	return &Middleware{service: service}
}

// Authenticate verifies the session cookie and populates the user into the request context.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		_, user, err := m.service.ValidateSession(cookie.Value)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:   "session_id",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole enforces role-based authorization for protected endpoints.
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := CurrentUser(r)
			if user == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			allowed := false
			for _, role := range roles {
				if user.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "403 Forbidden: You do not have permission to access this area.", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
