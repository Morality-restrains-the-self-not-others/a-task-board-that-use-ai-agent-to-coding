package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"daydaymoneymeta"
	"gatewayauth"
	"tracelog"
)

func main() {
	const svc = "task-cloud-service"
	stripOutboundProxyEnv()
	repoRoot, err := findMonorepoRoot()
	if err != nil {
		tracelog.Fatalf(svc, "[taskCloudService] monorepo root: %v", err)
	}
	loadConfig(repoRoot)
	loadStepFullCOSConf(repoRoot)
	initStepFullObjectStoreFromCfg()
	initLogging(svc)
	tracelog.Init(svc)
	if meta, err := daydaymoneymeta.LoadNearDir(filepath.Join(repoRoot, "taskCloudService")); err == nil {
		tracelog.SetDaydaymoneyMeta(meta.ServiceID, meta.Tags)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		// Explicit migrate CLI — also used if migrate.sh invokes go migrate.
		if err := openDB(cfg.DBPath); err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] migration: %v", err)
		}
		if err := runDataMigrate(repoRoot); err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] dataMigrate: %v", err)
		}
		if err := openBudgetDB(cfg.BudgetDBPath, repoRoot); err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] budget db: %v", err)
		}
		if err := runBudgetDataMigrate(budgetDB, repoRoot); err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] budget dataMigrate: %v", err)
		}
		log.Println("[taskCloudService] migration complete")
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "recycle-idle" {
		if err := openDB(cfg.DBPath); err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] recycle-idle db: %v", err)
		}
		n, err := recycleIdleMachines(time.Now().UTC())
		if err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] recycle-idle: %v", err)
		}
		log.Printf("[taskCloudService] recycle-idle recycled=%d", n)
		return
	}

	// OPT-20260827-012: 一次性回填 CLI — 双写只覆盖新 insert，存量分片日志缺 COS
	// 对象与指针。禁止业务 ticker 调用。
	if len(os.Args) > 1 && os.Args[1] == "backfill-ccb-startup-logs" {
		if err := openDB(cfg.DBPath); err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] backfill db: %v", err)
		}
		defer db.Close()
		res, err := backfillCCBStartupLogs(context.Background())
		if err != nil {
			tracelog.Fatalf(svc, "[taskCloudService] backfill: %v", err)
		}
		log.Printf("[taskCloudService] backfill-ccb-startup-logs tables=%d comments=%d archived=%d evicted=%d skipped=%d errors=%d",
			res.TablesScanned, res.CommentsScanned, res.Archived, res.Evicted, res.Skipped, res.Errors)
		for _, m := range res.ErrorMessages {
			log.Printf("[taskCloudService] backfill-ccb-startup-logs error=%s", m)
		}
		return
	}

	if err := openDB(cfg.DBPath); err != nil {
		tracelog.Fatalf(svc, "[taskCloudService] db: %v", err)
	}
	defer db.Close()
	// OPT-20260809-016: 启动时无条件执行 dataMigrate（幂等：data_migrate_log step_key 去重），
	// 避免新表上线依赖人工记得跑 migrate CLI 导致状态端点 500。
	if err := runDataMigrate(repoRoot); err != nil {
		tracelog.Fatalf(svc, "[taskCloudService] startup dataMigrate: %v", err)
	}
	if err := openBudgetDB(cfg.BudgetDBPath, repoRoot); err != nil {
		tracelog.Fatalf(svc, "[taskCloudService] budget db: %v", err)
	}
	defer closeBudgetDB()
	// budget 表同样接入启动自动迁移（与 migrate CLI 对齐）。
	if err := runBudgetDataMigrate(budgetDB, repoRoot); err != nil {
		tracelog.Fatalf(svc, "[taskCloudService] startup budget dataMigrate: %v", err)
	}
	initRelayRedis(repoRoot)
	// OPT-20260816-028：孤儿/泄漏 CSC 对账已迁到 taskEvents cloud_csc_reconcile timer，
	// 由该 timer 批量 POST 触发，禁止业务进程内 ticker。
	// OPT-20260827-029：看板 GET 只读 last_runtime_status；Describe/孤儿名对账同 tick 第三步。
	// OPT-20260816-025：空闲回收已迁到 taskEvents workspace_machine_idle timer（18044），
	// 进程内 IDLE_RECYCLE_TICK_SEC ticker 已删除，避免与 timer 双扫。

	mux := http.NewServeMux()
	mountRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	logInfo("listening on "+addr, "")
	log.Printf("[taskCloudService] listening on %s (db=%s)", addr, cfg.DBPath)
	if err := tracelog.ListenAndServe(addr, tracelog.Middleware(corsMiddleware(gatewayUserMiddleware(tracelog.MetricsMiddleware(mux))))); err != nil {
		tracelog.Fatalf(svc, "[taskCloudService] listen: %v", err)
	}
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/internal/budget/", handleInternalBudgetRoutes)
	mux.HandleFunc("/api/internal/budget", handleInternalBudgetRoutes)
	mux.HandleFunc("/api/internal/ai-endpoint/", handleInternalAIEndpointRoutes)
	mux.HandleFunc("/api/internal/ai-endpoint", handleInternalAIEndpointRoutes)
	mux.HandleFunc("/api/internal/feature-params-env/", handleInternalFeatureParamsEnv)
	mux.HandleFunc("/api/internal/feature-params-env", handleInternalFeatureParamsEnv)
	mux.HandleFunc("/api/internal/cloud-server-config/", handleInternalCloudServerConfig)
	mux.HandleFunc("/api/internal/cloud-server-config", handleInternalCloudServerConfig)
	mux.HandleFunc("/api/internal/cloud-server-events/", handleInternalCloudServerEventRoutes)
	mux.HandleFunc("/api/internal/cloud-server-events", handleInternalCloudServerEventRoutes)
	mux.HandleFunc("/api/internal/access-key-iam-associations/", handleInternalAccessKeyIAMRoutes)
	mux.HandleFunc("/api/internal/access-key-iam-associations", handleInternalAccessKeyIAMRoutes)
	mux.HandleFunc("/api/internal/tenant-member/", handleInternalTenantMember)
	mux.HandleFunc("/api/internal/tenant-member", handleInternalTenantMember)
	mux.HandleFunc("/api/internal/relay-startup-session/upsert/", handleInternalRelayStartupSessionUpsert)
	mux.HandleFunc("/api/internal/relay-startup-session/upsert", handleInternalRelayStartupSessionUpsert)
	mux.HandleFunc("/api/internal/cloud-platform-authorizations/import", handleInternalCloudPlatformAuthorizations)
	mux.HandleFunc("/api/internal/tenant-installed-images/import", handleInternalTenantInstalledImages)
	mux.HandleFunc("/api/internal/tenant-installed-images/lookup", handleInternalTenantInstalledImagesLookup)
	mux.HandleFunc("/api/internal/cloud-platform-authorizations/lookup", handleInternalCloudPlatformAuthorizationLookup)
	mux.HandleFunc("/api/internal/cloud-platform-authorizations/active-methods", handleInternalActiveMethodsList)
	mux.HandleFunc("/api/internal/cloud/oauth-token/lookup", handleInternalOAuthTokenLookup)
	mux.HandleFunc("/api/internal/cloud/oauth-token/lookup/", handleInternalOAuthTokenLookup)
	mux.HandleFunc("/api/internal/cloud/init-tenant-feature-params/", handleInternalInitTenantFeatureParams)
	mux.HandleFunc("/api/internal/cloud/init-tenant-feature-params", handleInternalInitTenantFeatureParams)
	mux.HandleFunc("/api/internal/cloud/tenant-budget-enabled/", handleInternalTenantBudgetEnabled)
	mux.HandleFunc("/api/internal/cloud/tenant-budget-enabled", handleInternalTenantBudgetEnabled)

	mux.HandleFunc("/api/internal/vendor-cloud-credentials/lookup", handleInternalVendorCloudCredentialLookup)

	mux.HandleFunc("/api/internal/layer-git-push/prepare", handleInternalLayerGitPushPrepare)
	mux.HandleFunc("/api/internal/layer-git-push/prepare/", handleInternalLayerGitPushPrepare)
	mux.HandleFunc("/api/internal/layer-git-push/complete", handleInternalLayerGitPushComplete)
	mux.HandleFunc("/api/internal/layer-git-push/complete/", handleInternalLayerGitPushComplete)
	mux.HandleFunc("/api/internal/layer-git-push/auth-context", handleInternalLayerGitPushAuthContext)
	mux.HandleFunc("/api/internal/layer-git-push/auth-context/", handleInternalLayerGitPushAuthContext)
	mux.HandleFunc("/api/internal/layer-git-repo-identities/prepare", handleInternalLayerGitRepoIdentitiesPrepare)
	mux.HandleFunc("/api/internal/layer-git-repo-identities/prepare/", handleInternalLayerGitRepoIdentitiesPrepare)

	mux.HandleFunc("/api/internal/runtime-session/open", handleInternalRuntimeSessionOpen)
	mux.HandleFunc("/api/internal/runtime-session/open/", handleInternalRuntimeSessionOpen)

	mux.HandleFunc("/api/internal/cloud/compute/recycle-idle-machines/", handleInternalRecycleIdleMachines)
	mux.HandleFunc("/api/internal/cloud/compute/recycle-idle-machines", handleInternalRecycleIdleMachines)
	mux.HandleFunc("/api/internal/cloud/workspace-machine-policy/", handleInternalWorkspaceMachinePolicy)
	mux.HandleFunc("/api/internal/cloud/workspace-machine-policy", handleInternalWorkspaceMachinePolicy)

	mux.HandleFunc("/api/internal/cloud/compute/instance-bindings/", handleInternalInstanceBindings)
	mux.HandleFunc("/api/internal/cloud/compute/instance-bindings", handleInternalInstanceBindings)
	mux.HandleFunc("/api/internal/cloud/compute/mark-terminal-released/", handleInternalMarkTerminalReleased)
	mux.HandleFunc("/api/internal/cloud/compute/mark-terminal-released", handleInternalMarkTerminalReleased)
	mux.HandleFunc("/api/internal/cloud/compute/mark-comment-terminal-released/", handleInternalMarkCommentTerminalReleased)
	mux.HandleFunc("/api/internal/cloud/compute/mark-comment-terminal-released", handleInternalMarkCommentTerminalReleased)
	mux.HandleFunc("/api/internal/cloud/compute/migrate-container-off-instance/", handleInternalMigrateContainerOffInstance)
	mux.HandleFunc("/api/internal/cloud/compute/migrate-container-off-instance", handleInternalMigrateContainerOffInstance)
	mux.HandleFunc("/api/internal/cloud/compute/clear-idle-bindings-on-instance/", handleInternalClearIdleBindingsOnInstance)
	mux.HandleFunc("/api/internal/cloud/compute/clear-idle-bindings-on-instance", handleInternalClearIdleBindingsOnInstance)
	mux.HandleFunc("/api/internal/cloud/compute/set-terminal-released-flag/", handleInternalSetTerminalReleasedFlag)
	mux.HandleFunc("/api/internal/cloud/compute/set-terminal-released-flag", handleInternalSetTerminalReleasedFlag)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-orphan-csc/", handleInternalReconcileOrphanCSC)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-orphan-csc", handleInternalReconcileOrphanCSC)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-leaked-servers/", handleInternalReconcileLeakedServers)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-leaked-servers", handleInternalReconcileLeakedServers)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-workspace-machine-runtimes/", handleInternalReconcileWorkspaceMachineRuntimes)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-workspace-machine-runtimes", handleInternalReconcileWorkspaceMachineRuntimes)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-stale-starting-bindings/", handleInternalReconcileStaleStartingBindings)
	mux.HandleFunc("/api/internal/cloud/compute/reconcile-stale-starting-bindings", handleInternalReconcileStaleStartingBindings)

	mux.HandleFunc("/api/internal/cloud/stops/", handleInternalStopRequestRoutes)
	mux.HandleFunc("/api/internal/cloud/stops", handleInternalStopRequestRoutes)
	mux.HandleFunc("/api/internal/cloud/users/", handleInternalCloudUsersRouter)
	mux.HandleFunc("/api/internal/cloud/users", handleInternalCloudUsersRouter)

	mux.HandleFunc("/api/internal/image/resolve", handleInternalImageResolve)
	mux.HandleFunc("/api/internal/image/resolve/", handleInternalImageResolve)
	mux.HandleFunc("/api/internal/extract-auto-run-steps", handleInternalExtractAutoRunSteps)
	mux.HandleFunc("/api/internal/extract-auto-run-steps/", handleInternalExtractAutoRunSteps)
	mux.HandleFunc("/api/internal/cloud/job-execution-events/", handleInternalJobExecutionEvents)
	mux.HandleFunc("/api/internal/cloud/job-execution-events", handleInternalJobExecutionEvents)

	mux.HandleFunc("/api/vendor/cloud-platform-credentials", handleVendorCloudCredentialRoutes)
	mux.HandleFunc("/api/vendor/cloud-platform-credentials/", handleVendorCloudCredentialRoutes)

	mux.HandleFunc("/api/vendor/cloud-server-images/", handleVendorCloudServerImageRoutes)
	mux.HandleFunc("/api/vendor/cloud-server-images", handleVendorCloudServerImageRoutes)

	mux.HandleFunc("/api/personal/", handlePersonalAPIRoutes)

	// OPT-20260730-001: sub-token-providers + recommended-llm-providers migrated from Django
	mux.HandleFunc("/api/system-admin/sub-token-providers/", handleSystemAdminSubTokenProviders)
	mux.HandleFunc("/api/system-admin/sub-token-providers", handleSystemAdminSubTokenProviders)
	mux.HandleFunc("/api/sub-token-providers/", handleTenantSubTokenProviders)
	mux.HandleFunc("/api/sub-token-providers", handleTenantSubTokenProviders)
	mux.HandleFunc("/api/system-admin/recommended-llm-providers/", handleSystemAdminRecommendedLLMProviders)
	mux.HandleFunc("/api/system-admin/recommended-llm-providers", handleSystemAdminRecommendedLLMProviders)
	mux.HandleFunc("/api/system-admin/step-full-cos/", handleSystemAdminStepFullCOS)
	mux.HandleFunc("/api/system-admin/step-full-cos", handleSystemAdminStepFullCOS)
	mux.HandleFunc("/api/recommended-llm-providers/", handleTenantRecommendedLLMProviders)
	mux.HandleFunc("/api/recommended-llm-providers", handleTenantRecommendedLLMProviders)

	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/health/", handleHealth)
	mux.Handle("GET /api/metrics", tracelog.MetricsHandler())
	mux.HandleFunc("/api/schema/", handleOpenAPISchema)
	mux.HandleFunc("/api/schema", handleOpenAPISchema)
	mux.HandleFunc("/api/swagger/", handleSwaggerUI)
	mux.HandleFunc("/api/swagger", handleSwaggerUI)
	mux.HandleFunc("/api/schema-internal/", handleOpenAPIInternalSchema)
	mux.HandleFunc("/api/schema-internal", handleOpenAPIInternalSchema)
	mux.HandleFunc("/api/swagger-internal/", handleSwaggerInternalUI)
	mux.HandleFunc("/api/swagger-internal", handleSwaggerInternalUI)
	mux.HandleFunc("/callback/cloudplatform/oauth2.0/aliyun", handleAliyunOAuthCallback)

	// System-admin cloud management → vendor credentials (no tenant required)
	mux.HandleFunc("/api/system-admin/cloud/", handleSystemAdminCloudRoutes)

	mux.HandleFunc("/api/cloud/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/cloud/")
		// Convention path: /api/cloud/{funcName}/tenant_id/{tid}/workspace_id/{wid}/task_id/{taskId}
		rest := gatewayauth.ParseConventionPath(r, path)
		parts := strings.SplitN(rest, "/", 4)
		if len(parts) >= 1 && parts[0] == "oauth" {
			handleCloudOAuthRoutes(w, r, parts[1:])
			return
		}
		taskID := r.Header.Get("X-Task-Id")
		workspaceID := r.Header.Get("X-Workspace-Id")
		if taskID != "" && workspaceID != "" {
			handleCloudTaskRoutes(w, r, rest)
			return
		}
		if workspaceID != "" {
			handleCloudWorkspaceRoutes(w, r, rest)
			return
		}
		handleCloudTenantRoutes(w, r, parts)
	})

	mux.HandleFunc("/api/cloud", func(w http.ResponseWriter, r *http.Request) {
		handleCloudTenantRoutes(w, r, nil)
	})
}

func handleCloudTenantRoutes(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		writeErrorJSON(w, r, 404, "not found")
		return
	}
	switch parts[0] {
	case "cloud-platform-authorizations":
		handleCloudAuthRoutes(w, r, parts[1:])
	case "toggle-active":
		handleToggleActive(w, r)
	case "active-list":
		handleActiveList(w, r)
	case "regions":
		handleCloudRegions(w, r)
	case "images":
		handleCloudImages(w, r)
	case "server-images":
		if len(parts) > 1 {
			switch parts[1] {
			case "vpcs":
				handleCloudNetworkList(w, r, "vpc")
			case "vswitches":
				handleCloudNetworkList(w, r, "vswitch")
			case "security-groups":
				handleCloudNetworkList(w, r, "security_group")
			case "create-vswitch":
				handleCloudCreateVswitch(w, r)
			case "update-vswitch":
				handleCloudUpdateVswitch(w, r)
			case "create-security-group":
				handleCloudCreateSecurityGroup(w, r)
			case "update-security-group":
				handleCloudUpdateSecurityGroup(w, r)
			default:
				writeErrorMapJSON(w, r, 404, map[string]interface{}{"error": "server-images sub-route not found", "path": parts[1]})
			}
		} else {
			handleCloudServerImages(w, r)
		}
	case "oauth-tokens":
		handleOAuthTokens(w, r, parts[1:])
	// 云平台授权查询子资源（Phase 3h 设计：cloud-platform/{authId}/cloud/{sub}；
	// 前端约定等价形式 cloud-platform/{authId}/{sub}/tenant_id/{tid}）
	case "cloud-platform":
		tenantID := getAuthTenant(r)
		if len(parts) < 2 {
			writeErrorJSON(w, r, 404, "cloud-platform authId required")
			return
		}
		authID := parts[1]
		sub := ""
		if len(parts) >= 3 {
			if parts[2] == "cloud" {
				sub = strings.Join(parts[3:], "/")
			} else {
				sub = strings.Join(parts[2:], "/")
			}
		}
		handleCloudPlatformRoutes(w, r, tenantID, authID, sub)
	case "server-config-default":
		handleServerConfigDefault(w, r, parts[1:])
	// 镜像市场租户 API（kv-last 形式 /api/cloud/installed-images/{sub}/tenant_id/{tid}/）
	case "installed-images":
		tenantID := getAuthTenant(r)
		sub := strings.Join(parts[1:], "/")
		switch {
		case sub == "catalog":
			handleInstalledImageCatalog(w, r, tenantID)
		case sub == "dev-catalog":
			handleInstalledImageDevCatalog(w, r, tenantID)
		case sub == "resolve-target-architectures":
			handleInstalledImageResolveTargetArchitectures(w, r)
		case sub == "":
			handleInstalledImageCollection(w, r, tenantID)
		default:
			handleInstalledImageDetail(w, r, tenantID, sub)
		}
	case "userdata-templates":
		handleUserDataTemplates(w, r)
	case "ai-model-authorizations":
		handleAIModelAuths(w, r)
	case "start-server":
		handleStartServer(w, r)
	// v64: /api/cloud/ 约定下恢复 budget-permissions / feature-params 租户 API
	//（前端已迁移 kv-last 路径；此前 ead5583 移除 /api/tenant/ 时未重新挂载）
	case "budget-permissions":
		handleUserTenantBudgetPermissions(w, r)
	case "feature-params":
		handleTenantFeatureParamsRoute(w, r, getAuthTenant(r), "feature-params")
	case "vpcs":
		handleCloudNetworkList(w, r, "vpc")
	case "vswitches":
		handleCloudNetworkList(w, r, "vswitch")
	case "security-groups":
		handleCloudNetworkList(w, r, "security_group")
	case "occupied-cidr-blocks":
		handleCloudOccupiedCidrBlocks(w, r)
	case "create-vpc":
		handleCloudCreateVpc(w, r)
	case "update-vpc":
		handleCloudUpdateVpc(w, r)
	default:
		writeErrorMapJSON(w, r, 404, map[string]interface{}{"error": "route not found", "path": parts[0]})
	}
}
