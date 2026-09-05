package main

// 约束 46 对称 UI：runAll 页头「推送 trae-agent 镜像」按钮。
//
// 运维侧与「精准编译重启」对称的一键入口：读取 .runall 下 trae-agent 镜像推送
// 状态（pending 登记 / 水位线 / 运行锁 / 日志尾部），并异步触发
// `scripts/trae-agent-docker-push.sh --if-pending --background`。
//
// 状态文件与 scripts/lib/trae-agent-docker-push.sh 保持一致：
//   - trae_agent_docker_push_pending   待推送登记（存在=有未推送镜像变更）
//   - trae_agent_docker_push_sha       最近一次成功推送的 trae-agent HEAD
//   - trae_agent_docker_push.lock      推送运行锁（存在=推送进行中）
//   - trae_agent_docker_push.log       推送输出日志（SSE 进度来源）
//
// API：
//   - GET  /api/trae-agent-push/status    读取状态
//   - POST /api/trae-agent-push           触发推送（异步，202 accepted）
//   - GET  /api/trae-agent-push/progress  SSE 进度（tail 日志文件）

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// registerTraeAgentPushHandlers 注册约束 46 对称 UI（「推送 trae-agent 镜像」按钮）的 API：
//   - GET  /api/trae-agent-push/status    读取状态
//   - POST /api/trae-agent-push           触发推送（异步，202 accepted）
//   - GET  /api/trae-agent-push/progress  SSE 进度（tail 日志文件）
func registerTraeAgentPushHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/trae-agent-push/status", func(w http.ResponseWriter, r *http.Request) {
		handleTraeAgentPushStatus(w, r, runner)
	})
	mux.HandleFunc("/api/trae-agent-push", func(w http.ResponseWriter, r *http.Request) {
		handleTraeAgentPushAction(w, r, runner)
	})
	mux.HandleFunc("/api/trae-agent-push/progress", func(w http.ResponseWriter, r *http.Request) {
		handleTraeAgentPushProgressSSE(w, r, runner)
	})
}

// traeAgentPushStateDir 返回 .runall 状态目录。优先尊重脚本的环境变量覆盖，便于测试隔离。
func traeAgentPushStateDir(runner *Runner) (string, error) {
	if env := strings.TrimSpace(os.Getenv("TRAE_AGENT_DOCKER_PUSH_STATE_DIR")); env != "" {
		return env, nil
	}
	if runner == nil {
		return "", fmt.Errorf("runner required")
	}
	root, err := runner.monorepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".runall"), nil
}

// traeAgentPushScriptPath 返回推送脚本绝对路径。支持环境变量覆盖（测试注入 stub）。
func traeAgentPushScriptPath(runner *Runner) (string, error) {
	if env := strings.TrimSpace(os.Getenv("RUNALL_TRAE_AGENT_PUSH_SCRIPT")); env != "" {
		return env, nil
	}
	if runner == nil {
		return "", fmt.Errorf("runner required")
	}
	root, err := runner.monorepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "scripts", "trae-agent-docker-push.sh"), nil
}

// traeAgentPushLockActive 判断推送运行锁是否仍代表一个存活进程。
// 锁文件存在且 PID 存活 → running；陈旧锁（进程已死）按未运行处理，避免 UI 卡死。
func traeAgentPushLockActive(stateDir string) bool {
	lock := filepath.Join(stateDir, "trae_agent_docker_push.lock")
	raw, err := os.ReadFile(lock)
	if err != nil {
		return false
	}
	pidStr := strings.TrimSpace(string(raw))
	if pidStr == "" {
		return true
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return true
	}
	// kill(pid, 0) 探测进程存活
	if processAlive(pid) {
		return true
	}
	return false
}

func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// traeAgentPushStatusView 是 /api/trae-agent-push/status 的响应体。
type traeAgentPushStatusView struct {
	Pending    bool     `json:"pending"`
	Running    bool     `json:"running"`
	LastSHA    string   `json:"last_sha"`
	LogTail    []string `json:"log_tail"`
	StateDir   string   `json:"state_dir"`
	ScriptPath string   `json:"script_path"`
}

// handleTraeAgentPushStatus 返回当前推送状态：是否待推送、是否运行中、水位线、日志尾部。
func handleTraeAgentPushStatus(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodGet {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	stateDir, err := traeAgentPushStateDir(runner)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	scriptPath, err := traeAgentPushScriptPath(runner)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	view := traeAgentPushStatusView{
		Pending:    fileExists(filepath.Join(stateDir, "trae_agent_docker_push_pending")),
		Running:    traeAgentPushLockActive(stateDir),
		LastSHA:    readFirstLine(filepath.Join(stateDir, "trae_agent_docker_push_sha")),
		LogTail:    readLogTail(filepath.Join(stateDir, "trae_agent_docker_push.log"), 40),
		StateDir:   stateDir,
		ScriptPath: scriptPath,
	}
	writeJSON(w, view)
}

// handleTraeAgentPushAction 触发一次 trae-agent 镜像推送：
// 校验运行锁（409 冲突）与 pending 登记（400 无待推送），随后异步执行推送脚本并立即 202。
func handleTraeAgentPushAction(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if r.Method != http.MethodPost {
		writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runner == nil {
		writeJSONErrorWithStatus(w, http.StatusServiceUnavailable, "runner not available")
		return
	}
	stateDir, err := traeAgentPushStateDir(runner)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	if traeAgentPushLockActive(stateDir) {
		writeJSONErrorWithStatus(w, http.StatusConflict, "trae-agent 镜像推送已在进行中")
		return
	}
	pendingFile := filepath.Join(stateDir, "trae_agent_docker_push_pending")
	if !fileExists(pendingFile) {
		writeJSONErrorWithStatus(w, http.StatusBadRequest,
			"无待推送变更（.runall/trae_agent_docker_push_pending 不存在）。"+
				"请先提交影响 onlineServiceJS 镜像的 trae-agent 变更，或检查 Stop/SessionEnd 扫描登记。")
		return
	}
	scriptPath, err := traeAgentPushScriptPath(runner)
	if err != nil {
		writeJSONError(w, err.Error())
		return
	}
	if !fileExists(scriptPath) {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError, "trae-agent 推送脚本不存在: "+scriptPath)
		return
	}

	// 脚本自身支持 --background（nohup + 锁 + 日志追加），父进程立即返回。
	cmd := exec.Command(getBashPath(), scriptPath, "--if-pending", "--background")
	cmd.Env = append(os.Environ(), "TRAE_AGENT_DOCKER_PUSH_STATE_DIR="+stateDir)
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		writeJSONErrorWithStatus(w, http.StatusInternalServerError,
			fmt.Sprintf("触发推送失败: %v: %s", runErr, strings.TrimSpace(string(out))))
		return
	}
	log.Printf("[api] trae-agent-push accepted: %s", strings.TrimSpace(string(out)))
	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]string{"status": "accepted"})
}

// handleTraeAgentPushProgressSSE 以 SSE 流式输出推送日志（tail 日志文件增量）。
// 结束条件：运行锁消失且 pending 已清空（推送完成）；超过 maxLifetime 兜底返回。
func handleTraeAgentPushProgressSSE(w http.ResponseWriter, r *http.Request, runner *Runner) {
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
	stateDir, err := traeAgentPushStateDir(runner)
	if err != nil {
		fmt.Fprintf(w, "data: {\"phase\":\"error\",\"error\":%q}\n\n", err.Error())
		flusher.Flush()
		return
	}
	logPath := filepath.Join(stateDir, "trae_agent_docker_push.log")
	pendingPath := filepath.Join(stateDir, "trae_agent_docker_push_pending")
	lockPath := filepath.Join(stateDir, "trae_agent_docker_push.lock")

	var offset int64
	started := time.Now()
	const maxLifetime = 5 * time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if chunk, next, rerr := readLogFromOffset(logPath, offset); rerr == nil && chunk != "" {
			fmt.Fprintf(w, "data: {\"phase\":\"progress\",\"log\":%q}\n\n", chunk)
			flusher.Flush()
			offset = next
		}
		running := fileExists(lockPath)
		done := !running && !fileExists(pendingPath)
		if done {
			fmt.Fprintf(w, "data: {\"phase\":\"idle\",\"done\":true}\n\n")
			flusher.Flush()
			return
		}
		if time.Since(started) > maxLifetime {
			fmt.Fprintf(w, "data: {\"phase\":\"timeout\",\"done\":true}\n\n")
			flusher.Flush()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// readLogFromOffset 从指定偏移读取日志新增内容，返回 (内容, 新偏移, err)。
func readLogFromOffset(path string, offset int64) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", offset, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", offset, err
	}
	if st.Size() <= offset {
		return "", offset, nil
	}
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return "", offset, err
	}
	// 单次轮询最多读 64KB，避免大日志一次性打爆浏览器
	toRead := st.Size() - offset
	if toRead > 64*1024 {
		toRead = 64 * 1024
	}
	buf := make([]byte, toRead)
	n, rerr := io.ReadFull(f, buf)
	if rerr != nil && rerr != io.EOF && rerr != io.ErrUnexpectedEOF {
		return "", offset, rerr
	}
	return string(buf[:n]), offset + int64(n), nil
}

// readFirstLine 读取文件首行（去空白）。
func readFirstLine(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.SplitN(strings.TrimSpace(string(raw)), "\n", 2)
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}

// readLogTail 读取日志最后 maxLines 行。
func readLogTail(path string, maxLines int) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	body := strings.TrimRight(string(raw), "\n")
	if body == "" {
		return nil
	}
	lines := strings.Split(body, "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return lines
}
