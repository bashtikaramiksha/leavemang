package employee

import (
	"context"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateEmployee validates and creates a new employee.
func (s *Service) CreateEmployee(ctx context.Context, input CreateEmployeeInput) (*Employee, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, input)
}

// GetEmployee retrieves an employee by ID.
func (s *Service) GetEmployee(ctx context.Context, id int64) (*Employee, error) {
	if id <= 0 {
		return nil, ErrEmployeeNotFound
	}
	return s.repo.GetByID(ctx, id)
}

// GetEmployeeByUserID retrieves an employee by linked User ID.
func (s *Service) GetEmployeeByUserID(ctx context.Context, userID int64) (*Employee, error) {
	if userID <= 0 {
		return nil, ErrEmployeeNotFound
	}
	return s.repo.GetByUserID(ctx, userID)
}

// ListEmployees retrieves all employee records.
func (s *Service) ListEmployees(ctx context.Context) ([]*Employee, error) {
	return s.repo.List(ctx)
}

// UpdateEmployee validates and updates an existing employee record.
func (s *Service) UpdateEmployee(ctx context.Context, id int64, input UpdateEmployeeInput) (*Employee, error) {
	if id <= 0 {
		return nil, ErrEmployeeNotFound
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, input)
}

// ActivateEmployee sets an employee's status to active.
func (s *Service) ActivateEmployee(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrEmployeeNotFound
	}
	return s.repo.SetStatus(ctx, id, StatusActive)
}

// DeactivateEmployee sets an employee's status to inactive (soft deactivation preserving historical records).
func (s *Service) DeactivateEmployee(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrEmployeeNotFound
	}
	return s.repo.SetStatus(ctx, id, StatusInactive)
}
