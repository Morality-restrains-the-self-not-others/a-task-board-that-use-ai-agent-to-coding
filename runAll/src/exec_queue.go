package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const execQueueFileName = "exec_queue.json"

const maxExecQueuePending = 16

// ExecQueuePendingItem is a bulk op waiting behind the in-flight one.
// Stored in-process so a page refresh can restore the queue bar.
type ExecQueuePendingItem struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// ExecutionQueueSnapshot is served on GET /api/status for UI resume.
type ExecutionQueueSnapshot struct {
	Current *ActiveBulkProgressSnapshot `json:"current,omitempty"`
	Pending []ExecQueuePendingItem      `json:"pending"`
}

type execQueueState struct {
	mu         sync.Mutex
	pending    []ExecQueuePendingItem
	devDBOp    string
	devDBRunID string
}

var knownExecQueueTypes = map[string]struct{}{
	"start-all": {}, "stop-all": {}, "restart-all": {}, "build-all": {},
	"precise-restart": {}, "clear-db": {}, "init-db": {},
}

func bulkOpLabel(kind string) string {
	switch kind {
	case "precise-restart":
		return "精准编译重启"
	case "restart-all":
		return "全部重启"
	case "build-all":
		return "全部重新编译"
	case "stop-all":
		return "全部关闭"
	case "start-all":
		return "全部启动"
	case "clear-db":
		return "清空全部数据库"
	case "init-db":
		return "初始化全部数据库"
	default:
		return kind
	}
}

func (r *Runner) SetExecQueuePending(items []ExecQueuePendingItem) {
	r.replaceExecQueuePending(items)
	r.persistExecQueueFile()
}

func (r *Runner) replaceExecQueuePending(items []ExecQueuePendingItem) {
	if r == nil {
		return
	}
	copied := make([]ExecQueuePendingItem, len(items))
	copy(copied, items)
	r.execQueue.mu.Lock()
	r.execQueue.pending = copied
	r.execQueue.mu.Unlock()
}

func (r *Runner) ExecQueuePending() []ExecQueuePendingItem {
	if r == nil {
		return nil
	}
	r.execQueue.mu.Lock()
	defer r.execQueue.mu.Unlock()
	out := make([]ExecQueuePendingItem, len(r.execQueue.pending))
	copy(out, r.execQueue.pending)
	return out
}

// HasPendingExecQueueType reports whether the given bulk op type already sits in
// the pending queue (OPT-20260819-040: 确认弹窗未关闭 / 多标签时二次入队拦截)。
func (r *Runner) HasPendingExecQueueType(kind string) bool {
	if r == nil || kind == "" {
		return false
	}
	r.execQueue.mu.Lock()
	defer r.execQueue.mu.Unlock()
	for _, item := range r.execQueue.pending {
		if item.Type == kind {
			return true
		}
	}
	return false
}

// ConsumePendingExecQueueType removes the first pending item of kind.
// Call after a bulk POST actually starts, so page refresh cannot replay it.
func (r *Runner) ConsumePendingExecQueueType(kind string) bool {
	if r == nil || kind == "" {
		return false
	}
	r.execQueue.mu.Lock()
	removed := false
	kept := make([]ExecQueuePendingItem, 0, len(r.execQueue.pending))
	for _, item := range r.execQueue.pending {
		if !removed && item.Type == kind {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if removed {
		r.execQueue.pending = kept
	}
	r.execQueue.mu.Unlock()
	if removed {
		log.Printf("[runall] exec-queue: consumed pending type=%s", kind)
		r.persistExecQueueFile()
	}
	return removed
}

func (r *Runner) SetActiveDevDBRun(op, runID string) {
	if r == nil {
		return
	}
	r.execQueue.mu.Lock()
	r.execQueue.devDBOp = op
	r.execQueue.devDBRunID = runID
	r.execQueue.mu.Unlock()
	log.Printf("[runall] exec-queue: dev-db op=%s run_id=%s started", op, runID)
	r.persistExecQueueFile()
}

func (r *Runner) GetActiveDevDBRun() (op, runID string) {
	if r == nil {
		return "", ""
	}
	r.execQueue.mu.Lock()
	defer r.execQueue.mu.Unlock()
	return r.execQueue.devDBOp, r.execQueue.devDBRunID
}

func (r *Runner) ClearActiveDevDBRun() {
	if r == nil {
		return
	}
	r.execQueue.mu.Lock()
	r.execQueue.devDBOp = ""
	r.execQueue.devDBRunID = ""
	r.execQueue.mu.Unlock()
	r.persistExecQueueFile()
}

func (r *Runner) executionQueueSnapshot() ExecutionQueueSnapshot {
	snap := ExecutionQueueSnapshot{Pending: []ExecQueuePendingItem{}}
	if r == nil {
		return snap
	}
	snap.Current = r.ActiveBulkProgress()
	snap.Pending = r.ExecQueuePending()
	if snap.Pending == nil {
		snap.Pending = []ExecQueuePendingItem{}
	}
	return snap
}

// ExecutionQueue returns the snapshot the status UI hydrates after refresh.
func (r *Runner) ExecutionQueue() ExecutionQueueSnapshot {
	snap := r.executionQueueSnapshot()
	r.persistExecQueueFile()
	return snap
}

func (r *Runner) execQueueFilePath() string {
	if regPath := os.Getenv("RUNALL_PRECISE_RESTART_FILE"); regPath != "" {
		return filepath.Join(filepath.Dir(regPath), execQueueFileName)
	}
	// Require cfgPath so unit tests without SetConfigPath cannot pollute the
	// real <repo>/.runall via monorepoRoot walking cwd.
	if r != nil && strings.TrimSpace(r.cfgPath) != "" && r.cfg != nil {
		if root, err := r.monorepoRoot(); err == nil && root != "" {
			return filepath.Join(root, ".runall", execQueueFileName)
		}
	}
	return filepath.Join(".runall", execQueueFileName)
}

func (r *Runner) persistExecQueueFile() {
	if r == nil {
		return
	}
	path := r.execQueueFilePath()
	if !filepath.IsAbs(path) {
		return
	}
	snap := r.executionQueueSnapshot()
	if snap.Current == nil && len(snap.Pending) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("[runall] exec-queue: remove %s: %v", path, err)
		}
		return
	}
	data, err := json.Marshal(snap)
	if err != nil {
		log.Printf("[runall] exec-queue: marshal: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("[runall] exec-queue: mkdir %s: %v", filepath.Dir(path), err)
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("[runall] exec-queue: write %s: %v", tmp, err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("[runall] exec-queue: rename %s: %v", path, err)
		_ = os.Remove(tmp)
	}
}

// LoadExecQueueFromDisk restores pending from exec_queue.json after UI-mode
// start / hot-replace. An in-flight "current" cannot resume (worker goroutine
// and cancel handle died with the previous process); it is abandoned and
// prepended to pending so the UI can replay. Missing or relative paths are a
// no-op.
func (r *Runner) LoadExecQueueFromDisk() {
	if r == nil {
		return
	}
	path := r.execQueueFilePath()
	if !filepath.IsAbs(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[runall] exec-queue: read %s: %v", path, err)
		}
		return
	}
	var snap ExecutionQueueSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		log.Printf("[runall] exec-queue: decode %s: %v", path, err)
		return
	}
	var orphan *ExecQueuePendingItem
	if snap.Current != nil && snap.Current.Kind != "" && snap.Current.RunID != "" {
		orphan = r.abandonOrphanedActiveRun(snap.Current)
	}
	cleaned := make([]ExecQueuePendingItem, 0, len(snap.Pending)+1)
	seenType := map[string]struct{}{}
	if orphan != nil {
		cleaned = append(cleaned, *orphan)
		seenType[orphan.Type] = struct{}{}
	}
	for _, item := range snap.Pending {
		if item.Type == "" {
			continue
		}
		if _, ok := knownExecQueueTypes[item.Type]; !ok {
			log.Printf("[runall] exec-queue: skip unknown pending type %q", item.Type)
			continue
		}
		if _, dup := seenType[item.Type]; dup {
			continue
		}
		if item.Label == "" {
			item.Label = bulkOpLabel(item.Type)
		}
		cleaned = append(cleaned, item)
		seenType[item.Type] = struct{}{}
	}
	r.replaceExecQueuePending(cleaned)
	r.persistExecQueueFile()
	orphanKind := ""
	if orphan != nil {
		orphanKind = orphan.Type
	}
	log.Printf("[runall] exec-queue: loaded pending=%d orphaned_current=%s from %s", len(cleaned), orphanKind, path)
}

// abandonOrphanedActiveRun discards a persisted "current" bulk op after
// hot-replace. The previous process's goroutine and cancel handle are gone;
// restoring active flags would freeze the progress panel and make cancel 404.
// Returns a pending item so the UI can re-queue the interrupted operation.
func (r *Runner) abandonOrphanedActiveRun(snap *ActiveBulkProgressSnapshot) *ExecQueuePendingItem {
	if r == nil || snap == nil || snap.Kind == "" {
		return nil
	}
	if _, ok := knownExecQueueTypes[snap.Kind]; !ok {
		log.Printf("[runall] exec-queue: abandon skipped unknown kind %q", snap.Kind)
		return nil
	}
	runID := snap.RunID
	ev := StartAllProgressEvent{
		Done:      true,
		Phase:     "interrupted",
		Operation: snap.Operation,
		Error:     "interrupted by runAll hot-replace; re-queued for replay",
	}
	if snap.Event != nil {
		ev.Total = snap.Event.Total
		ev.Started = snap.Event.Started
		ev.Failed = snap.Event.Failed
		ev.Remaining = snap.Event.Remaining
		ev.Skipped = snap.Event.Skipped
		if snap.Event.Operation != "" {
			ev.Operation = snap.Event.Operation
		}
	}
	if runID != "" {
		r.PublishProgress(runID, ev)
		go func(id string) {
			time.Sleep(2 * time.Second)
			if r.progressBroadcaster != nil {
				r.progressBroadcaster.CloseRun(id)
			}
		}(runID)
	}
	log.Printf("[runall] exec-queue: orphaned current kind=%s run_id=%s abandoned → pending", snap.Kind, runID)
	return &ExecQueuePendingItem{
		ID:    "orphan-" + snap.Kind,
		Type:  snap.Kind,
		Label: bulkOpLabel(snap.Kind),
	}
}

// ClearOrphanedBulkProgress clears in-memory "active" markers for a bulk kind
// when no cancellable lifecycle handle exists (hot-replace ghost). Returns true
// if something was cleared.
func (r *Runner) ClearOrphanedBulkProgress(kind string) bool {
	if r == nil || kind == "" {
		return false
	}
	cleared := false
	switch kind {
	case "precise-restart":
		if r.IsPreciseRestartActive() {
			runID := r.GetActivePreciseRestartRunID()
			r.endPreciseRestart()
			r.publishOrphanCleared(runID, "restart")
			cleared = true
		}
	case "restart-all":
		if r.IsRestartAllActive() {
			runID := r.GetActiveStopAllRunID()
			if runID == "" {
				runID = r.GetActiveStartAllRunID()
			}
			r.endRestartAll()
			r.clearActiveStartAllRunID()
			r.clearActiveStopAllRunID()
			r.publishOrphanCleared(runID, "restart")
			cleared = true
		}
	case "build-all":
		if runID := r.GetActiveBuildAllRunID(); runID != "" {
			r.clearActiveBuildAllRunID()
			r.EndBulk("build-all")
			r.publishOrphanCleared(runID, "build")
			cleared = true
		}
	case "start-all":
		if runID := r.GetActiveStartAllRunID(); runID != "" {
			r.clearActiveStartAllRunID()
			r.EndBulk("start-all")
			r.publishOrphanCleared(runID, "start")
			cleared = true
		}
	case "stop-all":
		if runID := r.GetActiveStopAllRunID(); runID != "" {
			r.clearActiveStopAllRunID()
			r.EndBulk("stop-all")
			r.publishOrphanCleared(runID, "stop")
			cleared = true
		}
	case "init-db", "clear-db":
		if op, runID := r.GetActiveDevDBRun(); runID != "" && (op == kind || op == "") {
			r.ClearActiveDevDBRun()
			r.publishOrphanCleared(runID, kind)
			cleared = true
		}
	}
	if cleared {
		r.persistExecQueueFile()
		log.Printf("[runall] exec-queue: cleared orphaned bulk kind=%s", kind)
	}
	return cleared
}

func (r *Runner) publishOrphanCleared(runID, operation string) {
	if r == nil || runID == "" {
		return
	}
	r.PublishProgress(runID, StartAllProgressEvent{
		Done: true, Phase: "cancelled", Operation: operation,
		Error: "cleared orphaned bulk progress after hot-replace",
	})
}

func (r *Runner) clearActiveBuildAllRunID() {
	r.activeBuildAllRunIDMu.Lock()
	r.activeBuildAllRunID = ""
	r.activeBuildAllRunIDMu.Unlock()
}

func (r *Runner) clearActiveStartAllRunID() {
	r.activeStartAllRunIDMu.Lock()
	r.activeStartAllRunID = ""
	r.activeStartAllRunIDMu.Unlock()
}

func (r *Runner) clearActiveStopAllRunID() {
	r.activeStopAllRunIDMu.Lock()
	r.activeStopAllRunID = ""
	r.activeStopAllRunIDMu.Unlock()
}

func registerExecQueueHandler(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/exec-queue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		var payload struct {
			Pending []ExecQueuePendingItem `json:"pending"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONErrorWithStatus(w, http.StatusBadRequest, "invalid json")
			return
		}
		if payload.Pending == nil {
			payload.Pending = []ExecQueuePendingItem{}
		}
		if len(payload.Pending) > maxExecQueuePending {
			writeJSONErrorWithStatus(w, http.StatusBadRequest, "pending too long")
			return
		}
		cleaned := make([]ExecQueuePendingItem, 0, len(payload.Pending))
		seenType := map[string]struct{}{}
		for _, item := range payload.Pending {
			if item.Type == "" {
				writeJSONErrorWithStatus(w, http.StatusBadRequest, "pending type is required")
				return
			}
			if _, ok := knownExecQueueTypes[item.Type]; !ok {
				writeJSONErrorWithStatus(w, http.StatusBadRequest, "unknown pending type")
				return
			}
			// OPT-20260819-040: 同类型 bulk 只允许排队一条，重复入队直接折叠，
			// 避免「全部重新编译」在确认连点/多标签下出现第二条 build-all。
			if _, dup := seenType[item.Type]; dup {
				log.Printf("[runall] exec-queue: drop duplicate pending type=%q id=%s", item.Type, item.ID)
				continue
			}
			seenType[item.Type] = struct{}{}
			if item.Label == "" {
				item.Label = bulkOpLabel(item.Type)
			}
			cleaned = append(cleaned, item)
		}
		runner.SetExecQueuePending(cleaned)
		log.Printf("[runall] exec-queue: pending=%d", len(cleaned))
		writeJSON(w, map[string]any{"status": "ok", "pending": cleaned})
	})
}
