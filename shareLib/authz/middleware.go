package authz

import (
	"encoding/json"
	"net/http"
)

// Middleware parses the gateway-injected headers once and stores the
// AuthContext in the request context. Wire this in each service's router:
//
//	router.Use(authz.Middleware)
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, WithContext(r, ParseFromHeaders(r)))
	})
}

func writeAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"detail": msg})
}

// RequirePerm requires the user to hold perm in companyID (O(1) set lookup).
// On failure writes a 403 JSON response and returns false.
func RequirePerm(w http.ResponseWriter, r *http.Request, perm PermCode, companyID string) bool {
	ac := AuthContextFromHeaders(r)
	if ac.HasPerm(companyID, perm) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "权限不足，需要权限码: "+string(perm))
	return false
}

// RequireRegion requires any access to the region (view or operate / legacy).
// Prefer RequireRegionOperate for mutating APIs (ADR-0004).
func RequireRegion(w http.ResponseWriter, r *http.Request, regionKey, companyID string) bool {
	if HasRegion(r, companyID, regionKey) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "权限不足，需要资源区域: "+regionKey)
	return false
}

// RequireRegionView requires view-level access (view, operate, or legacy bare code).
func RequireRegionView(w http.ResponseWriter, r *http.Request, regionKey, companyID string) bool {
	if HasRegionView(r, companyID, regionKey) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "权限不足，需要资源区域只读: "+regionKey)
	return false
}

// RequireRegionOperate requires operate-level access (edit/execute) or legacy bare code.
func RequireRegionOperate(w http.ResponseWriter, r *http.Request, regionKey, companyID string) bool {
	if HasRegionOperate(r, companyID, regionKey) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "权限不足，需要资源区域可编辑执行: "+regionKey)
	return false
}

// HasRegion reports whether the user has any access to region:<key> (view|operate|legacy).
func HasRegion(r *http.Request, companyID, regionKey string) bool {
	return HasRegionView(r, companyID, regionKey)
}

// HasRegionView is true for view, operate, or legacy region:<key>.
func HasRegionView(r *http.Request, companyID, regionKey string) bool {
	ac := AuthContextFromHeaders(r)
	return ac.HasPerm(companyID, RegionViewPermCode(regionKey)) ||
		ac.HasPerm(companyID, RegionOperatePermCode(regionKey)) ||
		ac.HasPerm(companyID, RegionPermCode(regionKey))
}

// HasRegionOperate is true for operate-qualified or legacy region:<key> (not view-only).
func HasRegionOperate(r *http.Request, companyID, regionKey string) bool {
	ac := AuthContextFromHeaders(r)
	return ac.HasPerm(companyID, RegionOperatePermCode(regionKey)) ||
		ac.HasPerm(companyID, RegionPermCode(regionKey))
}

// MethodRequiresOperate reports whether the HTTP method is a mutating
// (write/execute) method that requires operate-level region access.
// Read-only methods (GET/HEAD/OPTIONS) only need view.
//
// OPT-20260811-046 / v73 §8: auto-infer required_effect from the HTTP method so
// handlers stop hand-picking RequireRegionView vs RequireRegionOperate — a
// mutating handler that only checks view would let a view-only member write.
func MethodRequiresOperate(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// HasRegionByMethod enforces region access with the effect inferred from the
// request method: read-only methods need view; mutating methods need operate.
// Prefer this over hand-choosing HasRegionView/HasRegionOperate on write paths.
func HasRegionByMethod(r *http.Request, regionKey, companyID string) bool {
	if MethodRequiresOperate(r.Method) {
		return HasRegionOperate(r, companyID, regionKey)
	}
	return HasRegionView(r, companyID, regionKey)
}

// HasPeopleAccessWrite is the dual-read write gate for 访问管理:
// v73 operate on subject_list / region_matrix, or legacy save_actions, or member:manage.
func HasPeopleAccessWrite(r *http.Request, companyID string) bool {
	if HasPerm(r, companyID, PermMemberManage) {
		return true
	}
	return HasRegionByMethod(r, "people.access.subject_list", companyID) ||
		HasRegionByMethod(r, "people.access.region_matrix", companyID) ||
		HasRegionByMethod(r, "people.access.save_actions", companyID)
}

// RequireRegionByMethod is the 403-writing variant of HasRegionByMethod: it
// auto-requires operate for mutating methods and view for read-only methods.
func RequireRegionByMethod(w http.ResponseWriter, r *http.Request, regionKey, companyID string) bool {
	if MethodRequiresOperate(r.Method) {
		return RequireRegionOperate(w, r, regionKey, companyID)
	}
	return RequireRegionView(w, r, regionKey, companyID)
}

// HasPage reports whether the user holds any page access (legacy|view|operate).
func HasPage(r *http.Request, companyID, pageKey string) bool {
	ac := AuthContextFromHeaders(r)
	return ac.HasPerm(companyID, PagePermCode(pageKey)) ||
		ac.HasPerm(companyID, PageViewPermCode(pageKey)) ||
		ac.HasPerm(companyID, PageOperatePermCode(pageKey))
}

// RequirePlatformPerm requires a platform permission code (statically mapped
// from X-User-Roles).
func RequirePlatformPerm(w http.ResponseWriter, r *http.Request, perm PermCode) bool {
	ac := AuthContextFromHeaders(r)
	if ac.HasPlatformPerm(perm) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "权限不足，需要平台权限码: "+string(perm))
	return false
}

// RequireTenantMember requires the user to be an active member of companyID
// (holds at least one permission code, i.e. the member role baseline). Group
// membership is already reflected in the injected code set.
func RequireTenantMember(w http.ResponseWriter, r *http.Request, companyID string) bool {
	ac := AuthContextFromHeaders(r)
	if len(ac.TenantPerms[companyID]) > 0 {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "您不是该公司的成员")
	return false
}

// RequireGroupAdmin requires the user to be the admin of groupID. tenant_admin
// always passes (full tenant perms imply group scope). Requires the injected
// code set of the user to contain group-members:manage.
func RequireGroupAdmin(w http.ResponseWriter, r *http.Request, companyID, groupID string) bool {
	ac := AuthContextFromHeaders(r)
	// group-members:manage is a tenant-scoped code granted to group_admin
	// (PDP expands per-company, group scope is enforced by the handler
	// comparing the target groupID against the user's admin groups).
	if ac.HasPerm(companyID, PermGroupMembersManage) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "仅组长或租户管理员可执行此操作")
	return false
}

// HasPerm is the boolean form (non-handler / frontend-facing logic).
func HasPerm(r *http.Request, companyID string, perm PermCode) bool {
	return AuthContextFromHeaders(r).HasPerm(companyID, perm)
}

// HasPlatformPerm is the boolean form.
func HasPlatformPerm(r *http.Request, perm PermCode) bool {
	return AuthContextFromHeaders(r).HasPlatformPerm(perm)
}

// HasGroupResourceAccess reports whether the user's group memberships grant
// access to resourceType/resourceID in companyID. The PDP injects group
// resource inheritance as per-resource pseudo-codes:
//
//	group-res:<resourceType>:<resourceID>:view|manage
//
// in X-Tenant-Perms, so this is still an O(1) set lookup.
func HasGroupResourceAccess(r *http.Request, companyID, resourceType, resourceID string) bool {
	ac := AuthContextFromHeaders(r)
	if ac == nil {
		return false
	}
	set := ac.TenantPerms[companyID]
	for _, perm := range []PermCode{
		groupResourcePerm(resourceType, resourceID, "manage"),
		groupResourcePerm(resourceType, resourceID, "view"),
	} {
		if set[perm] {
			return true
		}
	}
	return false
}

// GroupResourcePermCode returns the pseudo-code for group resource access.
// Export for PDP side to emit the same key format.
func GroupResourcePermCode(resourceType, resourceID, access string) PermCode {
	return groupResourcePerm(resourceType, resourceID, access)
}

func groupResourcePerm(resourceType, resourceID, access string) PermCode {
	return PermCode("group-res:" + resourceType + ":" + resourceID + ":" + access)
}
