package employee_test

import (
	"context"
	"path/filepath"
	"testing"

	"leavemang/internal/shared/database"
	"leavemang/internal/slices/authentication"
	"leavemang/internal/slices/employee"
)

func setupTestDB(t *testing.T) *employee.Service {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test_employee.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	repo := employee.NewRepository(db)
	return employee.NewService(repo)
}

func TestAT01_CreateEmployee(t *testing.T) {
	svc := setupTestDB(t)
	ctx := context.Background()

	input := employee.CreateEmployeeInput{
		FirstName:   "Vikram",
		LastName:    "Rao",
		Email:       "vikram.rao@example.com",
		Phone:       "9876543299",
		Department:  "Engineering",
		Designation: "Backend Developer",
		JoiningDate: "2026-08-10",
		Role:        employee.RoleEmployee,
	}

	emp, err := svc.CreateEmployee(ctx, input)
	if err != nil {
		t.Fatalf("expected successful employee creation, got: %v", err)
	}

	if emp.ID == 0 {
		t.Errorf("expected non-zero employee ID")
	}
	if emp.FirstName != "Vikram" || emp.LastName != "Rao" {
		t.Errorf("unexpected employee name: %s %s", emp.FirstName, emp.LastName)
	}
	if emp.Email != "vikram.rao@example.com" {
		t.Errorf("unexpected employee email: %s", emp.Email)
	}
	if emp.Status != employee.StatusActive {
		t.Errorf("expected status active, got %s", emp.Status)
	}
	if emp.EmployeeCode == "" {
		t.Errorf("expected generated employee code, got empty")
	}
}

func TestAT02_DuplicateEmailRejection(t *testing.T) {
	svc := setupTestDB(t)
	ctx := context.Background()

	input := employee.CreateEmployeeInput{
		FirstName:   "Duplicate",
		LastName:    "Test",
		Email:       "rahul@example.com", // Seed user rahul already exists
		Phone:       "1234567890",
		Department:  "IT",
		Designation: "Developer",
		JoiningDate: "2026-08-10",
		Role:        employee.RoleEmployee,
	}

	_, err := svc.CreateEmployee(ctx, input)
	if err == nil {
		t.Fatalf("expected error when creating employee with duplicate email, got nil")
	}
	if err != employee.ErrDuplicateEmail {
		t.Errorf("expected ErrDuplicateEmail, got: %v", err)
	}
}

func TestAT03_UnauthorizedAuthorizationMatrix(t *testing.T) {
	adminUser := &authentication.User{ID: 1, Role: authentication.RoleAdmin}
	managerUser := &authentication.User{ID: 2, Role: authentication.RoleManager}
	employeeUser := &authentication.User{ID: 3, Role: authentication.RoleEmployee}

	targetEmp := &employee.Employee{ID: 10, UserID: 3}
	otherEmp := &employee.Employee{ID: 11, UserID: 99}

	// CanManageEmployee
	if !employee.CanManageEmployee(adminUser) {
		t.Errorf("Admin should be able to manage employees")
	}
	if employee.CanManageEmployee(managerUser) {
		t.Errorf("Manager should NOT be able to manage employees")
	}
	if employee.CanManageEmployee(employeeUser) {
		t.Errorf("Employee should NOT be able to manage employees")
	}

	// CanViewEmployeeList
	if !employee.CanViewEmployeeList(adminUser) || !employee.CanViewEmployeeList(managerUser) {
		t.Errorf("Admin and Manager should be able to view employee list")
	}
	if employee.CanViewEmployeeList(employeeUser) {
		t.Errorf("Employee should NOT be able to view employee list")
	}

	// CanViewEmployeeDetails
	if !employee.CanViewEmployeeDetails(employeeUser, targetEmp) {
		t.Errorf("Employee should be able to view their own profile")
	}
	if employee.CanViewEmployeeDetails(employeeUser, otherEmp) {
		t.Errorf("Employee should NOT be able to view another employee profile")
	}
}

func TestAT04_EditEmployee(t *testing.T) {
	svc := setupTestDB(t)
	ctx := context.Background()

	// Fetch existing seed employee Rahul (ID 1)
	emp, err := svc.GetEmployee(ctx, 1)
	if err != nil {
		t.Fatalf("failed to fetch seed employee: %v", err)
	}

	updateInput := employee.UpdateEmployeeInput{
		FirstName:   emp.FirstName,
		LastName:    emp.LastName,
		Email:       emp.Email,
		Phone:       emp.Phone,
		Department:  "DevOps", // Changed department
		Designation: "Senior Lead",
		JoiningDate: emp.JoiningDate,
		Role:        employee.RoleManager, // Changed role
		Status:      emp.Status,
	}

	updatedEmp, err := svc.UpdateEmployee(ctx, emp.ID, updateInput)
	if err != nil {
		t.Fatalf("failed to update employee: %v", err)
	}

	if updatedEmp.Department != "DevOps" {
		t.Errorf("expected department DevOps, got %s", updatedEmp.Department)
	}
	if updatedEmp.Role != employee.RoleManager {
		t.Errorf("expected updated role manager, got %s", updatedEmp.Role)
	}
}

func TestAT05_AT06_DeactivateAndActivateEmployee(t *testing.T) {
	svc := setupTestDB(t)
	ctx := context.Background()

	// Deactivate active employee (Rahul ID 1)
	err := svc.DeactivateEmployee(ctx, 1)
	if err != nil {
		t.Fatalf("failed to deactivate employee: %v", err)
	}

	emp, err := svc.GetEmployee(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get employee: %v", err)
	}
	if emp.Status != employee.StatusInactive {
		t.Errorf("expected inactive status, got %s", emp.Status)
	}

	// Re-activate employee
	err = svc.ActivateEmployee(ctx, 1)
	if err != nil {
		t.Fatalf("failed to activate employee: %v", err)
	}

	empActive, err := svc.GetEmployee(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get employee: %v", err)
	}
	if empActive.Status != employee.StatusActive {
		t.Errorf("expected active status, got %s", empActive.Status)
	}
}

func TestAT07_EmployeeViewOwnProfile(t *testing.T) {
	svc := setupTestDB(t)
	ctx := context.Background()

	// Rahul's user_id is 1
	emp, err := svc.GetEmployeeByUserID(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get employee profile by user ID: %v", err)
	}

	if emp.Email != "rahul@example.com" {
		t.Errorf("expected rahul@example.com, got %s", emp.Email)
	}
}
