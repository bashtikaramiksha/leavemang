package leave_request

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	sharedHttp "leavemang/internal/shared/http"
	"leavemang/internal/slices/authentication"
	"leavemang/internal/slices/leave"
)

type Handler struct {
	service      *Service
	leaveService *leave.Service
	layoutPath   string
	templateDir  string
}

func NewHandler(service *Service, leaveService *leave.Service, layoutPath, templateDir string) *Handler {
	return &Handler{
		service:      service,
		leaveService: leaveService,
		layoutPath:   layoutPath,
		templateDir:  templateDir,
	}
}

// HandleNewRequestForm renders the form for creating a new leave request.
func (h *Handler) HandleNewRequestForm(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	emp, balances, err := h.leaveService.GetEmployeeBalancesByUserID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load employee profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	leaveTypes, err := h.leaveService.ListLeaveTypes(r.Context())
	if err != nil {
		http.Error(w, "Failed to load leave types: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter for active leave types only (BR-03)
	var activeTypes []leave.LeaveType
	for _, lt := range leaveTypes {
		if lt.IsActive() {
			activeTypes = append(activeTypes, lt)
		}
	}

	templatePath := fmt.Sprintf("%s/form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":      "Apply for Leave - Leave Management System",
		"User":       user,
		"Employee":   emp,
		"Balances":   balances,
		"LeaveTypes": activeTypes,
		"Input":      CreateLeaveRequestInput{},
		"Error":      "",
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleCreateRequest processes the submission of a new leave request.
func (h *Handler) HandleCreateRequest(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, r, CreateLeaveRequestInput{}, "Invalid form data submission.")
		return
	}

	leaveTypeID, _ := strconv.ParseInt(r.FormValue("leave_type_id"), 10, 64)
	input := CreateLeaveRequestInput{
		LeaveTypeID: leaveTypeID,
		FromDate:    r.FormValue("from_date"),
		ToDate:      r.FormValue("to_date"),
		Reason:      r.FormValue("reason"),
	}

	createdReq, err := h.service.CreateRequest(r.Context(), user.ID, input)
	if err != nil {
		h.renderFormError(w, r, input, err.Error())
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/my/leave-requests/%d", createdReq.ID))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/my/leave-requests/%d", createdReq.ID), http.StatusSeeOther)
}

// renderFormError re-renders form with validation error message.
func (h *Handler) renderFormError(w http.ResponseWriter, r *http.Request, input CreateLeaveRequestInput, errMsg string) {
	user := authentication.CurrentUser(r)

	emp, balances, _ := h.leaveService.GetEmployeeBalancesByUserID(r.Context(), user.ID)
	leaveTypes, _ := h.leaveService.ListLeaveTypes(r.Context())

	var activeTypes []leave.LeaveType
	for _, lt := range leaveTypes {
		if lt.IsActive() {
			activeTypes = append(activeTypes, lt)
		}
	}

	templatePath := fmt.Sprintf("%s/form.html", h.templateDir)
	data := map[string]interface{}{
		"Title":      "Apply for Leave - Leave Management System",
		"User":       user,
		"Employee":   emp,
		"Balances":   balances,
		"LeaveTypes": activeTypes,
		"Input":      input,
		"Error":      errMsg,
	}

	sharedHttp.RenderHTML(w, r, http.StatusUnprocessableEntity, data, h.layoutPath, templatePath)
}

// HandleListEmployeeRequests lists the authenticated employee's submitted leave requests.
func (h *Handler) HandleListEmployeeRequests(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	requests, err := h.service.ListEmployeeRequests(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load leave requests: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templatePath := fmt.Sprintf("%s/list.html", h.templateDir)
	data := map[string]interface{}{
		"Title":    "My Leave Requests - Leave Management System",
		"User":     user,
		"Requests": requests,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleRequestDetails displays details of a specific leave request.
func (h *Handler) HandleRequestDetails(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	reqDetails, err := h.service.GetRequest(r.Context(), id, user.ID)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, "Forbidden: You are not authorized to view this leave request", http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	templatePath := fmt.Sprintf("%s/details.html", h.templateDir)
	data := map[string]interface{}{
		"Title":   "Leave Request Details - Leave Management System",
		"User":    user,
		"Request": reqDetails,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleListPendingRequests renders team leave requests with optional status and employee filtering (VS-05 / VS-06).
func (h *Handler) HandleListPendingRequests(w http.ResponseWriter, r *http.Request) {
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

	if statusFilter == "" && r.URL.Query().Get("employee_id") == "" {
		statusFilter = "pending"
	}

	requests, err := h.service.repo.ListWithFilters(r.Context(), statusFilter, empIDFilter)
	if err != nil {
		http.Error(w, "Failed to load leave requests: "+err.Error(), http.StatusInternalServerError)
		return
	}

	templatePath := fmt.Sprintf("%s/manager_list.html", h.templateDir)
	data := map[string]interface{}{
		"Title":            "Team Leave Requests - Manager Dashboard",
		"User":             user,
		"Requests":         requests,
		"StatusFilter":     statusFilter,
		"EmployeeIDFilter": empIDFilter,
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleManagerRequestDetails renders full request details for manager decision (AR-02).
func (h *Handler) HandleManagerRequestDetails(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	reqDetails, bal, err := h.service.GetManagerRequestDetails(r.Context(), id)
	if err != nil {
		http.Error(w, "Request not found: "+err.Error(), http.StatusNotFound)
		return
	}

	currentBal := 0
	if bal != nil {
		currentBal = bal.Remaining()
	}
	projectedBal := currentBal - reqDetails.Days

	templatePath := fmt.Sprintf("%s/manager_details.html", h.templateDir)
	data := map[string]interface{}{
		"Title":                "Review Leave Request - Leave Management System",
		"User":                 user,
		"Request":              reqDetails,
		"CurrentBalance":       currentBal,
		"ProjectedBalance":     projectedBal,
		"Error":                "",
	}

	sharedHttp.RenderHTML(w, r, http.StatusOK, data, h.layoutPath, templatePath)
}

// HandleApproveRequest processes approval of a pending leave request (AR-03, AR-05, AR-06).
func (h *Handler) HandleApproveRequest(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	if err := h.service.ApproveRequest(r.Context(), id, user.ID); err != nil {
		reqDetails, bal, _ := h.service.GetManagerRequestDetails(r.Context(), id)
		currentBal := 0
		if bal != nil {
			currentBal = bal.Remaining()
		}
		projectedBal := currentBal
		if reqDetails != nil {
			projectedBal = currentBal - reqDetails.Days
		}

		templatePath := fmt.Sprintf("%s/manager_details.html", h.templateDir)
		data := map[string]interface{}{
			"Title":            "Review Leave Request - Leave Management System",
			"User":             user,
			"Request":          reqDetails,
			"CurrentBalance":   currentBal,
			"ProjectedBalance": projectedBal,
			"Error":            err.Error(),
		}
		sharedHttp.RenderHTML(w, r, http.StatusUnprocessableEntity, data, h.layoutPath, templatePath)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/manager/leave-requests/%d", id))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/manager/leave-requests/%d", id), http.StatusSeeOther)
}

// HandleRejectRequest processes rejection of a pending leave request with reason (AR-04, AR-05, AR-07).
func (h *Handler) HandleRejectRequest(w http.ResponseWriter, r *http.Request) {
	user := authentication.CurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	rejectionReason := r.FormValue("rejection_reason")

	if err := h.service.RejectRequest(r.Context(), id, user.ID, rejectionReason); err != nil {
		reqDetails, bal, _ := h.service.GetManagerRequestDetails(r.Context(), id)
		currentBal := 0
		if bal != nil {
			currentBal = bal.Remaining()
		}
		projectedBal := currentBal
		if reqDetails != nil {
			projectedBal = currentBal - reqDetails.Days
		}

		templatePath := fmt.Sprintf("%s/manager_details.html", h.templateDir)
		data := map[string]interface{}{
			"Title":            "Review Leave Request - Leave Management System",
			"User":             user,
			"Request":          reqDetails,
			"CurrentBalance":   currentBal,
			"ProjectedBalance": projectedBal,
			"Error":            err.Error(),
		}
		sharedHttp.RenderHTML(w, r, http.StatusUnprocessableEntity, data, h.layoutPath, templatePath)
		return
	}

	if sharedHttp.IsHTMX(r) {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/manager/leave-requests/%d", id))
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/manager/leave-requests/%d", id), http.StatusSeeOther)
}
