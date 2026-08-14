# Implementation Plan - VS-03: Leave Types & Leave Balance

VS-03 introduces leave configuration and balance capabilities into the leave management system. It builds directly upon VS-02 (Employee Management) and provides essential foundation for VS-04 (Apply Leave).

## User Review Required
> [!IMPORTANT]
> - Database schema will be updated in `internal/shared/database/db.go` with `leave_types` and `leave_balances` tables.
> - Default seed data for initial leave types (Casual Leave, Sick Leave, Earned Leave) will be seeded on initialization.
> - Routes under `/leave-types*` and `/leave-balances/new` / `POST /leave-balances` are restricted to Admin role, while `/my/leave-balances` is accessible to all authenticated users.

## Proposed Changes

### Database Layer
#### [MODIFY] [db.go](file:///d:/Projects/leavemang/internal/shared/database/db.go)
- Add `leave_types` table schema: `id`, `code`, `name`, `description`, `default_allocation`, `status`, `created_at`, `updated_at`.
- Add `leave_balances` table schema: `id`, `employee_id`, `leave_type_id`, `allocated_days`, `used_days`, `created_at`, `updated_at`, with `UNIQUE(employee_id, leave_type_id)`.
- Update `Seed` method to seed default leave types (Casual Leave CL, Sick Leave SL, Earned Leave EL) and initial leave allocations for active employees.

### Domain Slice: Leave Management (`internal/slices/leave`)
#### [NEW] [model.go](file:///d:/Projects/leavemang/internal/slices/leave/model.go)
- Define `LeaveType` struct and `LeaveBalance` struct.
- Define `LeaveBalanceWithDetails` struct (joining employee and leave type info for display).
- Implement `Remaining()` method on `LeaveBalance` (`allocated_days - used_days`).

#### [NEW] [repository.go](file:///d:/Projects/leavemang/internal/slices/leave/repository.go)
- Implement `LeaveTypeRepository`:
  - `Create`, `GetByID`, `GetByCode`, `List`, `Update`, `Activate`, `Deactivate`.
- Implement `LeaveBalanceRepository`:
  - `CreateOrUpdateAllocation`, `GetByID`, `GetByEmployeeID`, `GetByEmployeeAndType`, `ListAllBalances`.
- Use transaction semantics where appropriate.

#### [NEW] [service.go](file:///d:/Projects/leavemang/internal/slices/leave/service.go)
- Implement `LeaveTypeService`:
  - Validate non-empty code and name.
  - Reject duplicate code on creation/update.
  - Enforce non-negative default allocation.
- Implement `LeaveBalanceService`:
  - Validate existing employee & active status (BR-08).
  - Validate existing leave type & active status (BR-07).
  - Validate non-negative allocation (BR-02).
  - Provide `GetBalance(employeeID, leaveTypeID)` and `GetEmployeeBalances(employeeID)` for VS-04 handoff and UI rendering.

#### [NEW] [handler.go](file:///d:/Projects/leavemang/internal/slices/leave/handler.go)
- Admin Handlers:
  - `HandleListLeaveTypes`, `HandleNewLeaveTypeForm`, `HandleCreateLeaveType`
  - `HandleEditLeaveTypeForm`, `HandleUpdateLeaveType`
  - `HandleActivateLeaveType`, `HandleDeactivateLeaveType`
  - `HandleListBalances`, `HandleNewAllocationForm`, `HandleCreateAllocation`
- Employee Handler:
  - `HandleMyLeaveBalances` (renders employee's balance view)

#### [NEW] [routes.go](file:///d:/Projects/leavemang/internal/slices/leave/routes.go)
- Register GET and POST routes for `/leave-types*`, `/leave-balances*`, and `/my/leave-balances`.
- Apply `m.Authenticate` and `m.RequireRole(authentication.RoleAdmin)` middleware.

#### [NEW] UI Templates (`internal/slices/leave/templates/`)
- `leave_types_list.html`: Table of leave types with Edit/Deactivate/Activate actions.
- `leave_type_form.html`: Form for creating/editing a leave type.
- `allocation_form.html`: Admin form to assign leave allocation to an employee.
- `admin_balances_list.html`: Admin overview of employee leave allocations.
- `my_balances.html`: Employee's view of leave allocations, used days, and remaining days.

#### [NEW] [leave_test.go](file:///d:/Projects/leavemang/internal/slices/leave/leave_test.go)
- Unit and integration tests for AT-01 through AT-08.

### Main Entrypoint
#### [MODIFY] [main.go](file:///d:/Projects/leavemang/cmd/server/main.go)
- Initialize `leave.Repository`, `leave.Service`, `leave.Handler`.
- Register routes for `leave.RegisterRoutes`.

## Verification Plan

### Automated Tests
- Run `go test ./... -v` to ensure all slice unit and integration tests (AT-01 to AT-08) pass.

### Manual Verification
- Login as Admin (amit / password123):
  - Access `/leave-types`, create new leave type (e.g. Maternity Leave ML, 180 days).
  - Edit existing leave type.
  - Deactivate / Activate leave type.
  - Access `/leave-balances/new`, assign Casual Leave 12 days to Rahul Patil.
  - Verify error when attempting to allocate to inactive employee or inactive leave type or negative days.
- Login as Employee (rahul / password123):
  - Access `/my/leave-balances` and verify allocated (12), used (0), remaining (12) display.
