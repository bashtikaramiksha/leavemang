package leave

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"leavemang/internal/slices/employee"
)

// Service provides business logic operations for leave type management and balance allocations.
type Service struct {
	leaveRepo *Repository
	empRepo   *employee.Repository
}

// NewService instantiates a new leave Service.
func NewService(leaveRepo *Repository, empRepo *employee.Repository) *Service {
	return &Service{
		leaveRepo: leaveRepo,
		empRepo:   empRepo,
	}
}

// CreateLeaveType validates and creates a new leave type. Enforces BR-06 (unique code).
func (s *Service) CreateLeaveType(ctx context.Context, input CreateLeaveTypeInput) (*LeaveType, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check if code already exists
	existing, err := s.leaveRepo.GetLeaveTypeByCode(ctx, input.Code)
	if err == nil && existing != nil {
		return nil, ErrDuplicateCode
	}
	if err != nil && !errors.Is(err, ErrLeaveTypeNotFound) {
		return nil, fmt.Errorf("failed to check existing leave type: %w", err)
	}

	lt := &LeaveType{
		Code:              input.Code,
		Name:              input.Name,
		Description:       input.Description,
		DefaultAllocation: input.DefaultAllocation,
		Status:            StatusActive,
	}

	if err := s.leaveRepo.CreateLeaveType(ctx, lt); err != nil {
		return nil, err
	}

	return lt, nil
}

// GetLeaveTypeByID retrieves a single leave type by ID.
func (s *Service) GetLeaveTypeByID(ctx context.Context, id int64) (*LeaveType, error) {
	return s.leaveRepo.GetLeaveTypeByID(ctx, id)
}

// ListLeaveTypes returns all leave types.
func (s *Service) ListLeaveTypes(ctx context.Context) ([]LeaveType, error) {
	return s.leaveRepo.ListLeaveTypes(ctx)
}

// UpdateLeaveType updates an existing leave type and validates code uniqueness.
func (s *Service) UpdateLeaveType(ctx context.Context, id int64, input UpdateLeaveTypeInput) (*LeaveType, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.leaveRepo.GetLeaveTypeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// If code changed, check that new code isn't taken
	if existing.Code != input.Code {
		byCode, err := s.leaveRepo.GetLeaveTypeByCode(ctx, input.Code)
		if err == nil && byCode != nil && byCode.ID != id {
			return nil, ErrDuplicateCode
		}
	}

	existing.Code = input.Code
	existing.Name = input.Name
	existing.Description = input.Description
	existing.DefaultAllocation = input.DefaultAllocation
	existing.Status = input.Status

	if err := s.leaveRepo.UpdateLeaveType(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// ActivateLeaveType sets status to active for the leave type.
func (s *Service) ActivateLeaveType(ctx context.Context, id int64) error {
	return s.leaveRepo.SetLeaveTypeStatus(ctx, id, StatusActive)
}

// DeactivateLeaveType sets status to inactive without deleting data (BR-09).
func (s *Service) DeactivateLeaveType(ctx context.Context, id int64) error {
	return s.leaveRepo.SetLeaveTypeStatus(ctx, id, StatusInactive)
}

// AllocateLeave assigns a leave allocation to an employee. Enforces BR-02, BR-07, BR-08.
func (s *Service) AllocateLeave(ctx context.Context, input AllocateLeaveInput) (*LeaveBalance, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Validate Employee (AR-09, BR-08)
	emp, err := s.empRepo.GetByID(ctx, input.EmployeeID)
	if err != nil {
		return nil, err
	}
	if emp.Status != employee.StatusActive {
		return nil, ErrInactiveEmployee
	}

	// Validate Leave Type (AR-10, BR-07)
	lt, err := s.leaveRepo.GetLeaveTypeByID(ctx, input.LeaveTypeID)
	if err != nil {
		return nil, err
	}
	if !lt.IsActive() {
		return nil, ErrInactiveLeaveType
	}

	// Check existing balance
	existingBal, err := s.leaveRepo.GetLeaveBalanceByEmployeeAndType(ctx, input.EmployeeID, input.LeaveTypeID)
	usedDays := 0
	balID := int64(0)
	if err == nil && existingBal != nil {
		usedDays = existingBal.UsedDays
		balID = existingBal.ID
	}

	if usedDays > input.AllocatedDays {
		return nil, ErrUsedExceedsAllocated
	}

	bal := &LeaveBalance{
		ID:            balID,
		EmployeeID:    input.EmployeeID,
		LeaveTypeID:   input.LeaveTypeID,
		AllocatedDays: input.AllocatedDays,
		UsedDays:      usedDays,
	}

	if err := s.leaveRepo.SaveLeaveBalance(ctx, bal); err != nil {
		return nil, err
	}

	return bal, nil
}

// GetEmployeeBalances returns all leave balances for a given employee (AR-06).
func (s *Service) GetEmployeeBalances(ctx context.Context, employeeID int64) ([]LeaveBalanceWithDetails, error) {
	_, err := s.empRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	return s.leaveRepo.GetBalancesByEmployeeID(ctx, employeeID)
}

// GetEmployeeBalancesByUserID returns leave balances using linked user ID (for logged-in user).
func (s *Service) GetEmployeeBalancesByUserID(ctx context.Context, userID int64) (*employee.Employee, []LeaveBalanceWithDetails, error) {
	emp, err := s.empRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	bals, err := s.leaveRepo.GetBalancesByEmployeeID(ctx, emp.ID)
	if err != nil {
		return nil, nil, err
	}

	return emp, bals, nil
}

// GetBalance gets a single LeaveBalance record for VS-04 handoff contract.
func (s *Service) GetBalance(ctx context.Context, employeeID, leaveTypeID int64) (*LeaveBalance, error) {
	return s.leaveRepo.GetLeaveBalanceByEmployeeAndType(ctx, employeeID, leaveTypeID)
}

// ListAllBalances returns all balance allocations for Admin view.
func (s *Service) ListAllBalances(ctx context.Context) ([]LeaveBalanceWithDetails, error) {
	return s.leaveRepo.ListAllBalances(ctx)
}

// IncrementUsedDaysTx increments used days for an employee balance in transaction (for VS-05).
func (s *Service) IncrementUsedDaysTx(ctx context.Context, tx *sql.Tx, employeeID, leaveTypeID int64, days int) error {
	return s.leaveRepo.IncrementUsedDaysTx(ctx, tx, employeeID, leaveTypeID, days)
}
