package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"gatewaycors"
	"tracelog"
)

// corsMiddleware adds CORS headers to every response.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeadersWith("X-Relay-To-Trae-Secret"))
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// secretOK checks the X-Relay-To-Trae-Secret header.
func secretOK(r *http.Request) bool {
	if secretToken == "" {
		return true
	}
	got := strings.TrimSpace(r.Header.Get("X-Relay-To-Trae-Secret"))
	return got == secretToken
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if !secretOK(r) {
		writeJSON(w, 401, map[string]string{"detail": "unauthorized"})
		return
	}

	var body struct {
		Env              map[string]string `json:"env"`
		TenantID         string            `json:"tenant_id"`
		TenantId         string            `json:"tenantId"`
		WorkspaceID      string            `json:"workspace_id"`
		WorkspaceId      string            `json:"workspaceId"`
		TaskID           string            `json:"task_id"`
		TaskId           string            `json:"taskId"`
		Image            string            `json:"image"`
		InstalledImageID string            `json:"installed_image_id"`
		InstalledImageId string            `json:"installedImageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body = struct {
			Env              map[string]string `json:"env"`
			TenantID         string            `json:"tenant_id"`
			TenantId         string            `json:"tenantId"`
			WorkspaceID      string            `json:"workspace_id"`
			WorkspaceId      string            `json:"workspaceId"`
			TaskID           string            `json:"task_id"`
			TaskId           string            `json:"taskId"`
			Image            string            `json:"image"`
			InstalledImageID string            `json:"installed_image_id"`
			InstalledImageId string            `json:"installedImageId"`
		}{}
	}

	envIn := body.Env
	if envIn == nil {
		envIn = map[string]string{}
	}

	tenantID := body.TenantID
	if tenantID == "" {
		tenantID = body.TenantId
	}
	workspaceID := body.WorkspaceID
	if workspaceID == "" {
		workspaceID = body.WorkspaceId
	}
	taskID := body.TaskID
	if taskID == "" {
		taskID = body.TaskId
	}
	imageRef := strings.TrimSpace(body.Image)
	installedImageID := strings.TrimSpace(body.InstalledImageID)
	if installedImageID == "" {
		installedImageID = strings.TrimSpace(body.InstalledImageId)
	}

	if !tryAcquireLifecycleLock() {
		log.Printf("[go_relayToTrae] event=lifecycle_lock_timeout op=start wait=%s", lifecycleLockWait())
		writeJSON(w, 503, map[string]string{
			"detail": "relay lifecycle busy; retry later",
			"error":  "LIFECYCLE_BUSY",
		})
		return
	}
	defer lifecycleMu.Unlock()

	traceID := tracelog.TraceIDFromContext(r.Context())
	corr := tracelog.CorrelationFromContext(r.Context())
	if traceID == "" {
		traceID = strings.TrimSpace(r.Header.Get(tracelog.Header))
	}

	// Stop existing service first.
	var needStop bool
	activeTask := trimTaskID(taskID)
	stateMu.Lock()
	needStop = state.Running
	state.Logs = nil
	state.Error = ""
	setActiveCorrelationLocked(corr)
	if activeTask != "" {
		state.ActiveTaskID = activeTask
		resetTaskLogsLocked(activeTask)
	}
	stateMu.Unlock()

	if needStop {
		stopRunning()
	}
	ensureOnlineServicePortFree()

	taskOrigin := extractOrigin(envIn["TASK_API_ENDPOINT_ORIGIN"])
	if taskOrigin == "" {
		taskOrigin = strings.TrimSpace(envIn["TASK_API_ENDPOINT_ORIGIN"])
	}

	// Copy env payload.
	envCopy := make(map[string]string)
	for k, v := range envIn {
		envCopy[k] = v
	}

	envCopy = tracelog.WithTraceEnv(envCopy, r.Header.Get(tracelog.Header))
	if ctxTid := tracelog.TraceIDFromContext(r.Context()); ctxTid != "" {
		envCopy = tracelog.WithTraceEnv(envCopy, ctxTid)
	}

	var result map[string]interface{}
	var err error
	if imageRef != "" {
		if installedImageID != "" {
			appendLog(fmt.Sprintf("[relayToTrae] start mode=selected_image installed_image_id=%s image=%s", installedImageID, imageRef))
		} else {
			appendLog(fmt.Sprintf("[relayToTrae] start mode=selected_image image=%s", imageRef))
		}
		result, err = startSelectedImageContainer(envCopy, tenantID, workspaceID, taskID, imageRef)
	} else {
		result, err = startOnlineService(envCopy, tenantID, workspaceID, taskID)
	}
	if err != nil {
		appendLog("[relayToTrae] start failed: " + err.Error())
		stateMu.Lock()
		// Prefer specific Error already set by startOnlineService (e.g. TOKEN_PERSIST_FAILED).
		if strings.TrimSpace(state.Error) == "" {
			state.Error = err.Error()
		}
		errMsg := state.Error
		stateMu.Unlock()
		resp := map[string]string{"status": "error", "message": errMsg}
		if strings.Contains(errMsg, "TOKEN_PERSIST_FAILED") {
			resp["error_code"] = "TOKEN_PERSIST_FAILED"
		}
		writeJSON(w, 400, resp)
		return
	}

	// Register with post-child-exchange token (synced from container_refresh_token.json).
	stateMu.Lock()
	exchangedToken := strings.TrimSpace(state.AccessToken)
	registerTaskLocked(tenantID, workspaceID, taskID, taskOrigin, exchangedToken, strings.TrimSpace(envCopy["COMMENT_ID"]))
	stateMu.Unlock()

	// 启动完成即同步推送一次状态（含 bootstrap 日志），替代 1.5s 轮询（OPT-20260816-032）。
	pushAllRegisteredTasks()

	resp := map[string]interface{}{"status": "ok"}
	for k, v := range result {
		resp[k] = v
	}
	writeJSON(w, 200, resp)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if !secretOK(r) {
		writeJSON(w, 401, map[string]string{"detail": "unauthorized"})
		return
	}

	stateMu.Lock()
	pending := state.TokenSyncPending
	stateMu.Unlock()
	if pending {
		// selected_image / child 换票完成前不接受 bootstrap 注册，避免 status-push 401。
		writeJSON(w, 202, map[string]string{
			"status":  "deferred",
			"message": "token exchange pending; register after start returns",
		})
		return
	}

	var body struct {
		TenantID         string `json:"tenant_id"`
		TenantId         string `json:"tenantId"`
		WorkspaceID      string `json:"workspace_id"`
		WorkspaceId      string `json:"workspaceId"`
		TaskID           string `json:"task_id"`
		TaskId           string `json:"taskId"`
		TaskAPIOrigin    string `json:"task_api_endpoint_origin"`
		TaskAPIOriginAlt string `json:"TASK_API_ENDPOINT_ORIGIN"`
		AccessToken      string `json:"access_token"`
		AccessTokenAlt   string `json:"ACCESS_TOKEN"`
		CommentID        string `json:"comment_id"`
		CommentIDAlt     string `json:"commentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body = struct {
			TenantID         string `json:"tenant_id"`
			TenantId         string `json:"tenantId"`
			WorkspaceID      string `json:"workspace_id"`
			WorkspaceId      string `json:"workspaceId"`
			TaskID           string `json:"task_id"`
			TaskId           string `json:"taskId"`
			TaskAPIOrigin    string `json:"task_api_endpoint_origin"`
			TaskAPIOriginAlt string `json:"TASK_API_ENDPOINT_ORIGIN"`
			AccessToken      string `json:"access_token"`
			AccessTokenAlt   string `json:"ACCESS_TOKEN"`
			CommentID        string `json:"comment_id"`
			CommentIDAlt     string `json:"commentId"`
		}{}
	}

	tenantID := body.TenantID
	if tenantID == "" {
		tenantID = body.TenantId
	}
	workspaceID := body.WorkspaceID
	if workspaceID == "" {
		workspaceID = body.WorkspaceId
	}
	taskID := body.TaskID
	if taskID == "" {
		taskID = body.TaskId
	}
	taskAPIOrigin := body.TaskAPIOrigin
	if taskAPIOrigin == "" {
		taskAPIOrigin = body.TaskAPIOriginAlt
	}
	accessToken := body.AccessToken
	if accessToken == "" {
		accessToken = body.AccessTokenAlt
	}
	commentID := body.CommentID
	if commentID == "" {
		commentID = body.CommentIDAlt
	}

	if taskID == "" {
		writeJSON(w, 400, map[string]string{"status": "error", "message": "missing task_id"})
		return
	}
	if taskAPIOrigin == "" {
		writeJSON(w, 400, map[string]string{"status": "error", "message": "missing task_api_endpoint_origin"})
		return
	}

	stateMu.Lock()
	resolvedToken := resolveAccessTokenForRegisterLocked(accessToken)
	if resolvedToken == "" {
		stateMu.Unlock()
		writeJSON(w, 200, map[string]string{"status": "deferred", "task_id": taskID})
		return
	}
	registerTaskLocked(tenantID, workspaceID, taskID, taskAPIOrigin, resolvedToken, commentID)
	stateMu.Unlock()

	// 注册即同步推送一次状态，替代 1.5s 轮询（OPT-20260816-032）。
	pushAllRegisteredTasks()

	writeJSON(w, 200, map[string]string{"status": "ok", "task_id": taskID})
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	if !secretOK(r) {
		writeJSON(w, 401, map[string]string{"detail": "unauthorized"})
		return
	}

	if !tryAcquireLifecycleLock() {
		log.Printf("[go_relayToTrae] event=lifecycle_lock_timeout op=stop wait=%s", lifecycleLockWait())
		writeJSON(w, 503, map[string]string{
			"detail": "relay lifecycle busy; retry later",
			"error":  "LIFECYCLE_BUSY",
		})
		return
	}
	defer lifecycleMu.Unlock()

	resetOK, killed := stopOnlineService()
	// 停机即同步推送一次终态，替代 1.5s 轮询（OPT-20260816-032）。
	pushAllRegisteredTasks()
	writeJSON(w, 200, map[string]interface{}{
		"status":      "ok",
		"reset_ok":    resetOK,
		"killed_pids": killed,
	})
}

func stopOnlineService() (resetOK bool, killed []string) {
	port := onlineServicePortForCleanup()

	accessToken := resolveAccessTokenForStop()
	resetOK = callOnlineServiceReset(port, accessToken)

	stopRunning()

	port = onlineServicePortForCleanup()
	if portListening(port) {
		killed = killPortListeners(port)
	}
	// Host-network orphans may still hold the port after stopRunning (invisible to lsof).
	if portListening(port) {
		stopOrphanRelayContainers()
	}

	stateMu.Lock()
	state.Error = ""
	stateMu.Unlock()

	appendLog("[relayToTrae] stopped")
	return resetOK, killed
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if !secretOK(r) {
		writeJSON(w, 401, map[string]string{"detail": "unauthorized"})
		return
	}

	cursorRaw := r.URL.Query().Get("cursor")
	cursor := 0
	if cursorRaw != "" {
		if c, err := strconv.Atoi(cursorRaw); err == nil && c > 0 {
			cursor = c
		}
	}
	viewerTaskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	if viewerTaskID == "" {
		viewerTaskID = strings.TrimSpace(r.URL.Query().Get("taskId"))
	}

	snapshot := collectStatusSnapshot(viewerTaskID, cursor)
	writeJSON(w, 200, snapshot)
}

func handleClearLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	if !secretOK(r) {
		writeJSON(w, 401, map[string]string{"detail": "unauthorized"})
		return
	}

	tenantID := strings.TrimSpace(r.PathValue("tenant_id"))
	workspaceID := strings.TrimSpace(r.PathValue("workspace_id"))
	taskID := trimTaskID(r.PathValue("task_id"))
	if tenantID == "" || workspaceID == "" || taskID == "" {
		missing := "task_id"
		if tenantID == "" {
			missing = "tenant_id"
		} else if workspaceID == "" {
			missing = "workspace_id"
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":  "error",
			"message": "missing " + missing,
		})
		return
	}

	stateMu.Lock()
	clearedTask, clearedLive, ok, mismatch := clearLogsForScopeLocked(tenantID, workspaceID, taskID)
	stateMu.Unlock()
	if mismatch {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"status":  "error",
			"message": "scope mismatch",
		})
		return
	}
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":  "error",
			"message": "missing tenant_id|workspace_id|task_id",
		})
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"status":       "ok",
		"cleared":      true,
		"tenant_id":    tenantID,
		"workspace_id": workspaceID,
		"task_id":      clearedTask,
		"cleared_live": clearedLive,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{
		"service": "go-relay",
		"ok":      true,
		"checks":  map[string]interface{}{},
	})
}

func setupRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/start", func(w http.ResponseWriter, r *http.Request) {
		handleStart(w, r)
	})
	mux.HandleFunc("/v1/register", func(w http.ResponseWriter, r *http.Request) {
		handleRegister(w, r)
	})
	mux.HandleFunc("/v1/stop", func(w http.ResponseWriter, r *http.Request) {
		handleStop(w, r)
	})
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, r *http.Request) {
		handleStatus(w, r)
	})
	mux.HandleFunc("POST /v1/tenant/{tenant_id}/workspace/{workspace_id}/task/{task_id}/clear-logs", handleClearLogs)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		handleHealth(w, r)
	})
	return tracelog.Middleware(corsMiddleware(mux))
}

func startServer(host string, port int) error {
	addr := host + ":" + strconv.Itoa(port)
	log.Printf("[relayToTrae] starting HTTP server on %s", addr)
	return tracelog.ListenAndServe(addr, setupRoutes())
}
