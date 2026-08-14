package leave_dashboard

import (
	"context"
	"fmt"

	"leavemang/internal/slices/employee"
	"leavemang/internal/slices/leave"
	"leavemang/internal/slices/leave_request"
)

type Service struct {
	leaveReqRepo *leave_request.Repository
	leaveRepo    *leave.Repository
	empRepo      *employee.Repository
	empService   *employee.Service
}

func NewService(
	leaveReqRepo *leave_request.Repository,
	leaveRepo *leave.Repository,
	empRepo *employee.Repository,
	empService *employee.Service,
) *Service {
	return &Service{
		leaveReqRepo: leaveReqRepo,
		leaveRepo:    leaveRepo,
		empRepo:      empRepo,
		empService:   empService,
	}
}

type EmployeeDashboardData struct {
	Employee       *employee.Employee
	Balances       []leave.LeaveBalanceWithDetails
	RecentRequests []*leave_request.LeaveRequestWithDetails
}

type ManagerDashboardData struct {
	Stats            *leave_request.DashboardStats
	Employees        []*employee.Employee
	Requests         []*leave_request.LeaveRequestWithDetails
	StatusFilter     string
	EmployeeIDFilter int64
}

// GetEmployeeDashboard retrieves leave balances and recent requests for the logged in employee.
func (s *Service) GetEmployeeDashboard(ctx context.Context, userID int64) (*EmployeeDashboardData, error) {
	emp, err := s.empService.GetEmployeeByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("employee record not found: %w", err)
	}

	balances, err := s.leaveRepo.GetBalancesByEmployeeID(ctx, emp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leave balances: %w", err)
	}

	requests, err := s.leaveReqRepo.ListByEmployee(ctx, emp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leave requests: %w", err)
	}

	recent := requests
	if len(recent) > 5 {
		recent = recent[:5]
	}

	return &EmployeeDashboardData{
		Employee:       emp,
		Balances:       balances,
		RecentRequests: recent,
	}, nil
}

// GetEmployeeHistory retrieves all leave requests for the logged in employee (BR-01, AR-03).
func (s *Service) GetEmployeeHistory(ctx context.Context, userID int64) (*employee.Employee, []*leave_request.LeaveRequestWithDetails, error) {
	emp, err := s.empService.GetEmployeeByUserID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("employee record not found: %w", err)
	}

	requests, err := s.leaveReqRepo.ListByEmployee(ctx, emp.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch leave requests: %w", err)
	}

	return emp, requests, nil
}

// GetEmployeeBalances retrieves detailed balances for the logged in employee (BR-01, AR-02, BR-06).
func (s *Service) GetEmployeeBalances(ctx context.Context, userID int64) (*employee.Employee, []leave.LeaveBalanceWithDetails, error) {
	emp, err := s.empService.GetEmployeeByUserID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("employee record not found: %w", err)
	}

	balances, err := s.leaveRepo.GetBalancesByEmployeeID(ctx, emp.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch leave balances: %w", err)
	}

	return emp, balances, nil
}

// GetManagerDashboard retrieves summary stats, permitted employee list, and filtered leave requests (AR-07 to AR-10, BR-02).
func (s *Service) GetManagerDashboard(ctx context.Context, statusFilter string, employeeIDFilter int64) (*ManagerDashboardData, error) {
	stats, err := s.leaveReqRepo.GetDashboardStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load dashboard statistics: %w", err)
	}

	employees, err := s.empRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}

	requests, err := s.leaveReqRepo.ListWithFilters(ctx, statusFilter, employeeIDFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list filtered requests: %w", err)
	}

	return &ManagerDashboardData{
		Stats:            stats,
		Employees:        employees,
		Requests:         requests,
		StatusFilter:     statusFilter,
		EmployeeIDFilter: employeeIDFilter,
	}, nil
}
