package leave_request

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Status constants
const (
	StatusPending  = "Pending"
	StatusApproved = "Approved"
	StatusRejected = "Rejected"
)

// Common errors
var (
	ErrLeaveRequestNotFound     = errors.New("leave request not found")
	ErrInvalidDateRange         = errors.New("the start date cannot be after the end date")
	ErrMissingReason            = errors.New("please provide a reason for your leave request")
	ErrInsufficientBalance      = errors.New("insufficient leave balance")
	ErrOverlappingRequest       = errors.New("you already have a leave request for part of this period")
	ErrInactiveEmployee         = errors.New("an inactive employee cannot create a new leave request")
	ErrInactiveLeaveType        = errors.New("this leave type is currently unavailable")
	ErrUnauthorized             = errors.New("unauthorized access to leave request")
	ErrAlreadyProcessed         = errors.New("this leave request has already been processed")
	ErrMissingRejectionReason   = errors.New("please provide a reason for rejection")
)

// LeaveRequest represents a leave request application.
type LeaveRequest struct {
	ID              int64          `json:"id"`
	EmployeeID      int64          `json:"employee_id"`
	LeaveTypeID     int64          `json:"leave_type_id"`
	FromDate        string         `json:"from_date"` // YYYY-MM-DD
	ToDate          string         `json:"to_date"`   // YYYY-MM-DD
	Days            int            `json:"days"`
	Reason          string         `json:"reason"`
	Status          string         `json:"status"` // Pending, Approved, Rejected
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	ReviewedBy      sql.NullInt64  `json:"reviewed_by,omitempty"`
	ReviewedAt      sql.NullTime   `json:"reviewed_at,omitempty"`
	RejectionReason sql.NullString `json:"rejection_reason,omitempty"`
}

// LeaveRequestWithDetails provides composite view of request with employee and leave type details.
type LeaveRequestWithDetails struct {
	LeaveRequest
	LeaveTypeCode string `json:"leave_type_code"`
	LeaveTypeName string `json:"leave_type_name"`
	EmployeeCode  string `json:"employee_code"`
	EmployeeName  string `json:"employee_name"`
	Department    string `json:"department"`
}

// CreateLeaveRequestInput contains input data for submitting a leave request.
type CreateLeaveRequestInput struct {
	LeaveTypeID int64  `json:"leave_type_id"`
	FromDate    string `json:"from_date"`
	ToDate      string `json:"to_date"`
	Reason      string `json:"reason"`
}

// Validate performs basic structural validation on input fields.
func (in *CreateLeaveRequestInput) Validate() error {
	in.FromDate = strings.TrimSpace(in.FromDate)
	in.ToDate = strings.TrimSpace(in.ToDate)
	in.Reason = strings.TrimSpace(in.Reason)

	if in.LeaveTypeID <= 0 {
		return errors.New("please select a leave type")
	}
	if in.FromDate == "" || in.ToDate == "" {
		return errors.New("please select valid start and end dates")
	}
	if in.Reason == "" {
		return ErrMissingReason
	}
	return nil
}

// DashboardStats represents derived leave statistics for manager dashboard reporting (AR-10, BR-07).
type DashboardStats struct {
	TotalRequests    int `json:"total_requests"`
	PendingRequests  int `json:"pending_requests"`
	ApprovedRequests int `json:"approved_requests"`
	RejectedRequests int `json:"rejected_requests"`
	ApprovedDays     int `json:"approved_days"`
}

