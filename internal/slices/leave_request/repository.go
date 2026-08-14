package leave_request

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying sql.DB connection for transaction management.
func (r *Repository) DB() *sql.DB {
	return r.db
}

// Create inserts a new leave request in the database.
func (r *Repository) Create(ctx context.Context, req *LeaveRequest) (*LeaveRequest, error) {
	query := `
		INSERT INTO leave_requests (employee_id, leave_type_id, from_date, to_date, days, reason, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(
		ctx, query,
		req.EmployeeID, req.LeaveTypeID, req.FromDate, req.ToDate, req.Days, req.Reason, StatusPending,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert leave request: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert ID: %w", err)
	}

	return r.GetByIDOnly(ctx, id)
}

// GetByIDOnly retrieves a basic LeaveRequest by ID.
func (r *Repository) GetByIDOnly(ctx context.Context, id int64) (*LeaveRequest, error) {
	query := `
		SELECT id, employee_id, leave_type_id, from_date, to_date, days, reason, status, created_at, updated_at, reviewed_by, reviewed_at, rejection_reason
		FROM leave_requests
		WHERE id = ?
	`
	req := &LeaveRequest{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.EmployeeID, &req.LeaveTypeID, &req.FromDate, &req.ToDate,
		&req.Days, &req.Reason, &req.Status, &req.CreatedAt, &req.UpdatedAt,
		&req.ReviewedBy, &req.ReviewedAt, &req.RejectionReason,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLeaveRequestNotFound
		}
		return nil, fmt.Errorf("failed to query leave request: %w", err)
	}
	return req, nil
}

// GetByID retrieves a LeaveRequest with details by ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*LeaveRequestWithDetails, error) {
	query := `
		SELECT 
			lr.id, lr.employee_id, lr.leave_type_id, lr.from_date, lr.to_date, lr.days, lr.reason, lr.status,
			lr.created_at, lr.updated_at, lr.reviewed_by, lr.reviewed_at, lr.rejection_reason,
			lt.code, lt.name,
			e.employee_code, (e.first_name || ' ' || e.last_name) AS employee_name, e.department
		FROM leave_requests lr
		JOIN leave_types lt ON lr.leave_type_id = lt.id
		JOIN employees e ON lr.employee_id = e.id
		WHERE lr.id = ?
	`
	req := &LeaveRequestWithDetails{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&req.ID, &req.EmployeeID, &req.LeaveTypeID, &req.FromDate, &req.ToDate, &req.Days, &req.Reason, &req.Status,
		&req.CreatedAt, &req.UpdatedAt, &req.ReviewedBy, &req.ReviewedAt, &req.RejectionReason,
		&req.LeaveTypeCode, &req.LeaveTypeName,
		&req.EmployeeCode, &req.EmployeeName, &req.Department,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLeaveRequestNotFound
		}
		return nil, fmt.Errorf("failed to query leave request details: %w", err)
	}
	return req, nil
}

// ListByEmployee retrieves all leave requests submitted by a specific employee.
func (r *Repository) ListByEmployee(ctx context.Context, employeeID int64) ([]*LeaveRequestWithDetails, error) {
	query := `
		SELECT 
			lr.id, lr.employee_id, lr.leave_type_id, lr.from_date, lr.to_date, lr.days, lr.reason, lr.status,
			lr.created_at, lr.updated_at, lr.reviewed_by, lr.reviewed_at, lr.rejection_reason,
			lt.code, lt.name,
			e.employee_code, (e.first_name || ' ' || e.last_name) AS employee_name, e.department
		FROM leave_requests lr
		JOIN leave_types lt ON lr.leave_type_id = lt.id
		JOIN employees e ON lr.employee_id = e.id
		WHERE lr.employee_id = ?
		ORDER BY lr.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query leave requests for employee: %w", err)
	}
	defer rows.Close()

	var requests []*LeaveRequestWithDetails
	for rows.Next() {
		req := &LeaveRequestWithDetails{}
		if err := rows.Scan(
			&req.ID, &req.EmployeeID, &req.LeaveTypeID, &req.FromDate, &req.ToDate, &req.Days, &req.Reason, &req.Status,
			&req.CreatedAt, &req.UpdatedAt, &req.ReviewedBy, &req.ReviewedAt, &req.RejectionReason,
			&req.LeaveTypeCode, &req.LeaveTypeName,
			&req.EmployeeCode, &req.EmployeeName, &req.Department,
		); err != nil {
			return nil, fmt.Errorf("failed to scan leave request row: %w", err)
		}
		requests = append(requests, req)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

// FindOverlapping checks for existing Pending or Approved leave requests that overlap with [fromDate, toDate].
func (r *Repository) FindOverlapping(ctx context.Context, employeeID int64, fromDate, toDate string) ([]*LeaveRequest, error) {
	query := `
		SELECT id, employee_id, leave_type_id, from_date, to_date, days, reason, status, created_at, updated_at, reviewed_by, reviewed_at, rejection_reason
		FROM leave_requests
		WHERE employee_id = ?
		  AND status IN (?, ?)
		  AND from_date <= ?
		  AND to_date >= ?
	`
	rows, err := r.db.QueryContext(ctx, query, employeeID, StatusPending, StatusApproved, toDate, fromDate)
	if err != nil {
		return nil, fmt.Errorf("failed to check overlapping requests: %w", err)
	}
	defer rows.Close()

	var requests []*LeaveRequest
	for rows.Next() {
		req := &LeaveRequest{}
		if err := rows.Scan(
			&req.ID, &req.EmployeeID, &req.LeaveTypeID, &req.FromDate, &req.ToDate, &req.Days, &req.Reason, &req.Status,
			&req.CreatedAt, &req.UpdatedAt, &req.ReviewedBy, &req.ReviewedAt, &req.RejectionReason,
		); err != nil {
			return nil, fmt.Errorf("failed to scan overlapping request: %w", err)
		}
		requests = append(requests, req)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

// ListPending retrieves all pending leave requests across all employees (for manager view).
func (r *Repository) ListPending(ctx context.Context) ([]*LeaveRequestWithDetails, error) {
	query := `
		SELECT 
			lr.id, lr.employee_id, lr.leave_type_id, lr.from_date, lr.to_date, lr.days, lr.reason, lr.status,
			lr.created_at, lr.updated_at, lr.reviewed_by, lr.reviewed_at, lr.rejection_reason,
			lt.code, lt.name,
			e.employee_code, (e.first_name || ' ' || e.last_name) AS employee_name, e.department
		FROM leave_requests lr
		JOIN leave_types lt ON lr.leave_type_id = lt.id
		JOIN employees e ON lr.employee_id = e.id
		WHERE lr.status = ?
		ORDER BY lr.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, StatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending leave requests: %w", err)
	}
	defer rows.Close()

	var requests []*LeaveRequestWithDetails
	for rows.Next() {
		req := &LeaveRequestWithDetails{}
		if err := rows.Scan(
			&req.ID, &req.EmployeeID, &req.LeaveTypeID, &req.FromDate, &req.ToDate, &req.Days, &req.Reason, &req.Status,
			&req.CreatedAt, &req.UpdatedAt, &req.ReviewedBy, &req.ReviewedAt, &req.RejectionReason,
			&req.LeaveTypeCode, &req.LeaveTypeName,
			&req.EmployeeCode, &req.EmployeeName, &req.Department,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pending request row: %w", err)
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// ApproveTx updates a pending request to Approved within a transaction.
func (r *Repository) ApproveTx(ctx context.Context, tx *sql.Tx, reqID, managerUserID int64) error {
	query := `
		UPDATE leave_requests
		SET status = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`
	res, err := tx.ExecContext(ctx, query, StatusApproved, managerUserID, reqID, StatusPending)
	if err != nil {
		return fmt.Errorf("failed to execute approve request query: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected on approve: %w", err)
	}
	if rows == 0 {
		return ErrAlreadyProcessed
	}
	return nil
}

// RejectTx updates a pending request to Rejected with rejection reason within a transaction.
func (r *Repository) RejectTx(ctx context.Context, tx *sql.Tx, reqID, managerUserID int64, rejectionReason string) error {
	query := `
		UPDATE leave_requests
		SET status = ?, rejection_reason = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`
	res, err := tx.ExecContext(ctx, query, StatusRejected, rejectionReason, managerUserID, reqID, StatusPending)
	if err != nil {
		return fmt.Errorf("failed to execute reject request query: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected on reject: %w", err)
	}
	if rows == 0 {
		return ErrAlreadyProcessed
	}
	return nil
}

// ListWithFilters retrieves leave requests filtered by status (All, Pending, Approved, Rejected) and/or employee ID.
func (r *Repository) ListWithFilters(ctx context.Context, status string, employeeID int64) ([]*LeaveRequestWithDetails, error) {
	query := `
		SELECT 
			lr.id, lr.employee_id, lr.leave_type_id, lr.from_date, lr.to_date, lr.days, lr.reason, lr.status,
			lr.created_at, lr.updated_at, lr.reviewed_by, lr.reviewed_at, lr.rejection_reason,
			lt.code, lt.name,
			e.employee_code, (e.first_name || ' ' || e.last_name) AS employee_name, e.department
		FROM leave_requests lr
		JOIN leave_types lt ON lr.leave_type_id = lt.id
		JOIN employees e ON lr.employee_id = e.id
		WHERE 1=1
	`
	var args []interface{}

	status = strings.TrimSpace(status)
	if status != "" && !strings.EqualFold(status, "all") {
		query += " AND LOWER(lr.status) = LOWER(?)"
		args = append(args, status)
	}

	if employeeID > 0 {
		query += " AND lr.employee_id = ?"
		args = append(args, employeeID)
	}

	query += " ORDER BY lr.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leave requests with filters: %w", err)
	}
	defer rows.Close()

	var requests []*LeaveRequestWithDetails
	for rows.Next() {
		req := &LeaveRequestWithDetails{}
		if err := rows.Scan(
			&req.ID, &req.EmployeeID, &req.LeaveTypeID, &req.FromDate, &req.ToDate, &req.Days, &req.Reason, &req.Status,
			&req.CreatedAt, &req.UpdatedAt, &req.ReviewedBy, &req.ReviewedAt, &req.RejectionReason,
			&req.LeaveTypeCode, &req.LeaveTypeName,
			&req.EmployeeCode, &req.EmployeeName, &req.Department,
		); err != nil {
			return nil, fmt.Errorf("failed to scan filtered leave request row: %w", err)
		}
		requests = append(requests, req)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

// GetDashboardStats calculates derived leave statistics dynamically from the underlying leave request data (AR-10, BR-07).
func (r *Repository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	query := `
		SELECT 
			COUNT(1) AS total,
			COALESCE(SUM(CASE WHEN LOWER(status) = 'pending' THEN 1 ELSE 0 END), 0) AS pending,
			COALESCE(SUM(CASE WHEN LOWER(status) = 'approved' THEN 1 ELSE 0 END), 0) AS approved,
			COALESCE(SUM(CASE WHEN LOWER(status) = 'rejected' THEN 1 ELSE 0 END), 0) AS rejected,
			COALESCE(SUM(CASE WHEN LOWER(status) = 'approved' THEN days ELSE 0 END), 0) AS approved_days
		FROM leave_requests
	`
	stats := &DashboardStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalRequests,
		&stats.PendingRequests,
		&stats.ApprovedRequests,
		&stats.RejectedRequests,
		&stats.ApprovedDays,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query dashboard statistics: %w", err)
	}
	return stats, nil
}

