package notification_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"leavemang/internal/shared/database"
	"leavemang/internal/slices/employee"
	"leavemang/internal/slices/leave"
	"leavemang/internal/slices/leave_request"
	"leavemang/internal/slices/notification"
)

func setupTestDB(t *testing.T) (*notification.Service, *leave_request.Service, *employee.Service, *leave.Service, int64, int64, int64, int64) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_notification.db")

	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	empRepo := employee.NewRepository(db)
	empService := employee.NewService(empRepo)

	leaveRepo := leave.NewRepository(db)
	leaveService := leave.NewService(leaveRepo, empRepo)

	notifRepo := notification.NewRepository(db)
	notifService := notification.NewService(notifRepo)

	leaveReqRepo := leave_request.NewRepository(db)
	leaveReqService := leave_request.NewService(leaveReqRepo, empService, leaveService)
	leaveReqService.SetNotificationService(notifService)

	// Fetch seeded user IDs
	// Rahul (employee, user_id=1, employee_id=1)
	// Priya (manager, user_id=2, employee_id=2)
	// Amit (admin, user_id=3, employee_id=3)
	var rahulUserID, priyaUserID, rahulEmpID, priyaEmpID int64
	err = db.QueryRow("SELECT id FROM users WHERE username = 'rahul'").Scan(&rahulUserID)
	if err != nil {
		t.Fatalf("failed to get rahul user id: %v", err)
	}
	err = db.QueryRow("SELECT id FROM users WHERE username = 'priya'").Scan(&priyaUserID)
	if err != nil {
		t.Fatalf("failed to get priya user id: %v", err)
	}
	err = db.QueryRow("SELECT id FROM employees WHERE user_id = ?", rahulUserID).Scan(&rahulEmpID)
	if err != nil {
		t.Fatalf("failed to get rahul emp id: %v", err)
	}
	err = db.QueryRow("SELECT id FROM employees WHERE user_id = ?", priyaUserID).Scan(&priyaEmpID)
	if err != nil {
		t.Fatalf("failed to get priya emp id: %v", err)
	}

	return notifService, leaveReqService, empService, leaveService, rahulUserID, priyaUserID, rahulEmpID, priyaEmpID
}

// AT-01 — New Leave Notification
func TestAT01_NewLeaveNotification(t *testing.T) {
	ctx := context.Background()
	notifService, leaveReqService, _, leaveService, rahulUserID, priyaUserID, _, _ := setupTestDB(t)

	// 1. Create a leave request for Rahul
	leaveTypes, err := leaveService.ListLeaveTypes(ctx)
	if err != nil || len(leaveTypes) == 0 {
		t.Fatalf("failed to fetch leave types: %v", err)
	}
	clType := leaveTypes[0]

	input := leave_request.CreateLeaveRequestInput{
		LeaveTypeID: clType.ID,
		FromDate:    "2026-08-20",
		ToDate:      "2026-08-22",
		Reason:      "Vacation",
	}

	req, err := leaveReqService.CreateRequest(ctx, rahulUserID, input)
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	// 2. Verify manager (Priya) receives a notification
	priyaNotifs, err := notifService.GetUserNotifications(ctx, priyaUserID)
	if err != nil {
		t.Fatalf("failed to fetch priya notifications: %v", err)
	}
	if len(priyaNotifs) == 0 {
		t.Fatalf("expected manager Priya to receive a notification, got 0")
	}

	n := priyaNotifs[0]
	if n.Type != notification.TypeLeaveRequestSubmitted {
		t.Errorf("expected notification type %s, got %s", notification.TypeLeaveRequestSubmitted, n.Type)
	}
	if n.ReferenceID != req.ID {
		t.Errorf("expected reference_id %d, got %d", req.ID, n.ReferenceID)
	}
	if n.IsRead {
		t.Errorf("expected is_read false, got true")
	}
}

// AT-02 — Approval Notification
func TestAT02_ApprovalNotification(t *testing.T) {
	ctx := context.Background()
	notifService, leaveReqService, _, leaveService, rahulUserID, priyaUserID, _, _ := setupTestDB(t)

	leaveTypes, _ := leaveService.ListLeaveTypes(ctx)
	req, err := leaveReqService.CreateRequest(ctx, rahulUserID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: leaveTypes[0].ID,
		FromDate:    "2026-08-20",
		ToDate:      "2026-08-22",
		Reason:      "Medical Leave",
	})
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	// Manager approves request
	err = leaveReqService.ApproveRequest(ctx, req.ID, priyaUserID)
	if err != nil {
		t.Fatalf("failed to approve leave request: %v", err)
	}

	// Verify Rahul (employee) receives approval notification
	rahulNotifs, err := notifService.GetUserNotifications(ctx, rahulUserID)
	if err != nil {
		t.Fatalf("failed to fetch rahul notifications: %v", err)
	}

	var approvalNotif *notification.Notification
	for _, n := range rahulNotifs {
		if n.Type == notification.TypeLeaveRequestApproved && n.ReferenceID == req.ID {
			approvalNotif = n
			break
		}
	}

	if approvalNotif == nil {
		t.Fatalf("expected employee Rahul to receive approval notification for request %d", req.ID)
	}
	if approvalNotif.IsRead {
		t.Errorf("expected new notification is_read to be false")
	}
}

// AT-03 — Rejection Notification
func TestAT03_RejectionNotification(t *testing.T) {
	ctx := context.Background()
	notifService, leaveReqService, _, leaveService, rahulUserID, priyaUserID, _, _ := setupTestDB(t)

	leaveTypes, _ := leaveService.ListLeaveTypes(ctx)
	req, err := leaveReqService.CreateRequest(ctx, rahulUserID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: leaveTypes[0].ID,
		FromDate:    "2026-08-20",
		ToDate:      "2026-08-22",
		Reason:      "Personal Work",
	})
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	rejectionReason := "Team deployment required."
	err = leaveReqService.RejectRequest(ctx, req.ID, priyaUserID, rejectionReason)
	if err != nil {
		t.Fatalf("failed to reject leave request: %v", err)
	}

	// Verify Rahul receives rejection notification with reason
	rahulNotifs, err := notifService.GetUserNotifications(ctx, rahulUserID)
	if err != nil {
		t.Fatalf("failed to fetch rahul notifications: %v", err)
	}

	var rejectionNotif *notification.Notification
	for _, n := range rahulNotifs {
		if n.Type == notification.TypeLeaveRequestRejected && n.ReferenceID == req.ID {
			rejectionNotif = n
			break
		}
	}

	if rejectionNotif == nil {
		t.Fatalf("expected employee Rahul to receive rejection notification for request %d", req.ID)
	}
	if rejectionNotif.Message == "" {
		t.Errorf("expected non-empty message in rejection notification")
	}
}

// AT-04 & AT-05 — Read Status & Mark as Read
func TestAT04_AT05_ReadStatusAndMarkAsRead(t *testing.T) {
	ctx := context.Background()
	notifService, leaveReqService, _, leaveService, rahulUserID, priyaUserID, _, _ := setupTestDB(t)

	leaveTypes, _ := leaveService.ListLeaveTypes(ctx)
	_, err := leaveReqService.CreateRequest(ctx, rahulUserID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: leaveTypes[0].ID,
		FromDate:    "2026-08-25",
		ToDate:      "2026-08-26",
		Reason:      "Test",
	})
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	priyaNotifs, err := notifService.GetUserNotifications(ctx, priyaUserID)
	if err != nil || len(priyaNotifs) == 0 {
		t.Fatalf("expected manager notification")
	}

	targetNotif := priyaNotifs[0]
	// AT-04: is_read = false
	if targetNotif.IsRead {
		t.Fatalf("expected new notification is_read = false")
	}
	if targetNotif.ReadAt != nil {
		t.Fatalf("expected read_at = nil for new notification")
	}

	// AT-05: Mark as read
	time.Sleep(10 * time.Millisecond)
	err = notifService.MarkAsRead(ctx, targetNotif.ID, priyaUserID)
	if err != nil {
		t.Fatalf("failed to mark notification as read: %v", err)
	}

	updatedNotifs, err := notifService.GetUserNotifications(ctx, priyaUserID)
	if err != nil {
		t.Fatalf("failed to fetch updated notifications: %v", err)
	}

	var updatedNotif *notification.Notification
	for _, n := range updatedNotifs {
		if n.ID == targetNotif.ID {
			updatedNotif = n
			break
		}
	}

	if updatedNotif == nil || !updatedNotif.IsRead {
		t.Errorf("expected notification is_read = true after marking read")
	}
	if updatedNotif.ReadAt == nil {
		t.Errorf("expected read_at timestamp to be populated after marking read")
	}
}

// AT-06 — User Isolation
func TestAT06_UserIsolation(t *testing.T) {
	ctx := context.Background()
	notifService, leaveReqService, _, leaveService, rahulUserID, priyaUserID, _, _ := setupTestDB(t)

	leaveTypes, _ := leaveService.ListLeaveTypes(ctx)

	// Submit leave for Rahul -> Priya receives notification
	req, _ := leaveReqService.CreateRequest(ctx, rahulUserID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: leaveTypes[0].ID,
		FromDate:    "2026-08-20",
		ToDate:      "2026-08-22",
		Reason:      "Isolation check",
	})

	// Rahul shouldn't see Priya's notification
	rahulNotifs, _ := notifService.GetUserNotifications(ctx, rahulUserID)
	for _, n := range rahulNotifs {
		if n.UserID != rahulUserID {
			t.Errorf("user isolation failure: Rahul received notification belonging to user_id %d", n.UserID)
		}
	}

	// Marking Priya's notification as read by Rahul should fail
	priyaNotifs, _ := notifService.GetUserNotifications(ctx, priyaUserID)
	if len(priyaNotifs) > 0 {
		err := notifService.MarkAsRead(ctx, priyaNotifs[0].ID, rahulUserID)
		if err == nil {
			t.Errorf("user isolation failure: Rahul was able to mark Priya's notification as read")
		}
	}

	_ = req
}

// AT-07 & AT-08 — Unread Count & Mark All Read
func TestAT07_AT08_UnreadCountAndMarkAllRead(t *testing.T) {
	ctx := context.Background()
	notifService, leaveReqService, _, leaveService, rahulUserID, priyaUserID, _, _ := setupTestDB(t)

	leaveTypes, _ := leaveService.ListLeaveTypes(ctx)

	// Submit 3 requests -> Priya receives 3 unread notifications
	for i := 0; i < 3; i++ {
		_, err := leaveReqService.CreateRequest(ctx, rahulUserID, leave_request.CreateLeaveRequestInput{
			LeaveTypeID: leaveTypes[0].ID,
			FromDate:    "2026-09-0" + string(rune('1'+i)),
			ToDate:      "2026-09-0" + string(rune('1'+i)),
			Reason:      "Batch leave request",
		})
		if err != nil {
			t.Fatalf("failed to create request %d: %v", i, err)
		}
	}

	// AT-07: Check unread count
	count, err := notifService.GetUnreadCount(ctx, priyaUserID)
	if err != nil {
		t.Fatalf("failed to get unread count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected unread count 3, got %d", count)
	}

	// AT-08: Mark all read
	err = notifService.MarkAllAsRead(ctx, priyaUserID)
	if err != nil {
		t.Fatalf("failed to mark all read: %v", err)
	}

	countAfter, err := notifService.GetUnreadCount(ctx, priyaUserID)
	if err != nil {
		t.Fatalf("failed to get unread count: %v", err)
	}
	if countAfter != 0 {
		t.Errorf("expected unread count 0 after mark all read, got %d", countAfter)
	}
}

// AT-09 — Notification Reference
func TestAT09_NotificationReference(t *testing.T) {
	ctx := context.Background()
	notifService, leaveReqService, _, leaveService, rahulUserID, priyaUserID, _, _ := setupTestDB(t)

	leaveTypes, _ := leaveService.ListLeaveTypes(ctx)
	req, _ := leaveReqService.CreateRequest(ctx, rahulUserID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: leaveTypes[0].ID,
		FromDate:    "2026-08-20",
		ToDate:      "2026-08-22",
		Reason:      "Reference check",
	})

	priyaNotifs, err := notifService.GetUserNotifications(ctx, priyaUserID)
	if err != nil || len(priyaNotifs) == 0 {
		t.Fatalf("expected notification for Priya")
	}

	n := priyaNotifs[0]
	if n.ReferenceType != notification.RefTypeLeaveRequest {
		t.Errorf("expected reference_type %s, got %s", notification.RefTypeLeaveRequest, n.ReferenceType)
	}
	if n.ReferenceID != req.ID {
		t.Errorf("expected reference_id %d, got %d", req.ID, n.ReferenceID)
	}
}
