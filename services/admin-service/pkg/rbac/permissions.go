package rbac

import (
	"fmt"
	"strings"

	"github.com/real-ass/admin-service/internal/domain"
)

// Resource represents an RBAC resource type.
type Resource string

const (
	ResourceUsers            Resource = "users"
	ResourceRoles            Resource = "roles"
	ResourceSubscriptions    Resource = "subscriptions"
	ResourceAuditLogs        Resource = "audit_logs"
	ResourceSystemSettings   Resource = "system_settings"
	ResourceReports          Resource = "reports"
)

// Action represents an RBAC action.
type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionList    Action = "list"
	ActionExport  Action = "export"
	ActionImport  Action = "import"
)

// Permission represents an RBAC permission (resource:action).
type Permission struct {
	Resource Resource
	Action   Action
}

// String returns the string representation of a permission.
func (p Permission) String() string {
	return fmt.Sprintf("%s:%s", p.Resource, p.Action)
}

// ParsePermission parses a permission string in the format "resource:action".
func ParsePermission(s string) (*Permission, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid permission format: %s, expected resource:action", s)
	}

	return &Permission{
		Resource: Resource(parts[0]),
		Action:   Action(parts[1]),
	}, nil
}

// RolePermissions defines the default permissions for each role.
var RolePermissions = map[domain.UserRole][]Permission{
	domain.RoleSuperAdmin: {
		{ResourceUsers, ActionCreate},
		{ResourceUsers, ActionRead},
		{ResourceUsers, ActionUpdate},
		{ResourceUsers, ActionDelete},
		{ResourceUsers, ActionList},
		{ResourceUsers, ActionExport},
		{ResourceRoles, ActionCreate},
		{ResourceRoles, ActionRead},
		{ResourceRoles, ActionUpdate},
		{ResourceRoles, ActionDelete},
		{ResourceRoles, ActionList},
		{ResourceSubscriptions, ActionCreate},
		{ResourceSubscriptions, ActionRead},
		{ResourceSubscriptions, ActionUpdate},
		{ResourceSubscriptions, ActionDelete},
		{ResourceSubscriptions, ActionList},
		{ResourceAuditLogs, ActionRead},
		{ResourceAuditLogs, ActionList},
		{ResourceAuditLogs, ActionExport},
		{ResourceSystemSettings, ActionCreate},
		{ResourceSystemSettings, ActionRead},
		{ResourceSystemSettings, ActionUpdate},
		{ResourceSystemSettings, ActionDelete},
		{ResourceReports, ActionCreate},
		{ResourceReports, ActionRead},
		{ResourceReports, ActionExport},
	},
	domain.RoleAdmin: {
		{ResourceUsers, ActionCreate},
		{ResourceUsers, ActionRead},
		{ResourceUsers, ActionUpdate},
		{ResourceUsers, ActionList},
		{ResourceUsers, ActionExport},
		{ResourceRoles, ActionRead},
		{ResourceRoles, ActionList},
		{ResourceSubscriptions, ActionCreate},
		{ResourceSubscriptions, ActionRead},
		{ResourceSubscriptions, ActionUpdate},
		{ResourceSubscriptions, ActionList},
		{ResourceAuditLogs, ActionRead},
		{ResourceAuditLogs, ActionList},
		{ResourceReports, ActionCreate},
		{ResourceReports, ActionRead},
		{ResourceReports, ActionExport},
	},
	domain.RoleModerator: {
		{ResourceUsers, ActionRead},
		{ResourceUsers, ActionUpdate},
		{ResourceUsers, ActionList},
		{ResourceSubscriptions, ActionRead},
		{ResourceSubscriptions, ActionList},
		{ResourceAuditLogs, ActionRead},
		{ResourceAuditLogs, ActionList},
		{ResourceReports, ActionRead},
	},
	domain.RoleUser: {
		{ResourceUsers, ActionRead},
		{ResourceSubscriptions, ActionRead},
		{ResourceReports, ActionRead},
	},
}

// HasPermission checks if a role has the specified permission.
func HasPermission(role domain.UserRole, permission Permission) bool {
	perms, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range perms {
		if p.Resource == permission.Resource && p.Action == permission.Action {
			return true
		}
	}

	return false
}

// HasPermissions checks if a role has all the specified permissions.
func HasPermissions(role domain.UserRole, permissions []Permission) bool {
	for _, perm := range permissions {
		if !HasPermission(role, perm) {
			return false
		}
	}

	return true
}

// GetPermissionsForRole returns all permissions for a given role.
func GetPermissionsForRole(role domain.UserRole) []Permission {
	perms, exists := RolePermissions[role]
	if !exists {
		return []Permission{}
	}

	return perms
}

// CanManageRole checks if a role can manage another role (hierarchy check).
func CanManageRole(managingRole, targetRole domain.UserRole) bool {
	roleHierarchy := map[domain.UserRole]int{
		domain.RoleSuperAdmin: 4,
		domain.RoleAdmin:      3,
		domain.RoleModerator:  2,
		domain.RoleUser:       1,
	}

	managingLevel, managingExists := roleHierarchy[managingRole]
	targetLevel, targetExists := roleHierarchy[targetRole]

	if !managingExists || !targetExists {
		return false
	}

	return managingLevel > targetLevel
}

// Enforce checks if the given role can perform the action on the resource.
// Returns an error if the permission is denied.
func Enforce(role domain.UserRole, resource Resource, action Action) error {
	perm := Permission{Resource: resource, Action: action}
	if !HasPermission(role, perm) {
		return &PermissionDeniedError{
			Role:       role,
			Resource:   resource,
			Action:     action,
			Permission: perm.String(),
		}
	}

	return nil
}

// PermissionDeniedError is returned when a role does not have the required permission.
type PermissionDeniedError struct {
	Role       domain.UserRole
	Resource   Resource
	Action     Action
	Permission string
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf("permission denied: role '%s' does not have permission '%s'", e.Role, e.Permission)
}
