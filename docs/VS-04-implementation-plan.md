# VS-04 Implementation Plan — Apply for Leave

Build the complete employee leave request workflow for authenticated active employees, incorporating validation against leave balances and overlapping active requests, saving requests with `Pending` status.

## User Review Required
> [!IMPORTANT]
> - **Balance Deduction Scope**: As specified in decision record ICM-03, submitting a leave request validates against available balance (`Allocated - Used`), but does **NOT** deduct `Used` days. Balance deduction will occur upon manager approval in VS-05.
> - **Calendar Day Calculation**: Standard day calculation formula `(End - Start) + 1` calendar days is applied without weekend or holiday exclusions in this slice.
> - **Session-based Authorization**: The employee identity is strictly derived from the authenticated session (`User -> Employee`) to avoid trusting client-supplied IDs.

## Open Questions
- None. Requirements are clear and completely covered by the VS-04 specification.

## Proposed Changes

### Database & Shared Schema

#### [MODIFY] [db.go](file:///d:/Projects/leavemang/internal/shared/database/db.go)
- Add `leave_requests` table to `Migrate()`:
  ```sql
  CREATE TABLE IF NOT EXISTS leave_requests (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      employee_id INTEGER NOT NULL,
      leave_type_id INTEGER NOT NULL,
      from_date TEXT NOT NULL,
      to_date TEXT NOT NULL,
      days INTEGER NOT NULL,
      reason TEXT NOT NULL,
      status TEXT NOT NULL DEFAULT 'Pending',
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      reviewed_by INTEGER,
      reviewed_at DATETIME,
      FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
      FOREIGN KEY (leave_type_id) REFERENCES leave_types(id) ON DELETE CASCADE,
      FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL
  );
  ```

---

### Leave Request Slice (`internal/slices/leave_request`)

#### [NEW] [model.go](file:///d:/Projects/leavemang/internal/slices/leave_request/model.go)
- Define `LeaveRequest` struct and `LeaveRequestWithDetails` composite struct.
- Define `CreateLeaveRequestInput` struct.
- Define status constants: `StatusPending = "Pending"`, `StatusApproved = "Approved"`, `StatusRejected = "Rejected"`.
- Define errors: `ErrInvalidDateRange`, `ErrMissingReason`, `ErrInsufficientBalance`, `ErrOverlappingRequest`, `ErrInactiveEmployee`, `ErrInactiveLeaveType`, `ErrLeaveRequestNotFound`, `ErrUnauthorized`.

#### [NEW] [repository.go](file:///d:/Projects/leavemang/internal/slices/leave_request/repository.go)
- Implement `LeaveRequestRepository`:
  - `Create(ctx, req)`
  - `GetByID(ctx, id)`
  - `ListByEmployee(ctx, employeeID)`
  - `FindOverlapping(ctx, employeeID, fromDate, toDate)` — checks for requests with status `Pending` or `Approved` overlapping given date range.
  - `ListPending(ctx)` — for VS-05 manager handoff.

#### [NEW] [service.go](file:///d:/Projects/leavemang/internal/slices/leave_request/service.go)
- Implement `LeaveRequestService`:
  - `CreateRequest(ctx, userID, input)`:
    1. Fetch `Employee` by `userID` via `employee.Service`. Reject if not active.
    2. Fetch `LeaveType` by ID via `leave.Service`. Reject if not active.
    3. Validate dates: Parse `from_date` and `to_date` (`YYYY-MM-DD`). Reject if `from_date > to_date`.
    4. Calculate days: `(to_date - from_date) + 1`.
    5. Validate reason: Non-empty trimmed text.
    6. Check balance: Fetch balance for `(employeeID, leaveTypeID)` via `leave.Service`. Ensure `days <= remaining`. Reject if insufficient.
    7. Check overlap: Call `repository.FindOverlapping(ctx, employeeID, fromDate, toDate)`. Reject if overlapping request exists.
    8. Insert `LeaveRequest` record with `Status = "Pending"`.
  - `GetRequest(ctx, id, userID)`: Fetch request and ensure employee owns it or user has permission.
  - `ListEmployeeRequests(ctx, userID)`: List requests for authenticated user's employee record.

#### [NEW] [handler.go](file:///d:/Projects/leavemang/internal/slices/leave_request/handler.go)
- Implement HTTP handlers:
  - `HandleNewRequestForm`: Render leave application form (`/my/leave-requests/new`), listing active leave types and employee's current balances.
  - `HandleCreateRequest`: Process form POST (`/my/leave-requests`), validate & create request. Handle HTMX & HTML form redirects/errors.
  - `HandleListEmployeeRequests`: Render employee request history (`/my/leave-requests`).
  - `HandleRequestDetails`: Render detailed request view (`/my/leave-requests/{id}`).

#### [NEW] [routes.go](file:///d:/Projects/leavemang/internal/slices/leave_request/routes.go)
- Register HTTP routes using Chi router and authentication middleware:
  - `GET /my/leave-requests`
  - `GET /my/leave-requests/new`
  - `POST /my/leave-requests`
  - `GET /my/leave-requests/{id}`

#### [NEW] [templates/form.html](file:///d:/Projects/leavemang/internal/slices/leave_request/templates/form.html)
- Leave Application Form template (`/my/leave-requests/new`) with leave type dropdown, date inputs, calculated days indicator, available balance display, reason textarea, submit button, and clear inline validation error alerts.

#### [NEW] [templates/list.html](file:///d:/Projects/leavemang/internal/slices/leave_request/templates/list.html)
- Employee's Leave Request History table (`/my/leave-requests`) displaying leave type, dates, days, status badges (Pending/Approved/Rejected), and view link.

#### [NEW] [templates/details.html](file:///d:/Projects/leavemang/internal/slices/leave_request/templates/details.html)
- Leave request detail view (`/my/leave-requests/{id}`) showing request metadata.

#### [NEW] [leave_request_test.go](file:///d:/Projects/leavemang/internal/slices/leave_request/leave_request_test.go)
- Comprehensive unit/integration tests for acceptance criteria AT-01 through AT-10.

---

### Integration & Navigation

#### [MODIFY] [main.go](file:///d:/Projects/leavemang/cmd/server/main.go)
- Wire up `leave_request.Repository`, `leave_request.Service`, `leave_request.Handler`, and call `leave_request.RegisterRoutes(...)`.

#### [MODIFY] [employee.html](file:///d:/Projects/leavemang/internal/slices/authentication/templates/employee.html)
- Add quick action cards for "Apply for Leave" and "My Leave Requests".

---

## Verification Plan

### Automated Tests
- Run full test suite: `& "C:\Users\basht\go_sdk\go\bin\go.exe" test ./... -v`
- Verify AT-01 through AT-10 pass cleanly:
  - AT-01 — Open Application Form
  - AT-02 — Submit Valid Request (Pending status created)
  - AT-03 — Insufficient Balance rejected
  - AT-04 — Invalid Date Range rejected
  - AT-05 — Missing Reason rejected
  - AT-06 — Overlapping Pending Request rejected
  - AT-07 — Rejected Request Does Not Block new request
  - AT-08 — Inactive Employee rejected
  - AT-09 — Inactive Leave Type rejected
  - AT-10 — Employee Ownership enforced

### Manual Verification
- Launch application server.
- Log in as employee `rahul` (password: `password123`).
- Navigate to `/my/leave-requests/new` and submit a valid leave request for 3 days.
- Verify status is `Pending`, available balance was checked but NOT deducted yet.
- Test date error validation (start date > end date).
- Test overlapping date validation.
