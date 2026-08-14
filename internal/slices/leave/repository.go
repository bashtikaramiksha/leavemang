package leave

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Repository handles persistence operations for LeaveType and LeaveBalance entities.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new instance of Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateLeaveType inserts a new LeaveType into the database.
func (r *Repository) CreateLeaveType(ctx context.Context, lt *LeaveType) error {
	query := `
		INSERT INTO leave_types (code, name, description, default_allocation, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	res, err := r.db.ExecContext(ctx, query, lt.Code, lt.Name, lt.Description, lt.DefaultAllocation, lt.Status)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "UNIQUE") {
			return ErrDuplicateCode
		}
		return fmt.Errorf("failed to create leave type: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to retrieve last insert id: %w", err)
	}
	lt.ID = id
	return nil
}

// GetLeaveTypeByID retrieves a LeaveType by its primary key ID.
func (r *Repository) GetLeaveTypeByID(ctx context.Context, id int64) (*LeaveType, error) {
	query := `
		SELECT id, code, name, description, default_allocation, status, created_at, updated_at
		FROM leave_types
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var lt LeaveType
	var createdAtStr, updatedAtStr string
	err := row.Scan(&lt.ID, &lt.Code, &lt.Name, &lt.Description, &lt.DefaultAllocation, &lt.Status, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLeaveTypeNotFound
		}
		return nil, fmt.Errorf("failed to query leave type by id: %w", err)
	}

	lt.CreatedAt, _ = parseTime(createdAtStr)
	lt.UpdatedAt, _ = parseTime(updatedAtStr)
	return &lt, nil
}

// GetLeaveTypeByCode retrieves a LeaveType by its unique code.
func (r *Repository) GetLeaveTypeByCode(ctx context.Context, code string) (*LeaveType, error) {
	query := `
		SELECT id, code, name, description, default_allocation, status, created_at, updated_at
		FROM leave_types
		WHERE UPPER(code) = UPPER(?)
	`
	row := r.db.QueryRowContext(ctx, query, strings.TrimSpace(code))

	var lt LeaveType
	var createdAtStr, updatedAtStr string
	err := row.Scan(&lt.ID, &lt.Code, &lt.Name, &lt.Description, &lt.DefaultAllocation, &lt.Status, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLeaveTypeNotFound
		}
		return nil, fmt.Errorf("failed to query leave type by code: %w", err)
	}

	lt.CreatedAt, _ = parseTime(createdAtStr)
	lt.UpdatedAt, _ = parseTime(updatedAtStr)
	return &lt, nil
}

// ListLeaveTypes retrieves all configured leave types.
func (r *Repository) ListLeaveTypes(ctx context.Context) ([]LeaveType, error) {
	query := `
		SELECT id, code, name, description, default_allocation, status, created_at, updated_at
		FROM leave_types
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list leave types: %w", err)
	}
	defer rows.Close()

	var leaveTypes []LeaveType
	for rows.Next() {
		var lt LeaveType
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&lt.ID, &lt.Code, &lt.Name, &lt.Description, &lt.DefaultAllocation, &lt.Status, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("failed to scan leave type row: %w", err)
		}
		lt.CreatedAt, _ = parseTime(createdAtStr)
		lt.UpdatedAt, _ = parseTime(updatedAtStr)
		leaveTypes = append(leaveTypes, lt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leave type rows: %w", err)
	}

	return leaveTypes, nil
}

// UpdateLeaveType updates an existing LeaveType record.
func (r *Repository) UpdateLeaveType(ctx context.Context, lt *LeaveType) error {
	query := `
		UPDATE leave_types
		SET code = ?, name = ?, description = ?, default_allocation = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query, lt.Code, lt.Name, lt.Description, lt.DefaultAllocation, lt.Status, lt.ID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "UNIQUE") {
			return ErrDuplicateCode
		}
		return fmt.Errorf("failed to update leave type: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return ErrLeaveTypeNotFound
	}
	return nil
}

// SetLeaveTypeStatus updates the active/inactive status of a leave type.
func (r *Repository) SetLeaveTypeStatus(ctx context.Context, id int64, status string) error {
	query := `
		UPDATE leave_types
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update leave type status: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return ErrLeaveTypeNotFound
	}
	return nil
}

// SaveLeaveBalance creates or updates an employee's leave allocation for a given leave type using an UPSERT pattern.
func (r *Repository) SaveLeaveBalance(ctx context.Context, bal *LeaveBalance) error {
	query := `
		INSERT INTO leave_balances (employee_id, leave_type_id, allocated_days, used_days, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(employee_id, leave_type_id) DO UPDATE SET
			allocated_days = excluded.allocated_days,
			updated_at = CURRENT_TIMESTAMP
	`
	res, err := r.db.ExecContext(ctx, query, bal.EmployeeID, bal.LeaveTypeID, bal.AllocatedDays, bal.UsedDays)
	if err != nil {
		return fmt.Errorf("failed to save leave balance: %w", err)
	}

	if bal.ID == 0 {
		id, err := res.LastInsertId()
		if err == nil && id > 0 {
			bal.ID = id
		}
	}
	return nil
}

// GetLeaveBalanceByEmployeeAndType retrieves the balance record for a specific employee and leave type.
func (r *Repository) GetLeaveBalanceByEmployeeAndType(ctx context.Context, employeeID, leaveTypeID int64) (*LeaveBalance, error) {
	query := `
		SELECT id, employee_id, leave_type_id, allocated_days, used_days, created_at, updated_at
		FROM leave_balances
		WHERE employee_id = ? AND leave_type_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, employeeID, leaveTypeID)

	var bal LeaveBalance
	var createdAtStr, updatedAtStr string
	err := row.Scan(&bal.ID, &bal.EmployeeID, &bal.LeaveTypeID, &bal.AllocatedDays, &bal.UsedDays, &createdAtStr, &updatedAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLeaveBalanceNotFound
		}
		return nil, fmt.Errorf("failed to query leave balance: %w", err)
	}

	bal.CreatedAt, _ = parseTime(createdAtStr)
	bal.UpdatedAt, _ = parseTime(updatedAtStr)
	return &bal, nil
}

// IncrementUsedDaysTx increments used_days for an employee's leave balance within an active SQL transaction.
func (r *Repository) IncrementUsedDaysTx(ctx context.Context, tx *sql.Tx, employeeID, leaveTypeID int64, days int) error {
	query := `
		UPDATE leave_balances
		SET used_days = used_days + ?, updated_at = CURRENT_TIMESTAMP
		WHERE employee_id = ? AND leave_type_id = ?
	`
	res, err := tx.ExecContext(ctx, query, days, employeeID, leaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to increment used days in transaction: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected balance rows: %w", err)
	}
	if rows == 0 {
		return ErrLeaveBalanceNotFound
	}
	return nil
}

// GetBalancesByEmployeeID returns all leave balance records with details for a single employee.
func (r *Repository) GetBalancesByEmployeeID(ctx context.Context, employeeID int64) ([]LeaveBalanceWithDetails, error) {
	query := `
		SELECT b.id, b.employee_id, b.leave_type_id, b.allocated_days, b.used_days, b.created_at, b.updated_at,
		       lt.code, lt.name, e.employee_code, (e.first_name || ' ' || e.last_name) AS employee_name, e.department
		FROM leave_balances b
		JOIN leave_types lt ON b.leave_type_id = lt.id
		JOIN employees e ON b.employee_id = e.id
		WHERE b.employee_id = ?
		ORDER BY lt.name ASC
	`
	rows, err := r.db.QueryContext(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query employee balances: %w", err)
	}
	defer rows.Close()

	var balances []LeaveBalanceWithDetails
	for rows.Next() {
		var item LeaveBalanceWithDetails
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&item.ID, &item.EmployeeID, &item.LeaveTypeID, &item.AllocatedDays, &item.UsedDays, &createdAtStr, &updatedAtStr,
			&item.LeaveTypeCode, &item.LeaveTypeName, &item.EmployeeCode, &item.EmployeeName, &item.Department,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leave balance row: %w", err)
		}
		item.CreatedAt, _ = parseTime(createdAtStr)
		item.UpdatedAt, _ = parseTime(updatedAtStr)
		balances = append(balances, item)
	}

	return balances, nil
}

// ListAllBalances returns all leave balance records across all employees with details for Admin view.
func (r *Repository) ListAllBalances(ctx context.Context) ([]LeaveBalanceWithDetails, error) {
	query := `
		SELECT b.id, b.employee_id, b.leave_type_id, b.allocated_days, b.used_days, b.created_at, b.updated_at,
		       lt.code, lt.name, e.employee_code, (e.first_name || ' ' || e.last_name) AS employee_name, e.department
		FROM leave_balances b
		JOIN leave_types lt ON b.leave_type_id = lt.id
		JOIN employees e ON b.employee_id = e.id
		ORDER BY e.employee_code ASC, lt.name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all leave balances: %w", err)
	}
	defer rows.Close()

	var balances []LeaveBalanceWithDetails
	for rows.Next() {
		var item LeaveBalanceWithDetails
		var createdAtStr, updatedAtStr string
		err := rows.Scan(
			&item.ID, &item.EmployeeID, &item.LeaveTypeID, &item.AllocatedDays, &item.UsedDays, &createdAtStr, &updatedAtStr,
			&item.LeaveTypeCode, &item.LeaveTypeName, &item.EmployeeCode, &item.EmployeeName, &item.Department,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leave balance row: %w", err)
		}
		item.CreatedAt, _ = parseTime(createdAtStr)
		item.UpdatedAt, _ = parseTime(updatedAtStr)
		balances = append(balances, item)
	}

	return balances, nil
}

// Helper to parse sqlite timestamps
func parseTime(val string) (time.Time, error) {
	if val == "" {
		return time.Time{}, nil
	}
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, val); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
