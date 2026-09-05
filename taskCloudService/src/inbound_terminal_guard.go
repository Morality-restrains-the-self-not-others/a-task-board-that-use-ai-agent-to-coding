package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// lookupTaskTerminalKindsFn is replaceable in tests.
var lookupTaskTerminalKindsFn = lookupTaskTerminalKindsHTTP

// —— 终态查找短 TTL 缓存（OPT-20260821-019）——
// 热路径（心跳/job-stream）每次同步都打 taskTaskService 会放大 task 抖动；
// 进程内按 task_id 缓存：终态（cancelled/completed）缓存更久，非终态/错误仅短缓存，
// 避免把空结果当成「非终态」长期缓存。

type terminalKindCacheEntry struct {
	kind      string
	expiresAt time.Time
}

var (
	terminalKindCacheMu sync.Mutex
	terminalKindCache   = map[string]terminalKindCacheEntry{}
)

const (
	terminalKindCacheTTLNormal   = 8 * time.Second
	terminalKindCacheTTLTerminal = 5 * time.Minute
)

func resetTerminalKindCache() {
	terminalKindCacheMu.Lock()
	terminalKindCache = map[string]terminalKindCacheEntry{}
	terminalKindCacheMu.Unlock()
}

// lookupTaskTerminalKindsCached 先查缓存，仅对未命中的 task_id 打上游，再把结果写回缓存。
// 终态结果缓存 5 分钟；非终态/错误只缓存 8 秒，错误 fail-open 不被当成长期非终态。
func lookupTaskTerminalKindsCached(taskIDs []string) (map[string]string, error) {
	clean := make([]string, 0, len(taskIDs))
	seen := map[string]bool{}
	for _, id := range taskIDs {
		tid := strings.TrimSpace(id)
		if tid == "" || seen[tid] {
			continue
		}
		seen[tid] = true
		clean = append(clean, tid)
	}
	if len(clean) == 0 {
		return map[string]string{}, nil
	}

	now := time.Now()
	result := map[string]string{}
	misses := make([]string, 0, len(clean))
	terminalKindCacheMu.Lock()
	for _, tid := range clean {
		if e, ok := terminalKindCache[tid]; ok && now.Before(e.expiresAt) {
			result[tid] = e.kind
		} else {
			misses = append(misses, tid)
		}
	}
	terminalKindCacheMu.Unlock()

	if len(misses) == 0 {
		return result, nil
	}

	kinds, err := lookupTaskTerminalKindsFn(misses)
	terminalKindCacheMu.Lock()
	if err != nil {
		// fail-open：错误不长期缓存，仅 8 秒短负缓存防止错误风暴
		for _, tid := range misses {
			terminalKindCache[tid] = terminalKindCacheEntry{kind: "", expiresAt: now.Add(terminalKindCacheTTLNormal)}
		}
		terminalKindCacheMu.Unlock()
		return nil, err
	}
	for _, tid := range misses {
		kind := strings.TrimSpace(kinds[tid])
		ttl := terminalKindCacheTTLNormal
		if isTaskTerminalKind(kind) {
			ttl = terminalKindCacheTTLTerminal
		}
		terminalKindCache[tid] = terminalKindCacheEntry{kind: kind, expiresAt: now.Add(ttl)}
		result[tid] = kind
	}
	terminalKindCacheMu.Unlock()
	return result, nil
}

func isTaskTerminalKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "cancelled", "canceled", "completed":
		return true
	default:
		return false
	}
}

// refuseInboundIfTaskTerminal returns true after writing HTTP 410 when the task
// is cancelled/completed. Leftover comment CSC terminal_released after a new
// instance bind must not fake-cancel an open task (boot-progress would 410
// then destroy the new VM). Empty-instance leftover flag still 410s old userdata.
func refuseInboundIfTaskTerminal(w http.ResponseWriter, r *http.Request, ctx context.Context, cfgRow *CloudServerConfig, tenantID, workspaceID, taskID, action string) bool {
	if strings.TrimSpace(action) == "request-machine-release" {
		return false
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}
	if ctx == nil && r != nil {
		ctx = r.Context()
	}

	flagSet := cfgRow != nil && cfgRow.TerminalReleasedFlag() == 1
	kind, lookupErr := lookupTaskTerminalKind(taskID)
	if lookupErr != nil {
		logWarn(fmt.Sprintf("event=terminal_kind_lookup_failed task_id=%s err=%v", taskID, lookupErr), taskID)
	}
	taskTerminal := isTaskTerminalKind(kind)
	live := cfgRow != nil && (strings.TrimSpace(cfgRow.InstanceID) != "" || strings.TrimSpace(cfgRow.ServerURL) != "")
	if !taskTerminal && !flagSet {
		return false
	}
	if !taskTerminal && flagSet && live {
		logInfo(fmt.Sprintf("event=inbound_stale_terminal_flag_ignored task_id=%s comment_id=%s instance_id=%s",
			taskID, strings.TrimSpace(cfgRow.CommentID), strings.TrimSpace(cfgRow.InstanceID)), taskID)
		return false
	}
	if kind == "" {
		kind = "cancelled"
	}

	alreadyReleased := flagSet && !live
	if !alreadyReleased && cfgRow != nil {
		if _, err := releaseMachineForTerminal(ctx, cfgRow, tenantID, workspaceID, taskID, kind, "inbound_task_terminal"); err != nil {
			logWarn(fmt.Sprintf("event=inbound_terminal_release_failed task_id=%s kind=%s err=%v", taskID, kind, err), taskID)
		} else {
			logInfo(fmt.Sprintf("event=inbound_terminal_release_ok task_id=%s kind=%s", taskID, kind), taskID)
		}
	}

	logInfo(fmt.Sprintf("event=inbound_blocked_task_terminal task_id=%s action=%s kind=%s terminal_released=%v",
		taskID, action, kind, flagSet), taskID)
	writeErrorMapJSON(w, r, http.StatusGone, map[string]interface{}{
		"detail":  "任务已处于终端状态（已取消/已完成），拒绝容器入站",
		"code":    "TASK_TERMINAL",
		"task_id": taskID,
	})
	return true
}

func lookupTaskTerminalKind(taskID string) (string, error) {
	kinds, err := lookupTaskTerminalKindsCached([]string{taskID})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(kinds[taskID]), nil
}

// lookupTaskTerminalKindsHTTP asks taskTaskService POST /api/internal/tasks/terminal-kinds/.
// Empty TaskServiceURL or transport errors fail-open (empty map) so live heartbeats
// are not 410'd when task service is briefly unreachable.
func lookupTaskTerminalKindsHTTP(taskIDs []string) (map[string]string, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return map[string]string{}, nil
	}
	clean := make([]string, 0, len(taskIDs))
	seen := map[string]bool{}
	for _, id := range taskIDs {
		tid := strings.TrimSpace(id)
		if tid == "" || seen[tid] {
			continue
		}
		seen[tid] = true
		clean = append(clean, tid)
	}
	if len(clean) == 0 {
		return map[string]string{}, nil
	}
	payload, err := json.Marshal(map[string]interface{}{"task_ids": clean})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/tasks/terminal-kinds/", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if v := strings.TrimSpace(cfg.InternalSecret); v != "" {
		req.Header.Set("X-Internal-Secret", v)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("terminal-kinds status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var body struct {
		Kinds map[string]string `json:"kinds"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if body.Kinds == nil {
		return map[string]string{}, nil
	}
	return body.Kinds, nil
}
