# Implementation Plan — VS-02 Employee Management

## 1. Overview
VS-02 introduces employee record management while integrating with VS-01's authentication and role-based access control. Admin users can create, view, edit, activate, and deactivate employees, while Managers and Employees are constrained to viewing permitted employee details.

## 2. Proposed Changes

### Database Schema Updates
- Update `internal/shared/database/db.go`:
  - Add `employees` table: `id`, `user_id`, `employee_code`, `first_name`, `last_name`, `email`, `phone`, `department`, `designation`, `joining_date`, `status`, `created_at`, `updated_at`.
  - Update `Seed(db)` to populate corresponding Employee records for default seed users (`rahul`, `priya`, `amit`, `inactive_user`).

### New Employee Slice (`internal/slices/employee`)
- `model.go`:
  - `Employee` struct.
  - `CreateEmployeeInput` and `UpdateEmployeeInput` DTOs with validation rules (BR-02, BR-03, BR-04).
- `repository.go`:
  - `EmployeeRepository` with DB methods: `Create`, `GetByID`, `GetByUserID`, `GetByEmail`, `List`, `Update`, `SetStatus`, auto-generating unique `employee_code` (e.g. `EMP-001`).
  - Atomically manages User + Employee creation/updates in SQLite transactions.
- `service.go`:
  - Enforces BR-01 to BR-07: Email uniqueness check, required fields validation, status rules, authorization validation.
- `authorization.go`:
  - Role-based authorization matrix enforcement (Admin full access; Manager view list/details; Employee view own details).
- `handler.go`:
  - Handlers for `/employees` (GET list, GET/POST form, GET details, GET/POST edit, POST activate/deactivate) & `/profile`.
  - HTMX fragment and full HTML page rendering.
- `routes.go`:
  - Mounts routes on Chi router with authentication and role middlewares.
- `employee_test.go`:
  - Comprehensive unit/integration tests covering AT-01 to AT-07.

### Application Integration
- Update `cmd/server/main.go`:
  - Wire `EmployeeRepository`, `EmployeeService`, `EmployeeHandler`, and `RegisterRoutes`.
- Update templates & CSS:
  - Add navigation bar links for Employees and Profile in existing portal templates.

## 3. Verification Plan

### Automated Tests
- Run `go test ./... -v` to ensure all VS-01 and VS-02 unit/integration tests pass.

### Manual Verification
- Log in as Admin (`amit`) -> Create new employee, edit employee details, deactivate/activate employee.
- Log in as Manager (`priya`) -> Verify view-only access to employee list/details, reject create/edit/deactivate attempts.
- Log in as Employee (`rahul`) -> Verify access to own profile, reject list/create/edit endpoints with 403.
