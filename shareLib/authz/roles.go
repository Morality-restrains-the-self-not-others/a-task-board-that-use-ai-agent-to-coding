package authz

// RoleDef describes a built-in system role (is_system=1, locked).
type RoleDef struct {
	Name        string
	DisplayName string
	Level       string // "platform" | "tenant"
	IsSystem    bool
}

// SystemRoles is the built-in role registry (v63 merged design §2.1).
// Permission mappings for platform roles are static (locked); tenant roles
// are expanded by the PDP into code sets injected via X-Tenant-Perms.
var SystemRoles = map[string]RoleDef{
	"super_admin":  {"super_admin", "超级管理员", "platform", true},
	"employee":     {"employee", "平台员工", "platform", true},
	"tenant_admin": {"tenant_admin", "租户管理员", "tenant", true},
	"group_admin":  {"group_admin", "小组管理员", "tenant", true},
	"member":       {"member", "成员", "tenant", true},
}

// platformRolePerms statically maps platform roles to their locked
// permission-code sets (v63 §2.1).
var platformRolePerms = map[string][]PermCode{
	"super_admin": {
		PermPlatformManage, PermTenantAudit, PermUserImpersonate,
		PermSystemConfig, PermBillingAudit, PermEmployeeManage,
	},
	"employee": {
		PermTenantAudit, PermUserImpersonate, PermBillingAudit,
	},
}

// platformRolePermSet is the static lookup set (fast path).
var platformRolePermSet = func() map[string]map[PermCode]bool {
	m := make(map[string]map[PermCode]bool, len(platformRolePerms))
	for role, perms := range platformRolePerms {
		set := make(map[PermCode]bool, len(perms))
		for _, p := range perms {
			set[p] = true
		}
		m[role] = set
	}
	return m
}()

// PlatformRolePerms returns the locked permission codes for a platform role.
func PlatformRolePerms(role string) map[PermCode]bool {
	return platformRolePermSet[role]
}

// HasPlatformRole reports whether the role exists in the built-in registry.
func HasPlatformRole(role string) bool {
	d, ok := SystemRoles[role]
	return ok && d.Level == "platform"
}
