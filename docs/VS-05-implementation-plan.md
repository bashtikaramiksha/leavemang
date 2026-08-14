# VS-05 — Implementation Plan: Manager Approval & Rejection

This implementation plan outlines the steps required to complete **VS-05: Manager Approval & Rejection**, finishing the core leave workflow lifecycle.

---

## Technical Overview

VS-05 allows authorized managers to review pending leave requests, inspect request details (including available vs. projected balance), and approve or reject them.

Key requirements implemented in this slice:
1. **Transactional Integrity**: Approval updates request status (`Pending` → `Approved`) and increments employee's `used_days` in a single atomic SQL transaction.
2. **Concurrency Control**: State changes check `WHERE id = ? AND status = 'Pending'`. If zero rows are affected, the operation aborts to prevent race conditions or double processing.
3. **Rejection Handling**: Rejection requires a `rejection_reason` and sets status (`Pending` → `Rejected`) without changing the leave balance.
4. **Role Authorization**: Access to `/manager/leave-requests*` routes is restricted to users with `manager` or `admin` roles (`RequireRole("manager", "admin")`).

---

## User Review Required

> [!IMPORTANT]
> - Database schema will be updated to add `rejection_reason TEXT` to the `leave_requests` table.
> - State transitions are strictly enforced: `Pending` → `Approved` or `Pending` → `Rejected`. Re-approving or re-rejecting already processed requests will return an error.

---

## Proposed Changes

### Database & Shared Layer

#### [MODIFY] [db.go](file:///d:/Projects/leavemang/internal/shared/database/db.go)
- Update `CREATE TABLE IF NOT EXISTS leave_requests` definition to include `rejection_reason TEXT`.
- Add an explicit `ALTER TABLE leave_requests ADD COLUMN rejection_reason TEXT;` migration execution step in `Migrate()` to update existing databases gracefully.

---

### Leave Request Slice

#### [MODIFY] [model.go](file:///d:/Projects/leavemang/internal/slices/leave_request/model.go)
- Add `RejectionReason sql.NullString` to `LeaveRequest` and `LeaveRequestWithDetails` structs.
- Add new domain error sentinel values: `ErrAlreadyProcessed` and `ErrMissingRejectionReason`.

#### [MODIFY] [repository.go](file:///d:/Projects/leavemang/internal/slices/leave_request/repository.go)
- Update SQL `SELECT` queries across `GetByIDOnly`, `GetByID`, `ListByEmployee`, `FindOverlapping`, and `ListPending` to include `rejection_reason`.
- Implement `ApproveRequestTx(ctx context.Context, tx *sql.Tx, reqID, managerUserID int64) error` to execute atomic conditional update `WHERE id = ? AND status = 'Pending'`.
- Implement `RejectRequestTx(ctx context.Context, tx *sql.Tx, reqID, managerUserID int64, rejectionReason string) error`.

#### [MODIFY] [repository.go (leave slice)](file:///d:/Projects/leavemang/internal/slices/leave/repository.go)
- Add `IncrementUsedDaysTx(ctx context.Context, tx *sql.Tx, employeeID, leaveTypeID int64, days int) error` to increment `used_days` in `leave_balances` within an active transaction.

#### [MODIFY] [service.go](file:///d:/Projects/leavemang/internal/slices/leave_request/service.go)
- Update `Service` to hold `*sql.DB` reference for transaction control.
- Implement `ApproveRequest(ctx context.Context, id int64, managerUserID int64) error`:
  1. Fetch request details and check `Pending` status.
  2. Verify employee's available balance (`remaining >= req.Days`).
  3. Begin `db.BeginTx`.
  4. Perform `ApproveRequestTx` (checks `RowsAffected == 1`).
  5. Perform `IncrementUsedDaysTx`.
  6. Commit transaction (or rollback on failure).
- Implement `RejectRequest(ctx context.Context, id int64, managerUserID int64, reason string) error`:
  1. Validate non-empty `reason`.
  2. Fetch request details and check `Pending` status.
  3. Begin `db.BeginTx`.
  4. Perform `RejectRequestTx` (checks `RowsAffected == 1`).
  5. Commit transaction (or rollback on failure).

#### [MODIFY] [handler.go](file:///d:/Projects/leavemang/internal/slices/leave_request/handler.go)
- Add `HandleListPendingRequests` (`GET /manager/leave-requests`) for Manager dashboard.
- Add `HandleManagerRequestDetails` (`GET /manager/leave-requests/{id}`) rendering request details, current balance, and projected balance after approval.
- Add `HandleApproveRequest` (`POST /manager/leave-requests/{id}/approve`).
- Add `HandleRejectRequest` (`POST /manager/leave-requests/{id}/reject`).

#### [MODIFY] [routes.go](file:///d:/Projects/leavemang/internal/slices/leave_request/routes.go)
- Register route group protected with `middleware.Authenticate` and `middleware.RequireRole("manager", "admin")`:
  - `GET /manager/leave-requests`
  - `GET /manager/leave-requests/{id}`
  - `POST /manager/leave-requests/{id}/approve`
  - `POST /manager/leave-requests/{id}/reject`

#### [NEW] [manager_list.html](file:///d:/Projects/leavemang/internal/slices/leave_request/templates/manager_list.html)
- Manager dashboard listing pending leave requests in a structured table format with employee details, leave type, date range, requested days, and action buttons.

#### [NEW] [manager_details.html](file:///d:/Projects/leavemang/internal/slices/leave_request/templates/manager_details.html)
- Manager detailed view showing request metadata, reason, current leave balance, calculated balance after approval, and interactive Approve/Reject forms/modals (with required rejection reason input).

---

### Tests

#### [MODIFY] [leave_request_test.go](file:///d:/Projects/leavemang/internal/slices/leave_request/leave_request_test.go)
- Implement tests covering AT-01 through AT-09:
  - `TestListPendingRequests` (AT-01)
  - `TestApproveRequest` (AT-02)
  - `TestRejectRequest` (AT-03)
  - `TestDoubleApproval` (AT-04)
  - `TestApproveRejectedRequest` (AT-05)
  - `TestEmployeeCannotApprove` (AT-06)
  - `TestInsufficientBalanceDuringApproval` (AT-07)
  - `TestTransactionConsistency` (AT-08)
  - `TestConcurrentApproval` (AT-09)

---

## Verification Plan

### Automated Tests
Execute all tests in the package to confirm acceptance criteria:
```bash
go test -v ./internal/slices/leave_request/...
```

### Manual Verification
1. Log in as Manager (`priya` / `password123`).
2. Navigate to `http://localhost:8080/manager/leave-requests`.
3. View pending request details, confirm current and post-approval remaining days calculations.
4. Test Approval flow, verify redirected page shows Approved status and updated employee balance.
5. Test Rejection flow with rejection reason, verify status set to Rejected and balance un-deducted.
6. Verify Non-Manager (`rahul`) accessing `/manager/leave-requests` receives 403 Forbidden.
