package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestExpandResourceGroupHits_PageExpandsRegions(t *testing.T) {
	hits := []struct{ Kind, Key, ParentPageKey, Effect string }{
		{Kind: "page", Key: "people.access", Effect: "operate"},
	}
	children := map[string][]string{
		"people.access": {"people.access.subject_list", "people.access.save_actions"},
	}
	codes := expandResourceGroupHits(hits, children)
	joined := strings.Join(codes, ",")
	if !strings.Contains(joined, "region:people.access.subject_list") ||
		!strings.Contains(joined, "region:people.access.save_actions") ||
		!strings.Contains(joined, "page:people.access") ||
		!strings.Contains(joined, "region:people.access.subject_list:view") ||
		!strings.Contains(joined, "region:people.access.subject_list:operate") {
		t.Fatalf("expected page+regions with effects, got %v", codes)
	}
	for _, c := range codes {
		if strings.HasPrefix(c, "project:") || strings.HasPrefix(c, "member:") {
			t.Fatalf("B2: must not expand coarse codes, got %s", c)
		}
	}
}

func TestExpandResourceGroupHits_ViewOnlyNoLegacyBare(t *testing.T) {
	hits := []struct{ Kind, Key, ParentPageKey, Effect string }{
		{Kind: "ui_region", Key: "billing.overview.main", ParentPageKey: "billing.overview", Effect: "view"},
	}
	codes := expandResourceGroupHits(hits, nil)
	joined := strings.Join(codes, ",")
	if !strings.Contains(joined, "region:billing.overview.main:view") ||
		!strings.Contains(joined, "page:billing.overview:view") {
		t.Fatalf("expected view codes, got %v", codes)
	}
	for _, c := range codes {
		if c == "region:billing.overview.main" || c == "page:billing.overview" {
			t.Fatalf("view-only must not emit legacy bare code, got %v", codes)
		}
		if strings.HasSuffix(c, ":operate") {
			t.Fatalf("view-only must not emit operate, got %v", codes)
		}
	}
}

func TestExpandResourceGroupHits_RegionImpliesPage(t *testing.T) {
	hits := []struct{ Kind, Key, ParentPageKey, Effect string }{
		{Kind: "ui_region", Key: "billing.overview.main", ParentPageKey: "billing.overview"},
	}
	codes := expandResourceGroupHits(hits, nil)
	joined := strings.Join(codes, ",")
	if !strings.Contains(joined, "region:billing.overview.main") || !strings.Contains(joined, "page:billing.overview") {
		t.Fatalf("expected region+page, got %v", codes)
	}
}

func TestComputePermSets_MergesResourceGroupCodes(t *testing.T) {
	oldAuthz, oldPerms, oldRG, oldRev := fetchAuthzStateFn, fetchRolePermsFn, fetchRoleResourceGroupCodesFn, membershipRevFn
	defer func() {
		fetchAuthzStateFn, fetchRolePermsFn, fetchRoleResourceGroupCodesFn, membershipRevFn = oldAuthz, oldPerms, oldRG, oldRev
	}()
	authzPermsCacheMu.Lock()
	authzPermsCache = map[string]authzPermsCacheEntry{}
	authzPermsCacheMu.Unlock()

	membershipRevFn = func(ctx context.Context, userID string) (int64, error) { return 1, nil }
	fetchAuthzStateFn = func(userID string) ([]authzStateCompany, error) {
		return []authzStateCompany{{CompanyID: "c1", IsActive: true, DirectRoles: []string{"custom_access"}}}, nil
	}
	fetchRolePermsFn = func(roleNames []string, companyID string) ([]string, error) {
		return []string{"member:view"}, nil
	}
	fetchRoleResourceGroupCodesFn = func(roleNames []string, companyID string) ([]string, error) {
		return []string{"region:people.access.save_actions", "page:people.access"}, nil
	}

	perms, err := computePermSets("u-rg")
	if err != nil {
		t.Fatal(err)
	}
	codes := strings.Join(perms["c1"], ",")
	if !strings.Contains(codes, "member:view") {
		t.Fatalf("legacy coarse should remain: %s", codes)
	}
	if !strings.Contains(codes, "region:people.access.save_actions") || !strings.Contains(codes, "page:people.access") {
		t.Fatalf("expected region/page injection: %s", codes)
	}
}

func TestSerializeTenantPerms(t *testing.T) {
	perms := map[string][]string{
		"t2": {"task:view"},
		"t1": {"company:view", "task:manage"},
	}
	out := serializeTenantPerms(perms)
	// 排序稳定: t1 在 t2 前
	if !strings.HasPrefix(out, "t1:company:view,task:manage;t2:task:view") {
		t.Fatalf("unexpected serialization: %s", out)
	}
	if serializeTenantPerms(nil) != "" {
		t.Fatal("nil → empty")
	}
	if serializeTenantPerms(map[string][]string{"t1": {}}) != "" {
		t.Fatal("empty codes → empty")
	}
}

func TestSerializeTenantPerms_Truncation(t *testing.T) {
	big := make([]string, 0, 1000)
	for i := 0; i < 1000; i++ {
		big = append(big, "project:manage")
	}
	out := serializeTenantPerms(map[string][]string{"t1": big})
	if len(out) > headerTenantPermsMaxBytes {
		t.Fatalf("serialized len %d exceeds cap %d", len(out), headerTenantPermsMaxBytes)
	}
}

func TestInPlaceholders(t *testing.T) {
	if got := inPlaceholders(3); got != "?,?,?" {
		t.Fatalf("expected ?,?,? got %s", got)
	}
}

func TestAuthzEmptyPermsCacheTTL(t *testing.T) {
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "")
	if got := authzEmptyPermsCacheTTL(); got != 2*time.Second {
		t.Fatalf("default empty TTL should be 2s, got %v", got)
	}
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "500")
	if got := authzEmptyPermsCacheTTL(); got != 500*time.Millisecond {
		t.Fatalf("env empty TTL should be 500ms, got %v", got)
	}
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "0")
	if got := authzEmptyPermsCacheTTL(); got != 0 {
		t.Fatalf("0 should disable empty caching, got %v", got)
	}
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "abc")
	if got := authzEmptyPermsCacheTTL(); got != 2*time.Second {
		t.Fatalf("invalid env should fall back to 2s, got %v", got)
	}
}

func TestPermsCacheEffectiveTTL(t *testing.T) {
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "500")
	// 空权限集 → 短 TTL（不再是标准 TTL，避免冻结陈旧空结果）
	if got := permsCacheEffectiveTTL(0, 30*time.Second); got != 500*time.Millisecond {
		t.Fatalf("empty result should use short TTL, got %v", got)
	}
	// 非空权限集 → 标准 TTL
	if got := permsCacheEffectiveTTL(1, 30*time.Second); got != 30*time.Second {
		t.Fatalf("non-empty result should use standard TTL, got %v", got)
	}
	// 标准缓存被显式关闭 → 一律不缓存
	if got := permsCacheEffectiveTTL(0, 0); got != 0 {
		t.Fatalf("standard disabled should disable all caching, got %v", got)
	}
	if got := permsCacheEffectiveTTL(3, 0); got != 0 {
		t.Fatalf("standard disabled should disable all caching (non-empty), got %v", got)
	}
}

func clearAuthzPermsCache() {
	authzPermsCacheMu.Lock()
	defer authzPermsCacheMu.Unlock()
	authzPermsCache = map[string]authzPermsCacheEntry{}
}

func cachedPermsExpiry(userID string) (time.Time, bool) {
	authzPermsCacheMu.Lock()
	defer authzPermsCacheMu.Unlock()
	ent, ok := authzPermsCache[userID]
	return ent.expiresAt, ok
}

// TestComputePermSetsEmptyResultShortTTL 复现生产事故（2026-08-06）：
// 新用户首次 forward-auth 在事件驱动建公司（USER_CREATED 链路）落库前
// 计算得到空权限集；若空结果按标准 TTL（30s）缓存，onboarding 改名 PATCH
// 会命中陈旧空权限 → 403「仅公司管理员可修改公司名称」。
// 修复后空结果仅短 TTL 缓存，过期后重算应立即看到新公司的新权限。
func TestComputePermSetsEmptyResultShortTTL(t *testing.T) {
	t.Setenv("TASKAUTH_AUTHZ_PERMS_CACHE_TTL_MS", "30000")
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "80")
	clearAuthzPermsCache()

	oldAuthz, oldPerms := fetchAuthzStateFn, fetchRolePermsFn
	defer func() { fetchAuthzStateFn, fetchRolePermsFn = oldAuthz, oldPerms }()
	fetchRolePermsFn = func(roleNames []string, companyID string) ([]string, error) {
		return []string{"member:manage", "member:view"}, nil
	}

	// 阶段 1：事件驱动建公司之前的窗口 — 首次 forward-auth 计算得到空权限集
	fetchAuthzStateFn = func(userID string) ([]authzStateCompany, error) {
		return nil, nil
	}
	if perms, err := computePermSets("u1"); err != nil || len(perms) != 0 {
		t.Fatalf("initial empty perms expected, got %v err=%v", perms, err)
	}
	// 空结果缓存 TTL 必须 ≤ 短 TTL（80ms），而非标准 30s
	expiry, ok := cachedPermsExpiry("u1")
	if !ok {
		t.Fatal("empty result should still be cached (short TTL)")
	}
	if d := time.Until(expiry); d < -1*time.Millisecond || d > 200*time.Millisecond {
		t.Fatalf("empty result cached with short TTL expected, got expiry in %v", d)
	}

	// 阶段 2：事件驱动建公司落库（公司+成员+tenant_admin 角色行）
	// 空结果短 TTL 过期后重算 → 应立即看到新公司的新权限，不再 403
	time.Sleep(150 * time.Millisecond)
	fetchAuthzStateFn = func(userID string) ([]authzStateCompany, error) {
		return []authzStateCompany{{CompanyID: "c1", IsActive: true, DirectRoles: []string{"tenant_admin"}}}, nil
	}
	perms, err := computePermSets("u1")
	if err != nil {
		t.Fatalf("recompute failed: %v", err)
	}
	has := false
	for _, code := range perms["c1"] {
		if code == "member:manage" {
			has = true
			break
		}
	}
	if !has {
		t.Fatalf("fresh perms after short-TTL expiry should include member:manage, got %v", perms)
	}
	// 非空结果应回到标准 TTL（30s）
	expiry, ok = cachedPermsExpiry("u1")
	if !ok {
		t.Fatal("non-empty result should be cached")
	}
	if d := time.Until(expiry); d < 25*time.Second {
		t.Fatalf("non-empty result should use standard TTL, got expiry in %v", d)
	}
}

// TestComputePermSetsRevInvalidation 验证 rev 事件失效（OPT-20260806-010）：
// 标准 TTL 期间（30s）成员/角色变化经 incrMembershipRev 递增 rev，命中缓存
// 时比较 rev 快照发现不一致 → 立即重算，而非等满 TTL。覆盖「已有公司的用户
// 新建公司后，新公司权限延迟 30s」核心场景。
func TestComputePermSetsRevInvalidation(t *testing.T) {
	t.Setenv("TASKAUTH_AUTHZ_PERMS_CACHE_TTL_MS", "30000")
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "2000")
	clearAuthzPermsCache()

	oldAuthz, oldPerms, oldRev := fetchAuthzStateFn, fetchRolePermsFn, membershipRevFn
	defer func() { fetchAuthzStateFn, fetchRolePermsFn, membershipRevFn = oldAuthz, oldPerms, oldRev }()
	fetchRolePermsFn = func(roleNames []string, companyID string) ([]string, error) {
		return []string{"member:manage", "member:view"}, nil
	}

	// 模拟 redis rev 计数（成员/角色变化由 taskTenantService incr 递增）
	rev := int64(1)
	membershipRevFn = func(_ context.Context, userID string) (int64, error) { return rev, nil }

	authzCalls := 0
	// 阶段 1: 用户已有公司 c1（tenant_admin）→ 计算并缓存（rev=1）
	fetchAuthzStateFn = func(userID string) ([]authzStateCompany, error) {
		authzCalls++
		return []authzStateCompany{{CompanyID: "c1", IsActive: true, DirectRoles: []string{"tenant_admin"}}}, nil
	}
	if _, err := computePermSets("u-rev"); err != nil {
		t.Fatalf("first compute failed: %v", err)
	}
	// 标准 TTL 内命中 → 不重算
	if _, err := computePermSets("u-rev"); err != nil {
		t.Fatalf("cached compute failed: %v", err)
	}
	if authzCalls != 1 {
		t.Fatalf("expected cache hit (1 authz call), got %d", authzCalls)
	}

	// 阶段 2: 用户新建公司 → taskTenant incr rev（1→2）→ 即使 TTL 未满也必须失效重算
	rev = 2
	fetchAuthzStateFn = func(userID string) ([]authzStateCompany, error) {
		authzCalls++
		return []authzStateCompany{
			{CompanyID: "c1", IsActive: true, DirectRoles: []string{"tenant_admin"}},
			{CompanyID: "c2", IsActive: true, DirectRoles: []string{"tenant_admin"}},
		}, nil
	}
	perms, err := computePermSets("u-rev")
	if err != nil {
		t.Fatalf("recompute after rev bump failed: %v", err)
	}
	if authzCalls != 2 {
		t.Fatalf("expected recompute after rev bump (2 authz calls), got %d", authzCalls)
	}
	if _, ok := perms["c2"]; !ok {
		t.Fatalf("new company c2 perms should be visible immediately after rev bump, got %v", perms)
	}

	// 阶段 3: rev 未再变化 → 命中新缓存
	if _, err := computePermSets("u-rev"); err != nil {
		t.Fatalf("cached compute after rev settle failed: %v", err)
	}
	if authzCalls != 2 {
		t.Fatalf("expected cache hit after rev settle (still 2 authz calls), got %d", authzCalls)
	}
}

// TestComputePermSetsRevUnknownDegradesToTTL rev 读取失败（redis 不可达）时
// 优雅降级：不因 revErr 强制重算，缓存仍按 TTL 生效。
func TestComputePermSetsRevUnknownDegradesToTTL(t *testing.T) {
	t.Setenv("TASKAUTH_AUTHZ_PERMS_CACHE_TTL_MS", "30000")
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "2000")
	clearAuthzPermsCache()

	oldAuthz, oldPerms, oldRev := fetchAuthzStateFn, fetchRolePermsFn, membershipRevFn
	defer func() { fetchAuthzStateFn, fetchRolePermsFn, membershipRevFn = oldAuthz, oldPerms, oldRev }()
	fetchRolePermsFn = func(roleNames []string, companyID string) ([]string, error) {
		return []string{"member:view"}, nil
	}
	membershipRevFn = func(_ context.Context, userID string) (int64, error) {
		return 0, errors.New("redis not configured")
	}
	authzCalls := 0
	fetchAuthzStateFn = func(userID string) ([]authzStateCompany, error) {
		authzCalls++
		return []authzStateCompany{{CompanyID: "c1", IsActive: true, DirectRoles: []string{"tenant_admin"}}}, nil
	}
	if _, err := computePermSets("u-ttl"); err != nil {
		t.Fatalf("compute failed: %v", err)
	}
	if _, err := computePermSets("u-ttl"); err != nil {
		t.Fatalf("cached compute failed: %v", err)
	}
	if authzCalls != 1 {
		t.Fatalf("rev read failure should fall back to TTL-only caching, got %d authz calls", authzCalls)
	}
}

// TestComputePermSetsStandardTTLDisabled 标准缓存关闭（TTL=0）时完全禁用缓存。
func TestComputePermSetsStandardTTLDisabled(t *testing.T) {
	t.Setenv("TASKAUTH_AUTHZ_PERMS_CACHE_TTL_MS", "0")
	t.Setenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS", "100")
	clearAuthzPermsCache()

	oldAuthz, oldPerms := fetchAuthzStateFn, fetchRolePermsFn
	defer func() { fetchAuthzStateFn, fetchRolePermsFn = oldAuthz, oldPerms }()
	fetchAuthzStateFn = func(userID string) ([]authzStateCompany, error) { return nil, nil }
	fetchRolePermsFn = func(roleNames []string, companyID string) ([]string, error) { return nil, nil }

	if _, err := computePermSets("u2"); err != nil {
		t.Fatalf("computePermSets failed: %v", err)
	}
	if _, ok := cachedPermsExpiry("u2"); ok {
		t.Fatal("caching should be fully disabled when standard TTL is 0")
	}
}
