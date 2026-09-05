package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newAuthedRequest builds a request with gateway-injected headers.
func newAuthedRequest(userID, roles, tenantPerms string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/todos/", nil)
	r.Header.Set(HeaderUserID, userID)
	if roles != "" {
		r.Header.Set(HeaderUserRoles, roles)
	}
	if tenantPerms != "" {
		r.Header.Set(HeaderTenantPerms, tenantPerms)
	}
	return r
}

func TestHasPerm_FromHeaders(t *testing.T) {
	r := newAuthedRequest("u1", "", "t1:company:view,task:manage;t2:project:view")
	ac := ParseFromHeaders(r)
	if !ac.HasPerm("t1", PermTaskManage) {
		t.Fatal("expected task:manage in t1")
	}
	if !ac.HasPerm("t1", PermCompanyView) {
		t.Fatal("expected company:view in t1")
	}
	if ac.HasPerm("t1", PermCloudManage) {
		t.Fatal("cloud:manage should not be granted")
	}
	if ac.HasPerm("t9", PermTaskManage) {
		t.Fatal("t9 not in set")
	}
	if ac.HasPerm("t2", PermCompanyView) {
		t.Fatal("company:view not in t2")
	}
}

func TestHasPlatformPerm_StaticMapping(t *testing.T) {
	// super_admin → all platform codes
	sa := ParseFromHeaders(newAuthedRequest("u1", "super_admin", ""))
	for _, p := range []PermCode{PermPlatformManage, PermTenantAudit, PermEmployeeManage} {
		if !sa.HasPlatformPerm(p) {
			t.Fatalf("super_admin should hold %s", p)
		}
	}
	// employee → audit subset, not platform:manage
	emp := ParseFromHeaders(newAuthedRequest("u2", "employee", ""))
	if !emp.HasPlatformPerm(PermTenantAudit) {
		t.Fatal("employee should hold tenant:audit")
	}
	if emp.HasPlatformPerm(PermPlatformManage) {
		t.Fatal("employee must not hold platform:manage")
	}
	// no roles → nothing
	none := ParseFromHeaders(newAuthedRequest("u3", "", ""))
	if none.HasPlatformPerm(PermTenantAudit) {
		t.Fatal("no roles → no platform perms")
	}
}

func TestMiddleware_WithContext(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ac, ok := FromContext(r)
		if !ok || ac == nil {
			t.Fatal("expected AuthContext in context")
		}
		if !ac.HasPerm("t1", PermTaskView) {
			t.Fatal("expected task:view via context")
		}
	})
	req := newAuthedRequest("u1", "", "t1:task:view")
	w := httptest.NewRecorder()
	Middleware(inner).ServeHTTP(w, req)
}

func TestRequirePerm_Writes403(t *testing.T) {
	r := newAuthedRequest("u1", "", "t1:task:view")
	w := httptest.NewRecorder()
	if RequirePerm(w, r, PermTaskManage, "t1") {
		t.Fatal("should be denied")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestHasGroupResourceAccess(t *testing.T) {
	// PDP injects pseudo-codes group-res:project:p1:view
	r := newAuthedRequest("u1", "", "t1:task:view,group-res:project:p1:view")
	if !HasGroupResourceAccess(r, "t1", "project", "p1") {
		t.Fatal("expected group-resource access to project p1")
	}
	if HasGroupResourceAccess(r, "t1", "project", "p9") {
		t.Fatal("p9 not assigned")
	}
	if HasGroupResourceAccess(r, "t2", "project", "p1") {
		t.Fatal("t2 not assigned")
	}
}

func TestParse_TenantPermsFormat(t *testing.T) {
	// Verify the documented wire format: cid:perm1,perm2;cid2:perm1
	ac := ParseFromHeaders(newAuthedRequest("u1", "super_admin", "c1:a:manage,b:view;c2:x:view"))
	if len(ac.TenantPerms) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(ac.TenantPerms))
	}
	if !ac.TenantPerms["c1"][PermCode("a:manage")] || !ac.TenantPerms["c1"][PermCode("b:view")] {
		t.Fatal("c1 codes mismatch")
	}
	if !ac.TenantPerms["c2"][PermCode("x:view")] {
		t.Fatal("c2 codes mismatch")
	}
}

// TestIsPlatformStaff — v63 平台角色判定取代遗留 X-Auth-Superuser/X-Auth-Staff。
func TestIsPlatformStaff(t *testing.T) {
	if !IsPlatformStaff(newAuthedRequest("u1", "super_admin", "")) {
		t.Fatal("super_admin should be platform staff")
	}
	if !IsPlatformStaff(newAuthedRequest("u2", "employee", "")) {
		t.Fatal("employee should be platform staff")
	}
	if IsPlatformStaff(newAuthedRequest("u3", "", "")) {
		t.Fatal("no roles → not platform staff")
	}
	if IsPlatformStaff(newAuthedRequest("u4", "tenant_admin", "")) {
		t.Fatal("tenant role → not platform staff")
	}
}

func TestHasRegion_HasPage_FromHeaders(t *testing.T) {
	r := newAuthedRequest("u1", "", "t1:region:people.access.save_actions,page:people.access,task:view")
	if !HasRegion(r, "t1", "people.access.save_actions") {
		t.Fatal("expected region people.access.save_actions")
	}
	if HasRegion(r, "t1", "people.access.subject_list") {
		t.Fatal("subject_list region must not be granted")
	}
	if !HasPage(r, "t1", "people.access") {
		t.Fatal("expected page people.access")
	}
	if HasPage(r, "t1", "billing.overview") {
		t.Fatal("billing.overview page must not be granted")
	}
	if !HasPerm(r, "t1", PermTaskView) {
		t.Fatal("legacy coarse codes must still parse alongside region/page")
	}
}

func TestHasRegionView_Operate_EffectCodes(t *testing.T) {
	viewOnly := newAuthedRequest("u1", "", "t1:region:settings.cloud.main:view,page:settings.cloud:view")
	if !HasRegionView(viewOnly, "t1", "settings.cloud.main") {
		t.Fatal("view code should grant HasRegionView")
	}
	if !HasRegion(viewOnly, "t1", "settings.cloud.main") {
		t.Fatal("HasRegion should accept view")
	}
	if HasRegionOperate(viewOnly, "t1", "settings.cloud.main") {
		t.Fatal("view-only must not grant operate")
	}
	if !HasPage(viewOnly, "t1", "settings.cloud") {
		t.Fatal("page:view should satisfy HasPage")
	}

	operate := newAuthedRequest("u2", "", "t1:region:settings.cloud.main:view,region:settings.cloud.main:operate,region:settings.cloud.main")
	if !HasRegionOperate(operate, "t1", "settings.cloud.main") {
		t.Fatal("operate code should grant HasRegionOperate")
	}
}

func TestHasPeopleAccessWrite(t *testing.T) {
	put := func(perms string) *http.Request {
		r := httptest.NewRequest(http.MethodPut, "/api/auth/roles/", nil)
		r.Header.Set(HeaderUserID, "u1")
		r.Header.Set(HeaderTenantPerms, perms)
		return r
	}
	if !HasPeopleAccessWrite(put("t1:region:people.access.subject_list:operate"), "t1") {
		t.Fatal("subject_list operate should write")
	}
	if !HasPeopleAccessWrite(put("t1:region:people.access.region_matrix:operate"), "t1") {
		t.Fatal("region_matrix operate should write")
	}
	if !HasPeopleAccessWrite(put("t1:region:people.access.save_actions:operate"), "t1") {
		t.Fatal("legacy save_actions operate should write")
	}
	if HasPeopleAccessWrite(put("t1:region:people.access.subject_list:view"), "t1") {
		t.Fatal("view-only must not write")
	}
}

func TestEmitLogicalRGCodes(t *testing.T) {
	codes := EmitLogicalRGCodes(
		map[string]string{"a.main": EffectView, "b.main": EffectOperate},
		map[string]string{"a": EffectView},
	)
	has := func(c string) bool {
		for _, x := range codes {
			if x == c {
				return true
			}
		}
		return false
	}
	if !has("region:a.main:view") || has("region:a.main") || has("region:a.main:operate") {
		t.Fatalf("view-only region mismatch: %v", codes)
	}
	if !has("region:b.main:view") || !has("region:b.main:operate") || !has("region:b.main") {
		t.Fatalf("operate region mismatch: %v", codes)
	}
	if !has("page:a:view") || has("page:a") {
		t.Fatalf("view-only page mismatch: %v", codes)
	}
}

func TestRequireRegion_Writes403(t *testing.T) {
	r := newAuthedRequest("u1", "", "t1:region:people.access.subject_list")
	w := httptest.NewRecorder()
	if RequireRegion(w, r, "people.access.save_actions", "t1") {
		t.Fatal("should be denied")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	w2 := httptest.NewRecorder()
	if !RequireRegion(w2, r, "people.access.subject_list", "t1") {
		t.Fatal("should allow subject_list region")
	}
}

func TestRegionPagePermCodeHelpers(t *testing.T) {
	if RegionPermCode("people.access.save_actions") != PermCode("region:people.access.save_actions") {
		t.Fatal("RegionPermCode mismatch")
	}
	if PagePermCode("people.access") != PermCode("page:people.access") {
		t.Fatal("PagePermCode mismatch")
	}
}

// OPT-20260811-046 — auto-infer required_effect from the HTTP method.
func TestMethodRequiresOperate(t *testing.T) {
	readOnly := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, m := range readOnly {
		if MethodRequiresOperate(m) {
			t.Fatalf("method %s must not require operate", m)
		}
	}
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if !MethodRequiresOperate(m) {
			t.Fatalf("method %s must require operate", m)
		}
	}
}

func TestRequireRegionByMethod_ReadVsWrite(t *testing.T) {
	// view-only member holds region:<key>:view but not operate/legacy.
	viewOnly := newAuthedRequest("u1", "", "t1:region:settings.cloud.main:view,page:settings.cloud:view")

	// GET (read) with view-only member → allowed.
	w := httptest.NewRecorder()
	if !RequireRegionByMethod(w, viewOnly, "settings.cloud.main", "t1") {
		t.Fatalf("GET with view-only should pass, got %d", w.Code)
	}

	// PUT (mutating) with view-only member → denied 403.
	put := httptest.NewRequest(http.MethodPut, "/api/x", nil)
	put.Header.Set(HeaderUserID, "u1")
	put.Header.Set(HeaderTenantPerms, "t1:region:settings.cloud.main:view")
	w2 := httptest.NewRecorder()
	if RequireRegionByMethod(w2, put, "settings.cloud.main", "t1") {
		t.Fatal("PUT with view-only must be denied")
	}
	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w2.Code)
	}

	// Operate member holds legacy region:<key> → PUT passes.
	op := httptest.NewRequest(http.MethodPut, "/api/x", nil)
	op.Header.Set(HeaderUserID, "u2")
	op.Header.Set(HeaderTenantPerms, "t1:region:settings.cloud.main")
	w3 := httptest.NewRecorder()
	if !RequireRegionByMethod(w3, op, "settings.cloud.main", "t1") {
		t.Fatalf("PUT with operate member should pass, got %d", w3.Code)
	}
}

func TestHasRegionByMethod(t *testing.T) {
	viewOnly := newAuthedRequest("u1", "", "t1:region:settings.cloud.main:view")
	viewOnly.Method = http.MethodPut
	if HasRegionByMethod(viewOnly, "settings.cloud.main", "t1") {
		t.Fatal("PUT with view-only must not satisfy HasRegionByMethod")
	}
	viewOnly.Method = http.MethodGet
	if !HasRegionByMethod(viewOnly, "settings.cloud.main", "t1") {
		t.Fatal("GET with view-only should satisfy HasRegionByMethod")
	}

	operate := newAuthedRequest("u2", "", "t1:region:settings.cloud.main:operate")
	operate.Method = http.MethodPatch
	if !HasRegionByMethod(operate, "settings.cloud.main", "t1") {
		t.Fatal("PATCH with operate member should satisfy HasRegionByMethod")
	}
}
