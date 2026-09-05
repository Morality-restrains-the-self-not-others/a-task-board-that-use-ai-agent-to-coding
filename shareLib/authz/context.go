package authz

import (
	"context"
	"net/http"
	"strings"
)

// Header names injected by APISIX forward-auth (v63 design).
const (
	HeaderUserID        = "X-User-Id"
	HeaderUserRoles     = "X-User-Roles"   // comma-separated platform roles
	HeaderTenantPerms   = "X-Tenant-Perms" // cid:perm1,perm2;cid2:perm1
	HeaderGatewayVerify = "X-Gateway-Auth-Verified"
)

// AuthContext is the per-request authorization context injected by the
// gateway headers (parsed once in middleware).
type AuthContext struct {
	UserID        string
	PlatformRoles []string
	// TenantPerms maps companyID → permission-code set. Custom roles and
	// group inheritance are already expanded by the PDP at forward-auth time.
	TenantPerms map[string]map[PermCode]bool
}

type ctxKey struct{}

// ParseFromHeaders builds an AuthContext from the injected gateway headers.
func ParseFromHeaders(r *http.Request) *AuthContext {
	ac := &AuthContext{
		UserID:      r.Header.Get(HeaderUserID),
		TenantPerms: make(map[string]map[PermCode]bool),
	}
	if v := r.Header.Get(HeaderUserRoles); v != "" {
		ac.PlatformRoles = splitCSV(v)
	}
	if v := r.Header.Get(HeaderTenantPerms); v != "" {
		for _, tenant := range strings.Split(v, ";") {
			tenant = strings.TrimSpace(tenant)
			if tenant == "" {
				continue
			}
			cid, perms, ok := strings.Cut(tenant, ":")
			if !ok || cid == "" {
				continue
			}
			set := make(map[PermCode]bool)
			for _, p := range splitCSV(perms) {
				set[PermCode(p)] = true
			}
			ac.TenantPerms[cid] = set
		}
	}
	return ac
}

// WithContext stores the AuthContext in the request context.
func WithContext(r *http.Request, ac *AuthContext) *http.Request {
	if ac == nil {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), ctxKey{}, ac))
}

// FromContext extracts the AuthContext stored by WithContext.
func FromContext(r *http.Request) (*AuthContext, bool) {
	ac, ok := r.Context().Value(ctxKey{}).(*AuthContext)
	return ac, ok && ac != nil
}

// AuthContextFromHeaders is the fallback: parse headers on demand when no
// middleware wrapped the request (internal calls, tests).
func AuthContextFromHeaders(r *http.Request) *AuthContext {
	if ac, ok := FromContext(r); ok {
		return ac
	}
	return ParseFromHeaders(r)
}

// HasPerm reports whether the user holds perm in companyID.
func (ac *AuthContext) HasPerm(companyID string, perm PermCode) bool {
	if ac == nil {
		return false
	}
	set, ok := ac.TenantPerms[companyID]
	return ok && set[perm]
}

// HasPlatformPerm reports whether the user holds a platform permission code
// (statically resolved from X-User-Roles).
func (ac *AuthContext) HasPlatformPerm(perm PermCode) bool {
	if ac == nil {
		return false
	}
	for _, role := range ac.PlatformRoles {
		if platformRolePermSet[role][perm] {
			return true
		}
	}
	return false
}

// HasPlatformRole reports whether the user holds the given platform role
// (e.g. "super_admin", "employee" from X-User-Roles).
func (ac *AuthContext) HasPlatformRole(role string) bool {
	if ac == nil || role == "" {
		return false
	}
	for _, r := range ac.PlatformRoles {
		if r == role {
			return true
		}
	}
	return false
}

// IsPlatformStaff reports whether the user holds any platform role
// (super_admin or employee). v63 统一入口：取代遗留
// X-Auth-Superuser / X-Auth-Staff 头的"超管或员工"判定。
func IsPlatformStaff(r *http.Request) bool {
	ac := AuthContextFromHeaders(r)
	return ac.HasPlatformRole("super_admin") || ac.HasPlatformRole("employee")
}

// TenantIDs returns the company IDs the user has any permission in.
func (ac *AuthContext) TenantIDs() []string {
	if ac == nil {
		return nil
	}
	ids := make([]string, 0, len(ac.TenantPerms))
	for cid := range ac.TenantPerms {
		ids = append(ids, cid)
	}
	return ids
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
