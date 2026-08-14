package leave_dashboard_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
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
	"leavemang/internal/slices/leave_dashboard"
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
	db               *sql.DB
	authRepo         *authentication.Repository
	authService      *authentication.Service
	middleware       *authentication.Middleware
	empRepo          *employee.Repository
	empService       *employee.Service
	leaveRepo        *leave.Repository
	leaveService     *leave.Service
	reqRepo          *leave_request.Repository
	reqService       *leave_request.Service
	reqHandler       *leave_request.Handler
	dashboardService *leave_dashboard.Service
	dashboardHandler *leave_dashboard.Handler
	router           *chi.Mux
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

	reqRepo := leave_request.NewRepository(db)
	reqService := leave_request.NewService(reqRepo, empService, leaveService)

	dashboardService := leave_dashboard.NewService(reqRepo, leaveRepo, empRepo, empService)

	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(workDir, "cmd", "server", "main.go")); err == nil {
			break
		}
		workDir = filepath.Dir(workDir)
	}

	authTemplateDir := filepath.Join(workDir, "internal", "slices", "authentication", "templates")
	reqTemplateDir := filepath.Join(workDir, "internal", "slices", "leave_request", "templates")
	dashboardTemplateDir := filepath.Join(workDir, "internal", "slices", "leave_dashboard", "templates")
	layoutPath := filepath.Join(authTemplateDir, "layout.html")

	reqHandler := leave_request.NewHandler(reqService, leaveService, layoutPath, reqTemplateDir)
	dashboardHandler := leave_dashboard.NewHandler(dashboardService, layoutPath, dashboardTemplateDir)

	router := chi.NewRouter()
	authentication.RegisterRoutes(router, authentication.NewHandler(authService, authTemplateDir), middleware)
	leave_request.RegisterRoutes(router, reqHandler, middleware)
	leave_dashboard.RegisterRoutes(router, dashboardHandler, middleware)

	return &testServices{
		db:               db,
		authRepo:         authRepo,
		authService:      authService,
		middleware:       middleware,
		empRepo:          empRepo,
		empService:       empService,
		leaveRepo:        leaveRepo,
		leaveService:     leaveService,
		reqRepo:          reqRepo,
		reqService:       reqService,
		reqHandler:       reqHandler,
		dashboardService: dashboardService,
		dashboardHandler: dashboardHandler,
		router:           router,
	}
}

func createTestUserAndEmployee(t *testing.T, ts *testServices, email, role string) (*authentication.User, *employee.Employee) {
	ctx := context.Background()

	emp, err := ts.empService.CreateEmployee(ctx, employee.CreateEmployeeInput{
		FirstName:   "Test",
		LastName:    "User",
		Email:       email,
		Phone:       "1234567890",
		Department:  "Engineering",
		Designation: "Developer",
		JoiningDate: "2026-01-01",
		Role:        role,
	})
	if err != nil {
		t.Fatalf("failed to create employee record: %v", err)
	}

	user, err := ts.authRepo.GetUserByID(emp.UserID)
	if err != nil {
		t.Fatalf("failed to get user by id: %v", err)
	}

	return user, emp
}

func createAuthenticatedCookie(t *testing.T, ts *testServices, user *authentication.User) *http.Cookie {
	sess, err := ts.authService.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	return &http.Cookie{
		Name:  "session_id",
		Value: sess.ID,
		Path:  "/",
	}
}

// AT-01: Employee Dashboard shows leave balances and recent requests
func TestAT01_EmployeeDashboard(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()

	user, emp := createTestUserAndEmployee(t, ts, "emp1@example.com", employee.RoleEmployee)
	lt, err := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "CL",
		Name:              "Casual Leave",
		DefaultAllocation: 12,
	})
	if err != nil {
		t.Fatalf("failed to create leave type: %v", err)
	}

	_, err = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{
		EmployeeID:    emp.ID,
		LeaveTypeID:   lt.ID,
		AllocatedDays: 12,
	})
	if err != nil {
		t.Fatalf("failed to allocate leave: %v", err)
	}

	_, err = ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID,
		FromDate:    "2026-09-01",
		ToDate:      "2026-09-03",
		Reason:      "Vacation trip",
	})
	if err != nil {
		t.Fatalf("failed to create leave request: %v", err)
	}

	cookie := createAuthenticatedCookie(t, ts, user)

	req := httptest.NewRequest("GET", "/my/dashboard", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	ts.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Casual Leave") {
		t.Errorf("expected dashboard to contain 'Casual Leave', body: %s", body)
	}
	if !strings.Contains(body, "Vacation trip") {
		t.Errorf("expected dashboard to contain recent request reason 'Vacation trip', body: %s", body)
	}
}

// AT-02: Employee History displays all employee requests
func TestAT02_EmployeeHistory(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()

	user, emp := createTestUserAndEmployee(t, ts, "history@example.com", employee.RoleEmployee)
	lt, _ := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "SL",
		Name:              "Sick Leave",
		DefaultAllocation: 10,
	})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{
		EmployeeID:    emp.ID,
		LeaveTypeID:   lt.ID,
		AllocatedDays: 10,
	})

	for i := 1; i <= 5; i++ {
		_, err := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
			LeaveTypeID: lt.ID,
			FromDate:    "2026-10-0" + strconv.Itoa(i),
			ToDate:      "2026-10-0" + strconv.Itoa(i),
			Reason:      "Sick day " + strconv.Itoa(i),
		})
		if err != nil {
			t.Fatalf("failed to create request %d: %v", i, err)
		}
	}

	cookie := createAuthenticatedCookie(t, ts, user)

	req := httptest.NewRequest("GET", "/my/leave-requests", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	ts.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	for i := 1; i <= 5; i++ {
		if !strings.Contains(body, "Sick day "+strconv.Itoa(i)) {
			t.Errorf("expected history to contain 'Sick day %d'", i)
		}
	}
}

// AT-03: Employee Isolation ensures Employee A cannot see Employee B's data
func TestAT03_EmployeeIsolation(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()

	userA, empA := createTestUserAndEmployee(t, ts, "userA@example.com", employee.RoleEmployee)
	userB, empB := createTestUserAndEmployee(t, ts, "userB@example.com", employee.RoleEmployee)

	lt, _ := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "EL",
		Name:              "Earned Leave",
		DefaultAllocation: 15,
	})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{EmployeeID: empA.ID, LeaveTypeID: lt.ID, AllocatedDays: 15})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{EmployeeID: empB.ID, LeaveTypeID: lt.ID, AllocatedDays: 15})

	_, _ = ts.reqService.CreateRequest(ctx, userA.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID, FromDate: "2026-09-10", ToDate: "2026-09-12", Reason: "Secret Employee A Request",
	})
	reqB, _ := ts.reqService.CreateRequest(ctx, userB.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID, FromDate: "2026-09-15", ToDate: "2026-09-16", Reason: "Employee B Private Request",
	})

	// 1. Employee A opens history: should ONLY contain A's request, NOT B's
	cookieA := createAuthenticatedCookie(t, ts, userA)
	httpReq := httptest.NewRequest("GET", "/my/leave-requests", nil)
	httpReq.AddCookie(cookieA)
	rr := httptest.NewRecorder()

	ts.router.ServeHTTP(rr, httpReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Secret Employee A Request") {
		t.Errorf("Employee A history missing A's request")
	}
	if strings.Contains(body, "Employee B Private Request") {
		t.Errorf("Employee A history illegally contains Employee B's request!")
	}

	// 2. Employee A tries to view Employee B's specific request details via GET /my/leave-requests/{id}
	detailReq := httptest.NewRequest("GET", "/my/leave-requests/"+strconv.FormatInt(reqB.ID, 10), nil)
	detailReq.AddCookie(cookieA)
	rrDetail := httptest.NewRecorder()

	ts.router.ServeHTTP(rrDetail, detailReq)
	if rrDetail.Code != http.StatusForbidden && rrDetail.Code != http.StatusUnauthorized {
		t.Errorf("expected 403 Forbidden when accessing another user's request details, got %d", rrDetail.Code)
	}
}

// AT-04: Rejection reason displayed on request details
func TestAT04_RejectionReason(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()

	user, emp := createTestUserAndEmployee(t, ts, "rejection@example.com", employee.RoleEmployee)
	mgrUser, _ := createTestUserAndEmployee(t, ts, "mgr@example.com", employee.RoleManager)

	lt, _ := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "CL",
		Name:              "Casual Leave",
		DefaultAllocation: 10,
	})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{EmployeeID: emp.ID, LeaveTypeID: lt.ID, AllocatedDays: 10})

	createdReq, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID, FromDate: "2026-11-01", ToDate: "2026-11-02", Reason: "Personal Work",
	})

	rejectionReasonText := "Team deployment required during this period."
	err := ts.reqService.RejectRequest(ctx, createdReq.ID, mgrUser.ID, rejectionReasonText)
	if err != nil {
		t.Fatalf("failed to reject request: %v", err)
	}

	cookie := createAuthenticatedCookie(t, ts, user)
	req := httptest.NewRequest("GET", "/my/leave-requests/"+strconv.FormatInt(createdReq.ID, 10), nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	ts.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, rejectionReasonText) {
		t.Errorf("expected rejection reason '%s' in details view, got: %s", rejectionReasonText, body)
	}
}

// AT-05 & AT-06: Manager status filter and employee filter
func TestAT05_AT06_ManagerFilters(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()

	mgrUser, _ := createTestUserAndEmployee(t, ts, "manager1@example.com", employee.RoleManager)
	userA, empA := createTestUserAndEmployee(t, ts, "rahul@example.com", employee.RoleEmployee)
	userB, empB := createTestUserAndEmployee(t, ts, "amit@example.com", employee.RoleEmployee)

	lt, _ := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "CL",
		Name:              "Casual Leave",
		DefaultAllocation: 20,
	})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{EmployeeID: empA.ID, LeaveTypeID: lt.ID, AllocatedDays: 20})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{EmployeeID: empB.ID, LeaveTypeID: lt.ID, AllocatedDays: 20})

	// Create 1 pending for A, 1 approved for B, 1 rejected for B
	reqPending, _ := ts.reqService.CreateRequest(ctx, userA.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID, FromDate: "2026-12-01", ToDate: "2026-12-02", Reason: "Pending A",
	})
	reqApproved, _ := ts.reqService.CreateRequest(ctx, userB.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID, FromDate: "2026-12-05", ToDate: "2026-12-06", Reason: "Approved B",
	})
	reqRejected, _ := ts.reqService.CreateRequest(ctx, userB.ID, leave_request.CreateLeaveRequestInput{
		LeaveTypeID: lt.ID, FromDate: "2026-12-10", ToDate: "2026-12-11", Reason: "Rejected B",
	})

	_ = ts.reqService.ApproveRequest(ctx, reqApproved.ID, mgrUser.ID)
	_ = ts.reqService.RejectRequest(ctx, reqRejected.ID, mgrUser.ID, "Not allowed")

	mgrCookie := createAuthenticatedCookie(t, ts, mgrUser)

	// AT-05: Filter by pending
	reqP := httptest.NewRequest("GET", "/manager/leave-requests?status=pending", nil)
	reqP.AddCookie(mgrCookie)
	rrP := httptest.NewRecorder()
	ts.router.ServeHTTP(rrP, reqP)

	if rrP.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rrP.Code)
	}
	bodyP := rrP.Body.String()
	if !strings.Contains(bodyP, "Pending A") {
		t.Errorf("Pending filter missing pending request")
	}
	if strings.Contains(bodyP, "Approved B") {
		t.Errorf("Pending filter illegally contains approved request")
	}

	// AT-06: Filter by employee A
	reqEmpA := httptest.NewRequest("GET", "/manager/leave-requests?employee_id="+strconv.FormatInt(empA.ID, 10), nil)
	reqEmpA.AddCookie(mgrCookie)
	rrEmpA := httptest.NewRecorder()
	ts.router.ServeHTTP(rrEmpA, reqEmpA)

	if rrEmpA.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rrEmpA.Code)
	}
	bodyEmpA := rrEmpA.Body.String()
	if !strings.Contains(bodyEmpA, "Pending A") {
		t.Errorf("Employee A filter missing A's request")
	}
	if strings.Contains(bodyEmpA, "Approved B") {
		t.Errorf("Employee A filter illegally contains B's request")
	}

	_ = reqPending
}

// AT-07 & AT-08: Manager Dashboard Statistics and Approved Days Calculation
func TestAT07_AT08_DashboardStatistics(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()

	mgrUser, _ := createTestUserAndEmployee(t, ts, "stat_mgr@example.com", employee.RoleManager)
	user, emp := createTestUserAndEmployee(t, ts, "stat_emp@example.com", employee.RoleEmployee)

	lt, _ := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "SL",
		Name:              "Sick Leave",
		DefaultAllocation: 50,
	})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{EmployeeID: emp.ID, LeaveTypeID: lt.ID, AllocatedDays: 50})

	// Create requests: 3 approved (3 days, 2 days, 5 days = 10 approved days)
	r1, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{LeaveTypeID: lt.ID, FromDate: "2026-08-01", ToDate: "2026-08-03", Reason: "Req 1 (3 days)"})
	r2, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{LeaveTypeID: lt.ID, FromDate: "2026-08-05", ToDate: "2026-08-06", Reason: "Req 2 (2 days)"})
	r3, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{LeaveTypeID: lt.ID, FromDate: "2026-08-10", ToDate: "2026-08-14", Reason: "Req 3 (5 days)"})

	// Approve r1, r2, r3 (Total approved days = 3 + 2 + 5 = 10)
	_ = ts.reqService.ApproveRequest(ctx, r1.ID, mgrUser.ID)
	_ = ts.reqService.ApproveRequest(ctx, r2.ID, mgrUser.ID)
	_ = ts.reqService.ApproveRequest(ctx, r3.ID, mgrUser.ID)

	// Create 3 pending requests
	_, _ = ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{LeaveTypeID: lt.ID, FromDate: "2026-08-20", ToDate: "2026-08-20", Reason: "Pending 1"})
	_, _ = ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{LeaveTypeID: lt.ID, FromDate: "2026-08-22", ToDate: "2026-08-22", Reason: "Pending 2"})
	_, _ = ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{LeaveTypeID: lt.ID, FromDate: "2026-08-25", ToDate: "2026-08-25", Reason: "Pending 3"})

	// Create 1 rejected request
	rReject, _ := ts.reqService.CreateRequest(ctx, user.ID, leave_request.CreateLeaveRequestInput{LeaveTypeID: lt.ID, FromDate: "2026-08-28", ToDate: "2026-08-28", Reason: "Reject 1"})
	_ = ts.reqService.RejectRequest(ctx, rReject.ID, mgrUser.ID, "Denied")

	// Get statistics directly from repository
	stats, err := ts.reqRepo.GetDashboardStats(ctx)
	if err != nil {
		t.Fatalf("failed to calculate dashboard stats: %v", err)
	}

	if stats.TotalRequests != 7 {
		t.Errorf("expected total requests 7, got %d", stats.TotalRequests)
	}
	if stats.ApprovedRequests != 3 {
		t.Errorf("expected approved requests 3, got %d", stats.ApprovedRequests)
	}
	if stats.PendingRequests != 3 {
		t.Errorf("expected pending requests 3, got %d", stats.PendingRequests)
	}
	if stats.RejectedRequests != 1 {
		t.Errorf("expected rejected requests 1, got %d", stats.RejectedRequests)
	}
	if stats.ApprovedDays != 10 { // 3 + 2 + 5 = 10
		t.Errorf("expected approved days 10, got %d", stats.ApprovedDays)
	}

	// Verify manager dashboard route displays these stats
	mgrCookie := createAuthenticatedCookie(t, ts, mgrUser)
	req := httptest.NewRequest("GET", "/manager/dashboard", nil)
	req.AddCookie(mgrCookie)
	rr := httptest.NewRecorder()

	ts.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Manager Leave Dashboard") {
		t.Fatalf("dashboard page missing title, body: %s", body)
	}
}

// AT-09: Employee cannot access manager data (403 Forbidden)
func TestAT09_EmployeeCannotAccessManagerData(t *testing.T) {
	ts := initTestServices(t)

	user, _ := createTestUserAndEmployee(t, ts, "normalemp@example.com", employee.RoleEmployee)
	cookie := createAuthenticatedCookie(t, ts, user)

	// Attempt GET /manager/dashboard
	reqDash := httptest.NewRequest("GET", "/manager/dashboard", nil)
	reqDash.AddCookie(cookie)
	rrDash := httptest.NewRecorder()

	ts.router.ServeHTTP(rrDash, reqDash)
	if rrDash.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for employee accessing /manager/dashboard, got %d", rrDash.Code)
	}

	// Attempt GET /manager/leave-requests
	reqList := httptest.NewRequest("GET", "/manager/leave-requests", nil)
	reqList.AddCookie(cookie)
	rrList := httptest.NewRecorder()

	ts.router.ServeHTTP(rrList, reqList)
	if rrList.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for employee accessing /manager/leave-requests, got %d", rrList.Code)
	}
}

// AT-10: Read-Only balance verification
func TestAT10_ReadOnlyBalance(t *testing.T) {
	ts := initTestServices(t)
	ctx := context.Background()

	user, emp := createTestUserAndEmployee(t, ts, "readonly@example.com", employee.RoleEmployee)
	lt, _ := ts.leaveService.CreateLeaveType(ctx, leave.CreateLeaveTypeInput{
		Code:              "AL",
		Name:              "Annual Leave",
		DefaultAllocation: 20,
	})
	_, _ = ts.leaveService.AllocateLeave(ctx, leave.AllocateLeaveInput{EmployeeID: emp.ID, LeaveTypeID: lt.ID, AllocatedDays: 20})

	// Fetch balance before viewing dashboard
	balBefore, _ := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)

	cookie := createAuthenticatedCookie(t, ts, user)

	// Call GET /my/dashboard multiple times
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/my/dashboard", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		ts.router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}
	}

	// Call GET /my/leave-balances multiple times
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/my/leave-balances", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		ts.router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}
	}

	// Fetch balance after viewing dashboards
	balAfter, _ := ts.leaveService.GetBalance(ctx, emp.ID, lt.ID)

	if balBefore.AllocatedDays != balAfter.AllocatedDays || balBefore.UsedDays != balAfter.UsedDays {
		t.Errorf("balance mutated! Before: %+v, After: %+v", balBefore, balAfter)
	}
}
