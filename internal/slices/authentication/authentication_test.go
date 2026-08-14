package authentication_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"leavemang/internal/shared/database"
	"leavemang/internal/slices/authentication"
)

func setupTestServer(t *testing.T) (http.Handler, func()) {
	t.Helper()

	// Create temporary SQLite DB for test isolation
	tmpDB := filepath.Join(t.TempDir(), "test_leavemang.db")
	db, err := database.InitDB(tmpDB)
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working dir: %v", err)
	}

	// Adjust template path if running within package dir
	templateDir := filepath.Join(workDir, "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		templateDir = filepath.Join(workDir, "..", "..", "..", "internal", "slices", "authentication", "templates")
	}

	repo := authentication.NewRepository(db)
	service := authentication.NewService(repo)
	handler := authentication.NewHandler(service, templateDir)
	middleware := authentication.NewMiddleware(service)

	r := chi.NewRouter()
	authentication.RegisterRoutes(r, handler, middleware)

	cleanup := func() {
		db.Close()
	}

	return r, cleanup
}

// AT-01 — Valid Login
func TestAT01_ValidLogin(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	form := url.Values{}
	form.Set("username", "rahul")
	form.Set("password", "password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("Expected status SeeOther (303), got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_id" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("Expected valid session_id cookie after successful login")
	}
}

// AT-02 — Invalid Password
func TestAT02_InvalidPassword(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	form := url.Values{}
	form.Set("username", "rahul")
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status Unauthorized (401), got %d", rec.Code)
	}

	for _, c := range rec.Result().Cookies() {
		if c.Name == "session_id" && c.Value != "" {
			t.Errorf("Did not expect valid session_id cookie on failed login")
		}
	}
}

// AT-03 — Unknown User
func TestAT03_UnknownUser(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	form := url.Values{}
	form.Set("username", "nonexistent_user")
	form.Set("password", "password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status Unauthorized (401), got %d", rec.Code)
	}
}

// AT-04 — Inactive User
func TestAT04_InactiveUser(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	form := url.Values{}
	form.Set("username", "inactive_user")
	form.Set("password", "password123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status Unauthorized (401) for inactive user, got %d", rec.Code)
	}
}

// AT-05 — Protected Route Rejection
func TestAT05_ProtectedRouteRejection(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/employee", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect to login (303), got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Expected redirect location /login, got %s", location)
	}
}

// AT-06 — Role Protection Rejection
func TestAT06_RoleProtectionRejection(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Log in as employee rahul
	form := url.Values{}
	form.Set("username", "rahul")
	form.Set("password", "password123")

	loginReq := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()

	router.ServeHTTP(loginRec, loginReq)

	var sessionCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "session_id" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("Failed to obtain session cookie for rahul")
	}

	// 2. Employee attempts to access admin route
	adminReq := httptest.NewRequest("GET", "/admin", nil)
	adminReq.AddCookie(sessionCookie)
	adminRec := httptest.NewRecorder()

	router.ServeHTTP(adminRec, adminReq)

	if adminRec.Code != http.StatusForbidden {
		t.Errorf("Expected status Forbidden (403) when employee accesses /admin, got %d", adminRec.Code)
	}
}

// AT-07 — Logout
func TestAT07_Logout(t *testing.T) {
	router, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Log in
	form := url.Values{}
	form.Set("username", "rahul")
	form.Set("password", "password123")

	loginReq := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()

	router.ServeHTTP(loginRec, loginReq)

	var sessionCookie *http.Cookie
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "session_id" {
			sessionCookie = c
			break
		}
	}

	// 2. Logout
	logoutReq := httptest.NewRequest("POST", "/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutRec := httptest.NewRecorder()

	router.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect after logout, got %d", logoutRec.Code)
	}

	// 3. Verify session cookie invalidated and protected access rejected
	accessReq := httptest.NewRequest("GET", "/", nil)
	accessReq.AddCookie(sessionCookie)
	accessRec := httptest.NewRecorder()

	router.ServeHTTP(accessRec, accessReq)

	if accessRec.Code != http.StatusSeeOther {
		t.Errorf("Expected redirect to login after session invalidation, got %d", accessRec.Code)
	}
}
