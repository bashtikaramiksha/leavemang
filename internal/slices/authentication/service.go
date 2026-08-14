package authentication

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInactiveUser       = errors.New("account is inactive")
	ErrSessionExpired     = errors.New("session has expired or is invalid")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// AuthenticateUser verifies user credentials and active status.
func (s *Service) AuthenticateUser(username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.Status != StatusActive {
		return nil, ErrInactiveUser
	}

	return user, nil
}

// CreateSession generates a secure session token and saves it.
func (s *Service) CreateSession(userID int64) (*Session, error) {
	token, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	session := &Session{
		ID:        token,
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

// ValidateSession verifies if a session token is valid and unexpired, returning the associated User.
func (s *Service) ValidateSession(sessionID string) (*Session, *User, error) {
	if sessionID == "" {
		return nil, nil, ErrSessionExpired
	}

	session, err := s.repo.GetSessionByID(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, nil, ErrSessionExpired
		}
		return nil, nil, err
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.repo.DeleteSession(sessionID)
		return nil, nil, ErrSessionExpired
	}

	user, err := s.repo.GetUserByID(session.UserID)
	if err != nil {
		return nil, nil, err
	}

	if user.Status != StatusActive {
		return nil, nil, ErrInactiveUser
	}

	return session, user, nil
}

// Logout revokes the specified session.
func (s *Service) Logout(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.repo.DeleteSession(sessionID)
}

func generateRandomToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
