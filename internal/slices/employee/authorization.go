package employee

import (
	"leavemang/internal/slices/authentication"
)

// CanViewEmployeeList checks whether the authenticated user has permission to view the employee list.
func CanViewEmployeeList(user *authentication.User) bool {
	if user == nil {
		return false
	}
	return user.Role == authentication.RoleAdmin || user.Role == authentication.RoleManager
}

// CanViewEmployeeDetails checks whether the authenticated user can view a specific employee's details.
func CanViewEmployeeDetails(user *authentication.User, targetEmployee *Employee) bool {
	if user == nil || targetEmployee == nil {
		return false
	}
	if user.Role == authentication.RoleAdmin || user.Role == authentication.RoleManager {
		return true
	}
	// Employee can view their own profile
	return user.ID == targetEmployee.UserID
}

// CanManageEmployee checks whether the authenticated user has full employee management rights (Create, Edit, Activate, Deactivate).
func CanManageEmployee(user *authentication.User) bool {
	if user == nil {
		return false
	}
	return user.Role == authentication.RoleAdmin
}
