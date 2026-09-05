#!/usr/bin/env python3
"""回归单测：check_go_routes_vs_apisix.py 中段通配符归一化（OPT-20260806-022）。

缺陷：`_strip_apisix_wildcard` 把中段通配符 `/*/` 折叠为 `/`，导致
`/api/system_admin/users/*/recharges`（网关已注册）无法匹配 Go 侧
`/api/system_admin/users/{uid}/recharges` 归一化路径 `/api/system_admin/users/*/recharges`，
check 脚本误报 MISSING（7 条中 3 条为假阳性）。
修复：保留中段 `*` 段，仅剥离尾部 `/*`。
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import check_go_routes_vs_apisix as m  # noqa: E402


def test_strip_apisix_wildcard_preserves_midpath_star():
    """中段通配符必须保留为字面 * 段（与 Go {param} → * 归一化对齐）。"""
    assert m._strip_apisix_wildcard("/api/system_admin/users/*/recharges") == "/api/system_admin/users/*/recharges"
    assert m._strip_apisix_wildcard("/api/system_admin/users/*/recharges/") == "/api/system_admin/users/*/recharges"
    assert m._strip_apisix_wildcard("/api/system_admin/users/*/recharges/*") == "/api/system_admin/users/*/recharges"


def test_strip_apisix_wildcard_trailing_wildcard():
    """尾部 /* 剥离（子树前缀语义），保持既有行为。"""
    assert m._strip_apisix_wildcard("/api/system-admin/license-agreement/*") == "/api/system-admin/license-agreement"
    assert m._strip_apisix_wildcard("/api/user/*") == "/api/user"


def test_path_covered_midpath_wildcard_roundtrip():
    """修复前：go /api/system_admin/users/*/recharges 对网关前缀（折叠版）返回 False；
    修复后：对保留 * 段的前缀返回 True。"""
    prefixes = {"/api/system_admin/users/*/recharges"}
    assert m._path_covered("/api/system_admin/users/*/recharges", prefixes) is True
    # 双形态：连字符与下划线分别覆盖
    prefixes2 = {"/api/system-admin/users/*/recharges", "/api/system_admin/users/*/recharges"}
    assert m._path_covered("/api/system-admin/users/*/recharges", prefixes2) is True
    assert m._path_covered("/api/system_admin/users/*/recharges", prefixes2) is True


def test_path_covered_prefix_walkup_still_works():
    """walk-up 前缀匹配不受影响（尾部 /* 语义）。"""
    prefixes = {"/api/system-admin/license-agreement"}
    assert m._path_covered("/api/system-admin/license-agreement/1", prefixes) is True


# --- OPT-20260824-072: parts-dispatch 路由提取 ---

TTS_LIKE_SNIPPET = '''
	mux.HandleFunc("/api/tenant/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/tenant/")
		if parts == nil || len(parts) < 1 {
			writeError(w, r, 404, "not found")
			return
		}
		tenantID := parts[0]
		// /api/tenant/<tid>/workspace/<wid>/todos[/<taskId>[/...]]
		if len(parts) >= 4 && parts[1] == "workspace" && parts[3] == "todos" {
			r.Header.Set("X-Workspace-Id", parts[2])
			handleTaskRoutes(w, r)
			return
		}
		// /api/tenant/<tid>/workspace/<wid>/queue-schedule
		if len(parts) >= 4 && parts[1] == "workspace" && parts[3] == "queue-schedule" {
			r.Header.Set("X-Workspace-Id", parts[2])
			handleWorkspaceQueueScheduleRoutes(w, r, tenantID, parts[2])
			return
		}
		// 租户级任务搜索
		if len(parts) >= 3 && parts[1] == "tasks" && parts[2] == "search" {
			handleSearchTasks(w, r, tenantID)
			return
		}
		writeError(w, r, 404, "not found")
	})

	mux.HandleFunc("/api/internal/tasks/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/internal/tasks/")
		if parts[0] == "batch-get" && r.Method == http.MethodPost {
			handleInternalTasksBatchGet(w, r)
			return
		}
	})
'''


def test_extract_dispatch_routes_synthesizes_tenant_workspace_routes():
    """/api/tenant/ 内 parts[N]=="literal" 分发须合成 */workspace/*/<literal> 路由。"""
    routes = m._extract_dispatch_routes(TTS_LIKE_SNIPPET)
    assert "/api/tenant/*/workspace/*/todos" in routes
    assert "/api/tenant/*/workspace/*/queue-schedule" in routes
    # 非 if 行（len 检查、空判）不合成
    assert "/api/tenant/*" not in routes
    assert "/api/tenant" not in routes


def test_extract_dispatch_routes_excludes_legacy_search():
    """legacy /api/tenant/*/tasks/search 被 DISPATCH_ROUTE_EXCLUSIONS 排除（约定路径走网关）。"""
    routes = m._extract_dispatch_routes(TTS_LIKE_SNIPPET)
    assert "/api/tenant/*/tasks/search" not in routes


def test_extract_dispatch_routes_internal_prefix():
    """/api/internal/ 分发合成后由 SKIPPABLE_PREFIXES 覆盖（is_skippable True）。"""
    routes = m._extract_dispatch_routes(TTS_LIKE_SNIPPET)
    assert "/api/internal/tasks/batch-get" in routes
    assert m.is_skippable("/api/internal/tasks/batch-get") is True


def test_synthesize_dispatch_path_wildcard_fill():
    """parts[1]=="workspace" && parts[3]=="todos" → /api/tenant/*/workspace/*/todos。"""
    assert m._synthesize_dispatch_path("/api/tenant/", {1: "workspace", 3: "todos"}) == (
        "/api/tenant/*/workspace/*/todos"
    )
    assert m._synthesize_dispatch_path("/api/tenant/", {1: "tasks", 2: "search"}) == (
        "/api/tenant/*/tasks/search"
    )


# --- OPT-20260824-073: /api/tenant_id 裸挂载点 ---


def test_tenant_id_bare_mount_skippable():
    """`mux.HandleFunc("/api/tenant_id/", ...)` 归一化后应落入 SKIPPABLE_EXACT。"""
    assert m.is_skippable("/api/tenant_id") is True


if __name__ == "__main__":
    import traceback

    failed = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
                print(f"PASS {name}")
            except AssertionError as e:
                failed += 1
                print(f"FAIL {name}: {e}")
                traceback.print_exc()
    sys.exit(1 if failed else 0)
