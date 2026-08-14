package leave

import (
	"errors"
	"strings"
	"time"
)

// Status constants
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

var (
	ErrLeaveTypeNotFound   = errors.New("leave type not found")
	ErrDuplicateCode       = errors.New("leave type code already exists")
	ErrLeaveBalanceNotFound = errors.New("leave balance record not found")
	ErrInvalidAllocation   = errors.New("allocated days cannot be negative")
	ErrInactiveEmployee    = errors.New("cannot allocate leave to an inactive employee")
	ErrInactiveLeaveType   = errors.New("cannot allocate leave for an inactive leave type")
	ErrRequiredFields      = errors.New("missing required fields")
	ErrUsedExceedsAllocated = errors.New("used days cannot exceed allocated days")
	ErrNegativeUsedDays    = errors.New("used days cannot be negative")
)

// LeaveType represents a leave category configured by an Admin.
type LeaveType struct {
	ID                int64     `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	DefaultAllocation int       `json:"default_allocation"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// IsActive returns whether the leave type status is active.
func (lt *LeaveType) IsActive() bool {
	return strings.ToLower(lt.Status) == StatusActive
}

// CreateLeaveTypeInput contains form input for creating a leave type.
type CreateLeaveTypeInput struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	DefaultAllocation int    `json:"default_allocation"`
}

// Validate checks required fields and constraints for creation.
func (in *CreateLeaveTypeInput) Validate() error {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)

	if in.Code == "" || in.Name == "" {
		return ErrRequiredFields
	}

	if in.DefaultAllocation < 0 {
		return ErrInvalidAllocation
	}

	return nil
}

// UpdateLeaveTypeInput contains form input for updating a leave type.
type UpdateLeaveTypeInput struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	DefaultAllocation int    `json:"default_allocation"`
	Status            string `json:"status"`
}

// Validate checks required fields and constraints for update.
func (in *UpdateLeaveTypeInput) Validate() error {
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))

	if in.Code == "" || in.Name == "" {
		return ErrRequiredFields
	}

	if in.DefaultAllocation < 0 {
		return ErrInvalidAllocation
	}

	if in.Status != StatusActive && in.Status != StatusInactive {
		in.Status = StatusActive
	}

	return nil
}

// LeaveBalance represents an employee's leave entitlement and usage.
type LeaveBalance struct {
	ID            int64     `json:"id"`
	EmployeeID    int64     `json:"employee_id"`
	LeaveTypeID   int64     `json:"leave_type_id"`
	AllocatedDays int       `json:"allocated_days"`
	UsedDays      int       `json:"used_days"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Remaining calculates the remaining leave days (Allocated - Used).
func (b *LeaveBalance) Remaining() int {
	return b.AllocatedDays - b.UsedDays
}

// AllocateLeaveInput contains fields required to set an employee's leave allocation.
type AllocateLeaveInput struct {
	EmployeeID    int64 `json:"employee_id"`
	LeaveTypeID   int64 `json:"leave_type_id"`
	AllocatedDays int   `json:"allocated_days"`
}

// Validate checks fields for allocating leave balance.
func (in *AllocateLeaveInput) Validate() error {
	if in.EmployeeID <= 0 || in.LeaveTypeID <= 0 {
		return ErrRequiredFields
	}

	if in.AllocatedDays < 0 {
		return ErrInvalidAllocation
	}

	return nil
}

// LeaveBalanceWithDetails provides a composite view of a leave balance alongside employee and leave type details.
type LeaveBalanceWithDetails struct {
	LeaveBalance
	LeaveTypeCode string `json:"leave_type_code"`
	LeaveTypeName string `json:"leave_type_name"`
	EmployeeCode  string `json:"employee_code"`
	EmployeeName  string `json:"employee_name"`
	Department    string `json:"department"`
}
