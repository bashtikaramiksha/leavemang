package employee

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new User and corresponding Employee record within a single database transaction.
func (r *Repository) Create(ctx context.Context, input CreateEmployeeInput) (*Employee, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if email already exists
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM employees WHERE email = ?", input.Email).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}
	if count > 0 {
		return nil, ErrDuplicateEmail
	}

	// Default initial password for newly created employee user accounts
	defaultPassword := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create linked User account with username = email prefix or email
	username := strings.Split(input.Email, "@")[0]
	// Ensure username uniqueness
	var userCount int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE username = ?", username).Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if userCount > 0 {
		username = input.Email // Fallback to full email if username prefix exists
	}

	res, err := tx.ExecContext(
		ctx,
		"INSERT INTO users (username, password_hash, role, status) VALUES (?, ?, ?, ?)",
		username, string(hash), input.Role, StatusActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user account: %w", err)
	}

	userID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Generate sequential employee code, e.g., EMP-005
	var maxID int
	_ = tx.QueryRowContext(ctx, "SELECT IFNULL(MAX(id), 0) FROM employees").Scan(&maxID)
	empCode := fmt.Sprintf("EMP-%03d", maxID+1)

	empRes, err := tx.ExecContext(
		ctx,
		`INSERT INTO employees (user_id, employee_code, first_name, last_name, email, phone, department, designation, joining_date, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, empCode, input.FirstName, input.LastName, input.Email, input.Phone, input.Department, input.Designation, input.JoiningDate, StatusActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create employee record: %w", err)
	}

	empID, err := empRes.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get employee ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(ctx, empID)
}

// GetByID retrieves an employee record by employee ID including linked user role.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Employee, error) {
	query := `
		SELECT e.id, e.user_id, e.employee_code, e.first_name, e.last_name, e.email, e.phone,
		       e.department, e.designation, e.joining_date, e.status, u.role, e.created_at, e.updated_at
		FROM employees e
		JOIN users u ON e.user_id = u.id
		WHERE e.id = ?`

	emp := &Employee{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName,
		&emp.Email, &emp.Phone, &emp.Department, &emp.Designation, &emp.JoiningDate,
		&emp.Status, &emp.Role, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to get employee by ID: %w", err)
	}

	return emp, nil
}

// GetByUserID retrieves an employee record by linked user ID.
func (r *Repository) GetByUserID(ctx context.Context, userID int64) (*Employee, error) {
	query := `
		SELECT e.id, e.user_id, e.employee_code, e.first_name, e.last_name, e.email, e.phone,
		       e.department, e.designation, e.joining_date, e.status, u.role, e.created_at, e.updated_at
		FROM employees e
		JOIN users u ON e.user_id = u.id
		WHERE e.user_id = ?`

	emp := &Employee{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName,
		&emp.Email, &emp.Phone, &emp.Department, &emp.Designation, &emp.JoiningDate,
		&emp.Status, &emp.Role, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to get employee by user ID: %w", err)
	}

	return emp, nil
}

// GetByEmail retrieves an employee record by email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*Employee, error) {
	query := `
		SELECT e.id, e.user_id, e.employee_code, e.first_name, e.last_name, e.email, e.phone,
		       e.department, e.designation, e.joining_date, e.status, u.role, e.created_at, e.updated_at
		FROM employees e
		JOIN users u ON e.user_id = u.id
		WHERE LOWER(e.email) = ?`

	emp := &Employee{}
	err := r.db.QueryRowContext(ctx, query, strings.ToLower(email)).Scan(
		&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName,
		&emp.Email, &emp.Phone, &emp.Department, &emp.Designation, &emp.JoiningDate,
		&emp.Status, &emp.Role, &emp.CreatedAt, &emp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to get employee by email: %w", err)
	}

	return emp, nil
}

// List returns all employee records joined with their user roles.
func (r *Repository) List(ctx context.Context) ([]*Employee, error) {
	query := `
		SELECT e.id, e.user_id, e.employee_code, e.first_name, e.last_name, e.email, e.phone,
		       e.department, e.designation, e.joining_date, e.status, u.role, e.created_at, e.updated_at
		FROM employees e
		JOIN users u ON e.user_id = u.id
		ORDER BY e.id ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}
	defer rows.Close()

	var employees []*Employee
	for rows.Next() {
		emp := &Employee{}
		if err := rows.Scan(
			&emp.ID, &emp.UserID, &emp.EmployeeCode, &emp.FirstName, &emp.LastName,
			&emp.Email, &emp.Phone, &emp.Department, &emp.Designation, &emp.JoiningDate,
			&emp.Status, &emp.Role, &emp.CreatedAt, &emp.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan employee row: %w", err)
		}
		employees = append(employees, emp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating employee rows: %w", err)
	}

	return employees, nil
}

// Update updates an employee's details and updates the linked User account role & status.
func (r *Repository) Update(ctx context.Context, id int64, input UpdateEmployeeInput) (*Employee, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check existing employee and email collision
	var existingID int64
	var userID int64
	err = tx.QueryRowContext(ctx, "SELECT id, user_id FROM employees WHERE id = ?", id).Scan(&existingID, &userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to find employee: %w", err)
	}

	// Check if email belongs to another employee
	var count int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM employees WHERE LOWER(email) = ? AND id != ?", input.Email, id).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate email: %w", err)
	}
	if count > 0 {
		return nil, ErrDuplicateEmail
	}

	// Update employees table
	_, err = tx.ExecContext(
		ctx,
		`UPDATE employees
		 SET first_name = ?, last_name = ?, email = ?, phone = ?, department = ?, designation = ?, joining_date = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		input.FirstName, input.LastName, input.Email, input.Phone, input.Department, input.Designation, input.JoiningDate, input.Status, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update employee record: %w", err)
	}

	// Update linked users table
	_, err = tx.ExecContext(
		ctx,
		`UPDATE users
		 SET role = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		input.Role, input.Status, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update user record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return r.GetByID(ctx, id)
}

// SetStatus updates the status (active/inactive) for both the employee and their user account.
func (r *Repository) SetStatus(ctx context.Context, id int64, status string) error {
	if status != StatusActive && status != StatusInactive {
		return ErrInvalidStatus
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var userID int64
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM employees WHERE id = ?", id).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrEmployeeNotFound
		}
		return fmt.Errorf("failed to get user ID for employee: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE employees SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", status, id)
	if err != nil {
		return fmt.Errorf("failed to update employee status: %w", err)
	}

	_, err = tx.ExecContext(ctx, "UPDATE users SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", status, userID)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return tx.Commit()
}
