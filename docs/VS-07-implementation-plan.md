# VS-07 — Leave Notifications Implementation Plan

## Proposed Architecture & Design

VS-07 introduces an in-app notification system that reacts to leave request events (submission, approval, rejection) while keeping notification handling decoupled from core leave transaction logic.

### 1. Database Schema & Migration (`internal/shared/database/db.go`)
Create `notifications` table:
```sql
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    reference_type TEXT NOT NULL,
    reference_id INTEGER NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### 2. Notification Slice (`internal/slices/notification/`)
- `model.go`: Notification data model, types (`leave_request_submitted`, `leave_request_approved`, `leave_request_rejected`), and constants.
- `repository.go`: Database operations (`Create`, `ListByUser`, `GetByID`, `MarkAsRead`, `MarkAllAsRead`, `CountUnread`, `ExistsDuplicate`).
- `service.go`: Business operations (`NotifyManagerNewLeaveRequest`, `NotifyEmployeeApproved`, `NotifyEmployeeRejected`, `GetUserNotifications`, `MarkAsRead`, `MarkAllAsRead`, `GetUnreadCount`).
- `handler.go`: HTTP handlers for notification listing, read actions, and HTMX badge endpoint.
- `routes.go`: Route registration for `/notifications*`.
- `templates/`: `list.html` and `partials/notification_badge.html`.
- `notification_test.go`: Unit and integration tests covering AT-01 through AT-09.

### 3. Integration with Leave Request Slice (`internal/slices/leave_request/`)
- Inject `NotificationService` into `leave_request.Service`.
- `CreateRequest`: Call `NotifyManagerNewLeaveRequest(ctx, reqDetail)` after request creation.
- `ApproveRequest`: Call `NotifyEmployeeApproved(ctx, reqDetail)` after approval transaction commits.
- `RejectRequest`: Call `NotifyEmployeeRejected(ctx, reqDetail, reason)` after rejection transaction commits.

### 4. UI & HTMX Dynamic Unread Badge
- HTMX polling / load trigger for `/notifications/unread-count`.
- Notification list UI displaying read/unread status (● unread, ○ read), formatted timestamp, title, message, and direct link to the referenced leave request.

### 5. Verification Plan
- Unit tests & integration tests in `notification_test.go`.
- Verification of leave request submission, approval, and rejection notifications.
- Verification of ownership isolation and read state transitions.
