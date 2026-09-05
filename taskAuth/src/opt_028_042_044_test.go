package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// ── OPT-20260806-028: wechatLoginPolicyEnabled 30s TTL 缓存 ──────────────
// 策略行不变时连续请求只产生 1 次 DB 查询；DB 报错 fail-closed 且清缓存；
// 管理员保存后缓存立即失效（无需等 TTL）。

func TestWechatPolicyCacheDedupesDBReads(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)

	// 查询计数接缝：连续两次读策略，第二次应命中缓存不再查库
	var queries atomic.Int64
	old := featurePolicyQueryHook
	featurePolicyQueryHook = func() { queries.Add(1) }
	t.Cleanup(func() { featurePolicyQueryHook = old })

	if !wechatLoginPolicyEnabled() {
		t.Fatal("first read should see enabled=true")
	}
	if !wechatLoginPolicyEnabled() {
		t.Fatal("cached read should still see enabled=true")
	}
	if n := queries.Load(); n != 1 {
		t.Fatalf("wechatLoginPolicyEnabled twice: %d DB queries, want 1 (TTL cache miss)", n)
	}
}

func TestWechatPolicyCacheInvalidatedOnSave(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)
	if !wechatLoginPolicyEnabled() {
		t.Fatal("enabled=true expected")
	}
	// 管理员保存关闭 → 缓存立即失效（无需等 30s TTL）
	setWechatPolicyEnabled(t, false)
	if wechatLoginPolicyEnabled() {
		t.Fatal("after save(false) policy should be false (cache invalidated on save)")
	}
}

func TestWechatPolicyFailClosedDB(t *testing.T) {
	setupAuthTestDB(t)
	setWechatPolicyEnabled(t, true)
	if !wechatLoginPolicyEnabled() {
		t.Fatal("enabled=true expected")
	}
	// 关闭底层连接模拟 DB 故障（先失效缓存，确保走 DB 路径）
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	invalidateWechatPolicyCache()
	if wechatLoginPolicyEnabled() {
		t.Fatal("DB error must be fail-closed (enabled=false)")
	}
	// fail-closed 已清缓存：再次调用仍返回 false（重新查询仍失败）
	if wechatLoginPolicyEnabled() {
		t.Fatal("fail-closed must keep rejecting on subsequent calls")
	}
}

// ── OPT-20260806-042: tenant members 瞬态失败重试 ────────────────────────
// 首次 503 → 重试成功 → 返回成员（回调落点不再误判 onboarding）。
// 两次均失败 → 返回空（落点兜底）。

func TestFetchCompanyNicknamesRetriesTransientFailure(t *testing.T) {
	oldDelay := tenantMembersRetryDelay
	tenantMembersRetryDelay = 10 * time.Millisecond
	t.Cleanup(func() { tenantMembersRetryDelay = oldDelay })

	var first atomic.Bool
	first.Store(true)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		if first.CompareAndSwap(true, false) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `[{"company_id":"c-1","company_name":"Acme","is_admin":true}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prev := cfg
	cfg = Config{TenantServiceURL: srv.URL, UserContentTypeID: prev.UserContentTypeID}
	defer func() { cfg = prev }()
	clearTenantMembersCache()

	result := fetchCompanyNicknames("u-retry")
	if len(result) != 1 {
		t.Fatalf("retry should succeed after transient 503, got %v", result)
	}
}

func TestFetchCompanyNicknamesFailsClosedAfterBothAttempts(t *testing.T) {
	oldDelay := tenantMembersRetryDelay
	tenantMembersRetryDelay = 10 * time.Millisecond
	t.Cleanup(func() { tenantMembersRetryDelay = oldDelay })

	var calls atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prev := cfg
	cfg = Config{TenantServiceURL: srv.URL, UserContentTypeID: prev.UserContentTypeID}
	defer func() { cfg = prev }()
	clearTenantMembersCache()

	result := fetchCompanyNicknames("u-fail")
	if len(result) != 0 {
		t.Fatalf("both attempts fail: want empty, got %v", result)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("expected exactly 2 attempts (initial + retry), got %d", n)
	}
}

// ── OPT-20260806-044: members 结果短 TTL 缓存 ────────────────────────────
// 同一用户 5s 内连续两次拉取：第二次命中缓存不产生外部请求，且落点一致。

func TestFetchCompanyNicknamesTTLCache(t *testing.T) {
	var requests atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `[{"company_id":"c-1","company_name":"Acme","is_admin":true}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prev := cfg
	cfg = Config{TenantServiceURL: srv.URL, UserContentTypeID: prev.UserContentTypeID}
	defer func() { cfg = prev }()
	clearTenantMembersCache()

	first := fetchCompanyNicknames("u-cache")
	second := fetchCompanyNicknames("u-cache")
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("both calls should return members, got %v / %v", first, second)
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("two calls within TTL: %d external requests, want 1 (second cached)", n)
	}
}

func TestFetchCompanyNicknamesCacheExpires(t *testing.T) {
	var requests atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `[{"company_id":"c-1","company_name":"Acme","is_admin":true}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	prev := cfg
	cfg = Config{TenantServiceURL: srv.URL, UserContentTypeID: prev.UserContentTypeID}
	defer func() { cfg = prev }()
	clearTenantMembersCache()

	fetchCompanyNicknames("u-expire")
	// 手动把缓存时间拨老，模拟 TTL 过期
	key := srv.URL + "|u-expire"
	tenantMembersCacheMu.Lock()
	if ent, ok := tenantMembersCache[key]; ok {
		tenantMembersCache[key] = tenantMembersCacheEntry{result: ent.result, fetchedAt: time.Now().Add(-tenantMembersCacheTTL - time.Second)}
	}
	tenantMembersCacheMu.Unlock()

	fetchCompanyNicknames("u-expire")
	if n := requests.Load(); n != 2 {
		t.Fatalf("after TTL expiry: %d requests, want 2 (refetch)", n)
	}
}

// ── helpers ─────────────────────────────────────────────────────────────

func clearTenantMembersCache() {
	tenantMembersCacheMu.Lock()
	tenantMembersCache = map[string]tenantMembersCacheEntry{}
	tenantMembersCacheMu.Unlock()
}
