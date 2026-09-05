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

func main() {
	const svc = "task-task-service"
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		tracelog.Fatalf(svc, "[taskTaskService] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	initLogging(svc)
	tracelog.Init(svc)
	if meta, err := daydaymoneymeta.LoadNearDir(filepath.Join(repoRoot, "taskTaskService")); err == nil {
		tracelog.SetDaydaymoneyMeta(meta.ServiceID, meta.Tags)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := openDB(cfg.DBPath); err != nil {
			tracelog.Fatalf(svc, "[taskTaskService] migration: %v", err)
		}
		if err := ensureCoreTablesExist(); err != nil {
			tracelog.Fatalf(svc, "[taskTaskService] schema check: %v", err)
		}
		if err := runDataMigrate(repoRoot); err != nil {
			tracelog.Fatalf(svc, "[taskTaskService] dataMigrate: %v", err)
		}
		log.Println("[taskTaskService] migration complete")
		return
	}

	if err := openDB(cfg.DBPath); err != nil {
		tracelog.Fatalf(svc, "[taskTaskService] db: %v", err)
	}
	defer db.Close()
	// OPT-20260816-030：排队自动开跑已迁到 taskEvents queued_auto_run_scan timer，
	// 由该 timer 批量 POST dispatch-once 触发，业务进程不再自带 30s ticker。

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logInfo("listening on "+addr, "")
	log.Printf("[taskTaskService] listening on %s (db=%s)", addr, cfg.DBPath)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(corsMiddleware(authMiddleware(tracelog.MetricsMiddleware(mux))))); err != nil {
		tracelog.Fatalf(svc, "[taskTaskService] listen: %v", err)
	}
}

func cleanPath(r *http.Request, prefix string) []string {
	p := strings.TrimPrefix(r.URL.Path, prefix)
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return nil
	}
	return strings.SplitN(p, "/", 8)
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/health/", handleHealth)
	mux.Handle("GET /api/metrics", tracelog.MetricsHandler())

	mux.HandleFunc("/api/internal/tasks/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/internal/tasks/")
		if parts == nil || len(parts) < 1 {
			writeError(w, r, 404, "not found")
			return
		}
		// POST /api/internal/tasks/batch-get/ — batch resolve task titles
		if parts[0] == "batch-get" && r.Method == http.MethodPost {
			handleInternalTasksBatchGet(w, r)
			return
		}
		// POST /api/internal/tasks/exists/ — batch task existence check（cloud 孤儿对账用）
		if parts[0] == "exists" && r.Method == http.MethodPost {
			handleInternalTasksExist(w, r)
			return
		}
		// POST /api/internal/tasks/terminal-kinds/ — 批量终态（已取消/已完成），入站守卫与 CSC 对账
		if parts[0] == "terminal-kinds" && r.Method == http.MethodPost {
			handleInternalTasksTerminalKinds(w, r)
			return
		}
		// POST /api/internal/tasks/expire-posts/ — 每日到期扫描（taskEvents 定时触发）
		if parts[0] == "expire-posts" && r.Method == http.MethodPost {
			handleInternalExpirePosts(w, r)
			return
		}
		// POST /api/internal/tasks/queued-schedule/dispatch-once/ — 排队调度分发 + 窗口自动关闭（OPT-20260816-030）
		if parts[0] == "queued-schedule" && len(parts) >= 2 && parts[1] == "dispatch-once" && r.Method == http.MethodPost {
			handleInternalQueuedScheduleDispatchOnce(w, r)
			return
		}
		r.Header.Set("X-Task-Id", parts[0])
		if len(parts) >= 2 && parts[1] == "container-snapshot" {
			handleInternalContainerSnapshot(w, r)
			return
		}
		if len(parts) >= 2 && parts[1] == "dequeue-queued-auto-run" {
			handleInternalDequeueQueuedAutoRun(w, r, parts[0])
			return
		}
		writeError(w, r, 404, "not found")
	})

	mux.HandleFunc("/api/tenant/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/tenant/")
		if parts == nil || len(parts) < 1 {
			writeError(w, r, 404, "not found")
			return
		}
		tenantID := parts[0]
		if getAuthTenant(r) == "" {
			r.Header.Set("X-Auth-Tenant-Id", tenantID)
		}

		// Tenant membership gate: all /api/tenant/* endpoints require the caller
		// to be a member of the target tenant. Internal calls bypass this check.
		if !requireTenantMember(w, r, getAuthUser(r), tenantID) {
			return
		}

		// translate-branch-title: retired on TTS (501); owner is task-project-service
		if len(parts) >= 3 && parts[1] == "projects" && parts[2] == "translate-branch-title" {
			handleTranslateBranchTitle(w, r, tenantID)
			return
		}

		// /api/tenant/<tid>/workspace/<wid>/todos[/<taskId>[/...]]
		if len(parts) >= 4 && parts[1] == "workspace" && parts[3] == "todos" {
			r.Header.Set("X-Workspace-Id", parts[2])
			if len(parts) >= 5 && parts[4] != "" {
				r.Header.Set("X-Resource-Id", parts[4])
			}
			handleTaskRoutes(w, r)
			return
		}

		// /api/tenant/<tid>/workspace/<wid>/queue-schedule — 工作空间级排队调度设置
		if len(parts) >= 4 && parts[1] == "workspace" && parts[3] == "queue-schedule" {
			r.Header.Set("X-Workspace-Id", parts[2])
			handleWorkspaceQueueScheduleRoutes(w, r, tenantID, parts[2])
			return
		}

		// /api/tenant/<tid>/tasks/search — 租户级任务搜索（计费筛选等）
		if len(parts) >= 3 && parts[1] == "tasks" && parts[2] == "search" {
			handleSearchTasks(w, r, tenantID)
			return
		}

		// /api/tenant/<tid>/tasks/<taskId>/comments
		if len(parts) >= 4 && parts[1] == "tasks" && parts[3] == "comments" {
			r.Header.Set("X-Task-Id", parts[2])
			handleCommentRoutes(w, r)
			return
		}

		writeError(w, r, 404, "not found")
	})

	// 评论回复（parent 在 path）：POST /api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent}/
	mux.HandleFunc("/api/tenant_id/", handleTenantIDPrefixedRoutes)

	// OPT-052: /api/git-identities/user/{userId}/ — user profile git identities CRUD
	// Also: tenant/{tid}/member/{mid}/ manage + internal ensure-default (v77).
	mux.HandleFunc("/api/git-identities/", dispatchGitIdentitiesRoutes)
	mux.HandleFunc("/api/internal/git-identities/ensure-default", handleInternalEnsureDefaultGitIdentity)
	mux.HandleFunc("/api/internal/git-identities/ensure-default/", handleInternalEnsureDefaultGitIdentity)
	mux.HandleFunc("/api/internal/git-identities/lookup", handleInternalLookupGitIdentity)
	mux.HandleFunc("/api/internal/git-identities/lookup/", handleInternalLookupGitIdentity)
	mux.HandleFunc("/api/internal/task-projects/detach-by-project", handleInternalDetachTaskProjects)
	mux.HandleFunc("/api/internal/task-projects/detach-by-project/", handleInternalDetachTaskProjects)

	mux.HandleFunc("/api/internal/comments/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/internal/comments/")
		if parts == nil || len(parts) < 1 {
			writeError(w, r, 404, "not found")
			return
		}
		// GET /api/internal/comments/{id}/created-by — comment author lookup for
		// owner-service HTTP fallback (OPT-20260820-021).
		if len(parts) >= 2 && parts[1] == "created-by" && r.Method == http.MethodGet {
			handleInternalCommentCreatedBy(w, r, parts[0])
			return
		}
		if len(parts) >= 2 && parts[1] == "git-oauth-grant" && r.Method == http.MethodPost {
			handleInternalMarkCommentGitOAuthGrant(w, r, parts[0])
			return
		}
		writeError(w, r, 404, "not found")
	})

	mux.HandleFunc("/api/user/", func(w http.ResponseWriter, r *http.Request) {
		parts := cleanPath(r, "/api/user/")
		if len(parts) >= 3 && parts[1] == "profile" && parts[2] == "git-identities" {
			handleProfileGitIdentities(w, r)
			return
		}
		writeError(w, r, 404, "not found")
	})

	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")

		// todos 家族（前端约定）：
		//   /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}[/sub...]
		// gatewayauth.ParseConventionPath 消费 kv 对并注入头，保留 kv 后位置段（taskId）。
		if path == "todos" || strings.HasPrefix(path, "todos/") {
			rest := gatewayauth.ParseConventionPath(r, path)
			// rest = "todos/{taskId}[/sub...]"；首个位置段即 taskId，其余子资源
			// （subtree/queued-auto-run 等）由 handleTaskRoutes 按 r.URL.Path 后缀分发。
			segs := strings.SplitN(strings.Trim(rest, "/"), "/", 3)
			if len(segs) >= 2 && segs[1] != "" {
				r.Header.Set("X-Resource-Id", segs[1])
			}
			handleTaskRoutes(w, r)
			return
		}

		// 租户级任务搜索（navbar / 计费筛选约定路径）：
		//   GET /api/tasks/search/tenant_id/{tid}/?q=&workspace_id=&limit=&assignee_ids=
		// 与 /api/tenant/{tid}/tasks/search/ 语义一致；须在 {taskId} 分支之前拦截，
		// 否则 "search" 会被误当作 taskId 走 handleGetTaskByID → 404。
		if path == "search" || strings.HasPrefix(path, "search/") {
			gatewayauth.ParseConventionPath(r, path)
			tenantID := strings.TrimSpace(getAuthTenant(r))
			if tenantID == "" {
				writeError(w, r, http.StatusBadRequest, "tenant_id required")
				return
			}
			if !requireTenantMember(w, r, getAuthUser(r), tenantID) {
				return
			}
			handleSearchTasks(w, r, tenantID)
			return
		}

		// Support convention path: /api/tasks/{taskId}/tenant_id/{tid}/...
		rest := gatewayauth.ParseConventionPath(r, path)
		parts := strings.SplitN(strings.Trim(rest, "/"), "/", 4)
		if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
			writeError(w, r, 404, "not found")
			return
		}
		if r.Header.Get("X-Task-Id") == "" {
			r.Header.Set("X-Task-Id", parts[0])
		}
		if len(parts) == 1 && r.Method == http.MethodGet {
			handleGetTaskByID(w, r)
			return
		}
		if len(parts) >= 2 && parts[1] == "feature-params" {
			handleFeatureParams(w, r)
			return
		}
		if len(parts) >= 2 && parts[1] == "feature-params-snapshots" {
			handleFeatureParamsSnapshots(w, r)
			return
		}
		// 人类评论子路由（与 /api/tenant/{tid}/tasks/{taskId}/comments 语义一致）：
		//   GET/POST /api/tasks/{taskId}/comments/tenant_id/{tid}
		//   PATCH    /api/tasks/{taskId}/comments/{commentId}/tenant_id/{tid}
		if len(parts) >= 2 && parts[1] == "revisions" {
			taskID := parts[0]
			r.Header.Set("X-Resource-Id", taskID)
			handleTaskRevisionRoutes(w, r, getAuthTenant(r), getAuthUser(r), r.Header.Get("X-Workspace-Id"), taskID)
			return
		}
		if len(parts) >= 2 && parts[1] == "comments" {
			commentID := ""
			if len(parts) >= 3 {
				commentID = parts[2]
			}
			tenantID := getAuthTenant(r)
			userID := getAuthUser(r)
			taskID := parts[0]
			if commentID != "" {
				if r.Method == http.MethodPatch {
					handlePatchComment(w, r, tenantID, userID, taskID, commentID)
					return
				}
				writeError(w, r, 404, "not found")
				return
			}
			switch r.Method {
			case http.MethodGet:
				handleListComments(w, r, tenantID, userID, taskID)
			case http.MethodPost:
				handleCreateComment(w, r, tenantID, userID, taskID)
			default:
				writeError(w, r, 405, "method not allowed")
			}
			return
		}
		writeError(w, r, 404, "not found")
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/health") || r.URL.Path == "/api/metrics" ||
			strings.HasPrefix(r.URL.Path, "/api/internal/") {
			next.ServeHTTP(w, r)
			return
		}

		if gatewayauth.ApplyGatewayUser(r, cfg.TaskGatewayInternalSecret) {
			next.ServeHTTP(w, r)
			return
		}

		userID := strings.TrimSpace(r.Header.Get(gatewayauth.HeaderAuthUserID))
		if userID != "" {
			next.ServeHTTP(w, r)
			return
		}
		tenantID := ""
		var err error
		userID, tenantID, err = verifyJWT(r)
		if err != nil {
			writeErrorMap(w, r, http.StatusUnauthorized, map[string]interface{}{
				"error":   "unauthorized",
				"message": err.Error(),
			})
			return
		}
		r.Header.Set("X-Auth-User-Id", userID)
		if tenantID != "" {
			r.Header.Set("X-Auth-Tenant-Id", tenantID)
		}
		next.ServeHTTP(w, r)
	})
}
