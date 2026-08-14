package leave_request_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"leavemang/internal/shared/database"
	"leavemang/internal/slices/authentication"
	"leavemang/internal/slices/employee"
	"leavemang/internal/slices/leave"
	"leavemang/internal/slices/leave_request"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	return db
}

type testServices struct {
	db           *sql.DB
	authRepo     *authentication.Repository
	authService  *authentication.Service
	middleware   *authentication.Middleware
	empRepo      *employee.Repository
	empService   *employee.Service
	leaveRepo    *leave.Repository
	leaveService *leave.Service
	reqRepo      *leave_request.Repository
	reqService   *leave_request.Service
	reqHandler   *leave_request.Handler
	router       *chi.Mux
}

func initTestServices(t *testing.T) *testServices {
	db := setupTestDB(t)

	authRepo := authentication.NewRepository(db)
	authService := authentication.NewService(authRepo)
	middleware := authentication.NewMiddleware(authService)

	empRepo := employee.NewRepository(db)
	empService := employee.NewService(empRepo)

	leaveRepo := leave.NewRepository(db)
	leaveService := leave.NewService(leaveRepo, empRepo)

	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	templateDir := filepath.Join(workDir, "templates")
	layoutPath := filepath.Join(workDir, "..", "authentication", "templates", "layout.html")

	reqRepo := leave_request.NewRepository(db)
	reqService := leave_request.NewService(reqRepo, empService, leaveService)
	reqHandler := leave_request.NewHandler(reqService, leaveService, layoutPath, templateDir)

	r := chi.NewRouter()
	leave_request.RegisterRoutes(r, reqHandler, middleware)

	return &testServices{
		db:           db,
		authRepo:     authRepo,
		authService:  authService,
		middleware:   middleware,
		empRepo:      empRepo,
		empService:   empService,
		leaveRepo:    leaveRepo,
		leaveService: leaveService,
		reqRepo:      reqRepo,
		reqService:   reqService,
		reqHandler:   reqHandler,
		router:       r,
	}
}

// createTestEmployeeHelper creates user, employee, and active leave type balance setup
func createTestEmployeeHelper(t *testing.T, ts *testServices, username, ltCode, ltStatus string, allocDays int) (*authentication.User, *employee.Employee, *leave.LeaveType) {
	ctx := context.Background()

	emp, err := ts.empService.CreateEmployee(ctx, employee.CreateEmployeeInput{
		FirstName:   username,
		LastName:    "User",
		Email:       username + "@example.com",
		Department:  "Engineering",
		Designation: "Developer",
		JoiningDate: "2026-01-01",
		Role:        employee.RoleEmployee,
	})
	if err != nil {
		t.Fatalf("failed to create employee: %v", err)
	}

	user, err := ts.authRepo.GetUserByID(emp.UserID)
	if err != nil {
		t.Fatalf("failed to get user for employee: %v", err)
	}

	lt, err := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              ltCode,
		Name:              ltCode + " Leave",
		Description:       "Test Leave Type",
		DefaultAllocation: allocDays,
	})
	if err != nil {
		t.Fatalf("failed to create leave type: %v", err)
	}

	if ltStatus == leave.StatusInactive {
		err = ts.leaveService.DeactivateLeaveType(ctx, lt.ID)
		if err != nil {
			t.Fatalf("failed to deactivate leave type: %v", err)
		}
		lt, _ = ts.leaveService.GetLeaveTypeByID(ctx, lt.ID)
	}

	if lt.IsActive() && emp.Status == employee.StatusActive {
		_, err = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{
			EmployeeID:    emp.ID,
			LeaveTypeID:   lt.ID,
			AllocatedDays: allocDays,
		})
		if err != nil {
			t.Fatalf("failed to allocate leave: %v", err)
		}
	}

	return user, emp, lt
}

func createAuthenticatedCookie(t *testing.T, ts *testServices, user *authentication.User) *http.Cookie {
	sess, err := ts.authService.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	return &http.Cookie{
		Name:  "session_id",
		Value: sess.ID,
	}
}

// AT-01 — Open Application Form
func TestAT01_OpenApplicationForm(t *testing.T) {
	ts := initTestServices(t)
	user, _, _ := createTestEmployeeHelper(t, ts, "emp1", "CL1", leave.StatusActive, 10)
	cookie := createAuthenticatedCookie(t, ts, user)

	req := httptest.NewRequest("GET", "/my/leave-requests/new", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Code)
	}
}

// AT-02 — Submit Valid Request
func TestAT02_SubmitValidRequest(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp2", "CL2", leave.StatusActive, 8)

	req, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-12",
		Reason:      "Personal work",
	})
	if err != nil {
		t.Fatalf("expected successful creation, got err: %v", err)
	}

	if req.Days != 3 {
		t.Errorf("expected 3 days, got %d", req.Days)
	}
	if req.Status != leave_request.StatusPending {
		t.Errorf("expected Pending status, got %s", req.Status)
	}
	if req.EmployeeID != emp.ID {
		t.Errorf("expected employee ID %d, got %d", emp.ID, req.EmployeeID)
	}

	// Verify balance is NOT deducted yet (Allocated=8, Used=0)
	bal, err := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	if err != nil {
		t.Fatalf("failed to fetch balance: %v", err)
	}
	if bal.UsedDays != 0 || bal.Remaining() != 8 {
		t.Errorf("balance should remain un-deducted (used: %d, remaining: %d)", bal.UsedDays, bal.Remaining())
	}
}

// AT-03 — Insufficient Balance
func TestAT03_InsufficientBalance(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, _, lt := createTestEmployeeHelper(t, ts, "emp3", "SL3", leave.StatusActive, 2)

	_, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-14", // 5 days requested vs 2 remaining
		Reason:      "Medical recovery",
	})
	if err == nil {
		t.Fatal("expected request to be rejected due to insufficient balance, but it succeeded")
	}

	if !strings.Contains(err.Error(), "remaining") {
		t.Errorf("expected error message mentioning remaining balance, got: %v", err)
	}
}

// AT-04 — Invalid Date Range
func TestAT04_InvalidDateRange(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, _, lt := createTestEmployeeHelper(t, ts, "emp4", "CL4", leave.StatusActive, 10)

	_, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-15",
		ToDate:      "2026-08-10", // Start after end
		Reason:      "Vacation",
	})
	if err == nil {
		t.Fatal("expected invalid date range error, but request succeeded")
	}
}

// AT-05 — Missing Reason
func TestAT05_MissingReason(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, _, lt := createTestEmployeeHelper(t, ts, "emp5", "CL5", leave.StatusActive, 10)

	_, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-12",
		Reason:      "   ", // Empty after trim
	})
	if err == nil {
		t.Fatal("expected missing reason error, but request succeeded")
	}
}

// AT-06 — Overlapping Pending Request
func TestAT06_OverlappingPendingRequest(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, _, lt := createTestEmployeeHelper(t, ts, "emp6", "CL6", leave.StatusActive, 15)

	// Create initial pending request (10 Aug - 12 Aug)
	_, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-12",
		Reason:      "First request",
	})
	if err != nil {
		t.Fatalf("failed to create first request: %v", err)
	}

	// Submit overlapping request (11 Aug - 13 Aug)
	_, err = ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-11",
		ToDate:      "2026-08-13",
		Reason:      "Second overlapping request",
	})
	if err == nil {
		t.Fatal("expected overlapping request error, but it succeeded")
	}
}

// AT-07 — Rejected Request Does Not Block
func TestAT07_RejectedRequestDoesNotBlock(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp7", "CL7", leave.StatusActive, 15)

	// Manually insert a Rejected request in DB for 10-12 Aug
	_, err := ts.db.Exec(`
		INSERT INTO leave_requests (employee_id, leave_type_id, from_date, to_date, days, reason, status)
		VALUES (?, ?, '2026-08-10', '2026-08-12', 3, 'Old request', 'Rejected')
	`, emp.ID, lt.ID)
	if err != nil {
		t.Fatalf("failed to seed rejected request: %v", err)
	}

	// Submit new request for 11-13 Aug (overlaps with rejected request)
	req, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-11",
		ToDate:      "2026-08-13",
		Reason:      "New request after rejection",
	})
	if err != nil {
		t.Fatalf("expected new request to succeed despite old rejected request, got err: %v", err)
	}
	if req.Status != leave_request.StatusPending {
		t.Errorf("expected Pending status, got %s", req.Status)
	}
}

// AT-08 — Inactive Employee
func TestAT08_InactiveEmployee(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp8", "CL8", leave.StatusActive, 10)

	// Soft-deactivate employee
	err := ts.empService.DeactivateEmployee(ctx, emp.ID)
	if err != nil {
		t.Fatalf("failed to deactivate employee: %v", err)
	}

	_, err = ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-12",
		Reason:      "Attempt leave while inactive",
	})
	if err == nil {
		t.Fatal("expected inactive employee request to be rejected, but it succeeded")
	}
}

// AT-09 — Inactive Leave Type
func TestAT09_InactiveLeaveType(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	user, _, lt := createTestEmployeeHelper(t, ts, "emp9", "EL9", leave.StatusInactive, 10)

	_, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-12",
		Reason:      "Attempt inactive leave type",
	})
	if err == nil {
		t.Fatal("expected inactive leave type request to be rejected, but it succeeded")
	}
}

// AT-10 — Employee Ownership
func TestAT10_EmployeeOwnership(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()
	userA, empA, ltA := createTestEmployeeHelper(t, ts, "userA", "CLA", leave.StatusActive, 10)
	userB, empB, _ := createTestEmployeeHelper(t, ts, "userB", "SLB", leave.StatusActive, 10)

	// User A submits request
	reqA, err := ts.reqService.CreateRequest(ctx, userA.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: ltA.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-12",
		Reason:      "User A leave",
	})
	if err != nil {
		t.Fatalf("user A create failed: %v", err)
	}

	if reqA.EmployeeID != empA.ID {
		t.Errorf("request employee ID should be %d, got %d", empA.ID, reqA.EmployeeID)
	}

	// User B tries to view User A's request details -> should be rejected with Unauthorized error
	_, err = ts.reqService.GetRequest(ctx, reqA.ID, userB.ID)
	if err == nil {
		t.Fatal("user B should not be able to view user A's request details")
	}

	// List User B's requests -> should not contain User A's request
	listB, err := ts.reqService.ListEmployeeRequests(ctx, userB.ID)
	if err != nil {
		t.Fatalf("failed to list user B requests: %v", err)
	}
	if len(listB) != 0 {
		t.Errorf("expected 0 requests for User B, got %d", len(listB))
	}

	// List User A's requests -> should contain 1 request
	listA, err := ts.reqService.ListEmployeeRequests(ctx, userA.ID)
	if err != nil {
		t.Fatalf("failed to list user A requests: %v", err)
	}
	if len(listA) != 1 || listA[0].ID != reqA.ID {
		t.Errorf("expected User A to have request %d, got list %v", reqA.ID, listA)
	}

	_ = empB
}

// Test HTTP Endpoint End-to-End Submission
func TestHTTPEndpoint_SubmitAndListRequests(t *testing.T) {
	ts := initTestServices(t)
	user, _, lt := createTestEmployeeHelper(t, ts, "httpuser", "CLHTTP", leave.StatusActive, 10)
	cookie := createAuthenticatedCookie(t, ts, user)

	// 1. Submit POST request
	formValues := url.Values{}
	formValues.Set("leave_type_id", strconv.FormatInt(lt.ID, 10))
	formValues.Set("from_date", "2026-08-10")
	formValues.Set("to_date", "2026-08-12")
	formValues.Set("reason", "HTTP Test Reason")

	postReq := httptest.NewRequest("POST", "/my/leave-requests", strings.NewReader(formValues.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(cookie)
	wPost := httptest.NewRecorder()

	ts.router.ServeHTTP(wPost, postReq)

	if wPost.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 SeeOther redirect after creation, got status %d", wPost.Code)
	}

	// 2. Fetch list view
	getReq := httptest.NewRequest("GET", "/my/leave-requests", nil)
	getReq.AddCookie(cookie)
	wGet := httptest.NewRecorder()

	ts.router.ServeHTTP(wGet, getReq)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list view, got status %d", wGet.Code)
	}
}

// Helper to create a manager user for test assertions
func createTestManagerHelper(t *testing.T, ts *testServices, username string) *authentication.User {
	ctx := context.Background()
	emp, err := ts.empService.CreateEmployee(ctx, employee.CreateEmployeeInput{
		FirstName:   username,
		LastName:    "Manager",
		Email:       username + "@example.com",
		Department:  "Management",
		Designation: "Manager",
		JoiningDate: "2026-01-01",
		Role:        employee.RoleManager,
	})
	if err != nil {
		t.Fatalf("failed to create manager employee: %v", err)
	}

	user, err := ts.authRepo.GetUserByID(emp.UserID)
	if err != nil {
		t.Fatalf("failed to get user for manager: %v", err)
	}
	return user
}

// AT-01: Manager can view pending leave requests
func TestAT01_ViewPendingRequests(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_pending", "CL01", leave.StatusActive, 10)
	manager := createTestManagerHelper(t, ts, "manager01")
	cookie := createAuthenticatedCookie(t, ts, manager)

	ctx := context.Background()
	_, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-20",
		ToDate:      "2026-08-22",
		Reason:      "Vacation",
	})
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	// HTTP GET /manager/leave-requests
	req := httptest.NewRequest("GET", "/manager/leave-requests", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	ts.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for manager pending list, got %d", w.Code)
	}

	pendingList, err := ts.reqService.ListPendingRequests(ctx)
	if err != nil {
		t.Fatalf("failed to list pending requests: %v", err)
	}

	if len(pendingList) != 1 {
		t.Fatalf("expected 1 pending request, got %d", len(pendingList))
	}
	if pendingList[0].EmployeeID != emp.ID {
		t.Errorf("expected employee ID %d, got %d", emp.ID, pendingList[0].EmployeeID)
	}
}

// AT-02: Approve Request updates status to Approved, increases used days, records reviewer & timestamp
func TestAT02_ApproveRequest(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_approve", "CL02", leave.StatusActive, 12)
	manager := createTestManagerHelper(t, ts, "manager02")

	ctx := context.Background()
	req, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-20",
		ToDate:      "2026-08-22", // 3 days
		Reason:      "Personal work",
	})
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	err = ts.reqService.ApproveRequest(ctx, req.ID, manager.ID)
	if err != nil {
		t.Fatalf("expected approval to succeed, got: %v", err)
	}

	// Verify updated request
	updatedReq, err := ts.reqRepo.GetByIDOnly(ctx, req.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated request: %v", err)
	}
	if updatedReq.Status != leave_request.StatusApproved {
		t.Errorf("expected status Approved, got %s", updatedReq.Status)
	}
	if !updatedReq.ReviewedBy.Valid || updatedReq.ReviewedBy.Int64 != manager.ID {
		t.Errorf("expected reviewed_by %d, got %v", manager.ID, updatedReq.ReviewedBy)
	}
	if !updatedReq.ReviewedAt.Valid {
		t.Error("expected reviewed_at to be populated")
	}

	// Verify balance update
	bal, err := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	if err != nil {
		t.Fatalf("failed to get balance: %v", err)
	}
	if bal.UsedDays != 3 {
		t.Errorf("expected used_days = 3, got %d", bal.UsedDays)
	}
	if bal.Remaining() != 9 { // 12 - 3
		t.Errorf("expected remaining = 9, got %d", bal.Remaining())
	}
}

// AT-03: Reject Request updates status to Rejected, stores rejection_reason, balance remains unchanged
func TestAT03_RejectRequest(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_reject", "CL03", leave.StatusActive, 10)
	manager := createTestManagerHelper(t, ts, "manager03")

	ctx := context.Background()
	req, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-25",
		ToDate:      "2026-08-25", // 1 day
		Reason:      "Sick leave",
	})
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	err = ts.reqService.RejectRequest(ctx, req.ID, manager.ID, "Insufficient justification provided")
	if err != nil {
		t.Fatalf("expected rejection to succeed, got: %v", err)
	}

	// Verify updated request
	updatedReq, err := ts.reqRepo.GetByIDOnly(ctx, req.ID)
	if err != nil {
		t.Fatalf("failed to fetch updated request: %v", err)
	}
	if updatedReq.Status != leave_request.StatusRejected {
		t.Errorf("expected status Rejected, got %s", updatedReq.Status)
	}
	if !updatedReq.RejectionReason.Valid || updatedReq.RejectionReason.String != "Insufficient justification provided" {
		t.Errorf("expected rejection_reason 'Insufficient justification provided', got %v", updatedReq.RejectionReason)
	}

	// Verify balance remains unchanged
	bal, err := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	if err != nil {
		t.Fatalf("failed to get balance: %v", err)
	}
	if bal.UsedDays != 0 {
		t.Errorf("expected used_days = 0, got %d", bal.UsedDays)
	}
}

// AT-04: Double Approval attempt fails and does not deduct balance again
func TestAT04_DoubleApproval(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_dbl_app", "CL04", leave.StatusActive, 10)
	manager := createTestManagerHelper(t, ts, "manager04")

	ctx := context.Background()
	req, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-15",
		ToDate:      "2026-08-16", // 2 days
		Reason:      "Vacation",
	})
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	// First approval
	if err := ts.reqService.ApproveRequest(ctx, req.ID, manager.ID); err != nil {
		t.Fatalf("first approval failed: %v", err)
	}

	// Second approval attempt
	err = ts.reqService.ApproveRequest(ctx, req.ID, manager.ID)
	if err == nil {
		t.Fatal("expected second approval to fail, but it succeeded")
	}

	// Check balance is deducted only once (2 days)
	bal, _ := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	if bal.UsedDays != 2 {
		t.Errorf("expected used_days = 2 after double approval attempt, got %d", bal.UsedDays)
	}
}

// AT-05: Cannot approve an already rejected request
func TestAT05_ApproveRejectedRequest(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_rej_app", "CL05", leave.StatusActive, 10)
	manager := createTestManagerHelper(t, ts, "manager05")

	ctx := context.Background()
	req, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-15",
		ToDate:      "2026-08-15",
		Reason:      "Off",
	})

	// Reject first
	_ = ts.reqService.RejectRequest(ctx, req.ID, manager.ID, "Not allowed")

	// Attempt approval
	err := ts.reqService.ApproveRequest(ctx, req.ID, manager.ID)
	if err == nil {
		t.Fatal("expected approving a rejected request to fail, but it succeeded")
	}

	bal, _ := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	if bal.UsedDays != 0 {
		t.Errorf("expected used_days = 0, got %d", bal.UsedDays)
	}
}

// AT-06: Employee role cannot approve requests (403 Forbidden)
func TestAT06_EmployeeCannotApprove(t *testing.T) {
	ts := initTestServices(t)
	user, _, lt := createTestEmployeeHelper(t, ts, "emp_unauth", "CL06", leave.StatusActive, 10)
	employeeCookie := createAuthenticatedCookie(t, ts, user)

	ctx := context.Background()
	req, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-15",
		ToDate:      "2026-08-15",
		Reason:      "Off",
	})

	// Employee POST /manager/leave-requests/{id}/approve
	postReq := httptest.NewRequest("POST", "/manager/leave-requests/"+strconv.FormatInt(req.ID, 10)+"/approve", nil)
	postReq.AddCookie(employeeCookie)
	w := httptest.NewRecorder()

	ts.router.ServeHTTP(w, postReq)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for employee role, got %d", w.Code)
	}
}

// AT-07: Insufficient balance during approval fails and leaves request Pending
func TestAT07_InsufficientBalanceDuringApproval(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_low_bal", "CL07", leave.StatusActive, 5)
	manager := createTestManagerHelper(t, ts, "manager07")

	ctx := context.Background()
	req, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-13", // 4 days requested
		Reason:      "Leave",
	})

	// Manually reduce allocation so available balance is only 2 days
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{
		EmployeeID:    emp.ID,
		LeaveTypeID:   lt.ID,
		AllocatedDays: 2,
	})

	// Manager attempts approval
	err := ts.reqService.ApproveRequest(ctx, req.ID, manager.ID)
	if err == nil {
		t.Fatal("expected approval to fail due to insufficient balance, but it succeeded")
	}

	// Verify request remains Pending
	updatedReq, _ := ts.reqRepo.GetByIDOnly(ctx, req.ID)
	if updatedReq.Status != leave_request.StatusPending {
		t.Errorf("expected status to remain Pending, got %s", updatedReq.Status)
	}

	// Verify balance remains 0 used
	bal, _ := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	if bal.UsedDays != 0 {
		t.Errorf("expected used_days = 0, got %d", bal.UsedDays)
	}
}

// AT-08: Transaction Consistency - rollback ensures request remains Pending if balance update fails
func TestAT08_TransactionConsistency(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_tx", "CL08", leave.StatusActive, 10)
	manager := createTestManagerHelper(t, ts, "manager08")

	ctx := context.Background()
	req, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-11", // 2 days
		Reason:      "Leave",
	})

	// Delete leave balance row to simulate failure during balance update
	_, err := ts.db.Exec("DELETE FROM leave_balances WHERE employee_id = ? AND leave_type_id = ?", emp.ID, lt.ID)
	if err != nil {
		t.Fatalf("failed to delete balance: %v", err)
	}

	// Attempt approval
	err = ts.reqService.ApproveRequest(ctx, req.ID, manager.ID)
	if err == nil {
		t.Fatal("expected approval to fail due to missing balance record")
	}

	// Verify transaction rolled back status to Pending
	updatedReq, _ := ts.reqRepo.GetByIDOnly(ctx, req.ID)
	if updatedReq.Status != leave_request.StatusPending {
		t.Errorf("expected status Pending after transaction rollback, got %s", updatedReq.Status)
	}
}

// AT-09: Concurrent Approval - only one approval succeeds
func TestAT09_ConcurrentApproval(t *testing.T) {
	ts := initTestServices(t)
	user, emp, lt := createTestEmployeeHelper(t, ts, "emp_concurrent", "CL09", leave.StatusActive, 10)
	manager1 := createTestManagerHelper(t, ts, "manager09_a")
	manager2 := createTestManagerHelper(t, ts, "manager09_b")

	ctx := context.Background()
	req, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-08-10",
		ToDate:      "2026-08-12", // 3 days
		Reason:      "Leave",
	})

	err1 := ts.reqService.ApproveRequest(ctx, req.ID, manager1.ID)
	err2 := ts.reqService.ApproveRequest(ctx, req.ID, manager2.ID)

	if err1 != nil && err2 != nil {
		t.Fatalf("expected at least one approval to succeed, both failed: err1=%v, err2=%v", err1, err2)
	}
	if err1 == nil && err2 == nil {
		t.Fatal("expected only one approval to succeed, both succeeded")
	}

	// Verify balance is deducted only once (3 days)
	bal, _ := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)
	if bal.UsedDays != 3 {
		t.Errorf("expected used_days = 3 after concurrent attempt, got %d", bal.UsedDays)
	}
}
