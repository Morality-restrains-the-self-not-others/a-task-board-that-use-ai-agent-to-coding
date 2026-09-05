// Package sessionhub — 多智能体会话互知与冲突防护（agent-session-coordination v14/v66）。
//
// 提供：
//   - 会话注册表（logs/sessions/registry/）：每会话 JSON（kind/pid/心跳/状态/持有锁/等待集）
//   - 仓库级锁（logs/sessions/locks/）：租约（TTL 10min）+ 心跳（30s）+ 僵尸窃取
//   - 死锁检测：持有者等待集相交 → 依赖环报告
//   - 暂停/恢复（SIGSTOP/SIGCONT 进程组）
//   - Shadow Edit（快照拷贝 → 临时目录编辑 → 校验和原子拷回）
//   - 审计日志（logs/sessions/audit/）
//
// 目录根：meta root（含 .gitmodules 的仓库根），可用 SESSION_HUB_DIR 覆盖。
package sessionhub

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// TTL 锁租约时长 — 心跳超过 TTL 未刷新视为过期（配合 pid 存活判断决定可否窃取）
	TTL = 10 * time.Minute
	// HeartbeatInterval 心跳守护刷新间隔
	HeartbeatInterval = 30 * time.Second
	// LockWaitPoll acquire 等待轮询间隔
	LockWaitPoll = 2 * time.Second
)

// AcquireResult 获取锁的结果。
type AcquireResult int

const (
	Acquired AcquireResult = iota
	HeldBy
	Stolen
	DeadlockDetected
)

func (r AcquireResult) String() string {
	switch r {
	case Acquired:
		return "ACQUIRED"
	case HeldBy:
		return "HELD_BY"
	case Stolen:
		return "STOLEN"
	case DeadlockDetected:
		return "DEADLOCK"
	}
	return "UNKNOWN"
}

// Session 会话注册记录。
type Session struct {
	SessionID   string   `json:"session_id"`
	Kind        string   `json:"kind"` // interactive | headless | sweep | ci
	PID         int      `json:"pid"`
	StartCwd    string   `json:"start_cwd"`
	RepoRoot    string   `json:"repo_root"`
	StartedAt   string   `json:"started_at"`
	HeartbeatAt string   `json:"heartbeat_at"`
	Status      string   `json:"status"` // active | paused | borrowing | done | dead
	Locks       []string `json:"locks"`
	HoldingWait []string `json:"holding_wait"`
	Note        string   `json:"note"`
}

// Lock 仓库级锁记录。
type Lock struct {
	Repo       string `json:"repo"`
	Holder     string `json:"holder"`
	AcquiredAt string `json:"acquired_at"`
	ExpiresAt  string `json:"expires_at"`
}

// AcquireItem AcquireMany 的单项结果。
type AcquireItem struct {
	Repo   string
	Result AcquireResult
}

// ConflictError Shadow Apply 时的原文件已变更冲突。
type ConflictError struct {
	Files []string
}

func (e *ConflictError) Error() string {
	return "shadow apply conflict — 原文件已被他人修改: " + strings.Join(e.Files, ", ")
}

// Hub 会话互知中枢。
type Hub struct {
	root string // meta root
	dir  string // <root>/logs/sessions
}

// DiscoverRoot 从 start 向上查找含 .gitmodules 的 meta root。
func DiscoverRoot(start string) (string, error) {
	d, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, ".gitmodules")); err == nil {
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("meta root not found (no .gitmodules) from %s; set SESSION_HUB_DIR", start)
		}
		d = parent
	}
}

// New 创建 Hub（确保目录结构存在）。
func New(root string) (*Hub, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	h := &Hub{root: abs, dir: filepath.Join(abs, "logs", "sessions")}
	for _, sub := range []string{"registry", "locks", "shadow", "audit"} {
		if err := os.MkdirAll(filepath.Join(h.dir, sub), 0o755); err != nil {
			return nil, err
		}
	}
	return h, nil
}

// NewFromEnv 从环境 SESSION_HUB_DIR（优先）或 cwd 向上发现创建 Hub。
func NewFromEnv(start string) (*Hub, error) {
	if d := os.Getenv("SESSION_HUB_DIR"); d != "" {
		return New(d)
	}
	root, err := DiscoverRoot(start)
	if err != nil {
		return nil, err
	}
	return New(root)
}

// PIDAlive 检查进程是否存活（僵尸/死亡进程视为不存活）。
func PIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == syscall.ESRCH {
		return false
	}
	if err != nil && err != syscall.EPERM {
		return false
	}
	// 僵尸进程 kill(pid,0) 返回 nil — 读 /proc/<pid>/stat 的 state 字段排除
	// 格式: "pid (comm) state ..." — 取最后一个 ')' 之后的首字符
	if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		if i := strings.LastIndexByte(string(b), ')'); i >= 0 {
			tail := strings.TrimSpace(string(b[i+1:]))
			if len(tail) > 0 {
				st := tail[0]
				if st == 'Z' || st == 'X' {
					return false
				}
			}
		}
	}
	return true
}

// ---- 注册表 ----

func (h *Hub) sidPath(sid string) string { return filepath.Join(h.dir, "registry", sid+".json") }

func (h *Hub) get(sid string) (*Session, error) {
	b, err := os.ReadFile(h.sidPath(sid))
	if err != nil {
		return nil, fmt.Errorf("session %s: %w", sid, err)
	}
	var s Session
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (h *Hub) put(s *Session) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := h.sidPath(s.SessionID) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, h.sidPath(s.SessionID))
}

func newSID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("sess_%d_%s", time.Now().Unix(), hex.EncodeToString(b))
}

// Get 读取指定会话记录。
func (h *Hub) Get(sid string) (*Session, error) {
	return h.get(sid)
}

// Register 注册会话，返回 session_id。
func (h *Hub) Register(s Session) (string, error) {
	sid := newSID()
	s.SessionID = sid
	s.StartedAt = time.Now().Format(time.RFC3339)
	s.HeartbeatAt = s.StartedAt
	if s.Status == "" {
		s.Status = "active"
	}
	if err := h.put(&s); err != nil {
		return "", err
	}
	_ = h.Audit(sid, "register", "kind="+s.Kind)
	return sid, nil
}

// Unregister 注销会话：释放其全部锁 + 删除注册记录。
func (h *Hub) Unregister(sid string) error {
	s, err := h.get(sid)
	if err != nil {
		return err
	}
	for _, repo := range s.Locks {
		_ = h.Release(repo, sid)
	}
	_ = h.Audit(sid, "unregister", "")
	return os.Remove(h.sidPath(sid))
}

// List 列出会话；activeOnly=true 仅返回活跃会话。
// 检测到 pid 死亡 → 标记 dead、释放其锁、删除记录（僵尸回收）。
func (h *Hub) List(activeOnly bool) ([]Session, error) {
	entries, err := os.ReadDir(filepath.Join(h.dir, "registry"))
	if err != nil {
		return nil, err
	}
	var out []Session
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		sid := strings.TrimSuffix(e.Name(), ".json")
		s, err := h.get(sid)
		if err != nil {
			continue
		}
		if !PIDAlive(s.PID) {
			s.Status = "dead"
			_ = h.Audit(sid, "reap", "pid dead; releasing locks")
			for _, repo := range s.Locks {
				_ = h.Release(repo, sid)
			}
			_ = os.Remove(h.sidPath(sid))
			continue
		}
		if activeOnly && s.Status != "active" {
			continue
		}
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt < out[j].StartedAt })
	return out, nil
}

// Heartbeat 刷新会话心跳（须 pid 存活）。
func (h *Hub) Heartbeat(sid string, pid int) error {
	s, err := h.get(sid)
	if err != nil {
		return err
	}
	if !PIDAlive(pid) {
		return fmt.Errorf("session %s pid %d not alive", sid, pid)
	}
	s.HeartbeatAt = time.Now().Format(time.RFC3339)
	s.PID = pid
	return h.put(s)
}

// Dir 返回 hub 目录（<meta root>/logs/sessions），供 CLI 层放置守护 pidfile 等。
func (h *Hub) Dir() string { return h.dir }

// HeartbeatDaemon 长驻心跳守护（OPT-20260806-003）。
//
// 以 interval 周期刷新会话心跳（严格租约：TTL 10min 内持续刷新，即使长思考/无事件）。
// 退出条件（返回 nil）：被监护 pid 死亡、stop 通道关闭（SIGTERM/外层调用方）。
// 返回错误：会话已注销或 pid 失效（守护使命完成，调用方应静默退出）。
func (h *Hub) HeartbeatDaemon(sid string, pid int, interval time.Duration, stop <-chan struct{}) error {
	if interval <= 0 {
		interval = HeartbeatInterval
	}
	for {
		if err := h.Heartbeat(sid, pid); err != nil {
			return err
		}
		select {
		case <-stop:
			return nil
		case <-time.After(interval):
		}
		if !PIDAlive(pid) {
			return nil // 被监护进程已退出 — 守护完成使命
		}
	}
}

// ---- 锁 ----

func sanitizeRepo(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("repo name empty")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("repo must be relative: %s", name)
	}
	if strings.ContainsAny(name, " \t\n") {
		return "", fmt.Errorf("repo must not contain spaces: %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid repo path: %s", name)
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.') {
				return "", fmt.Errorf("invalid char %q in repo: %s", c, name)
			}
		}
	}
	return name, nil
}

func (h *Hub) lockPath(repo string) string { return filepath.Join(h.dir, "locks", repo+".json") }

func (h *Hub) writeLock(repo string, l *Lock) error {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(h.lockPath(repo)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(h.lockPath(repo), b, 0o644)
}

func (h *Hub) readLock(repo string) (*Lock, error) {
	b, err := os.ReadFile(h.lockPath(repo))
	if err != nil {
		return nil, err
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// withLockFile 以 flock 序列化锁目录操作（原子性保证）。
func (h *Hub) withLockFile(fn func() error) error {
	ctl := filepath.Join(h.dir, "locks", ".ctl")
	f, err := os.OpenFile(ctl, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func (h *Hub) addSessionLock(holder, repo string) {
	if s, err := h.get(holder); err == nil {
		for _, r := range s.Locks {
			if r == repo {
				return
			}
		}
		s.Locks = append(s.Locks, repo)
		_ = h.put(s)
	}
}

// tryAcquire 单次尝试获取锁（须在 withLockFile 内调用）。
func (h *Hub) tryAcquire(repo, holder string) (AcquireResult, *Lock, error) {
	lock, err := h.readLock(repo)
	if os.IsNotExist(err) {
		lock = &Lock{Repo: repo, Holder: holder, AcquiredAt: time.Now().Format(time.RFC3339), ExpiresAt: time.Now().Add(TTL).Format(time.RFC3339)}
		if err := h.writeLock(repo, lock); err != nil {
			return 0, nil, err
		}
		h.addSessionLock(holder, repo)
		_ = h.Audit(holder, "acquire", repo)
		return Acquired, lock, nil
	}
	if err != nil {
		return 0, nil, err
	}

	holderS, err := h.get(lock.Holder)
	if err != nil {
		// 孤儿锁（持有者注册表已消失）→ 窃取
		lock.Holder = holder
		lock.AcquiredAt = time.Now().Format(time.RFC3339)
		lock.ExpiresAt = time.Now().Add(TTL).Format(time.RFC3339)
		if err := h.writeLock(repo, lock); err != nil {
			return 0, nil, err
		}
		h.addSessionLock(holder, repo)
		_ = h.Audit(holder, "steal", repo+" orphan-holder="+lock.Holder)
		return Stolen, lock, nil
	}

	if !PIDAlive(holderS.PID) {
		// 持有者进程已死 → 窃取
		old := lock.Holder
		lock.Holder = holder
		lock.AcquiredAt = time.Now().Format(time.RFC3339)
		lock.ExpiresAt = time.Now().Add(TTL).Format(time.RFC3339)
		if err := h.writeLock(repo, lock); err != nil {
			return 0, nil, err
		}
		h.addSessionLock(holder, repo)
		_ = h.Audit(holder, "steal", repo+" holder="+old+" (pid dead)")
		return Stolen, lock, nil
	}

	// 持有者存活：心跳过期仅告警，不窃取
	if ts, err := time.Parse(time.RFC3339, holderS.HeartbeatAt); err == nil && time.Since(ts) > TTL {
		_ = h.Audit(holder, "warn", repo+" held-by="+lock.Holder+" (stale heartbeat)")
	}
	return HeldBy, lock, nil
}

// Acquire 获取仓库锁。wait>0 时轮询等待；超时后返回 HeldBy（调用方可用 DetectDeadlock 判环）。
func (h *Hub) Acquire(repo, holder string, wait time.Duration) (AcquireResult, *Lock, error) {
	repo, err := sanitizeRepo(repo)
	if err != nil {
		return 0, nil, err
	}
	if _, err := h.get(holder); err != nil {
		return 0, nil, fmt.Errorf("holder session not registered: %s", holder)
	}
	deadline := time.Now().Add(wait)
	for {
		var res AcquireResult
		var lock *Lock
		err := h.withLockFile(func() error {
			var e error
			res, lock, e = h.tryAcquire(repo, holder)
			return e
		})
		if err != nil {
			return 0, nil, err
		}
		if res != HeldBy || time.Now().After(deadline) {
			if res == HeldBy {
				_ = h.SetHoldingWait(holder, repo)
			}
			return res, lock, nil
		}
		_ = h.SetHoldingWait(holder, repo)
		time.Sleep(LockWaitPoll)
	}
}

// AcquireMany 按字典序依次获取多仓锁（规范加锁序 → 无环等待，防死锁）。
func (h *Hub) AcquireMany(repos []string, holder string, wait time.Duration) ([]AcquireItem, error) {
	if _, err := h.get(holder); err != nil {
		return nil, fmt.Errorf("holder session not registered: %s", holder)
	}
	sorted := append([]string(nil), repos...)
	sort.Strings(sorted)
	out := make([]AcquireItem, 0, len(sorted))
	for _, repo := range sorted {
		res, _, err := h.Acquire(repo, holder, wait)
		if err != nil {
			return out, err
		}
		out = append(out, AcquireItem{Repo: repo, Result: res})
	}
	return out, nil
}

// Release 释放锁（仅持有者可释放；非持有者返回错误）。
func (h *Hub) Release(repo, holder string) error {
	repo, err := sanitizeRepo(repo)
	if err != nil {
		return err
	}
	return h.withLockFile(func() error {
		lock, err := h.readLock(repo)
		if os.IsNotExist(err) {
			return nil // 已释放，幂等
		}
		if err != nil {
			return err
		}
		if lock.Holder != holder {
			return fmt.Errorf("lock %s held by %s, not %s", repo, lock.Holder, holder)
		}
		if err := os.Remove(h.lockPath(repo)); err != nil {
			return err
		}
		if s, err := h.get(holder); err == nil {
			var kept []string
			for _, r := range s.Locks {
				if r != repo {
					kept = append(kept, r)
				}
			}
			s.Locks = kept
			_ = h.put(s)
		}
		_ = h.Audit(holder, "release", repo)
		return nil
	})
}

// Check 检查仓库锁状态。held=false 表示空闲或僵尸可窃取。
func (h *Hub) Check(repo string) (*Lock, bool, error) {
	repo, err := sanitizeRepo(repo)
	if err != nil {
		return nil, false, err
	}
	lock, err := h.readLock(repo)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	holderS, err := h.get(lock.Holder)
	if err != nil || !PIDAlive(holderS.PID) {
		return lock, false, nil // 僵尸/孤儿 → 可窃取
	}
	return lock, true, nil
}

// ---- 死锁 ----

// SetHoldingWait 记录会话正在等待的仓库（用于死锁检测）。
func (h *Hub) SetHoldingWait(sid, repo string) error {
	s, err := h.get(sid)
	if err != nil {
		return err
	}
	for _, r := range s.HoldingWait {
		if r == repo {
			return nil
		}
	}
	s.HoldingWait = append(s.HoldingWait, repo)
	return h.put(s)
}

// DetectDeadlock 从 holder 出发沿等待图 DFS；回到自身即判死锁，返回环路径。
func (h *Hub) DetectDeadlock(holder string) ([]string, bool) {
	if _, err := h.get(holder); err != nil {
		return nil, false
	}
	visited := map[string]bool{}
	var dfs func(node string) ([]string, bool)
	dfs = func(node string) ([]string, bool) {
		if visited[node] {
			return nil, false
		}
		visited[node] = true
		ns, err := h.get(node)
		if err != nil {
			return nil, false
		}
		for _, repo := range ns.HoldingWait {
			lock, err := h.readLock(repo)
			if err != nil {
				continue
			}
			if lock.Holder == holder {
				return []string{node}, true // 回到起始 → 环（起点为隐式终点，不重复）
			}
			if next, ok := dfs(lock.Holder); ok {
				return append([]string{node}, next...), true
			}
		}
		return nil, false
	}
	return dfs(holder)
}

// ---- 暂停/恢复 ----

func processGroupSignal(pid int, sig syscall.Signal) error {
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		return err
	}
	return syscall.Kill(-pgid, sig)
}

// PauseSession 暂停会话（SIGSTOP 目标 pid）。锁保持持有。
// 默认仅冻结会话主进程（安全：交互会话可能与终端共享进程组，组信号会波及 shell）。
// 需要冻结整个进程组时用 PauseSessionGroup。
func PauseSession(s *Session) error {
	return syscall.Kill(s.PID, syscall.SIGSTOP)
}

// ResumeSession 恢复会话（SIGCONT 目标 pid）。
func ResumeSession(s *Session) error {
	return syscall.Kill(s.PID, syscall.SIGCONT)
}

// PauseSessionGroup 暂停会话整个进程组（SIGSTOP）。
func PauseSessionGroup(s *Session) error {
	return processGroupSignal(s.PID, syscall.SIGSTOP)
}

// ResumeSessionGroup 恢复会话整个进程组（SIGCONT）。
func ResumeSessionGroup(s *Session) error {
	return processGroupSignal(s.PID, syscall.SIGCONT)
}

// ---- 祖先链（pre-commit 自我识别）----

func ppidOf(pid int) int {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return -1
	}
	// 格式: "pid (comm) state ppid ..." — 取最后一个 ')' 之后的字段
	s := string(b)
	if i := strings.LastIndexByte(s, ')'); i >= 0 {
		f := strings.Fields(s[i+1:])
		if len(f) >= 2 { // f[0]=state, f[1]=ppid
			n, err := strconv.Atoi(f[1])
			if err == nil {
				return n
			}
		}
	}
	return -1
}

// ProcessHasAncestor 检查 pid 的祖先链（沿 /proc ppid 上溯）中是否包含 want。
// 用于 pre-commit 自我识别：提交进程是会话主进程的后代时视为「本会话自己」。
func ProcessHasAncestor(pid, want int) bool {
	seen := map[int]bool{}
	for cur := pid; cur > 0 && !seen[cur]; cur = ppidOf(cur) {
		if cur == want {
			return true
		}
		seen[cur] = true
	}
	return false
}

// ---- Shadow Edit ----

type shadowManifest struct {
	Repo  string                     `json:"repo"`
	Files map[string]shadowFileEntry `json:"files"` // key: repo 相对路径
}

type shadowFileEntry struct {
	Path   string `json:"path"` // 相对 repo 的路径
	SHA256 string `json:"sha256"`
}

func (h *Hub) shadowDir(sid string) string { return filepath.Join(h.dir, "shadow", sid) }
func (h *Hub) shadowManifest(sid string) string {
	return filepath.Join(h.shadowDir(sid), "manifest.json")
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hsh := sha256.New()
	if _, err := io.Copy(hsh, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hsh.Sum(nil)), nil
}

func sanitizeRelPath(rel string) error {
	if rel == "" || filepath.IsAbs(rel) {
		return fmt.Errorf("path must be repo-relative: %s", rel)
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." {
			return fmt.Errorf("path traversal rejected: %s", rel)
		}
	}
	return nil
}

// ShadowBegin 将文件快照拷贝至 shadow 区并记录校验和；返回 shadow 绝对路径。
func (h *Hub) ShadowBegin(sid, repo string, paths []string) ([]string, error) {
	repo, err := sanitizeRepo(repo)
	if err != nil {
		return nil, err
	}
	m := &shadowManifest{Repo: repo, Files: map[string]shadowFileEntry{}}
	destDir := filepath.Join(h.shadowDir(sid), repo)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	var shadowPaths []string
	for _, rel := range paths {
		if err := sanitizeRelPath(rel); err != nil {
			return nil, err
		}
		src := filepath.Join(h.root, repo, rel)
		sum, err := fileSHA256(src)
		if err != nil {
			continue // 源不存在则跳过（不阻塞批量）
		}
		dest := filepath.Join(destDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, err
		}
		b, err := os.ReadFile(src)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(dest, b, 0o644); err != nil {
			return nil, err
		}
		m.Files[rel] = shadowFileEntry{Path: rel, SHA256: sum}
		shadowPaths = append(shadowPaths, dest)
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(h.shadowManifest(sid), b, 0o644); err != nil {
		return nil, err
	}
	if len(m.Files) == 0 {
		_ = h.Audit(sid, "shadow-begin", repo+" no-existing-files: "+fmt.Sprint(paths))
		return nil, nil
	}
	_ = h.Audit(sid, "shadow-begin", repo+" "+fmt.Sprint(paths))
	return shadowPaths, nil
}

// ShadowApply 校验原文件未变后原子拷回。force=true 时对冲突文件留 .bak 后覆盖。
func (h *Hub) ShadowApply(sid, repo string, paths []string, force bool) error {
	m, err := h.readManifest(sid)
	if err != nil {
		return err
	}
	repo, err = sanitizeRepo(repo)
	if err != nil {
		return err
	}
	if m.Repo != repo {
		return fmt.Errorf("manifest repo %s != requested %s", m.Repo, repo)
	}
	var conflicts []string
	for rel, entry := range m.Files {
		src := filepath.Join(h.root, repo, rel)
		sum, err := fileSHA256(src)
		if err != nil {
			conflicts = append(conflicts, rel+" (missing)")
			continue
		}
		if sum != entry.SHA256 {
			conflicts = append(conflicts, rel)
		}
	}
	if len(conflicts) > 0 && !force {
		return &ConflictError{Files: conflicts}
	}

	shadowRoot := filepath.Join(h.shadowDir(sid), repo)
	for rel, entry := range m.Files {
		shadowSrc := filepath.Join(shadowRoot, rel)
		target := filepath.Join(h.root, repo, rel)
		if sum, err := fileSHA256(target); err == nil && sum != entry.SHA256 {
			// force 覆盖：原文件留 .bak 备份
			_ = os.Rename(target, target+".bak-"+sid)
		}
		tmp := target + ".shadow-" + sid
		b, err := os.ReadFile(shadowSrc)
		if err != nil {
			return err
		}
		if err := os.WriteFile(tmp, b, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmp, target); err != nil {
			return err
		}
	}
	_ = h.Audit(sid, "shadow-apply", repo+" force="+fmt.Sprint(force))
	return os.Remove(h.shadowManifest(sid))
}

// ShadowAbort 丢弃 shadow 快照。
func (h *Hub) ShadowAbort(sid string) error {
	_ = h.Audit(sid, "shadow-abort", "")
	return os.RemoveAll(h.shadowDir(sid))
}

func (h *Hub) readManifest(sid string) (*shadowManifest, error) {
	b, err := os.ReadFile(h.shadowManifest(sid))
	if err != nil {
		return nil, fmt.Errorf("shadow manifest: %w", err)
	}
	var m shadowManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ---- 审计 ----

// Audit 追加审计日志（audit/<YYYYMMDD>.log）。detail 中的换行与敏感分隔符被净化。
func (h *Hub) Audit(sid, op, detail string) error {
	clean := strings.NewReplacer("\n", " ", "\r", " ").Replace(detail)
	line := fmt.Sprintf("%s %s %s %s\n", time.Now().Format(time.RFC3339), sid, op, clean)
	f, err := os.OpenFile(filepath.Join(h.dir, "audit", time.Now().Format("20060102")+".log"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}
