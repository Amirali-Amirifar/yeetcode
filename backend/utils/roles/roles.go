package roles

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// Permissions represents the set of permissions for a role
type Permissions struct {
	CanCreateProblems  bool
	CanEditProblems    bool
	CanDeleteProblems  bool
	CanViewAllProblems bool
	CanManageUsers     bool
	CanPublishProblems bool
}

// GetPermissions returns the permissions for a given role
func GetPermissions(role string) Permissions {
	switch role {
	case RoleAdmin:
		return Permissions{
			CanCreateProblems:  true,
			CanEditProblems:    true,
			CanDeleteProblems:  true,
			CanViewAllProblems: true,
			CanManageUsers:     true,
			CanPublishProblems: true,
		}
	case RoleUser:
		return Permissions{
			CanCreateProblems:  true,
			CanEditProblems:    false,
			CanDeleteProblems:  false,
			CanViewAllProblems: false,
			CanManageUsers:     false,
			CanPublishProblems: false,
		}
	default:
		return Permissions{} // No permissions for unknown roles
	}
}

// HasPermission checks if a role has a specific permission
func HasPermission(role string, permission string) bool {
	perms := GetPermissions(role)
	switch permission {
	case "create_problems":
		return perms.CanCreateProblems
	case "edit_problems":
		return perms.CanEditProblems
	case "delete_problems":
		return perms.CanDeleteProblems
	case "view_all_problems":
		return perms.CanViewAllProblems
	case "manage_users":
		return perms.CanManageUsers
	case "publish_problems":
		return perms.CanPublishProblems
	default:
		return false
	}
}
