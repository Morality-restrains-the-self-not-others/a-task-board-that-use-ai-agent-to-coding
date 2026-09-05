// Design (v63 RBAC merged design, 2026-08-05 + v72 logical resource groups):
//   - APISIX forward-auth → taskAuth PDP computes the user's permission-code
//     sets per tenant (direct roles ∪ group-role inheritance ∪ group-resource
//     inheritance ∪ region:*/page:* from auth_role_resource_group) and injects:
//     X-User-Roles     comma-separated platform roles (super_admin,employee)
//     X-Tenant-Perms   cid:perm1,perm2;cid2:perm1  (semicolon=tenant,
//     colon=separator after company id, comma=permission codes;
//     codes themselves may contain colons e.g. region:x.y)
//   - Services read the headers once in middleware, store an AuthContext in
//     the request context, and call RequirePerm / RequireRegion / HasPerm (O(1)).
//   - CheckPermission HTTP fallback to the PDP exists for header-less flows
//     (internal calls, tests).
//
// Platform permissions are statically mapped from X-User-Roles (5 built-in
// roles are locked); tenant permissions are injected as code sets (custom
// roles are expanded by the PDP at forward-auth time).
package authz

import (
	"sort"
	"strings"
)

// PermCode is a compile-time-safe permission code.
type PermCode string

// ════════════════ Platform ════════════════
const (
	PermPlatformManage  PermCode = "platform:manage"  // 平台全局管理
	PermTenantAudit     PermCode = "tenant:audit"     // 跨租户审计
	PermUserImpersonate PermCode = "user:impersonate" // 模拟用户登录
	PermSystemConfig    PermCode = "system:config"    // 系统配置管理
	PermBillingAudit    PermCode = "billing:audit"    // 跨租户账单审计
	PermEmployeeManage  PermCode = "employee:manage"  // 员工管理
)

// ════════════════ Tenant resources ════════════════
const (
	PermCompanyManage   PermCode = "company:manage"
	PermCompanyView     PermCode = "company:view"
	PermMemberManage    PermCode = "member:manage"
	PermMemberView      PermCode = "member:view"
	PermGroupManage     PermCode = "group:manage"
	PermProjectManage   PermCode = "project:manage"
	PermProjectView     PermCode = "project:view"
	PermTaskManage      PermCode = "task:manage"
	PermTaskView        PermCode = "task:view"
	PermCloudManage     PermCode = "cloud:manage"
	PermCloudView       PermCode = "cloud:view"
	PermBillingManage   PermCode = "billing:manage"
	PermBillingView     PermCode = "billing:view"
	PermWorkspaceManage PermCode = "workspace:manage"
)

// ════════════════ Tenant groups (built-in roles only) ════════════════
const (
	PermGroupMembersManage PermCode = "group-members:manage"   // 管理组内成员 (group_admin)
	PermGroupResourcesView PermCode = "group-resources:view"   // 查看组关联资源 (group_admin)
	PermGroupResourcesMng  PermCode = "group-resources:manage" // 管理组关联资源 (tenant_admin)
)

// ════════════════ Logical resource groups (v72 + v73 effects) ════════════════
// Injected by taskAuth PDP as region:<group_key> / page:<group_key> (legacy full),
// plus effect-qualified codes region:<key>:view|operate (ADR-0004).
// Not stored in auth_permission; orthogonal to coarse tenant codes.

// Grant effect on auth_role_resource_group (operate ⊃ view).
const (
	EffectView    = "view"
	EffectOperate = "operate"
)

// RegionPermCode returns the legacy PDP code for a ui_region (operate grants).
func RegionPermCode(regionKey string) PermCode {
	return PermCode("region:" + regionKey)
}

// RegionViewPermCode / RegionOperatePermCode are effect-qualified region codes.
func RegionViewPermCode(regionKey string) PermCode {
	return PermCode("region:" + regionKey + ":view")
}

func RegionOperatePermCode(regionKey string) PermCode {
	return PermCode("region:" + regionKey + ":operate")
}

// PagePermCode returns the legacy PDP code for a page (operate grants).
func PagePermCode(pageKey string) PermCode {
	return PermCode("page:" + pageKey)
}

func PageViewPermCode(pageKey string) PermCode {
	return PermCode("page:" + pageKey + ":view")
}

func PageOperatePermCode(pageKey string) PermCode {
	return PermCode("page:" + pageKey + ":operate")
}

// NormalizeGrantEffect returns view|operate; empty/unknown → operate (v72 compat).
func NormalizeGrantEffect(effect string) string {
	switch strings.ToLower(strings.TrimSpace(effect)) {
	case EffectView:
		return EffectView
	default:
		return EffectOperate
	}
}

// MaxGrantEffect returns the stronger of two effects (operate > view).
// Empty side is ignored (does not default the other side to operate).
func MaxGrantEffect(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return NormalizeGrantEffect(b)
	}
	if b == "" {
		return NormalizeGrantEffect(a)
	}
	a, b = NormalizeGrantEffect(a), NormalizeGrantEffect(b)
	if a == EffectOperate || b == EffectOperate {
		return EffectOperate
	}
	return EffectView
}

// EmitLogicalRGCodes expands region/page effect maps into sorted PDP codes.
// operate → :view + :operate + legacy bare code; view → :view only.
func EmitLogicalRGCodes(regionEffects, pageEffects map[string]string) []string {
	set := map[string]bool{}
	for key, eff := range regionEffects {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		eff = NormalizeGrantEffect(eff)
		set[string(RegionViewPermCode(key))] = true
		if eff == EffectOperate {
			set[string(RegionOperatePermCode(key))] = true
			set[string(RegionPermCode(key))] = true
		}
	}
	for key, eff := range pageEffects {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		eff = NormalizeGrantEffect(eff)
		set[string(PageViewPermCode(key))] = true
		if eff == EffectOperate {
			set[string(PageOperatePermCode(key))] = true
			set[string(PagePermCode(key))] = true
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// AllTenantPerms returns the 17 tenant-level permission codes (custom-role pick list).
func AllTenantPerms() []PermCode {
	return []PermCode{
		PermCompanyManage, PermCompanyView,
		PermMemberManage, PermMemberView,
		PermGroupManage,
		PermProjectManage, PermProjectView,
		PermTaskManage, PermTaskView,
		PermCloudManage, PermCloudView,
		PermBillingManage, PermBillingView,
		PermWorkspaceManage,
	}
}

// AllPerms returns all 23 permission codes (seed data source).
func AllPerms() []PermCode {
	all := []PermCode{
		PermPlatformManage, PermTenantAudit, PermUserImpersonate,
		PermSystemConfig, PermBillingAudit, PermEmployeeManage,
	}
	all = append(all, AllTenantPerms()...)
	all = append(all, PermGroupMembersManage, PermGroupResourcesView, PermGroupResourcesMng)
	return all
}
