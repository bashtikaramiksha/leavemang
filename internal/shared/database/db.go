package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

// InitDB initializes the SQLite database connection, applies migrations, and seeds initial test users.
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	if err := Seed(db); err != nil {
		return nil, fmt.Errorf("seeding failed: %w", err)
	}

	return db, nil
}

// Migrate creates the required database schema if it doesn't exist.
func Migrate(db *sql.DB) error {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	sessionTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	employeeTable := `
	CREATE TABLE IF NOT EXISTS employees (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER UNIQUE NOT NULL,
		employee_code TEXT UNIQUE NOT NULL,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		phone TEXT,
		department TEXT NOT NULL,
		designation TEXT NOT NULL,
		joining_date TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	leaveTypeTable := `
	CREATE TABLE IF NOT EXISTS leave_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		default_allocation INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	leaveBalanceTable := `
	CREATE TABLE IF NOT EXISTS leave_balances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		employee_id INTEGER NOT NULL,
		leave_type_id INTEGER NOT NULL,
		allocated_days INTEGER NOT NULL DEFAULT 0,
		used_days INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
		FOREIGN KEY (leave_type_id) REFERENCES leave_types(id) ON DELETE CASCADE,
		UNIQUE(employee_id, leave_type_id)
	);`

	leaveRequestTable := `
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
		rejection_reason TEXT,
		FOREIGN KEY (employee_id) REFERENCES employees(id) ON DELETE CASCADE,
		FOREIGN KEY (leave_type_id) REFERENCES leave_types(id) ON DELETE CASCADE,
		FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL
	);`

	notificationTable := `
	CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		message TEXT NOT NULL,
		reference_type TEXT NOT NULL,
		reference_id INTEGER NOT NULL,
		is_read BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		read_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	if _, err := db.Exec(userTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := db.Exec(sessionTable); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	if _, err := db.Exec(employeeTable); err != nil {
		return fmt.Errorf("failed to create employees table: %w", err)
	}

	if _, err := db.Exec(leaveTypeTable); err != nil {
		return fmt.Errorf("failed to create leave_types table: %w", err)
	}

	if _, err := db.Exec(leaveBalanceTable); err != nil {
		return fmt.Errorf("failed to create leave_balances table: %w", err)
	}

	if _, err := db.Exec(leaveRequestTable); err != nil {
		return fmt.Errorf("failed to create leave_requests table: %w", err)
	}

	if _, err := db.Exec(notificationTable); err != nil {
		return fmt.Errorf("failed to create notifications table: %w", err)
	}

	// Migration for existing tables: add rejection_reason column if missing
	_, _ = db.Exec("ALTER TABLE leave_requests ADD COLUMN rejection_reason TEXT;")

	return nil
}

// Seed populates the database with initial seed users, employees, leave types, and balances.
func Seed(db *sql.DB) error {
	type seedUser struct {
		username     string
		password     string
		role         string
		status       string
		employeeCode string
		firstName    string
		lastName     string
		email        string
		phone        string
		department   string
		designation  string
		joiningDate  string
	}

	users := []seedUser{
		{
			username: "rahul", password: "password123", role: "employee", status: "active",
			employeeCode: "EMP-001", firstName: "Rahul", lastName: "Patil", email: "rahul@example.com",
			phone: "9876543210", department: "IT", designation: "Software Developer", joiningDate: "2026-08-01",
		},
		{
			username: "priya", password: "password123", role: "manager", status: "active",
			employeeCode: "EMP-002", firstName: "Priya", lastName: "Shah", email: "priya@example.com",
			phone: "9876543211", department: "HR", designation: "HR Manager", joiningDate: "2026-07-15",
		},
		{
			username: "amit", password: "password123", role: "admin", status: "active",
			employeeCode: "EMP-003", firstName: "Amit", lastName: "Joshi", email: "amit@example.com",
			phone: "9876543212", department: "Sales", designation: "Sales Admin", joiningDate: "2026-06-01",
		},
		{
			username: "inactive_user", password: "password123", role: "employee", status: "inactive",
			employeeCode: "EMP-004", firstName: "Inactive", lastName: "User", email: "inactive@example.com",
			phone: "9876543213", department: "Operations", designation: "Staff", joiningDate: "2026-05-01",
		},
	}

	for _, u := range users {
		var userID int64
		err := db.QueryRow("SELECT id FROM users WHERE username = ?", u.username).Scan(&userID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		if err == sql.ErrNoRows {
			hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("failed to hash password for %s: %w", u.username, err)
			}

			res, err := db.Exec(
				"INSERT INTO users (username, password_hash, role, status) VALUES (?, ?, ?, ?)",
				u.username, string(hash), u.role, u.status,
			)
			if err != nil {
				return fmt.Errorf("failed to seed user %s: %w", u.username, err)
			}
			userID, err = res.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get user ID for %s: %w", u.username, err)
			}
			log.Printf("[SEED] Created user: %s (Role: %s, Status: %s)", u.username, u.role, u.status)
		}

		// Ensure matching employee record exists
		var empCount int
		err = db.QueryRow("SELECT COUNT(1) FROM employees WHERE user_id = ?", userID).Scan(&empCount)
		if err != nil {
			return err
		}
		if empCount == 0 {
			_, err = db.Exec(
				`INSERT INTO employees (user_id, employee_code, first_name, last_name, email, phone, department, designation, joining_date, status)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				userID, u.employeeCode, u.firstName, u.lastName, u.email, u.phone, u.department, u.designation, u.joiningDate, u.status,
			)
			if err != nil {
				return fmt.Errorf("failed to seed employee record for %s: %w", u.username, err)
			}
			log.Printf("[SEED] Created employee: %s (%s)", u.employeeCode, u.email)
		}
	}

	// Seed default leave types
	type seedLeaveType struct {
		code              string
		name              string
		description       string
		defaultAllocation int
		status            string
	}

	leaveTypes := []seedLeaveType{
		{code: "CL", name: "Casual Leave", description: "Casual leave for short absences", defaultAllocation: 12, status: "active"},
		{code: "SL", name: "Sick Leave", description: "Leave for medical reasons", defaultAllocation: 10, status: "active"},
		{code: "EL", name: "Earned Leave", description: "Annual privilege leave", defaultAllocation: 15, status: "active"},
	}

	for _, lt := range leaveTypes {
		var count int
		err := db.QueryRow("SELECT COUNT(1) FROM leave_types WHERE code = ?", lt.code).Scan(&count)
		if err != nil {
			return err
		}
		if count == 0 {
			_, err = db.Exec(
				`INSERT INTO leave_types (code, name, description, default_allocation, status) VALUES (?, ?, ?, ?, ?)`,
				lt.code, lt.name, lt.description, lt.defaultAllocation, lt.status,
			)
			if err != nil {
				return fmt.Errorf("failed to seed leave type %s: %w", lt.code, err)
			}
			log.Printf("[SEED] Created leave type: %s (%s)", lt.code, lt.name)
		}
	}

	// Seed initial leave balances for active employee Rahul (EMP-001)
	var rahulEmpID int64
	err := db.QueryRow("SELECT id FROM employees WHERE employee_code = 'EMP-001'").Scan(&rahulEmpID)
	if err == nil && rahulEmpID > 0 {
		for _, lt := range leaveTypes {
			var ltID int64
			err := db.QueryRow("SELECT id FROM leave_types WHERE code = ?", lt.code).Scan(&ltID)
			if err == nil && ltID > 0 {
				var balCount int
				err := db.QueryRow("SELECT COUNT(1) FROM leave_balances WHERE employee_id = ? AND leave_type_id = ?", rahulEmpID, ltID).Scan(&balCount)
				if err == nil && balCount == 0 {
					_, err = db.Exec(
						`INSERT INTO leave_balances (employee_id, leave_type_id, allocated_days, used_days) VALUES (?, ?, ?, 0)`,
						rahulEmpID, ltID, lt.defaultAllocation,
					)
					if err != nil {
						log.Printf("[SEED] Failed to seed balance for employee %d, type %s: %v", rahulEmpID, lt.code, err)
					}
				}
			}
		}
	}

	return nil
}
