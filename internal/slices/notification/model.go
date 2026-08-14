package notification

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrUnauthorized        = errors.New("unauthorized access to notification")
)

const (
	TypeLeaveRequestSubmitted = "leave_request_submitted"
	TypeLeaveRequestApproved  = "leave_request_approved"
	TypeLeaveRequestRejected  = "leave_request_rejected"

	RefTypeLeaveRequest = "leave_request"
)

type Notification struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	Message       string     `json:"message"`
	ReferenceType string     `json:"reference_type"`
	ReferenceID   int64      `json:"reference_id"`
	IsRead        bool       `json:"is_read"`
	CreatedAt     time.Time  `json:"created_at"`
	ReadAt        *time.Time `json:"read_at,omitempty"`
}

// DisplayTimeFormatted returns a human-readable string for CreatedAt
func (n *Notification) DisplayTimeFormatted() string {
	return n.CreatedAt.Format("02 Jan 2006, 15:04")
}

// TargetURL returns the correct destination URL based on notification type and reference.
func (n *Notification) TargetURL() string {
	if n.ReferenceType == RefTypeLeaveRequest {
		if n.Type == TypeLeaveRequestSubmitted {
			return fmt.Sprintf("/manager/leave-requests/%d", n.ReferenceID)
		}
		return fmt.Sprintf("/my/leave-requests/%d", n.ReferenceID)
	}
	return "/"
}
