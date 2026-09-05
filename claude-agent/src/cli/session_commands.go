// 会话互知与冲突防护 CLI 子命令（agent-session-coordination v14/v66）。
//
// 用法:
//
//	claude-agent session register  [--kind interactive|headless|sweep|ci] [--pid N] [--cwd DIR] [--note S]
//	claude-agent session list      [--active] [--repo R] [--json]
//	claude-agent session acquire   <repo...> --sid S [--wait SECS]
//	claude-agent session release   <repo...>|--all --sid S
//	claude-agent session heartbeat --sid S [--pid N]
//	claude-agent session heartbeat --daemon [--pid N] [--interval 30]  # 长驻守护（OPT-20260806-003）
//	claude-agent session check     <repo> [--pid N]       # FREE / HELD_BY / HELD_BY_SELF
//	claude-agent session pause     <sid>
//	claude-agent session resume    <sid>
//	claude-agent session shadow-begin <repo> <path...> --sid S
//	claude-agent session shadow-apply  --sid S --repo R [--force]
//	claude-agent session shadow-abort  --sid S
//
// Hub 目录：SESSION_HUB_DIR 或从 cwd 向上发现 .gitmodules 的 meta root。
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"claudeAgent/src/sessionhub"
)

func sessionHub() (*sessionhub.Hub, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return sessionhub.NewFromEnv(cwd)
}

// reorderFlags 将 args 中散落的 -flag 及其值移到前面 — Go flag 包在首个
// 位置参数后停止解析，而本 CLI 约定位置参数（仓库/路径）可能出现在 flag 前。
func reorderFlags(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		tok := args[i]
		if strings.HasPrefix(tok, "-") && tok != "-" {
			flags = append(flags, tok)
			// 无 "=" 形式的 flag：若下一 token 非 flag，视为其值一并前移
			if !strings.Contains(tok, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			pos = append(pos, tok)
		}
	}
	return append(flags, pos...)
}

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "session: "+format+"\n", a...)
	os.Exit(1)
}

// CmdSession 实现 `session` 子命令。
func CmdSession(args []string) {
	if len(args) < 1 {
		printSessionUsage()
		os.Exit(1)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "register":
		cmdSessionRegister(rest)
	case "list":
		cmdSessionList(rest)
	case "acquire":
		cmdSessionAcquire(rest)
	case "release":
		cmdSessionRelease(rest)
	case "heartbeat":
		cmdSessionHeartbeat(rest)
	case "check":
		cmdSessionCheck(rest)
	case "precheck":
		cmdSessionPrecheck(rest)
	case "pause":
		cmdSessionPause(rest, true)
	case "resume":
		cmdSessionPause(rest, false)
	case "shadow-begin":
		cmdShadowBegin(rest)
	case "shadow-apply":
		cmdShadowApply(rest)
	case "shadow-abort":
		cmdShadowAbort(rest)
	default:
		fmt.Fprintf(os.Stderr, "session: unknown subcommand: %s\n\n", sub)
		printSessionUsage()
		os.Exit(1)
	}
}

func printSessionUsage() {
	fmt.Println(`claude-agent session — 多智能体会话互知与冲突防护

Usage:
  claude-agent session <subcommand> [flags]

Subcommands:
  register   注册会话（SessionStart hook / headless run 入口调用）
  list       列出会话（--active 仅活跃；--repo R 过滤占用该仓的会话）
  acquire    获取仓库锁（多仓自动字典序；--wait 秒内轮询等待）
  release    释放仓库锁（--all 释放全部）
  heartbeat  刷新心跳（--daemon 长驻守护 / UserPromptSubmit hook 调用）
  check      检查仓库锁状态（FREE / HELD_BY / HELD_BY_SELF）
  precheck   PreToolUse hook: 探测当前工作区是否有其他活跃会话（有则输出警告）
  pause      暂停会话（SIGSTOP；锁保持持有）
  resume     恢复会话（SIGCONT）
  shadow-begin / shadow-apply / shadow-abort  影子编辑（临时目录编辑→原子拷回）`)
}

func cmdSessionRegister(args []string) {
	fs := flag.NewFlagSet("session register", flag.ExitOnError)
	kind := fs.String("kind", "interactive", "会话类型: interactive|headless|sweep|ci")
	pid := fs.Int("pid", os.Getpid(), "会话主进程 pid")
	note := fs.String("note", "", "备注")
	cwd := fs.String("cwd", "", "起始工作目录（默认当前目录）")
	fs.Parse(reorderFlags(args))

	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	if *cwd == "" {
		*cwd, _ = os.Getwd()
	}
	sid, err := h.Register(sessionhub.Session{
		Kind:     *kind,
		PID:      *pid,
		StartCwd: *cwd,
		Note:     *note,
	})
	if err != nil {
		die("register: %v", err)
	}
	fmt.Println(sid)
	// 互知摘要：SessionStart 时告知当前活跃会话
	sessions, _ := h.List(true)
	if len(sessions) > 1 {
		fmt.Fprintf(os.Stderr, "[session-hub] 当前活跃会话 %d 个（含本会话）:\n", len(sessions))
		for _, s := range sessions {
			if s.SessionID == sid {
				continue
			}
			fmt.Fprintf(os.Stderr, "  • %s kind=%s pid=%d cwd=%s locks=%v note=%s\n",
				s.SessionID, s.Kind, s.PID, s.StartCwd, s.Locks, s.Note)
		}
	}
}

func cmdSessionList(args []string) {
	fs := flag.NewFlagSet("session list", flag.ExitOnError)
	active := fs.Bool("active", false, "仅活跃会话")
	repo := fs.String("repo", "", "过滤占用该仓的会话")
	asJSON := fs.Bool("json", false, "JSON 输出")
	fs.Parse(reorderFlags(args))

	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	sessions, err := h.List(*active)
	if err != nil {
		die("list: %v", err)
	}
	if *repo != "" {
		root, _ := sessionhub.DiscoverRoot(mustCwd())
		prefix := root + "/" + strings.TrimPrefix(*repo, "./") + "/"
		var filtered []sessionhub.Session
		for _, s := range sessions {
			inRepo := false
			for _, l := range s.Locks {
				if l == *repo {
					inRepo = true
					break
				}
			}
			if !inRepo && strings.HasPrefix(s.StartCwd+"/", prefix) {
				inRepo = true
			}
			if inRepo {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}
	if *asJSON {
		b, _ := json.MarshalIndent(sessions, "", "  ")
		fmt.Println(string(b))
		return
	}
	for _, s := range sessions {
		fmt.Printf("%s kind=%s pid=%d status=%s started=%s heartbeat=%s locks=%v cwd=%s note=%s\n",
			s.SessionID, s.Kind, s.PID, s.Status, s.StartedAt, s.HeartbeatAt, s.Locks, s.StartCwd, s.Note)
	}
}

func mustCwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}

func cmdSessionAcquire(args []string) {
	fs := flag.NewFlagSet("session acquire", flag.ExitOnError)
	sid := fs.String("sid", "", "会话 id（须已注册）")
	wait := fs.Int("wait", 0, "等待秒数（超时后报告 HELD_BY / 死锁检测）")
	fs.Parse(reorderFlags(args))
	repos := fs.Args()
	if len(repos) == 0 {
		die("acquire: 至少一个仓库名")
	}
	if *sid == "" {
		die("acquire: --sid 必填")
	}
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	items, err := h.AcquireMany(repos, *sid, time.Duration(*wait)*time.Second)
	if err != nil {
		die("acquire: %v", err)
	}
	held := false
	for _, it := range items {
		fmt.Printf("%s %s\n", it.Repo, it.Result)
		if it.Result == sessionhub.HeldBy {
			held = true
			if lock, _, err := h.Check(it.Repo); err == nil && lock != nil {
				fmt.Fprintf(os.Stderr, "  %s 由 %s 持有（详情: claude-agent session list --repo %s）\n", it.Repo, lock.Holder, it.Repo)
			}
		}
	}
	if held {
		// 死锁检测：等待超时后检查等待环
		if cycle, ok := h.DetectDeadlock(*sid); ok {
			fmt.Fprintf(os.Stderr, "⚠ 检测到死锁环: %v\n", cycle)
		}
		os.Exit(2)
	}
}

func cmdSessionRelease(args []string) {
	fs := flag.NewFlagSet("session release", flag.ExitOnError)
	sid := fs.String("sid", "", "会话 id")
	pid := fs.Int("pid", 0, "会话主进程 pid（hooks: 自动解析 sid）")
	all := fs.Bool("all", false, "释放全部锁")
	fs.Parse(reorderFlags(args))
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	if *sid == "" {
		if *pid == 0 {
			die("release: --sid 或 --pid 必填")
		}
		s, err := lookupSessionByPID(h, *pid)
		if err != nil {
			die("release: %v", err)
		}
		*sid = s.SessionID
	}
	if *all {
		sessions, _ := h.List(false)
		for _, s := range sessions {
			if s.SessionID != *sid {
				continue
			}
			for _, repo := range s.Locks {
				if err := h.Release(repo, *sid); err != nil {
					fmt.Fprintf(os.Stderr, "release %s: %v\n", repo, err)
				} else {
					fmt.Printf("released %s\n", repo)
				}
			}
			return
		}
		die("release: session %s not found", *sid)
	}
	for _, repo := range fs.Args() {
		if err := h.Release(repo, *sid); err != nil {
			fmt.Fprintf(os.Stderr, "release %s: %v\n", repo, err)
			os.Exit(1)
		}
		fmt.Printf("released %s\n", repo)
	}
}

func cmdSessionHeartbeat(args []string) {
	fs := flag.NewFlagSet("session heartbeat", flag.ExitOnError)
	sid := fs.String("sid", "", "会话 id")
	pid := fs.Int("pid", os.Getpid(), "会话主进程 pid")
	daemon := fs.Bool("daemon", false, "长驻守护：周期刷新心跳直至 pid 死亡或 SIGTERM/SIGINT")
	interval := fs.Int("interval", 0, "守护刷新间隔（秒，默认 30）")
	fs.Parse(reorderFlags(args))
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	if *sid == "" {
		s, err := lookupSessionByPID(h, *pid)
		if err != nil {
			os.Exit(0) // 会话未注册（SessionStart 未跑）— hook 静默
		}
		*sid = s.SessionID
	}
	if !*daemon {
		if err := h.Heartbeat(*sid, *pid); err != nil {
			die("heartbeat: %v", err)
		}
		return
	}
	// 单实例守卫（OPT-20260806-003）：<hub>/daemon-<sid>.pid flock —
	// 已有存活守护在跑则本进程静默退出（心跳幂等，多守护纯属浪费）
	pidfile := filepath.Join(h.Dir(), "daemon-"+*sid+".pid")
	f, err := os.OpenFile(pidfile, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		os.Exit(0)
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		os.Exit(0) // 另一守护持有锁
	}
	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		close(stop)
	}()
	if err := h.HeartbeatDaemon(*sid, *pid, time.Duration(*interval)*time.Second, stop); err != nil {
		os.Exit(0) // 会话已注销/pid 失效 — 静默退出
	}
	os.Exit(0)
}

func cmdSessionCheck(args []string) {
	fs := flag.NewFlagSet("session check", flag.ExitOnError)
	pid := fs.Int("pid", os.Getpid(), "发起方 pid（祖先链自我识别）")
	fs.Parse(reorderFlags(args))
	repos := fs.Args()
	if len(repos) != 1 {
		die("check: 需要一个仓库名")
	}
	repo := repos[0]
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	lock, held, err := h.Check(repo)
	if err != nil {
		die("check: %v", err)
	}
	if !held || lock == nil {
		fmt.Println("FREE")
		return
	}
	holderS, err := h.Get(lock.Holder)
	if err == nil && sessionhub.ProcessHasAncestor(*pid, holderS.PID) {
		fmt.Println("HELD_BY_SELF")
		return
	}
	if err == nil {
		fmt.Printf("HELD_BY %s kind=%s pid=%d note=%s\n", lock.Holder, holderS.Kind, holderS.PID, holderS.Note)
		return
	}
	fmt.Printf("HELD_BY %s\n", lock.Holder)
}

// lookupSessionByPID 按 pid 定位会话（hooks 场景：仅知 CLAUDE_PROCESS_ID）。
func lookupSessionByPID(h *sessionhub.Hub, pid int) (*sessionhub.Session, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid %d", pid)
	}
	sessions, err := h.List(false)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		if sessions[i].PID == pid {
			return &sessions[i], nil
		}
	}
	return nil, fmt.Errorf("no session registered for pid %d (SessionStart hook 未执行?)", pid)
}

// cmdSessionPrecheck — PreToolUse hook 探测：当前工作区是否有其他活跃会话。
// 输出为空 = 无冲突（hook 保持静默）；有冲突则输出警告（会被回传给 agent）。
func cmdSessionPrecheck(args []string) {
	fs := flag.NewFlagSet("session precheck", flag.ExitOnError)
	pid := fs.Int("pid", os.Getpid(), "本会话主进程 pid")
	fs.Parse(reorderFlags(args))

	h, err := sessionHub()
	if err != nil {
		os.Exit(0) // hub 不可用时静默放行
	}
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(0)
	}
	root, err := sessionhub.DiscoverRoot(cwd)
	if err != nil {
		os.Exit(0)
	}
	// 当前仓库 = cwd 相对 meta root 的首段（meta root 自身用其 basename）
	repo := filepath.Base(root)
	if rel, err := filepath.Rel(root, cwd); err == nil && rel != "." && rel != "" {
		repo = strings.Split(rel, string(filepath.Separator))[0]
	}

	sessions, err := h.List(true)
	if err != nil {
		os.Exit(0)
	}
	prefix := root + "/" + repo + "/"
	var others []sessionhub.Session
	for _, s := range sessions {
		if s.PID == *pid {
			continue // 自己
		}
		ownRepo := false
		for _, l := range s.Locks {
			if l == repo {
				ownRepo = true
				break
			}
		}
		if !ownRepo && strings.HasPrefix(s.StartCwd+"/", prefix) {
			ownRepo = true
		}
		if ownRepo {
			others = append(others, s)
		}
	}
	if len(others) > 0 {
		fmt.Printf("[session-hub] ⚠ 当前仓库 %s 有 %d 个其他活跃会话:\n", repo, len(others))
		for _, s := range others {
			fmt.Printf("  • %s kind=%s pid=%d note=%s\n", s.SessionID, s.Kind, s.PID, s.Note)
		}
		fmt.Println("  编辑前建议: claude-agent session list 查看详情；必要时 pause/shadow edit。")
	}
	// 无冲突 → 静默退出
}

func cmdSessionPause(args []string, pause bool) {
	if len(args) < 1 {
		die("pause/resume: 需要会话 id")
	}
	sid := args[0]
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	sessions, _ := h.List(false)
	var target *sessionhub.Session
	for i := range sessions {
		if sessions[i].SessionID == sid {
			target = &sessions[i]
			break
		}
	}
	if target == nil {
		die("%s: session not found", sid)
	}
	if pause {
		if err := sessionhub.PauseSession(target); err != nil {
			die("pause %s: %v", sid, err)
		}
		target.Status = "paused"
		_ = h.Audit(sid, "pause", fmt.Sprintf("pid=%d", target.PID))
		fmt.Printf("paused %s (pid=%d) — 锁保持持有\n", sid, target.PID)
	} else {
		if err := sessionhub.ResumeSession(target); err != nil {
			die("resume %s: %v", sid, err)
		}
		target.Status = "active"
		_ = h.Audit(sid, "resume", fmt.Sprintf("pid=%d", target.PID))
		fmt.Printf("resumed %s (pid=%d)\n", sid, target.PID)
	}
}

func cmdShadowBegin(args []string) {
	fs := flag.NewFlagSet("session shadow-begin", flag.ExitOnError)
	sid := fs.String("sid", "", "会话 id")
	fs.Parse(reorderFlags(args))
	rest := fs.Args()
	if *sid == "" || len(rest) < 2 {
		die("shadow-begin: --sid + <repo> <path...>")
	}
	repo, paths := rest[0], rest[1:]
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	shadowPaths, err := h.ShadowBegin(*sid, repo, paths)
	if err != nil {
		die("shadow-begin: %v", err)
	}
	for _, p := range shadowPaths {
		fmt.Println(p)
	}
}

func cmdShadowApply(args []string) {
	fs := flag.NewFlagSet("session shadow-apply", flag.ExitOnError)
	sid := fs.String("sid", "", "会话 id")
	repo := fs.String("repo", "", "仓库名")
	force := fs.Bool("force", false, "冲突时留 .bak 后强制覆盖")
	fs.Parse(reorderFlags(args))
	if *sid == "" || *repo == "" {
		die("shadow-apply: --sid + --repo 必填")
	}
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	if err := h.ShadowApply(*sid, *repo, nil, *force); err != nil {
		fmt.Fprintf(os.Stderr, "shadow-apply: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("shadow applied")
}

func cmdShadowAbort(args []string) {
	fs := flag.NewFlagSet("session shadow-abort", flag.ExitOnError)
	sid := fs.String("sid", "", "会话 id")
	fs.Parse(reorderFlags(args))
	if *sid == "" {
		die("shadow-abort: --sid 必填")
	}
	h, err := sessionHub()
	if err != nil {
		die("%v", err)
	}
	if err := h.ShadowAbort(*sid); err != nil {
		die("shadow-abort: %v", err)
	}
	fmt.Println("shadow aborted")
}
