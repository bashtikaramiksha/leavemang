package leave_test

import (
	"context"
	"path/filepath"
	"testing"

	"leavemang/internal/shared/database"
	"leavemang/internal/slices/employee"
	"leavemang/internal/slices/leave"
)

func setupTestDB(t *testing.T) (*leave.Service, *employee.Service) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_leavemang.db")

	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	empRepo := employee.NewRepository(db)
	empService := employee.NewService(empRepo)

	leaveRepo := leave.NewRepository(db)
	leaveService := leave.NewService(leaveRepo, empRepo)

	return leaveService, empService
}

// AT-01 — Create Leave Type
func TestAT01_CreateLeaveType(t *testing.T) {
	svc, _ := setupTestDB(t)
	ctx := context.Background()

	input := leave.CreateLeaveTypeInput{
		Code:              "MAT",
		Name:              "Maternity Leave",
		Description:       "Maternity entitlement for eligible staff",
		DefaultAllocation: 180,
	}

	lt, err := svc.CreateLeaveType(ctx, input)
	if err != nil {
		t.Fatalf("Expected create leave type to succeed, got error: %v", err)
	}

	if lt.ID <= 0 {
		t.Errorf("Expected valid ID, got %d", lt.ID)
	}
	if lt.Code != "MAT" {
		t.Errorf("Expected code MAT, got %s", lt.Code)
	}

	// Verify retrieval
	fetched, err := svc.GetLeaveTypeByID(ctx, lt.ID)
	if err != nil {
		t.Fatalf("Failed to fetch created leave type: %v", err)
	}
	if fetched.Name != "Maternity Leave" {
		t.Errorf("Expected Maternity Leave, got %s", fetched.Name)
	}
}

// AT-02 — Reject Duplicate Leave Type Code
func TestAT02_DuplicateLeaveTypeCode(t *testing.T) {
	svc, _ := setupTestDB(t)
	ctx := context.Background()

	// Initial creation
	_, err := svc.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "TEST_DUP",
		Name:              "Test Duplicate",
		DefaultAllocation: 5,
	})
	if err != nil {
		t.Fatalf("Initial creation failed: %v", err)
	}

	// Duplicate attempt
	_, err = svc.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "TEST_DUP",
		Name:              "Another Duplicate",
		DefaultAllocation: 10,
	})
	if err != leave.ErrDuplicateCode {
		t.Errorf("Expected ErrDuplicateCode, got: %v", err)
	}
}

// AT-03 — Prevent Invalid Allocation (Negative Days)
func TestAT03_InvalidAllocation(t *testing.T) {
	svc, empSvc := setupTestDB(t)
	ctx := context.Background()

	// Get seed employee Rahul
	employees, err := empSvc.ListEmployees(ctx)
	if err != nil || len(employees) == 0 {
		t.Fatalf("Failed to load seed employees: %v", err)
	}

	rahul := employees[0]

	// Get seed leave type Casual Leave
	leaveTypes, err := svc.ListLeaveTypes(ctx)
	if err != nil || len(leaveTypes) == 0 {
		t.Fatalf("Failed to load seed leave types: %v", err)
	}

	cl := leaveTypes[0]

	// Attempt negative allocation
	_, err = svc.AllocateLeave(ctx, leave.AllocateLeaveInput{
		EmployeeID:    rahul.ID,
		LeaveTypeID:   cl.ID,
		AllocatedDays: -5,
	})
	if err != leave.ErrInvalidAllocation {
		t.Errorf("Expected ErrInvalidAllocation for negative days, got: %v", err)
	}
}

// AT-04 & AT-05 & AT-08 — Assign Balance, View Balance, & Calculate Remaining
func TestAT04_AT05_AT08_AssignAndViewBalance(t *testing.T) {
	svc, empSvc := setupTestDB(t)
	ctx := context.Background()

	// Find active employee Rahul (EMP-001)
	employees, err := empSvc.ListEmployees(ctx)
	if err != nil {
		t.Fatalf("Failed to list employees: %v", err)
	}
	var rahul *employee.Employee
	for _, e := range employees {
		if e.EmployeeCode == "EMP-001" {
			rahul = e
			break
		}
	}
	if rahul == nil {
		t.Fatalf("Rahul (EMP-001) not found in seed data")
	}

	// Find active Casual Leave
	leaveTypes, err := svc.ListLeaveTypes(ctx)
	if err != nil {
		t.Fatalf("Failed to list leave types: %v", err)
	}
	var casualLeave *leave.LeaveType
	for _, lt := range leaveTypes {
		if lt.Code == "CL" {
			casualLeave = &lt;
			break
		}
	}
	if casualLeave == nil {
		t.Fatalf("Casual Leave (CL) not found in seed data")
	}

	// AT-04: Assign 12 days
	bal, err := svc.AllocateLeave(ctx, leave.AllocateLeaveInput{
		EmployeeID:    rahul.ID,
		LeaveTypeID:   casualLeave.ID,
		AllocatedDays: 12,
	})
	if err != nil {
		t.Fatalf("Failed to allocate leave: %v", err)
	}

	if bal.AllocatedDays != 12 {
		t.Errorf("Expected 12 allocated days, got %d", bal.AllocatedDays)
	}
	if bal.UsedDays != 0 {
		t.Errorf("Expected 0 used days initially, got %d", bal.UsedDays)
	}

	// AT-08: Check remaining calculation (12 - 0 = 12)
	if bal.Remaining() != 12 {
		t.Errorf("Expected remaining 12 days, got %d", bal.Remaining())
	}

	// AT-05: View balance for Rahul
	bals, err := svc.GetEmployeeBalances(ctx, rahul.ID)
	if err != nil {
		t.Fatalf("Failed to view employee balances: %v", err)
	}

	found := false
	for _, b := range bals {
		if b.LeaveTypeCode == "CL" {
			found = true
			if b.AllocatedDays != 12 || b.UsedDays != 0 || b.Remaining() != 12 {
				t.Errorf("Balance details mismatch: Allocated=%d, Used=%d, Remaining=%d", b.AllocatedDays, b.UsedDays, b.Remaining())
			}
		}
	}
	if !found {
		t.Errorf("Expected Casual Leave balance record in employee balances view")
	}
}

// AT-06 — Inactive Employee Rejection
func TestAT06_InactiveEmployeeAllocation(t *testing.T) {
	svc, empSvc := setupTestDB(t)
	ctx := context.Background()

	// Find inactive employee (EMP-004)
	employees, err := empSvc.ListEmployees(ctx)
	if err != nil {
		t.Fatalf("Failed to list employees: %v", err)
	}
	var inactiveEmp *employee.Employee
	for _, e := range employees {
		if e.Status == employee.StatusInactive {
			inactiveEmp = e
			break
		}
	}
	if inactiveEmp == nil {
		t.Fatalf("Inactive employee not found in seed data")
	}

	leaveTypes, _ := svc.ListLeaveTypes(ctx)

	// Attempt allocation
	_, err = svc.AllocateLeave(ctx, leave.AllocateLeaveInput{
		EmployeeID:    inactiveEmp.ID,
		LeaveTypeID:   leaveTypes[0].ID,
		AllocatedDays: 10,
	})
	if err != leave.ErrInactiveEmployee {
		t.Errorf("Expected ErrInactiveEmployee, got: %v", err)
	}
}

// AT-07 — Inactive Leave Type Rejection
func TestAT07_InactiveLeaveTypeAllocation(t *testing.T) {
	svc, empSvc := setupTestDB(t)
	ctx := context.Background()

	// Create and deactivate a leave type
	lt, err := svc.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "SPECIAL",
		Name:              "Special Project Leave",
		DefaultAllocation: 5,
	})
	if err != nil {
		t.Fatalf("Failed to create leave type: %v", err)
	}

	err = svc.DeactivateLeaveType(ctx, lt.ID)
	if err != nil {
		t.Fatalf("Failed to deactivate leave type: %v", err)
	}

	employees, _ := empSvc.ListEmployees(ctx)
	var activeEmp *employee.Employee
	for _, e := range employees {
		if e.Status == employee.StatusActive {
			activeEmp = e
			break
		}
	}

	// Attempt allocation for inactive leave type
	_, err = svc.AllocateLeave(ctx, leave.AllocateLeaveInput{
		EmployeeID:    activeEmp.ID,
		LeaveTypeID:   lt.ID,
		AllocatedDays: 5,
	})
	if err != leave.ErrInactiveLeaveType {
		t.Errorf("Expected ErrInactiveLeaveType, got: %v", err)
	}
}
