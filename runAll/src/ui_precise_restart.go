package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// registerPreciseRestartHandlers 注册精准编译重启相关 API：
//   - GET  /api/precise-restart/registrations   读取登记列表
//   - POST /api/precise-restart/register        追加登记（校验服务名）
//   - POST /api/precise-restart                 触发：编译+重启登记的服务，完成后清空文件
//   - POST /api/precise-restart/cancel          中断当前精准编译重启
//   - GET  /api/precise-restart/progress        SSE 进度
func registerPreciseRestartHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/precise-restart/registrations", func(w http.ResponseWriter, r *http.Request) {
		handlePreciseRestartRegistrations(w, r, runner)
	})
	mux.HandleFunc("/api/precise-restart/register", func(w http.ResponseWriter, r *http.Request) {
		handlePreciseRestartRegister(w, r, runner)
	})
	mux.HandleFunc("/api/precise-restart", func(w http.ResponseWriter, r *http.Request) {
		handlePreciseRestartAction(w, r, runner)
	})
	mux.HandleFunc("/api/precise-restart/cancel", func(w http.ResponseWriter, r *http.Request) {
		handlePreciseRestartCancelAction(w, r, runner)
	})
	mux.HandleFunc("/api/precise-restart/progress", func(w http.ResponseWriter, r *http.Request) {
		handlePreciseRestartProgressSSE(w, r, runner)
	})
}

func preciseRestartFilePath(runner *Runner) string {
	if runner == nil {
		return preciseRestartFile("")
	}
	return preciseRestartFile(runner.cfgPath)
}

// preciseRestartRegistrationView 登记条目的 API 视图（OPT-20260807-024/025/026）：
// UI 需要区分 pending/failed、识别不可编译服务（直接重启）、判断是否超过 TTL 过期。
type preciseRestartRegistrationView struct {
	Name         string `json:"name"`
	State        string `json:"state"`
	RegisteredAt int64  `json:"registered_at"`
	Expired      bool   `json:"expired"`
	Resolvable   bool   `json:"resolvable"`
	Buildable    bool   `json:"buildable"`
}

// handlePreciseRestartRegistrations 返回当前登记的服务列表与文件路径。
// OPT-20260807-024/026：附带回退字段 entries（name/state/registered_at/expired/buildable），
// 供 UI 区分失败保留项、展示登记时间并对过期登记置灰；services 保持旧接口兼容。
func handlePreciseRestartRegistrations(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodGet {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := preciseRestartFilePath(runner)
	cfgPath := ""
	if runner != nil {
		cfgPath = runner.cfgPath
	}
	if r.URL.Query().Get("fill_from_scan") == "1" {
		peek, rerr := readRegistrationEntries(path)
		if rerr == nil && len(peek) == 0 {
			fillPreciseRestartFromScanFn(cfgPath)
		}
	}
	entries, err := readRegistrationEntries(path)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	now := time.Now()
	// 存在 unresolved 条目时做一次磁盘重载（仅当有 unresolved 才触发，避免每次 2s 轮询都 LoadConfig），
	// 否则 UI 会把进程启动后才写入 YAML 的可热加载新服务显示成「未知服务」（OPT-20260816-046）。
	if runner != nil {
		hasUnresolved := false
		for _, e := range entries {
			if len(runner.resolveRegisteredServices(e.Name)) == 0 {
				hasUnresolved = true
				break
			}
		}
		if hasUnresolved {
			added, rerr := runner.reloadConfigFromDisk()
			logPreciseRestartReload(added, rerr)
		}
	}
	services := make([]string, 0, len(entries))
	views := make([]preciseRestartRegistrationView, 0, len(entries))
	for _, e := range entries {
		services = append(services, e.Name)
		view := preciseRestartRegistrationView{
			Name:         e.Name,
			State:        string(e.State),
			RegisteredAt: e.RegisteredAt,
			Expired:      registrationExpired(e, now),
		}
		if runner != nil {
			svc := runner.resolveRegisteredService(e.Name)
			view.Resolvable = svc != nil
			view.Buildable = serviceBuildable(svc)
		}
		views = append(views, view)
	}
	writeJSON(w, map[string]any{
		"entries":     views,
		"services":    services,
		"file_path":   path,
		"source_root": resolveSourceRoot(cfgPath),
		"ttl_hours":   int(preciseRestartTTL / time.Hour),
	})
}

// handlePreciseRestartRegister 追加登记（body: {"services": ["task-auth", ...]}），
// 服务名必须能在 runAll 配置中解析（runAll.yaml name / conf_app / 工作目录名）。
func handlePreciseRestartRegister(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	payload, err := readJSONPayload(r)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	raw, ok := payload["services"]
	if !ok {
		writeJSONError(w, "services field is required")
		return
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		writeJSONError(w, "services must be a non-empty array")
		return
	}
	if runner == nil {
		writeJSONError(w, "runner is required")
		return
	}
	var names []string
	var unknown []string
	seen := make(map[string]bool)
	for _, it := range items {
		name := strings.TrimSpace(fmt.Sprintf("%v", it))
		if name == "" {
			continue
		}
		svcs := runner.resolveRegisteredServices(name)
		if len(svcs) == 0 {
			unknown = append(unknown, name)
			continue
		}
		// 别名归一化：taskAuth / task-auth / conf_app 均落盘为服务 name；
		// taskEvents 等共享 working_dir 别名展开为全部 task-events-*。
		for _, svc := range svcs {
			if seen[svc.Name] {
				continue
			}
			seen[svc.Name] = true
			names = append(names, svc.Name)
		}
	}
	if len(unknown) > 0 {
		writeJSONError(w, fmt.Sprintf("unknown services (not in runAll config): %s", strings.Join(unknown, ", ")))
		return
	}
	if len(names) == 0 {
		writeJSONError(w, "no valid service names provided")
		return
	}
	merged, err := appendRegisteredServices(preciseRestartFilePath(runner), names)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	log.Printf("[api] precise-restart register: +%v -> %d registered", names, len(merged))
	writeJSON(w, map[string]any{"status": "ok", "services": merged})
}

// handlePreciseRestartAction 触发精准编译重启：读取登记 → 异步执行 → 202 accepted。
func handlePreciseRestartAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil {
		writeJSONError(w, "runner is required")
		return
	}
	payload, err := readJSONPayload(r)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	sessionID, _ := payload["session_id"].(string)
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = defaultOwnershipSessionID
	}

	path := preciseRestartFilePath(runner)
	registrations, err := readRegisteredServices(path)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	if len(registrations) == 0 {
		log.Printf("[api] precise-restart empty registry (no-op): %s", path)
		writeJSON(w, map[string]any{
			"status":    "empty",
			"file_path": path,
			"message":   "no registered services",
		})
		return
	}

	runID := fmt.Sprintf("precise-restart-%d", time.Now().UnixNano())
	if !runner.TryBeginPreciseRestart(runID) {
		writeJSONConflictWithBulkProgress(w, runner, "precise-restart already in progress")
		return
	}
	runCancellableLifecycleActionAsync("precise-restart", func(ctx context.Context) error {
		defer runner.endPreciseRestart()
		_, err := runner.PreciseRestart(ctx, sessionID)
		return err
	}, "precise-restart", strings.Join(registrations, ", "))
	log.Printf("[api] precise-restart accepted for: %v", registrations)
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
}

func handlePreciseRestartCancelAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if cancelLifecycleAction("precise-restart") {
		writeJSON(w, map[string]string{"status": "cancelled"})
		return
	}
	// Hot-replace orphan: active flag restored/left without a cancel handle.
	if runner != nil && runner.ClearOrphanedBulkProgress("precise-restart") {
		writeJSON(w, map[string]string{"status": "cleared"})
		return
	}
	writeJSONErrorWithStatus(w, http.StatusNotFound, "no active precise-restart operation")
}

// handlePreciseRestartProgressSSE 流式输出精准编译重启进度（单阶段事件流）。
func handlePreciseRestartProgressSSE(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if runner == nil {
		writeJSONErrorWithStatus(w, http.StatusServiceUnavailable, "runner not available")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := r.Context()

	// 等待 runID 出现（按钮 POST 先返回 202，SSE 后连接；或重启后晚订阅）。
	for i := 0; i < 40; i++ {
		runID := runner.GetActivePreciseRestartRunID()
		if runID != "" && runner.GetProgressBroadcaster() != nil {
			ch := runner.GetProgressBroadcaster().Subscribe(runID)
			defer runner.GetProgressBroadcaster().Unsubscribe(runID, ch)
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-ch:
					if !ok {
						return
					}
					fmt.Fprint(w, ev.ToSSE())
					flusher.Flush()
					if ev.Done {
						return
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(250 * time.Millisecond):
		}
	}
	fmt.Fprintf(w, "data: {\"phase\":\"idle\",\"done\":true,\"operation\":\"restart\"}\n\n")
	flusher.Flush()
}
