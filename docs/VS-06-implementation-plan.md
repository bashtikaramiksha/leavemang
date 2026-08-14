# VS-06 — Leave History & Dashboard Implementation Plan

## Proposed Architecture & Design

VS-06 provides read-only dashboards and reporting for both Employees and Managers.

### 1. Data Layer & Repository Additions (`internal/slices/leave_request/repository.go` and/or `internal/slices/leave/repository.go`)
- `ListByScope(ctx context.Context, status string, employeeID int64) ([]*LeaveRequestWithDetails, error)`: Fetches team leave requests with optional filters for status (`pending`, `approved`, `rejected`, or `all`) and `employee_id`.
- `GetStatistics(ctx context.Context) (*DashboardStats, error)`: Computes `TotalRequests`, `PendingRequests`, `ApprovedRequests`, `RejectedRequests`, and `ApprovedLeaveDays`.
- `GetEmployeeBalances(ctx context.Context, employeeID int64)`: Retrieves balances for an employee with leave type metadata.

### 2. Service Layer (`internal/slices/leave_dashboard` or `internal/slices/leave_request`)
Create `LeaveDashboardService` or extend slice services to provide:
- `GetEmployeeDashboard(ctx context.Context, userID int64)`: Returns employee balances and recent leave requests.
- `GetEmployeeHistory(ctx context.Context, userID int64)`: Returns complete list of employee leave requests.
- `GetEmployeeBalances(ctx context.Context, userID int64)`: Returns leave balances for the logged-in employee.
- `GetManagerDashboard(ctx context.Context, managerUserID int64, statusFilter string, employeeIDFilter int64)`: Returns list of managed employees, filtered leave requests, and summary statistics.

### 3. Handlers & Templates (`internal/slices/leave_dashboard/` or `internal/slices/leave_request/`)
- `GET /my/dashboard`: Renders employee dashboard template (`employee_dashboard.html`).
- `GET /my/leave-requests`: Renders employee history template (`employee_history.html`).
- `GET /my/leave-balances`: Renders employee leave balances template (`employee_balances.html`).
- `GET /manager/dashboard`: Renders manager dashboard template (`manager_dashboard.html`).
- `GET /manager/leave-requests`: Renders full or HTMX partial requests table (`manager_requests_table.html`).

### 4. Security & Data Isolation (BR-01, BR-02)
- `/my/*` routes verify the session user ID belongs to an active employee and scoped strictly to that employee's data.
- `/manager/*` routes enforce `RequireRole(RoleManager, RoleAdmin)`.

### 5. Verification Plan
- Unit tests & Integration tests covering AT-01 through AT-10.
- Verify HTMX filtering and dashboard statistics calculation.
