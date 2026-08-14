package leave_request

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"leavemang/internal/slices/employee"
	"leavemang/internal/slices/leave"
	"leavemang/internal/slices/notification"
)

type Service struct {
	repo         *Repository
	empService   *employee.Service
	leaveService *leave.Service
	notifService *notification.Service
}

func NewService(repo *Repository, empService *employee.Service, leaveService *leave.Service) *Service {
	return &Service{
		repo:         repo,
		empService:   empService,
		leaveService: leaveService,
	}
}

func (s *Service) SetNotificationService(notifService *notification.Service) {
	s.notifService = notifService
}

// CalculateDays calculates calendar days between fromDate and toDate inclusive: (toDate - fromDate) + 1
func CalculateDays(fromDateStr, toDateStr string) (int, error) {
	from, err := time.Parse("2006-01-02", fromDateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid start date format (expected YYYY-MM-DD): %w", err)
	}
	to, err := time.Parse("2006-01-02", toDateStr)
	if err != nil {
		return 0, fmt.Errorf("invalid end date format (expected YYYY-MM-DD): %w", err)
	}

	if from.After(to) {
		return 0, ErrInvalidDateRange
	}

	days := int(to.Sub(from).Hours()/24) + 1
	return days, nil
}

// CreateRequest handles full business logic for submitting a leave request.
func (s *Service) CreateRequest(ctx context.Context, userID int64, input CreateLeaveRequestInput) (*LeaveRequest, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// 1. Fetch & Validate Employee identity from session user ID (BR-01, BR-02)
	emp, err := s.empService.GetEmployeeByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("employee record not found: %w", err)
	}
	if emp.Status != employee.StatusActive {
		return nil, ErrInactiveEmployee
	}

	// 2. Fetch & Validate Leave Type (BR-03)
	lt, err := s.leaveService.GetLeaveTypeByID(ctx, input.LeaveTypeID)
	if err != nil {
		return nil, fmt.Errorf("invalid leave type: %w", err)
	}
	if !lt.IsActive() {
		return nil, ErrInactiveLeaveType
	}

	// 3. Validate Date Range & Calculate Requested Days (BR-04, BR-05)
	days, err := CalculateDays(input.FromDate, input.ToDate)
	if err != nil {
		return nil, err
	}

	// 4. Check Available Balance (BR-06, Balance Rule)
	bal, err := s.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	remaining := 0
	if err == nil && bal != nil {
		remaining = bal.Remaining()
	}
	if days > remaining {
		return nil, fmt.Errorf("You have only %d days remaining for %s. You requested %d days.", remaining, lt.Name, days)
	}

	// 5. Check Overlapping Active Requests (BR-08, Overlap Logic)
	overlaps, err := s.repo.FindOverlapping(ctx, emp.ID, input.FromDate, input.ToDate)
	if err != nil {
		return nil, fmt.Errorf("failed to check overlapping requests: %w", err)
	}
	if len(overlaps) > 0 {
		return nil, ErrOverlappingRequest
	}

	// 6. Construct & Create Pending Request
	req := &LeaveRequest{
		EmployeeID:  emp.ID,
		LeaveTypeID: lt.ID,
		FromDate:    input.FromDate,
		ToDate:      input.ToDate,
		Days:        days,
		Reason:      input.Reason,
		Status:      StatusPending,
	}

	createdReq, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	if s.notifService != nil {
		if reqDetail, err := s.repo.GetByID(ctx, createdReq.ID); err == nil && reqDetail != nil {
			_ = s.notifService.NotifyManagerNewLeaveRequest(ctx, notification.LeaveEventPayload{
				RequestID:     reqDetail.ID,
				EmployeeID:    reqDetail.EmployeeID,
				EmployeeName:  reqDetail.EmployeeName,
				LeaveTypeName: reqDetail.LeaveTypeName,
				FromDate:      reqDetail.FromDate,
				ToDate:        reqDetail.ToDate,
				Days:          reqDetail.Days,
			})
		}
	}

	return createdReq, nil
}

// GetRequest retrieves details of a leave request and verifies user ownership.
func (s *Service) GetRequest(ctx context.Context, id int64, userID int64) (*LeaveRequestWithDetails, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	emp, err := s.empService.GetEmployeeByUserID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// Employee can view their own request; Managers/Admins can view any
	if req.EmployeeID != emp.ID {
		var role string
		_ = s.repo.DB().QueryRowContext(ctx, "SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if role != "manager" && role != "admin" {
			return nil, ErrUnauthorized
		}
	}

	return req, nil
}

// ListEmployeeRequests retrieves leave request history for the logged in user.
func (s *Service) ListEmployeeRequests(ctx context.Context, userID int64) ([]*LeaveRequestWithDetails, error) {
	emp, err := s.empService.GetEmployeeByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("employee record not found: %w", err)
	}

	return s.repo.ListByEmployee(ctx, emp.ID)
}

// ListPendingRequests retrieves all pending leave requests for manager approval (VS-05).
func (s *Service) ListPendingRequests(ctx context.Context) ([]*LeaveRequestWithDetails, error) {
	return s.repo.ListPending(ctx)
}

// GetManagerRequestDetails retrieves full details for manager review, along with the employee's current balance.
func (s *Service) GetManagerRequestDetails(ctx context.Context, id int64) (*LeaveRequestWithDetails, *leave.LeaveBalance, error) {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	bal, err := s.leaveService.GetBalance(ctx, req.EmployeeID, req.LeaveTypeID)
	if err != nil && !errors.Is(err, leave.ErrLeaveBalanceNotFound) {
		return nil, nil, fmt.Errorf("failed to load leave balance: %w", err)
	}

	return req, bal, nil
}

// ApproveRequest handles manager approval of a pending request in a transaction (BR-01, BR-02, BR-03, BR-05, BR-06, BR-07).
func (s *Service) ApproveRequest(ctx context.Context, id int64, managerUserID int64) error {
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Status != StatusPending {
		return ErrAlreadyProcessed
	}

	bal, err := s.leaveService.GetBalance(ctx, req.EmployeeID, req.LeaveTypeID)
	if err != nil {
		return fmt.Errorf("failed to fetch employee leave balance: %w", err)
	}

	remaining := bal.Remaining()
	if req.Days > remaining {
		return ErrInsufficientBalance
	}

	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.ApproveTx(ctx, tx, id, managerUserID); err != nil {
		return err
	}

	if err := s.leaveService.IncrementUsedDaysTx(ctx, tx, req.EmployeeID, req.LeaveTypeID, req.Days); err != nil {
		return fmt.Errorf("failed to update leave balance in transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit approval transaction: %w", err)
	}

	if s.notifService != nil {
		_ = s.notifService.NotifyEmployeeApproved(ctx, notification.LeaveEventPayload{
			RequestID:     req.ID,
			EmployeeID:    req.EmployeeID,
			EmployeeName:  req.EmployeeName,
			LeaveTypeName: req.LeaveTypeName,
			FromDate:      req.FromDate,
			ToDate:        req.ToDate,
			Days:          req.Days,
		})
	}

	return nil
}

// RejectRequest handles manager rejection of a pending request in a transaction (BR-01, BR-02, BR-04, BR-05, BR-06, BR-08).
func (s *Service) RejectRequest(ctx context.Context, id int64, managerUserID int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrMissingRejectionReason
	}

	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if req.Status != StatusPending {
		return ErrAlreadyProcessed
	}

	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.RejectTx(ctx, tx, id, managerUserID, reason); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rejection transaction: %w", err)
	}

	if s.notifService != nil {
		_ = s.notifService.NotifyEmployeeRejected(ctx, notification.LeaveEventPayload{
			RequestID:       req.ID,
			EmployeeID:      req.EmployeeID,
			EmployeeName:    req.EmployeeName,
			LeaveTypeName:   req.LeaveTypeName,
			FromDate:        req.FromDate,
			ToDate:          req.ToDate,
			Days:            req.Days,
			RejectionReason: reason,
		})
	}

	return nil
}
