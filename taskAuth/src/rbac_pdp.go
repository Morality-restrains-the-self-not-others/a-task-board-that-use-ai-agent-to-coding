package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// taskAuth PDP — 权限码集合计算 (v63 design §4)
//
// 计算链: forward-auth → taskTenant /api/internal/tenant/authz-state
//   → auth_role_permission 展开 → 组资源伪码 → X-Tenant-Perms 注入
// 缓存: 进程内存短 TTL + 事件失效 rev 快照（taskTenant 成员/角色/组/资源授权
//   变化时 incrMembershipRev 递增 redis 计数，本进程命中缓存时比较 rev，
//   不一致即失效重算 — 非空权限集不再需要等满标准 TTL）。
//   空权限集（用户尚无任何公司成员）用短 TTL（默认 2s）而非标准 TTL：
//   事件驱动的公司创建（USER_CREATED → 异步建 company+member+role 行）可能
//   晚于新用户首次 forward-auth 数百毫秒落库，若把空结果冻结 30s，
//   onboarding 改名等紧随其后的请求会命中陈旧空权限 → 误报 403。
//   注意：rev 存储在 redis 的 membership_rev:<user_id>（与 taskTenantService
//   TENANT_REDIS_HOST/TENANT_REDIS_DB 同实例同库，两服务 env 需对齐）；
//   redis 不可达时优雅降级为纯 TTL 判定（rev 检查跳过）。
// ═══════════════════════════════════════════════════════════════

const (
	// headerTenantPermsMaxBytes 注入上限保护（8KB HTTP header 限制内留余量）
	headerTenantPermsMaxBytes = 7000
	// memberDefaultRole / tenantAdminRole 内置角色名
	memberDefaultRole = "member"
	tenantAdminRole   = "tenant_admin"
	groupAdminRole    = "group_admin"
)

// authzStateCompany 对应 taskTenant authz-state 响应
type authzStateCompany struct {
	CompanyID     string `json:"company_id"`
	MemberID      string `json:"member_id"`
	IsActive      bool   `json:"is_active"`
	IsAdmin       bool   `json:"is_admin"`
	DirectRoles   []string `json:"direct_roles"`
	GroupRoles    []struct {
		GroupID string `json:"group_id"`
		Role    string `json:"role"`
	} `json:"group_roles"`
	GroupAdmins []struct {
		GroupID string `json:"group_id"`
	} `json:"group_admins"`
	GroupResource []struct {
		GroupID      string `json:"group_id"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		Permission   string `json:"permission"`
	} `json:"group_resources"`
}

type authzPermsCacheEntry struct {
	perms     map[string][]string // companyID → sorted perm codes
	expiresAt time.Time
	rev       int64 // membership rev 快照（taskTenant 变更递增）；-1 = 未知（redis 不可用）
}

var (
	authzPermsCacheMu sync.Mutex
	authzPermsCache   = map[string]authzPermsCacheEntry{}
)

func authzPermsCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TASKAUTH_AUTHZ_PERMS_CACHE_TTL_MS"))
	if raw == "" {
		return 30 * time.Second
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return 30 * time.Second
	}
	if ms == 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// authzEmptyPermsCacheTTL 空权限集（用户尚无公司成员）的缓存 TTL。
// 新用户注册 → 事件驱动建公司（USER_CREATED 链路）可能晚于首次 forward-auth
// 数百毫秒才落库；空结果若按标准 TTL（默认 30s）缓存，紧随其后的操作会命中
// 冻结的空权限集 → 权限误判（典型：onboarding 改名报「仅公司管理员可修改公司名称」）。
// 空结果短 TTL 保证新公司/角色变化最多延迟该 TTL 即可见。
// 0 表示完全不缓存空结果。
func authzEmptyPermsCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TASKAUTH_AUTHZ_EMPTY_PERMS_CACHE_TTL_MS"))
	if raw == "" {
		return 2 * time.Second
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return 2 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

// permsCacheEffectiveTTL 根据权限集是否为空选择缓存 TTL：
// 空（公司数 0）→ 短 TTL（authzEmptyPermsCacheTTL）；非空 → 标准 TTL。
// standardTTL <= 0（标准缓存被显式关闭）→ 不缓存，尊重运维意图。
func permsCacheEffectiveTTL(companyCount int, standardTTL time.Duration) time.Duration {
	if standardTTL <= 0 {
		return 0
	}
	if companyCount == 0 {
		return authzEmptyPermsCacheTTL()
	}
	return standardTTL
}

// fetchAuthzStateFn / fetchRolePermsFn 是 computePermSets 的外部依赖接缝，
// 单元测试中替换以隔离 HTTP/DB（生产路径恒为原函数）。
var (
	fetchAuthzStateFn = fetchAuthzState
	fetchRolePermsFn  = fetchRolePerms
	// membershipRevFn 是 rev 读取接缝（生产路径为 getMembershipRev — redis 读取；
	// 测试中替换以模拟成员/角色变更的 rev 递增）。
	membershipRevFn = getMembershipRev
)

// fetchAuthzState 调 taskTenantService 拉取用户全部公司的授权状态。
func fetchAuthzState(userID string) ([]authzStateCompany, error) {
	base := strings.TrimRight(cfg.TenantServiceURL, "/")
	if base == "" {
		return nil, errors.New("tenant service not configured")
	}
	url := base + "/api/internal/tenant/authz-state?user_id=" + userID
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenant service returned %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out []authzStateCompany
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// fetchRolePerms 按角色名批量查权限码。
// companyID 限定自定义角色归属（内置角色 company_id IS NULL 恒匹配）。
func fetchRolePerms(roleNames []string, companyID string) ([]string, error) {
	if len(roleNames) == 0 {
		return nil, nil
	}
	unique := make([]string, 0, len(roleNames))
	seen := map[string]bool{}
	for _, n := range roleNames {
		if n != "" && !seen[n] {
			seen[n] = true
			unique = append(unique, n)
		}
	}
	q := `SELECT DISTINCT p.codename FROM auth_permission p
		JOIN auth_role_permission rp ON rp.permission_id = p.id
		JOIN auth_role r ON r.id = rp.role_id
		WHERE r.name IN (` + inPlaceholders(len(unique)) + `)
		AND (r.company_id IS NULL OR r.company_id = ?)`
	args := make([]any, 0, len(unique)+1)
	for _, n := range unique {
		args = append(args, n)
	}
	args = append(args, companyID)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err == nil {
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

// computePermSets 计算用户全部公司的权限码集合（缓存优先）。
func computePermSets(userID string) (map[string][]string, error) {
	ttl := authzPermsCacheTTL()
	if ttl > 0 {
		now := time.Now()
		// 事件失效: redis 可用时比较当前 rev 与缓存快照，变化即失效重算；
		// redis 不可达（revErr != nil）→ 跳过 rev 判定，仅依赖 TTL（优雅降级，
		// 与 membership JWT 路径的 rev=0 兜底一致）。
		curRev, revErr := membershipRevFn(context.Background(), userID)
		authzPermsCacheMu.Lock()
		if ent, ok := authzPermsCache[userID]; ok && now.Before(ent.expiresAt) {
			if revErr != nil || ent.rev == curRev {
				authzPermsCacheMu.Unlock()
				return ent.perms, nil
			}
			log.Printf("[taskAuth] event=rbac_perm_sets_cache_rev_invalidate user_id=%s cached_rev=%d current_rev=%d",
				userID, ent.rev, curRev)
			delete(authzPermsCache, userID)
		}
		authzPermsCacheMu.Unlock()
	}

	entries, err := fetchAuthzStateFn(userID)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for _, e := range entries {
		if !e.IsActive {
			continue
		}
		roleNames := append([]string{}, e.DirectRoles...)
		for _, gr := range e.GroupRoles {
			roleNames = append(roleNames, gr.Role)
		}
		// group_admin 指派 → 隐含 group_admin 角色
		if len(e.GroupAdmins) > 0 {
			roleNames = append(roleNames, groupAdminRole)
		}
		// v63 硬切换完成: is_admin 列不再参与判定（迁移 004 回填 + 成员创建同步角色行），
		// tenant_admin 完全由 DirectRoles 提供。

		perms, err := fetchRolePermsFn(roleNames, e.CompanyID)
		if err != nil {
			log.Printf("[taskAuth] event=rbac_fetch_role_perms status=error user_id=%s cid=%s err=%v", userID, e.CompanyID, err)
			continue
		}
		set := make(map[string]bool, len(perms)+4)
		for _, p := range perms {
			set[p] = true
		}

		// v72: 逻辑资源组 → region:*/page:*（B2：不展开旧粗码）
		if rgCodes, err := fetchRoleResourceGroupCodesFn(roleNames, e.CompanyID); err != nil {
			log.Printf("[taskAuth] event=rbac_fetch_resource_groups status=error user_id=%s cid=%s err=%v", userID, e.CompanyID, err)
		} else {
			for _, c := range rgCodes {
				set[c] = true
			}
		}

		// 组资源分配 → 伪码 group-res:{type}:{id}:{view|manage}
		for _, res := range e.GroupResource {
			switch res.Permission {
			case "manage":
				set["group-res:"+res.ResourceType+":"+res.ResourceID+":manage"] = true
				set["group-res:"+res.ResourceType+":"+res.ResourceID+":view"] = true
			default: // view
				set["group-res:"+res.ResourceType+":"+res.ResourceID+":view"] = true
			}
		}

		// 无任何角色 → 默认 member 基线（存量成员过渡）
		if len(roleNames) == 0 {
			if mperms, err := fetchRolePermsFn([]string{memberDefaultRole}, e.CompanyID); err == nil {
				for _, p := range mperms {
					set[p] = true
				}
			}
		}

		sorted := make([]string, 0, len(set))
		for p := range set {
			sorted = append(sorted, p)
		}
		sort.Strings(sorted)
		result[e.CompanyID] = sorted
	}

	// 空权限集（无任何公司成员）用短 TTL 缓存，避免事件驱动建公司窗口期
	// 冻结陈旧空结果（见 authzEmptyPermsCacheTTL 注释）；TTL 0 则不缓存。
	cacheTTL := permsCacheEffectiveTTL(len(result), ttl)
	if cacheTTL > 0 {
		// rev 快照: 成员/角色变更后命中比较失效；redis 不可用 → -1 占位
		// （下一次命中若 redis 恢复且 rev 非 -1 即触发一次重算，随后恢复快照）
		rev := int64(-1)
		if r, err := membershipRevFn(context.Background(), userID); err == nil {
			rev = r
		}
		authzPermsCacheMu.Lock()
		authzPermsCache[userID] = authzPermsCacheEntry{perms: result, expiresAt: time.Now().Add(cacheTTL), rev: rev}
		if len(authzPermsCache) > 4096 {
			now := time.Now()
			for k, v := range authzPermsCache {
				if now.After(v.expiresAt) {
					delete(authzPermsCache, k)
				}
			}
		}
		authzPermsCacheMu.Unlock()
		log.Printf("[taskAuth] event=rbac_perm_sets_cache_store user_id=%s companies=%d ttl=%s", userID, len(result), cacheTTL)
	}
	return result, nil
}

// serializeTenantPerms 序列化 X-Tenant-Perms: cid:code1,code2;cid2:code1
// 超上限截断（后续请求走 PDP HTTP fallback 兜底）。
func serializeTenantPerms(perms map[string][]string) string {
	parts := make([]string, 0, len(perms))
	for cid, codes := range perms {
		if len(codes) == 0 {
			continue
		}
		parts = append(parts, cid+":"+strings.Join(codes, ","))
	}
	sort.Strings(parts)
	out := strings.Join(parts, ";")
	if len(out) > headerTenantPermsMaxBytes {
		return out[:headerTenantPermsMaxBytes]
	}
	return out
}

// platformRolesFromFlags 过渡期由 is_superuser/is_staff 推导平台角色。
func platformRolesFromFlags(isSuperuser, isStaff bool) []string {
	var roles []string
	if isSuperuser {
		roles = append(roles, "super_admin")
	} else if isStaff {
		roles = append(roles, "employee")
	}
	return roles
}

// handlePDPCheck — POST /api/internal/authz/check（HTTP fallback 兜底）
func handlePDPCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var req struct {
		UserID       string `json:"user_id"`
		CompanyID    string `json:"company_id"`
		PermCode     string `json:"perm_code"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || req.PermCode == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id and perm_code required")
		return
	}
	perms, err := computePermSets(req.UserID)
	if err != nil {
		log.Printf("[taskAuth] event=pdp_check status=error user_id=%s err=%v", req.UserID, err)
		writeErrorDetail(w, r, http.StatusBadGateway, "授权状态不可用")
		return
	}
	allowed := false
	if req.CompanyID != "" {
		for _, code := range perms[req.CompanyID] {
			if code == req.PermCode {
				allowed = true
				break
			}
		}
	}
	// 组资源伪码兜底
	if !allowed && req.ResourceType != "" && req.ResourceID != "" {
		pseudo := "group-res:" + req.ResourceType + ":" + req.ResourceID + ":view"
		for _, code := range perms[req.CompanyID] {
			if code == pseudo {
				allowed = true
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"allowed": allowed,
		"reason":  map[bool]string{true: "ok", false: "permission denied"}[allowed],
	})
}

func inPlaceholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}
