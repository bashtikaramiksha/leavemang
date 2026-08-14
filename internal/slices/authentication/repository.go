package authentication

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, password_hash, role, status, created_at, updated_at FROM users WHERE username = ?`
	row := r.db.QueryRow(query, username)

	var u User
	var createdAtStr, updatedAtStr string
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Status, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to fetch user by username: %w", err)
	}

	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	return &u, nil
}

func (r *Repository) GetUserByID(id int64) (*User, error) {
	query := `SELECT id, username, password_hash, role, status, created_at, updated_at FROM users WHERE id = ?`
	row := r.db.QueryRow(query, id)

	var u User
	var createdAtStr, updatedAtStr string
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Status, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to fetch user by id: %w", err)
	}

	u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	return &u, nil
}

func (r *Repository) CreateSession(s *Session) error {
	query := `INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`
	_, err := r.db.Exec(query, s.ID, s.UserID, s.ExpiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

func (r *Repository) GetSessionByID(sessionID string) (*Session, error) {
	query := `SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?`
	row := r.db.QueryRow(query, sessionID)

	var s Session
	var expiresAtStr, createdAtStr string
	err := row.Scan(&s.ID, &s.UserID, &expiresAtStr, &createdAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to fetch session: %w", err)
	}

	s.ExpiresAt = parseTime(expiresAtStr)
	s.CreatedAt = parseTime(createdAtStr)
	return &s, nil
}

func parseTime(str string) time.Time {
	if t, err := time.Parse(time.RFC3339, str); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", str); err == nil {
		return t
	}
	return time.Time{}
}

func (r *Repository) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}
