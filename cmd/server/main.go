package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"leavemang/internal/shared/database"
	"leavemang/internal/slices/authentication"
	"leavemang/internal/slices/employee"
	"leavemang/internal/slices/leave"
	"leavemang/internal/slices/leave_dashboard"
	"leavemang/internal/slices/leave_request"
	"leavemang/internal/slices/notification"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "leavemang.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[INIT] Initializing SQLite database at %s...", dbPath)
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}
	defer db.Close()

	// Find template directory path relative to current working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	authTemplateDir := filepath.Join(workDir, "internal", "slices", "authentication", "templates")
	empTemplateDir := filepath.Join(workDir, "internal", "slices", "employee", "templates")
	leaveTemplateDir := filepath.Join(workDir, "internal", "slices", "leave", "templates")
	leaveReqTemplateDir := filepath.Join(workDir, "internal", "slices", "leave_request", "templates")
	dashboardTemplateDir := filepath.Join(workDir, "internal", "slices", "leave_dashboard", "templates")
	notifTemplateDir := filepath.Join(workDir, "internal", "slices", "notification", "templates")
	layoutPath := filepath.Join(authTemplateDir, "layout.html")
	staticDir := filepath.Join(workDir, "static")

	authRepo := authentication.NewRepository(db)
	authService := authentication.NewService(authRepo)
	authHandler := authentication.NewHandler(authService, authTemplateDir)
	middleware := authentication.NewMiddleware(authService)

	empRepo := employee.NewRepository(db)
	empService := employee.NewService(empRepo)
	empHandler := employee.NewHandler(empService, layoutPath, empTemplateDir)

	leaveRepo := leave.NewRepository(db)
	leaveService := leave.NewService(leaveRepo, empRepo)
	leaveHandler := leave.NewHandler(leaveService, empService, layoutPath, leaveTemplateDir)

	notifRepo := notification.NewRepository(db)
	notifService := notification.NewService(notifRepo)
	notifHandler := notification.NewHandler(notifService, layoutPath, notifTemplateDir)

	leaveReqRepo := leave_request.NewRepository(db)
	leaveReqService := leave_request.NewService(leaveReqRepo, empService, leaveService)
	leaveReqService.SetNotificationService(notifService)
	leaveReqHandler := leave_request.NewHandler(leaveReqService, leaveService, layoutPath, leaveReqTemplateDir)

	dashboardService := leave_dashboard.NewService(leaveReqRepo, leaveRepo, empRepo, empService)
	dashboardHandler := leave_dashboard.NewHandler(dashboardService, layoutPath, dashboardTemplateDir)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// Serve static assets (CSS, JS, images)
	fileServer := http.FileServer(http.Dir(staticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Register slice routes
	authentication.RegisterRoutes(r, authHandler, middleware)
	employee.RegisterRoutes(r, empHandler, middleware)
	leave.RegisterRoutes(r, leaveHandler, middleware)
	leave_request.RegisterRoutes(r, leaveReqHandler, middleware)
	leave_dashboard.RegisterRoutes(r, dashboardHandler, middleware)
	notification.RegisterRoutes(r, notifHandler, middleware)

	log.Printf("[SERVER] Mini Leave Management System listening on http://localhost:%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
