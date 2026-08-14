package leave

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	sharedHttp "leavemang/internal/shared/http"
	"leavemang/internal/slices/authentication"
	"leavemang/internal/slices/employee"
)

type Handler struct {
	service     *Service
	empService  *employee.Service
	layoutPath  string
	templateDir string
}

func NewHandler(service *Service, empService *employee.Service, layoutPath string, templateDir string) *Handler {
	return &Handler{
		service:     service,
		empService:  empService,
		layoutPath:  layoutPath,
		templateDir: templateDir,
	}
}

// HandleListLeaveTypes displays the list of leave types for Admin.
func (h *Handler) HandleListLeaveTypes(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)

	leaveTypes, err := h.service.ListLeaveTypes(r.Context())
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templatePath := fmt.Sprintf("%s/leave_types_list.html", h.templateDir)
	data := map[string]interface{}{
		"Title":      "Leave Types - Leave Management System",
		"User":       user,
		"LeaveTypes": leaveTypes,
		"IsAdmin":    user != nil && user.Role == authentication.RoleAdmin,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleNewLeaveTypeForm renders the form to create a new leave type.
func (h *Handler) HandleNewLeaveTypeForm(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)

	templatePath := fmt.Sprintf("%s/leave_type_form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":  "Create Leave Type - Leave Management System",
		"User":   user,
		"IsEdit": false,
		"Input":  CreateLeaveTypeInput{DefaultAllocation: 12},
		"Error":  "",
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleCreateLeaveType processes leave type creation.
func (h *Handler) HandleCreateLeaveType(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderLeaveTypeFormError(w, r, false, nil, CreateLeaveTypeInput{}, "Invalid form data")
		return
	}

	defaultAlloc, _ := strconv.Atoi(r.FormValue("default_allocation"))
	input := CreateLeaveTypeInput{
		Code:              r.FormValue("code"),
		Name:              r.FormValue("name"),
		Description:       r.FormValue("description"),
		DefaultAllocation: defaultAlloc,
	}

	_, err := h.service.CreateLeaveType(r.Context(), input)
	if err != nil {
		h.renderLeaveTypeFormError(w, r, false, nil, input, err.Error())
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/leave-types")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/leave-types", http.StatusSeeOther)
}

// HandleEditLeaveTypeForm renders the edit form for a leave type.
func (h *Handler) HandleEditLeaveTypeForm(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid leave type ID", http.StatusBadRequest)
		return
	}

	lt, err := h.service.GetLeaveTypeByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Leave type not found: "+err.Error(), http.StatusNotFound)
		return
	}

	templatePath := fmt.Sprintf("%s/leave_type_form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":     fmt.Sprintf("Edit %s - Leave Management System", lt.Name),
		"User":      user,
		"IsEdit":    true,
		"LeaveType": lt,
		"Error":     "",
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleUpdateLeaveType processes updates to a leave type.
func (h *Handler) HandleUpdateLeaveType(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid leave type ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	defaultAlloc, _ := strconv.Atoi(r.FormValue("default_allocation"))
	input := UpdateLeaveTypeInput{
		Code:              r.FormValue("code"),
		Name:              r.FormValue("name"),
		Description:       r.FormValue("description"),
		DefaultAllocation: defaultAlloc,
		Status:            r.FormValue("status"),
	}

	_, err = h.service.UpdateLeaveType(r.Context(), id, input)
	if err != nil {
		existing, _ := h.service.GetLeaveTypeByID(r.Context(), id)
		h.renderLeaveTypeFormError(w, r, true, existing, CreateLeaveTypeInput{}, err.Error())
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/leave-types")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/leave-types", http.StatusSeeOther)
}

// HandleActivateLeaveType activates a leave type.
func (h *Handler) HandleActivateLeaveType(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid leave type ID", http.StatusBadRequest)
		return
	}

	if err := h.service.ActivateLeaveType(r.Context(), id); err != nil {
		http.Error(w, "Failed to activate leave type: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/leave-types")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/leave-types", http.StatusSeeOther)
}

// HandleDeactivateLeaveType deactivates a leave type.
func (h *Handler) HandleDeactivateLeaveType(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid leave type ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeactivateLeaveType(r.Context(), id); err != nil {
		http.Error(w, "Failed to deactivate leave type: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/leave-types")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/leave-types", http.StatusSeeOther)
}

// HandleListBalances renders all employee leave allocations for Admin overview.
func (h *Handler) HandleListBalances(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)

	balances, err := h.service.ListAllBalances(r.Context())
	if err != nil {
		http.Error(w, "Failed to query leave balances: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templatePath := fmt.Sprintf("%s/admin_balances_list.html", h.templateDir)
	data := map[string]interface{}{
		"Title":    "Employee Leave Allocations - Leave Management System",
		"User":     user,
		"Balances": balances,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleNewAllocationForm renders the form to assign leave balance to an employee.
func (h *Handler) HandleNewAllocationForm(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)

	employees, err := h.empService.ListEmployees(r.Context())
	if err != nil {
		http.Error(w, "Failed to load employees: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter active employees only for allocation dropdown
	var activeEmployees []*employee.Employee
	for _, emp := range employees {
		if emp.Status == employee.StatusActive {
			activeEmployees = append(activeEmployees, emp)
		}
	}

	leaveTypes, err := h.service.ListLeaveTypes(r.Context())
	if err != nil {
		http.Error(w, "Failed to load leave types: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter active leave types only
	var activeLeaveTypes []LeaveType
	for _, lt := range leaveTypes {
		if lt.IsActive() {
			activeLeaveTypes = append(activeLeaveTypes, lt)
		}
	}

	templatePath := fmt.Sprintf("%s/allocation_form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":      "Assign Leave Allocation - Leave Management System",
		"User":       user,
		"Employees":  activeEmployees,
		"LeaveTypes": activeLeaveTypes,
		"Error":      "",
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleCreateAllocation processes assign leave balance submission.
func (h *Handler) HandleCreateAllocation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAllocationFormError(w, r, "Invalid form submission")
		return
	}

	empID, _ := strconv.ParseInt(r.FormValue("employee_id"), 10, 64)
	ltID, _ := strconv.ParseInt(r.FormValue("leave_type_id"), 10, 64)
	allocDays, err := strconv.Atoi(r.FormValue("allocated_days"))
	if err != nil {
		h.renderAllocationFormError(w, r, "Allocated days must be a valid number")
		return
	}

	input := AllocateLeaveInput{
		EmployeeID:    empID,
		LeaveTypeID:   ltID,
		AllocatedDays: allocDays,
	}

	_, err = h.service.AllocateLeave(r.Context(), input)
	if err != nil {
		h.renderAllocationFormError(w, r, err.Error())
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/leave-balances")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/leave-balances", http.StatusSeeOther)
}

// HandleMyLeaveBalances displays leave balance cards for the logged-in employee.
func (h *Handler) HandleMyLeaveBalances(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emp, balances, err := h.service.GetEmployeeBalancesByUserID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load leave balances for employee: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templatePath := fmt.Sprintf("%s/my_balances.html", h.templateDir)
	data := map[string]interface{}{
		"Title":    "My Leave Balance - Leave Management System",
		"User":     user,
		"Employee": emp,
		"Balances": balances,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

func (h *Handler) renderLeaveTypeFormError(w http.ResponseWriter, r *http.Request, isEdit bool, lt *LeaveType, input CreateLeaveTypeInput, errorMsg string) {
	if sharedHttp.IsHTMX(r) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="alert alert-danger" style="margin-bottom: 1.25rem;">%s</div>`, errorMsg)
		return
	}

	user := authentication.CurrentUser(r)
	templatePath := fmt.Sprintf("%s/leave_type_form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":     "Leave Type Form - Leave Management System",
		"User":      user,
		"IsEdit":    isEdit,
		"LeaveType": lt,
		"Input":     input,
		"Error":     errorMsg,
	}
	sharedHttp.RenderHTML(w, r, http.StatusBadRequest, data, h.layoutPath, templatePath)
}

func (h *Handler) renderAllocationFormError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	if sharedHttp.IsHTMX(r) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="alert alert-danger" style="margin-bottom: 1.25rem;">%s</div>`, errorMsg)
		return
	}

	user := authentication.CurrentUser(r)
	employees, _ := h.empService.ListEmployees(r.Context())
	var activeEmployees []*employee.Employee
	for _, emp := range employees {
		if emp.Status == employee.StatusActive {
			activeEmployees = append(activeEmployees, emp)
		}
	}

	leaveTypes, _ := h.service.ListLeaveTypes(r.Context())
	var activeLeaveTypes []LeaveType
	for _, lt := range leaveTypes {
		if lt.IsActive() {
			activeLeaveTypes = append(activeLeaveTypes, lt)
		}
	}

	templatePath := fmt.Sprintf("%s/allocation_form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":      "Assign Leave Allocation - Leave Management System",
		"User":       user,
		"Employees":  activeEmployees,
		"LeaveTypes": activeLeaveTypes,
		"Error":      errorMsg,
	}
	sharedHttp.RenderHTML(w, r, http.StatusBadRequest, data, h.layoutPath, templatePath)
}
