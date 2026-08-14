# Implementation Plan - VS-01: Project Setup & Authentication

This document outlines the detailed execution plan for **VS-01 — Project Setup & Authentication** for the Mini Leave Management System.

---

## 1. Objectives & Scope

### In Scope
- Setup Go application structure (`cmd/server/main.go`, `internal/...`).
- Initialize SQLite database using CGO-free / pure Go driver (`modernc.org/sqlite` or `github.com/mattn/go-sqlite3`).
- Seed initial users with bcrypt password hashing:
  - `rahul` (role: `employee`, status: `active`, password: `password123`)
  - `priya` (role: `manager`, status: `active`, password: `password123`)
  - `amit` (role: `admin`, status: `active`, password: `password123`)
  - `inactive_user` (role: `employee`, status: `inactive`, password: `password123`)
- Server-side Session Management using cryptographic session tokens stored in SQLite or memory store with HTTP-only cookies (`session_id`).
- HTMX + HTML styled templates using modern typography (Inter/Roboto), clean layout, and HTMX progressive enhancement for login submission.
- Middleware:
  - `AuthMiddleware`: Validates session cookie, attaches `User` to request context, redirects unauthenticated requests to `/login`.
  - `RequireRole(roles...)`: Ensures authenticated user has appropriate role; returns `403 Forbidden` if unauthorized.
- Routes:
  - `GET /login` — Login page
  - `POST /login` — Process login (HTMX / Form request)
  - `POST /logout` — Terminate session
  - `GET /` — Authenticated home page
  - `GET /employee` — Protected Employee dashboard stub
  - `GET /manager` — Protected Manager dashboard stub
  - `GET /admin` — Protected Admin dashboard stub
- Automated unit and integration tests covering AT-01 through AT-07.

### Out of Scope
- Employee CRUD, leave applications, leave balances, manager approvals, email notifications.

---

## 2. Technical Stack & Dependencies

- **Go**: 1.21+
- **HTTP Server**: Go standard library (`net/http`) or lightweight router (`go-chi/chi/v5`)
- **Database**: SQLite (`modernc.org/sqlite` pure-Go driver for zero CGO setup dependency on Windows)
- **Password Hashing**: `golang.org/x/crypto/bcrypt`
- **Frontend**: HTML5 + HTMX (`https://unpkg.com/htmx.org`) + Custom CSS design system

---

## 3. Directory Structure

```text
leavemang/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── shared/
│   │   ├── database/
│   │   │   └── db.go
│   │   └── http/
│   │       └── response.go
│   └── slices/
│       └── authentication/
│           ├── model.go
│           ├── repository.go
│           ├── service.go
│           ├── handler.go
│           ├── middleware.go
│           ├── routes.go
│           ├── templates/
│           │   ├── layout.html
│           │   ├── login.html
│           │   ├── home.html
│           │   ├── employee.html
│           │   ├── manager.html
│           │   └── admin.html
│           └── authentication_test.go
├── static/
│   └── css/
│       └── style.css
├── migrations/
│   └── 0001_initial_schema.sql
├── docs/
│   ├── prd.md
│   ├── slices/
│   │   └── VS-01.md
│   └── VS-01-implementation-plan.md
├── go.mod
└── go.sum
```

---

## 4. Proposed Changes & Implementation Steps

### Step 1: Go Environment & Dependencies Setup
- Install Go SDK if not present on environment.
- Initialize Go module: `go mod init leavemang`.
- Add dependencies: `modernc.org/sqlite`, `golang.org/x/crypto/bcrypt`, `github.com/go-chi/chi/v5` (or standard `net/http`).

### Step 2: Shared Database Package (`internal/shared/database/db.go`)
- Create DB initialization function opening SQLite database (`leavemang.db`).
- Execute initial schema migrations (User and Session tables).
- Seed default users (`rahul`, `priya`, `amit`, `inactive_user`) with hashed passwords.

### Step 3: Domain Models & Repository (`internal/slices/authentication/`)
- `model.go`: Define `User` and `Session` structs.
- `repository.go`: Implement database operations for `GetUserByUsername`, `CreateSession`, `GetSessionByID`, `DeleteSession`, `DeleteExpiredSessions`.

### Step 4: Service Layer (`internal/slices/authentication/service.go`)
- `AuthenticateUser(username, password)`: Verify user existence, status (`active`), and password hash (`bcrypt.CompareHashAndPassword`).
- `CreateSession(userID)`: Generate secure 32-byte hex session token with 24-hour expiration.
- `ValidateSession(sessionID)`: Retrieve and validate unexpired session, fetching attached `User`.
- `Logout(sessionID)`: Invalidate session.

### Step 5: Middleware (`internal/slices/authentication/middleware.go`)
- `AuthMiddleware`: Extract `session_id` cookie, validate session via service, put `User` in `context.Context`. Redirect unauthenticated users to `/login`.
- `RequireRole(roles ...string)`: Inspect `User` from context. Reject with `403 Forbidden` if role is not permitted.

### Step 6: Templates & UI (`internal/slices/authentication/templates/` & `static/css/style.css`)
- Implement sleek, modern CSS with clean layout, dark/light theme options, responsive card containers, and HTMX error messaging.
- `login.html`: Form with username, password, login submit button, and `#error-message` container for HTMX response targeting.
- `home.html`, `employee.html`, `manager.html`, `admin.html`: Display user identity, role tag, and HTMX-enabled logout form.

### Step 7: Handlers & Routes (`internal/slices/authentication/handler.go` & `routes.go`)
- `HandleLoginGet`, `HandleLoginPost`, `HandleLogoutPost`, `HandleHomeGet`, `HandleEmployeeGet`, `HandleManagerGet`, `HandleAdminGet`.

### Step 8: Comprehensive Tests (`internal/slices/authentication/authentication_test.go`)
- Unit/Integration tests using `httptest` server covering:
  - **AT-01**: Valid login returns session cookie and redirects.
  - **AT-02**: Invalid password returns error status/message.
  - **AT-03**: Non-existent user returns authentication error.
  - **AT-04**: Inactive user login attempt rejected.
  - **AT-05**: Accessing `/employee` without session redirects to `/login`.
  - **AT-06**: Authenticated employee requesting `/admin` gets 403 Forbidden.
  - **AT-07**: POST `/logout` clears session and subsequent requests fail.

---

## 5. Verification Plan

### Automated Tests
- Run `go test -v ./internal/slices/authentication/...` to verify all acceptance criteria tests (AT-01 to AT-07).

### Manual Verification
- Start server via `go run cmd/server/main.go`.
- Open browser at `http://localhost:8080/login`.
- Verify logging in as `rahul` (employee), `priya` (manager), and `amit` (admin).
- Test role protection by navigating `rahul` to `/admin`.
- Test logging out and attempting to access protected routes.
