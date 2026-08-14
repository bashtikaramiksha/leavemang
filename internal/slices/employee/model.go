package employee

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Status constants
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

// Role constants
const (
	RoleEmployee = "employee"
	RoleManager  = "manager"
	RoleAdmin    = "admin"
)

var (
	ErrEmployeeNotFound = errors.New("employee not found")
	ErrDuplicateEmail   = errors.New("employee email address already exists")
	ErrRequiredFields   = errors.New("missing required fields")
	ErrInvalidEmail     = errors.New("invalid email address format")
	ErrInvalidRole      = errors.New("invalid role specified")
	ErrInvalidStatus    = errors.New("invalid status specified")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Employee represents an employee record in the organization directory.
type Employee struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	EmployeeCode string    `json:"employee_code"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Department   string    `json:"department"`
	Designation  string    `json:"designation"`
	JoiningDate  string    `json:"joining_date"`
	Status       string    `json:"status"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FullName returns the combined first and last name.
func (e *Employee) FullName() string {
	return strings.TrimSpace(e.FirstName + " " + e.LastName)
}

// CreateEmployeeInput contains fields for creating a new employee.
type CreateEmployeeInput struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Department  string `json:"department"`
	Designation string `json:"designation"`
	JoiningDate string `json:"joining_date"`
	Role        string `json:"role"`
}

// Validate checks required fields and formats for CreateEmployeeInput.
func (in *CreateEmployeeInput) Validate() error {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.Department = strings.TrimSpace(in.Department)
	in.Designation = strings.TrimSpace(in.Designation)
	in.JoiningDate = strings.TrimSpace(in.JoiningDate)
	in.Role = strings.ToLower(strings.TrimSpace(in.Role))

	if in.FirstName == "" || in.LastName == "" || in.Email == "" || in.JoiningDate == "" {
		return ErrRequiredFields
	}

	if in.Role == "" {
		in.Role = RoleEmployee
	}

	if in.Role != RoleEmployee && in.Role != RoleManager && in.Role != RoleAdmin {
		return ErrInvalidRole
	}

	if !emailRegex.MatchString(in.Email) {
		return ErrInvalidEmail
	}

	return nil
}

// UpdateEmployeeInput contains fields for updating an existing employee.
type UpdateEmployeeInput struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Department  string `json:"department"`
	Designation string `json:"designation"`
	JoiningDate string `json:"joining_date"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

// Validate checks required fields and formats for UpdateEmployeeInput.
func (in *UpdateEmployeeInput) Validate() error {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.Department = strings.TrimSpace(in.Department)
	in.Designation = strings.TrimSpace(in.Designation)
	in.JoiningDate = strings.TrimSpace(in.JoiningDate)
	in.Role = strings.ToLower(strings.TrimSpace(in.Role))
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))

	if in.FirstName == "" || in.LastName == "" || in.Email == "" || in.JoiningDate == "" {
		return ErrRequiredFields
	}

	if in.Role != RoleEmployee && in.Role != RoleManager && in.Role != RoleAdmin {
		return ErrInvalidRole
	}

	if in.Status != StatusActive && in.Status != StatusInactive {
		return ErrInvalidStatus
	}

	if !emailRegex.MatchString(in.Email) {
		return ErrInvalidEmail
	}

	return nil
}
