package leave_dashboard

import (
	"fmt"
	"net/http"
	"strconv"

	sharedHttp "leavemang/internal/shared/http"
	"leavemang/internal/slices/authentication"
)

type Handler struct {
	service     *Service
	layoutPath  string
	templateDir string
}

func NewHandler(service *Service, layoutPath, templateDir string) *Handler {
	return &Handler{
		service:     service,
		layoutPath:  layoutPath,
		templateDir: templateDir,
	}
}

// HandleEmployeeDashboard renders the employee dashboard (AR-01, AR-02, AR-03, BR-01).
func (h *Handler) HandleEmployeeDashboard(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data, err := h.service.GetEmployeeDashboard(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load dashboard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pageData := map[string]interface{}{
		"Title":          "My Leave Dashboard - Leave Management System",
		"User":           user,
		"Employee":       data.Employee,
		"Balances":       data.Balances,
		"RecentRequests": data.RecentRequests,
	}

	templatePath := fmt.Sprintf("%s/employee_dashboard.html", h.templateDir)
	sharedHttp.RenderHTML(w, r, http.StatusOK, pageData, h.layoutPath, templatePath)
}

// HandleEmployeeHistory renders full leave history for the employee (AR-03, AR-04, AR-05, BR-01).
func (h *Handler) HandleEmployeeHistory(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emp, requests, err := h.service.GetEmployeeHistory(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load leave history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pageData := map[string]interface{}{
		"Title":    "My Leave History - Leave Management System",
		"User":     user,
		"Employee": emp,
		"Requests": requests,
	}

	templatePath := fmt.Sprintf("%s/employee_history.html", h.templateDir)
	sharedHttp.RenderHTML(w, r, http.StatusOK, pageData, h.layoutPath, templatePath)
}

// HandleEmployeeBalances renders employee leave balances summary (AR-02, BR-01, BR-06).
func (h *Handler) HandleEmployeeBalances(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emp, balances, err := h.service.GetEmployeeBalances(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load leave balances: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pageData := map[string]interface{}{
		"Title":    "My Leave Balances - Leave Management System",
		"User":     user,
		"Employee": emp,
		"Balances": balances,
	}

	templatePath := fmt.Sprintf("%s/employee_balances.html", h.templateDir)
	sharedHttp.RenderHTML(w, r, http.StatusOK, pageData, h.layoutPath, templatePath)
}

// HandleManagerDashboard renders the manager dashboard with stats and team requests (AR-07 to AR-10, BR-02).
func (h *Handler) HandleManagerDashboard(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	var empIDFilter int64
	if empIDStr := r.URL.Query().Get("employee_id"); empIDStr != "" {
		empIDFilter, _ = strconv.ParseInt(empIDStr, 10, 64)
	}

	data, err := h.service.GetManagerDashboard(r.Context(), statusFilter, empIDFilter)
	if err != nil {
		http.Error(w, "Failed to load manager dashboard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pageData := map[string]interface{}{
		"Title":            "Manager Leave Dashboard - Leave Management System",
		"User":             user,
		"Stats":            data.Stats,
		"Employees":        data.Employees,
		"Requests":         data.Requests,
		"StatusFilter":     data.StatusFilter,
		"EmployeeIDFilter": data.EmployeeIDFilter,
	}

	templatePath := fmt.Sprintf("%s/manager_dashboard.html", h.templateDir)
	tablePath := fmt.Sprintf("%s/manager_requests_table.html", h.templateDir)
	sharedHttp.RenderHTML(w, r, http.StatusOK, pageData, h.layoutPath, templatePath, tablePath)
}

// HandleManagerRequests returns team requests table (supports HTMX partial updates for filtering).
func (h *Handler) HandleManagerRequests(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	var empIDFilter int64
	if empIDStr := r.URL.Query().Get("employee_id"); empIDStr != "" {
		empIDFilter, _ = strconv.ParseInt(empIDStr, 10, 64)
	}

	data, err := h.service.GetManagerDashboard(r.Context(), statusFilter, empIDFilter)
	if err != nil {
		http.Error(w, "Failed to load team requests: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pageData := map[string]interface{}{
		"Title":            "Team Leave Requests - Leave Management System",
		"User":             user,
		"Stats":            data.Stats,
		"Employees":        data.Employees,
		"Requests":         data.Requests,
		"StatusFilter":     data.StatusFilter,
		"EmployeeIDFilter": data.EmployeeIDFilter,
	}

	tablePath := fmt.Sprintf("%s/manager_requests_table.html", h.templateDir)
	if sharedHttp.IsHTMX(r) {
		sharedHttp.RenderHTML(w, r, http.StatusOK, pageData, tablePath)
		return
	}

	templatePath := fmt.Sprintf("%s/manager_dashboard.html", h.templateDir)
	sharedHttp.RenderHTML(w, r, http.StatusOK, pageData, h.layoutPath, templatePath, tablePath)
}
