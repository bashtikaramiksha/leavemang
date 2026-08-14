package authentication

import "time"

// Role constants
const (
	RoleEmployee = "employee"
	RoleManager  = "manager"
	RoleAdmin    = "admin"
)

// Status constants
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// User represents an authenticated identity in the system.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Session represents an active authenticated browser session.
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
