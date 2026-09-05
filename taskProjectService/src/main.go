package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"daydaymoneymeta"
	"gatewayauth"
	"tracelog"
)

func cleanPath(r *http.Request, prefix string) []string {
	p := strings.TrimPrefix(r.URL.Path, prefix)
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return nil
	}
	return strings.SplitN(p, "/", 8)
}

func cleanPathStr(p string) []string {
	p = strings.TrimSuffix(strings.TrimSpace(p), "/")
	if p == "" {
		return nil
	}
	return strings.SplitN(p, "/", 8)
}

func main() {
	const svc = "task-project-service"
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		tracelog.Fatalf(svc, "[taskProjectService] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	initLogging(svc)
	tracelog.Init(svc)
	if meta, err := daydaymoneymeta.LoadNearDir(filepath.Join(repoRoot, "taskProjectService")); err == nil {
		tracelog.SetDaydaymoneyMeta(meta.ServiceID, meta.Tags)
	} else if meta, err := daydaymoneymeta.LoadNearDir("."); err == nil {
		tracelog.SetDaydaymoneyMeta(meta.ServiceID, meta.Tags)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := openDB(cfg.DBPath); err != nil {
			tracelog.Fatalf(svc, "[taskProjectService] migration failed: %v", err)
		}
		if err := runDataMigrate(repoRoot); err != nil {
			tracelog.Fatalf(svc, "[taskProjectService] dataMigrate: %v", err)
		}
		log.Println("[taskProjectService] migration complete")
		return
	}

	if err := openDB(cfg.DBPath); err != nil {
		tracelog.Fatalf(svc, "[taskProjectService] db: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logInfo("listening on "+addr, "")
	log.Printf("[taskProjectService] listening on %s (db=%s)", addr, cfg.DBPath)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(corsMiddleware(gatewayUserMiddleware(tracelog.MetricsMiddleware(mux))))); err != nil {
		tracelog.Fatalf(svc, "[taskProjectService] listen: %v", err)
	}
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", handleHealth)
	mux.Handle("GET /api/metrics", tracelog.MetricsHandler())
	mux.HandleFunc("/api/schema/", handleOpenAPISchema)
	mux.HandleFunc("/api/schema", handleOpenAPISchema)
	mux.HandleFunc("/api/swagger/", handleSwaggerUI)
	mux.HandleFunc("/api/swagger", handleSwaggerUI)

	mux.HandleFunc("/api/switch-workspace/", handleLegacySwitchWorkspace)
	mux.HandleFunc("/api/switch-workspace", handleLegacySwitchWorkspace)

	mux.HandleFunc("/api/internal/nested-git-repos", handleInternalNestedGitRepos)
	mux.HandleFunc("/api/internal/nested-git-repos/", handleInternalNestedGitRepos)
	mux.HandleFunc("/api/internal/tenants/", handleInternalTenantGitlabLocalDiskUsage)
	mux.HandleFunc("/api/internal/taskproject/tenant-gitlab-cache/invalidate", handleInternalTenantGitLabCacheInvalidate)
	mux.HandleFunc("/api/internal/taskproject/tenant-gitlab-cache/invalidate/", handleInternalTenantGitLabCacheInvalidate)

	mux.HandleFunc("GET /api/internal/projects/git-oauth-grant/", handleInternalLookupProjectGitOAuthGrant)
	mux.HandleFunc("GET /api/internal/projects/git-oauth-grant", handleInternalLookupProjectGitOAuthGrant)
	mux.HandleFunc("POST /api/internal/projects/git-oauth-grant/", handleInternalUpsertProjectGitOAuthGrant)
	mux.HandleFunc("POST /api/internal/projects/git-oauth-grant", handleInternalUpsertProjectGitOAuthGrant)
	mux.HandleFunc("/api/internal/projects/git-oauth-grant/", handleInternalMarkProjectGitOAuthGrant)
	mux.HandleFunc("/api/internal/projects/git-oauth-grant", handleInternalMarkProjectGitOAuthGrant)

	mux.HandleFunc("/api/internal/projects/batch-get/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleInternalProjectsBatchGet(w, r)
			return
		}
		writeError(w, r, 405, "method not allowed")
	})
	mux.HandleFunc("/api/internal/projects/batch-get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleInternalProjectsBatchGet(w, r)
			return
		}
		writeError(w, r, 405, "method not allowed")
	})

	mux.HandleFunc("/api/internal/workspaces/batch-get/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleInternalWorkspacesBatchGet(w, r)
			return
		}
		writeError(w, r, 405, "method not allowed")
	})
	mux.HandleFunc("/api/internal/workspaces/batch-get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleInternalWorkspacesBatchGet(w, r)
			return
		}
		writeError(w, r, 405, "method not allowed")
	})

	mux.HandleFunc("/api/internal/workspaces/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/internal/workspaces/")
		if len(parts) == 0 {
			writeError(w, r, 404, "not found")
			return
		}
		handleInternalGetWorkspace(w, r, parts[0])
	})

	mux.HandleFunc("/api/internal/reload-config", handleInternalReloadConfig)
	mux.HandleFunc("/api/internal/reload-config/", handleInternalReloadConfig)

	mux.HandleFunc("/api/internal/progress-columns/validate", handleInternalValidateProgressColumn)
	mux.HandleFunc("/api/internal/progress-columns/validate/", handleInternalValidateProgressColumn)
	mux.HandleFunc("/api/internal/progress-columns/first", handleInternalResolveFirstProgressColumn)
	mux.HandleFunc("/api/internal/progress-columns/first/", handleInternalResolveFirstProgressColumn)

	// Internal deliverable system lookup (used by taskTaskService for deliverable_obj enrichment)
	mux.HandleFunc("/api/internal/deliverable-systems/lookup", handleInternalDeliverableSystemLookup)
	mux.HandleFunc("/api/internal/deliverable-systems/lookup/", handleInternalDeliverableSystemLookup)

	// System-admin progress systems
	mux.HandleFunc("/api/system-admin/progress-systems/", handleSystemProgressSystemRoute)
	mux.HandleFunc("/api/system-admin/progress-systems", handleSystemProgressSystemRoute)
	// Legacy system alias
	mux.HandleFunc("/api/system/progress-systems/", handleSystemProgressSystemRoute)
	mux.HandleFunc("/api/system/progress-systems", handleSystemProgressSystemRoute)

	// System-admin deliverable systems (dash)
	mux.HandleFunc("/api/system-admin/deliverable-systems/", handleSystemDeliverableSystemsRoute)
	mux.HandleFunc("/api/system-admin/deliverable-systems", handleSystemDeliverableSystemsRoute)
	mux.HandleFunc("/api/system-admin/tenant-workspaces/", handleAdminTenantWorkspaces)
	mux.HandleFunc("/api/system-admin/tenant-workspaces", handleAdminTenantWorkspaces)
	mux.HandleFunc("/api/system_admin/tenant-workspaces/", handleAdminTenantWorkspaces)
	mux.HandleFunc("/api/system_admin/tenant-workspaces", handleAdminTenantWorkspaces)

	// Convention-compliant /api/projects/ handler — supports key=value path params.
	// Examples:
	//   /api/projects/tenant_id/{tid}[/{projectId}]
	//   /api/projects/workspaces/tenant_id/{tid}[/{wsId}]
	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		path = strings.Trim(path, "/")
		// Parse convention key=value params (sets X-Auth-Tenant-Id etc.)
		rest := gatewayauth.ParseConventionPath(r, path)
		tenantID := r.Header.Get("X-Auth-Tenant-Id")
		if tenantID == "" {
			tenantID = getAuthTenant(r)
		}
		if tenantID == "" {
			writeError(w, r, 400, "tenant_id required")
			return
		}
		// OPT-20260820-015: 前端守卫可被 accessCode/直链绕过；在 /api/projects/ 分发层
		// 统一做租户成员校验（internal 旁路保留，TaskTenantURL 未配置时跳过）。
		if !requireTenantMember(w, r, getAuthUser(r), tenantID) {
			return
		}
		parts := cleanPathStr(rest)
		switch {
		case len(parts) == 0 || parts[0] == "":
			// Collection URL /api/projects/tenant_id/{tid}: dispatch by method
			// (GET list / POST create / else 405). ParseConventionPath consumes
			// the tenant_id pair, so empty parts here must NOT short-circuit to
			// handleListProjects — that silently turned POST into a list query
			// returning 200 [] (OPT-20260807-019, regressed by 6edee88).
			handleProjectsRoute(w, r, tenantID, nil)
		case parts[0] == "workspaces":
			handleWorkspacesRoute(w, r, tenantID, parts[1:])
		case parts[0] == "deliverable-systems":
			handleDeliverableSystemsRoute(w, r, tenantID, parts[1:])
		case parts[0] == "progress-systems":
			handleTenantProgressSystems(w, r, tenantID, parts[1:])
		case parts[0] == "daydaymoney":
			handleAidevRoute(w, r, tenantID, parts[1:])
		case parts[0] == "default-deliverable-system":
			handleGetDefaultDeliverableSystem(w, r, tenantID)
		case parts[0] == "settings":
			handleTenantDefaultProgressSystem(w, r, tenantID)
		case parts[0] == "switch":
			// 必须直达 handleSwitchWorkspace：handleLegacySwitchWorkspace 会先消费
			// request body 再委托，导致二次 readJSONBody 拿到空 body、workspace_id
			// 恒为空 → 400（线上"切换工作空间失败"根因）
			handleSwitchWorkspace(w, r, tenantID)
		default:
			handleProjectsRoute(w, r, tenantID, parts)
		}
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		handleListProjects(w, r)
	})

	// /api/workspace/* 遗留路径（WorkspaceSettings.vue，OPT-049 迁移后回归）
	mux.HandleFunc("/api/workspace/info/", handleWorkspaceInfo)
	mux.HandleFunc("/api/workspace/info", handleWorkspaceInfo)
	mux.HandleFunc("/api/workspace/settings/", handleWorkspaceSettings)
	mux.HandleFunc("/api/workspace/settings", handleWorkspaceSettings)

	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("/api/live", handleHealth)
}

func handleLegacySwitchWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, 405, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	wsID := strField(body, "workspace_id")
	if wsID == "" {
		writeError(w, r, 400, "工作空间ID不能为空")
		return
	}
	var tenantID string
	if err := db.QueryRow("SELECT company_id FROM project_workspace_entries WHERE id=?", wsID).Scan(&tenantID); err != nil {
		writeError(w, r, 404, "工作空间不存在或无权访问")
		return
	}
	r.Header.Set("X-Auth-Tenant-Id", tenantID)
	// body 已在本 handler 消费，委托核心逻辑时不得二次 readJSONBody
	switchWorkspaceCore(w, r, tenantID, userID, wsID)
}

func handleProjectsRoute(w http.ResponseWriter, r *http.Request, tenantID string, segs []string) {
	n := len(segs)
	if n == 0 {
		switch r.Method {
		case http.MethodGet:
			handleListProjects(w, r)
		case http.MethodPost:
			handleCreateProject(w, r)
		default:
			writeError(w, r, 405, "method not allowed")
		}
		return
	}
	head := segs[0]
	if isProjectActionSegment(head) {
		switch head {
		case "switch":
			handleSwitchWorkspace(w, r, tenantID)
		case "batch-delete":
			handleBatchDeleteProjects(w, r, tenantID)
		case "batch-get":
			handleBatchGetProjects(w, r, tenantID)
		case "gitlab-remote-repos":
			handleGitlabRemoteRepos(w, r, tenantID)
		case "batch-from-gitlab-repos":
			handleBatchFromGitlabRepos(w, r, tenantID)
		case "combined-from-gitlab-repos":
			handleCombinedFromGitlabRepos(w, r, tenantID)
		case "workspace-access":
			handleProjectsWorkspaceAccess(w, r, tenantID, segs[1:])
		case "validate-git-repo":
			handleValidateGitRepo(w, r)
		case "validate-git-repos":
			handleValidateGitRepos(w, r)
		case "translate-branch-title":
			handleTranslateBranchTitle(w, r, tenantID)
		case "manage-deliverable-system":
			handleLegacyManageDeliverableSystem(w, r, tenantID)
		case "manage-progress-column":
			handleLegacyManageProgressColumn(w, r, tenantID)
		default:
			writeNotImplemented(w, r)
		}
		return
	}
	// project ID or sub-resource
	if !strings.HasPrefix(head, "proj_") {
		// 鉴别日志（OPT-20260808-011）：非动作段、且不形如 proj_ 项目 ID 的头段落入
		// 「项目 ID 分支」多半是未知动作段（6edee88 分发丢失类 bug 特征：落此分支后
		// GET 返回 404 project not found）。记录 seg+租户+方法便于线上快速定位缺注册路由。
		logWarn(fmt.Sprintf("handleProjectsRoute: seg %q treated as project ID (tenant=%s method=%s path=%s) — if this was an action segment it is not registered",
			head, tenantID, r.Method, r.URL.Path), r.Header.Get("X-Trace-Id"))
	}
	head = resolveRequestID(idAliasKindProject, head, r.Header.Get("X-Trace-Id"))
	r.Header.Set("X-Resource-Id", head)
	if n == 1 {
		switch r.Method {
		case http.MethodGet:
			handleGetProject(w, r)
		case http.MethodPut, http.MethodPatch:
			handleUpdateProject(w, r)
		case http.MethodDelete:
			handleDeleteProject(w, r)
		default:
			writeError(w, r, 405, "method not allowed")
		}
		return
	}
	sub := segs[1]
	switch sub {
	case "revisions":
		handleProjectRevisionRoutes(w, r, tenantID, head, segs[2:])
	case "repo-access-check":
		handleRepoAccessCheck(w, r, tenantID, head)
	case "branches":
		handleProjectBranches(w, r, tenantID, head)
	case "resolve-ref":
		handleProjectResolveRef(w, r, tenantID, head)
	case "nested-git-repos":
		handleProjectNestedGitRepos(w, r, tenantID, head)
	case "git-repo-disk-sizes":
		handleProjectGitRepoDiskSizes(w, r, tenantID, head)
	case "associateWorkspace":
		handleAssociateWorkspace(w, r, tenantID, head)
	default:
		writeNotImplemented(w, r)
	}
}

func handleWorkspacesRoute(w http.ResponseWriter, r *http.Request, tenantID string, segs []string) {
	n := len(segs)
	if n == 0 {
		switch r.Method {
		case http.MethodGet:
			handleListWorkspaces(w, r)
		case http.MethodPost:
			handleCreateWorkspace(w, r)
		default:
			writeError(w, r, 405, "method not allowed")
		}
		return
	}
	head := segs[0]
	if head == "batch-get" {
		handleBatchGetWorkspaces(w, r, tenantID)
		return
	}
	if head == "switch" {
		handleSwitchWorkspace(w, r, tenantID)
		return
	}
	head = resolveRequestID(idAliasKindWorkspace, head, r.Header.Get("X-Trace-Id"))
	if n >= 2 {
		sub := segs[1]
		switch sub {
		case "work-panel-filters":
			handleWorkPanelFilters(w, r, tenantID, head)
			return
		case "task-kind-options":
			handleTaskKindOptions(w, r, tenantID, head)
			return
		case "code-lang-options":
			handleCodeLangOptions(w, r, tenantID, head)
			return
		case "create-task-field-settings":
			handleCreateTaskFieldSettings(w, r, tenantID, head)
			return
		case "progress-system", "column-system":
			handleWorkspaceProgressSystem(w, r, tenantID, head)
			return
		}
	}
	r.Header.Set("X-Resource-Id", head)
	switch r.Method {
	case http.MethodGet:
		handleGetWorkspace(w, r)
	case http.MethodPut, http.MethodPatch:
		handleUpdateWorkspace(w, r)
	case http.MethodDelete:
		handleDeleteWorkspace(w, r)
	default:
		writeError(w, r, 405, "method not allowed")
	}
}

func handleSystemProgressSystemRoute(w http.ResponseWriter, r *http.Request) {
	// path: /api/system-admin/progress-systems/[/{id}[/set-default]]
	// or:   /api/system/progress-systems/[/{id}[/set-default]]
	p := strings.TrimPrefix(r.URL.Path, "/api/system-admin/progress-systems")
	p = strings.TrimPrefix(p, "/api/system/progress-systems")
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		handleSystemProgressSystems(w, r)
		return
	}
	segs := strings.SplitN(strings.TrimPrefix(p, "/"), "/", 2)
	if len(segs) == 1 {
		// /{id}
		handleSystemProgressSystemByID(w, r)
	} else if segs[1] == "set-default" {
		handleSystemProgressSystemSetDefault(w, r)
	} else {
		writeError(w, r, 404, "not found")
	}
}

func handleSystemDeliverableSystemsRoute(w http.ResponseWriter, r *http.Request) {
	// path: /api/system-admin/deliverable-systems/[/{id}]
	p := strings.TrimPrefix(r.URL.Path, "/api/system-admin/deliverable-systems")
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		handleSystemDeliverableSystems(w, r)
		return
	}
	systemID := strings.TrimPrefix(p, "/")
	handleSystemDeliverableSystemByID(w, r, systemID)
}
