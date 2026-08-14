package employee

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	sharedHttp "leavemang/internal/shared/http"
	"leavemang/internal/slices/authentication"
)

type Handler struct {
	service     *Service
	layoutPath  string
	templateDir string
}

func NewHandler(service *Service, layoutPath string, templateDir string) *Handler {
	return &Handler{
		service:     service,
		layoutPath:  layoutPath,
		templateDir: templateDir,
	}
}

// HandleListEmployees displays the employee directory. (Admin & Manager)
func (h *Handler) HandleListEmployees(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if !CanViewEmployeeList(user) {
		http.Error(w, "403 Forbidden: You do not have permission to view the employee directory.", http.StatusForbidden)
		return
	}

	employees, err := h.service.ListEmployees(r.Context())
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	listPath := fmt.Sprintf("%s/list.html", h.templateDir)
	data := map[string]interface{}{
		"Title":     "Employees - Leave Management System",
		"User":      user,
		"Employees": employees,
		"CanManage": CanManageEmployee(user),
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, listPath)
}

// HandleNewEmployeeForm renders the employee creation form. (Admin)
func (h *Handler) HandleNewEmployeeForm(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if !CanManageEmployee(user) {
		http.Error(w, "403 Forbidden: Only Administrators can create new employees.", http.StatusForbidden)
		return
	}

	formPath := fmt.Sprintf("%s/form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":  "Create Employee - Leave Management System",
		"User":   user,
		"IsEdit": false,
		"Input":  CreateEmployeeInput{Role: RoleEmployee},
		"Error":  "",
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, formPath)
}

// HandleCreateEmployee processes new employee creation. (Admin)
func (h *Handler) HandleCreateEmployee(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if !CanManageEmployee(user) {
		http.Error(w, "403 Forbidden: Only Administrators can create new employees.", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, r, false, nil, CreateEmployeeInput{}, "Invalid form data submitted")
		return
	}

	input := CreateEmployeeInput{
		FirstName:   r.FormValue("first_name"),
		LastName:    r.FormValue("last_name"),
		Email:       r.FormValue("email"),
		Phone:       r.FormValue("phone"),
		Department:  r.FormValue("department"),
		Designation: r.FormValue("designation"),
		JoiningDate: r.FormValue("joining_date"),
		Role:        r.FormValue("role"),
	}

	emp, err := h.service.CreateEmployee(r.Context(), input)
	if err != nil {
		h.renderFormError(w, r, false, nil, input, err.Error())
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/employees/%d", emp.ID))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/employees/%d", emp.ID), http.StatusSeeOther)
}

// HandleViewEmployee displays detailed information for a specific employee.
func (h *Handler) HandleViewEmployee(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}

	emp, err := h.service.GetEmployee(r.Context(), id)
	if err != nil {
		http.Error(w, "Employee not found: "+err.Error(), http.StatusNotFound)
		return
	}

	if !CanViewEmployeeDetails(user, emp) {
		http.Error(w, "403 Forbidden: You do not have permission to view this employee's details.", http.StatusForbidden)
		return
	}

	detailsPath := fmt.Sprintf("%s/details.html", h.templateDir)
	data := map[string]interface{}{
		"Title":     fmt.Sprintf("Employee %s - Leave Management System", emp.EmployeeCode),
		"User":      user,
		"Employee":  emp,
		"CanManage": CanManageEmployee(user),
		"IsSelf":    user.ID == emp.UserID,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, detailsPath)
}

// HandleEditEmployeeForm renders the employee edit form. (Admin)
func (h *Handler) HandleEditEmployeeForm(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if !CanManageEmployee(user) {
		http.Error(w, "403 Forbidden: Only Administrators can edit employees.", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}

	emp, err := h.service.GetEmployee(r.Context(), id)
	if err != nil {
		http.Error(w, "Employee not found: "+err.Error(), http.StatusNotFound)
		return
	}

	formPath := fmt.Sprintf("%s/form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":    fmt.Sprintf("Edit Employee %s - Leave Management System", emp.EmployeeCode),
		"User":     user,
		"IsEdit":   true,
		"Employee": emp,
		"Error":    "",
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, formPath)
}

// HandleUpdateEmployee processes employee update form submission. (Admin)
func (h *Handler) HandleUpdateEmployee(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if !CanManageEmployee(user) {
		http.Error(w, "403 Forbidden: Only Administrators can edit employees.", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	input := UpdateEmployeeInput{
		FirstName:   r.FormValue("first_name"),
		LastName:    r.FormValue("last_name"),
		Email:       r.FormValue("email"),
		Phone:       r.FormValue("phone"),
		Department:  r.FormValue("department"),
		Designation: r.FormValue("designation"),
		JoiningDate: r.FormValue("joining_date"),
		Role:        r.FormValue("role"),
		Status:      r.FormValue("status"),
	}

	emp, err := h.service.UpdateEmployee(r.Context(), id, input)
	if err != nil {
		existingEmp, _ := h.service.GetEmployee(r.Context(), id)
		h.renderFormError(w, r, true, existingEmp, CreateEmployeeInput{}, err.Error())
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/employees/%d", emp.ID))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/employees/%d", emp.ID), http.StatusSeeOther)
}

// HandleActivateEmployee sets an employee to active status. (Admin)
func (h *Handler) HandleActivateEmployee(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if !CanManageEmployee(user) {
		http.Error(w, "403 Forbidden: Only Administrators can activate employees.", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}

	if err := h.service.ActivateEmployee(r.Context(), id); err != nil {
		http.Error(w, "Failed to activate employee: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/employees")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/employees", http.StatusSeeOther)
}

// HandleDeactivateEmployee sets an employee to inactive status. (Admin)
func (h *Handler) HandleDeactivateEmployee(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if !CanManageEmployee(user) {
		http.Error(w, "403 Forbidden: Only Administrators can deactivate employees.", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid employee ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeactivateEmployee(r.Context(), id); err != nil {
		http.Error(w, "Failed to deactivate employee: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", "/employees")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/employees", http.StatusSeeOther)
}

// HandleViewProfile allows an authenticated user to view their own profile.
func (h *Handler) HandleViewProfile(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emp, err := h.service.GetEmployeeByUserID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Employee profile not found for user: "+err.Error(), http.StatusNotFound)
		return
	}

	detailsPath := fmt.Sprintf("%s/details.html", h.templateDir)
	data := map[string]interface{}{
		"Title":     "My Profile - Leave Management System",
		"User":      user,
		"Employee":  emp,
		"CanManage": CanManageEmployee(user),
		"IsSelf":    true,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, detailsPath)
}

// renderFormError handles validation errors during employee creation and updates.
func (h *Handler) renderFormError(w http.ResponseWriter, r *http.Request, isEdit bool, emp *Employee, input CreateEmployeeInput, errorMsg string) {
	if sharedHttp.IsHTMX(r) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="alert alert-danger" style="margin-bottom: 1.25rem;">%s</div>`, errorMsg)
		return
	}

	user := authentication.CurrentUser(r)
	formPath := fmt.Sprintf("%s/form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":    "Employee Form - Leave Management System",
		"User":     user,
		"IsEdit":   isEdit,
		"Employee": emp,
		"Input":    input,
		"Error":    errorMsg,
	}
	sharedHttp.RenderHTML(w, r, http.StatusBadRequest, data, h.layoutPath, formPath)
}
