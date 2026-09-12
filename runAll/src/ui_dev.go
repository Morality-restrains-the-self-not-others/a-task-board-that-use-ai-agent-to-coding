package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"runAll/src/domain"
)

func registerDevHandlers(mux *http.ServeMux, runner *Runner) {
	registerDevDatabaseHandler(mux, runner, "/api/dev/clear-databases", "clear-db", domain.ToolDbClear, "CLEAR_ALL", func(ctx context.Context) any {
		return runner.ClearAllDatabases(ctx, "")
	})
	registerDevDatabaseHandler(mux, runner, "/api/dev/init-databases", "init-db", domain.ToolDbInit, "INIT_ALL", func(ctx context.Context) any {
		return runner.InitAllDatabases(ctx, "")
	})
	mux.HandleFunc("/api/dev/bootstrap-admin-email", func(w http.ResponseWriter, r *http.Request) {
		handleBootstrapAdminEmail(w, r, runner)
	})
	mux.HandleFunc("/api/dev/logs", func(w http.ResponseWriter, r *http.Request) {
		handleDevToolLogs(w, r, runner)
	})
	mux.HandleFunc("/api/dev/logs/clear", func(w http.ResponseWriter, r *http.Request) {
		handleDevToolLogsClear(w, r, runner)
	})
}

// devDBBeforeTerminalEventHook, when set (tests only), is invoked inside the dev
// clear/init background goroutine immediately after the dev db reset lock has
// been released and immediately before the terminal progress event is published.
// It gives tests a deterministic point to assert the lock is already re-entrant
// by the time a subscriber can observe Done (regression for the ordering bug
// where the lock was only released at goroutine exit, i.e. after PublishProgress).
var devDBBeforeTerminalEventHook func(runner *Runner)

func registerDevDatabaseHandler(
	mux *http.ServeMux,
	runner *Runner,
	path string,
	op string, // progress Operation label, e.g. "clear-db"/"init-db"
	tool string, // dev tool name for status file, e.g. domain.ToolDbClear
	confirmToken string,
	run func(context.Context) any,
) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		if !allowDevDatabaseReset(r) {
			writeJSONErrorWithStatus(w, http.StatusForbidden, "dev database operation not allowed on this host")
			return
		}
		if strings.TrimSpace(r.URL.Query().Get("confirm")) != confirmToken {
			writeJSONErrorWithStatus(w, http.StatusBadRequest, "confirm="+confirmToken+" is required")
			return
		}
		if !runner.TryAcquireDevDatabaseReset() {
			// OPT-20260820-007: 409 也带回 active_bulk_progress，前端可挂上进度条/SSE
			// 而不只是显示「已在进行中」横幅。
			writeJSONConflictWithBulkProgress(w, runner, "database operation already in progress")
			return
		}

		// OPT-20260812-049: clear/init 可能耗时数分钟，同步长阻塞 handler 在代理/浏览器
		// 超时断开时脆弱；进程 OOM 退出会直接中断 HTTP 连接（OPT-20260813-002）。
		// 改为 HTTP 202 立即 accepted + 后台 goroutine 执行，进度/结果走
		// /api/progress?run_id=... SSE（复用 generic ProgressBroadcaster）。
		runID := runner.GenerateSingleRunID(op, "dev-db")
		runner.SetActiveDevDBRun(op, runID)
		// 开始标记写入 .runall/db-<op>.last_status：若进程中途 OOM/panic，该文件停留在
		// running，作为诊断线索（OPT-20260813-002 #4）。
		runner.RecordDevDBLastStatus(tool, "running", "")
		runner.PublishProgress(runID, StartAllProgressEvent{
			Total: 1, Started: 0, Remaining: 1,
			Phase: "starting", Operation: op, Current: "开始执行...",
		})
		// 后台执行，不绑定请求 context：客户端/代理超时断开不得取消中途清库/初始化。
		reqCtx := r.Context()
		go func() {
			// 互斥锁须在终态事件发布前释放（且全程只释放一次）。run() 返回即代表
			// 清库/初始化主体已结束，此刻就应让新一轮操作可重入；若像旧实现那样把
			// ReleaseDevDatabaseReset 作为 defer 拖到 goroutine 末尾，实际会发生在
			// PublishProgress(done) 之后——订阅者观察到 Done 后立刻发起新一轮
			// clear/init 仍会误撞 409（panic 路径因 defer LIFO 恰好先释放，而
			// 正常/blocked 路径不一致）。releaseResetLock 幂等：panic 兜底（下方
			// recover defer）不会在显式释放后再释放一次，避免误清新一轮已获取的锁。
			var released bool
			releaseResetLock := func() {
				if !released {
					released = true
					runner.ReleaseDevDatabaseReset()
				}
			}
			// panic 不得击穿 runAll 主进程（OPT-20260813-002 #3）
			defer func() {
				if rec := recover(); rec != nil {
					msg := fmt.Sprintf("dev %s panic: %v", op, rec)
					log.Printf("[ui] %s", msg)
					releaseResetLock()
					runner.RecordDevDBLastStatus(tool, "panic", msg)
					runner.PublishProgress(runID, StartAllProgressEvent{
						Total: 1, Failed: 1, Remaining: 0, Done: true,
						Error: msg, Errors: []string{msg},
						Phase: "error", Operation: op, Current: msg,
					})
					runner.CloseProgressRun(runID, 30*time.Second)
				}
			}()
			ctx := context.WithoutCancel(reqCtx)
			payload := run(ctx)
			releaseResetLock()
			if devDBBeforeTerminalEventHook != nil {
				devDBBeforeTerminalEventHook(runner)
			}
			if initResult, ok := payload.(domain.DatabasePlatformInitResult); ok && initResult.Status == "blocked" {
				msg := "stop running services before init: " + strings.Join(initResult.BlockedServices, ", ")
				runner.RecordDevDBLastStatus(tool, "blocked", msg)
				runner.PublishProgress(runID, StartAllProgressEvent{
					Total: 1, Failed: 1, Remaining: 0, Done: true,
					Error: msg, Errors: []string{msg},
					Phase: "error", Operation: op, Current: msg,
					Detail: initResult,
				})
				runner.CloseProgressRun(runID, 30*time.Second)
				return
			}
			status := devDBResultStatus(payload)
			doneEv := StartAllProgressEvent{
				Total: 1, Started: 1, Remaining: 0, Done: true,
				Phase: "done", Operation: op, Detail: payload,
			}
			if status != "ok" && status != "completed" {
				msg := fmt.Sprintf("dev %s status=%s", op, status)
				doneEv.Failed = 1
				doneEv.Error = msg
				doneEv.Errors = []string{msg}
				doneEv.Phase = "error"
				doneEv.Current = msg
			}
			runner.RecordDevDBLastStatus(tool, status, "")
			runner.PublishProgress(runID, doneEv)
			runner.CloseProgressRun(runID, 30*time.Second)
		}()

		w.WriteHeader(http.StatusAccepted)
		writeJSON(w, map[string]string{"status": "accepted", "run_id": runID})
	})
}

// devDBResultStatus extracts the Status field from either dev database result type.
func devDBResultStatus(payload any) string {
	switch v := payload.(type) {
	case domain.DatabasePlatformClearResult:
		return v.Status
	case domain.DatabasePlatformInitResult:
		return v.Status
	default:
		return ""
	}
}

func allowDevDatabaseReset(r *http.Request) bool {
	if strings.TrimSpace(os.Getenv("RUNALL_ALLOW_DEV_DB_RESET")) == "1" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	if host == "127.0.0.1" || host == "localhost" || host == "" || host == "::1" {
		return true
	}
	return isPrivateNetworkIP(host)
}

// isPrivateNetworkIP checks whether the given IP address belongs to RFC 1918
// private network ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16).
func isPrivateNetworkIP(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 10 ||
		(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
		(ip4[0] == 192 && ip4[1] == 168)
}

func handleDevToolLogs(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodGet {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil || runner.devToolLogRecorder == nil {
		writeJSONError(w, "dev tool log recorder is required")
		return
	}

	tool := strings.TrimSpace(r.URL.Query().Get("tool"))
	if tool == "" {
		writeJSONError(w, "tool is required")
		return
	}
	if !domain.IsValidDevTool(tool) {
		writeJSONError(w, fmt.Sprintf("unknown tool: %q", tool))
		return
	}

	linesRaw := r.URL.Query().Get("lines")
	if linesRaw == "" {
		writeJSONError(w, "lines is required")
		return
	}
	lines, err := strconv.Atoi(linesRaw)
	if err != nil || lines <= 0 {
		writeJSONError(w, "lines must be a positive integer")
		return
	}
	if lines > maxLogsLines {
		writeJSONError(w, fmt.Sprintf("lines must be <= %d", maxLogsLines))
		return
	}

	logLines, err := runner.devToolLogRecorder.Tail(tool, lines)
	if err != nil {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, fmt.Sprintf("failed to read logs: %v", err))
		return
	}

	writeJSON(w, map[string]any{
		"tool":  tool,
		"lines": logLines,
	})
}

func handleDevToolLogsClear(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil || runner.devToolLogRecorder == nil {
		writeJSONError(w, "dev tool log recorder is required")
		return
	}

	var body struct {
		Tool string `json:"tool"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid json")
		return
	}
	body.Tool = strings.TrimSpace(body.Tool)
	if body.Tool == "" {
		writeJSONError(w, "tool is required")
		return
	}

	var err error
	if body.Tool == "all" {
		err = runner.devToolLogRecorder.ClearAll()
	} else {
		if !domain.IsValidDevTool(body.Tool) {
			writeJSONError(w, fmt.Sprintf("unknown tool: %q", body.Tool))
			return
		}
		err = runner.devToolLogRecorder.Clear(body.Tool)
	}
	if err != nil {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, fmt.Sprintf("failed to clear logs: %v", err))
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

// logCollectionEntry is the JSON-serializable log collection status for a single service.
