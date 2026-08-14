package notification

import (
	"context"
	"fmt"
	"log"
)

type LeaveEventPayload struct {
	RequestID       int64
	EmployeeID      int64
	EmployeeName    string
	LeaveTypeName   string
	FromDate        string
	ToDate          string
	Days            int
	RejectionReason string
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// NotifyManagerNewLeaveRequest creates a notification for all managers when an employee submits a leave request (AR-02).
func (s *Service) NotifyManagerNewLeaveRequest(ctx context.Context, payload LeaveEventPayload) error {
	managerUserIDs, err := s.repo.GetManagerUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get manager user IDs for notification: %w", err)
	}

	daysStr := "days"
	if payload.Days == 1 {
		daysStr = "day"
	}

	title := "New Leave Request"
	message := fmt.Sprintf("%s submitted a %s request for %d %s.", payload.EmployeeName, payload.LeaveTypeName, payload.Days, daysStr)

	for _, mgrID := range managerUserIDs {
		// Prevent duplicate notifications
		exists, err := s.repo.ExistsDuplicate(ctx, mgrID, TypeLeaveRequestSubmitted, RefTypeLeaveRequest, payload.RequestID)
		if err != nil {
			log.Printf("[NOTIFICATION WARNING] Check duplicate failed for manager %d: %v", mgrID, err)
		}
		if exists {
			continue
		}

		n := &Notification{
			UserID:        mgrID,
			Type:          TypeLeaveRequestSubmitted,
			Title:         title,
			Message:       message,
			ReferenceType: RefTypeLeaveRequest,
			ReferenceID:   payload.RequestID,
		}

		if _, err := s.repo.Create(ctx, n); err != nil {
			log.Printf("[NOTIFICATION ERROR] Failed to create submission notification for manager %d: %v", mgrID, err)
		}
	}

	return nil
}

// NotifyEmployeeApproved creates a notification for the employee when their leave request is approved (AR-03).
func (s *Service) NotifyEmployeeApproved(ctx context.Context, payload LeaveEventPayload) error {
	employeeUserID, err := s.repo.GetEmployeeUserID(ctx, payload.EmployeeID)
	if err != nil {
		return fmt.Errorf("failed to find user_id for employee %d: %w", payload.EmployeeID, err)
	}

	exists, err := s.repo.ExistsDuplicate(ctx, employeeUserID, TypeLeaveRequestApproved, RefTypeLeaveRequest, payload.RequestID)
	if err != nil {
		log.Printf("[NOTIFICATION WARNING] Check duplicate failed for employee user %d: %v", employeeUserID, err)
	}
	if exists {
		return nil
	}

	title := "Leave Request Approved"
	message := fmt.Sprintf("Your %s request for %s → %s has been approved.", payload.LeaveTypeName, payload.FromDate, payload.ToDate)

	n := &Notification{
		UserID:        employeeUserID,
		Type:          TypeLeaveRequestApproved,
		Title:         title,
		Message:       message,
		ReferenceType: RefTypeLeaveRequest,
		ReferenceID:   payload.RequestID,
	}

	_, err = s.repo.Create(ctx, n)
	if err != nil {
		log.Printf("[NOTIFICATION ERROR] Failed to create approval notification for user %d: %v", employeeUserID, err)
		return err
	}

	return nil
}

// NotifyEmployeeRejected creates a notification for the employee when their leave request is rejected (AR-04).
func (s *Service) NotifyEmployeeRejected(ctx context.Context, payload LeaveEventPayload) error {
	employeeUserID, err := s.repo.GetEmployeeUserID(ctx, payload.EmployeeID)
	if err != nil {
		return fmt.Errorf("failed to find user_id for employee %d: %w", payload.EmployeeID, err)
	}

	exists, err := s.repo.ExistsDuplicate(ctx, employeeUserID, TypeLeaveRequestRejected, RefTypeLeaveRequest, payload.RequestID)
	if err != nil {
		log.Printf("[NOTIFICATION WARNING] Check duplicate failed for employee user %d: %v", employeeUserID, err)
	}
	if exists {
		return nil
	}

	title := "Leave Request Rejected"
	message := fmt.Sprintf("Your %s request for %s → %s was rejected.", payload.LeaveTypeName, payload.FromDate, payload.ToDate)
	if payload.RejectionReason != "" {
		message += fmt.Sprintf(" Reason: %s", payload.RejectionReason)
	}

	n := &Notification{
		UserID:        employeeUserID,
		Type:          TypeLeaveRequestRejected,
		Title:         title,
		Message:       message,
		ReferenceType: RefTypeLeaveRequest,
		ReferenceID:   payload.RequestID,
	}

	_, err = s.repo.Create(ctx, n)
	if err != nil {
		log.Printf("[NOTIFICATION ERROR] Failed to create rejection notification for user %d: %v", employeeUserID, err)
		return err
	}

	return nil
}

// GetUserNotifications returns notification history for a user (AR-05, AR-08, BR-02).
func (s *Service) GetUserNotifications(ctx context.Context, userID int64) ([]*Notification, error) {
	return s.repo.ListByUser(ctx, userID)
}

// MarkAsRead marks a notification as read (AR-06, BR-03, BR-04).
func (s *Service) MarkAsRead(ctx context.Context, id int64, userID int64) error {
	return s.repo.MarkAsRead(ctx, id, userID)
}

// MarkAllAsRead marks all unread notifications for a user as read (AR-06, BR-03, BR-04).
func (s *Service) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

// GetUnreadCount returns the count of unread notifications for a user (AR-07).
func (s *Service) GetUnreadCount(ctx context.Context, userID int64) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}
