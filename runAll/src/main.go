package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

// capabilitySkipOrphanOnShutdownSelf is the stable marker that this binary preserves
// managed services after /api/shutdown-self hot-replace. Agents must rebuild via
// ./build.sh (or run.sh) so the deployed binary contains this string before
// replacing a live runAll that used shutdown-self.
const capabilitySkipOrphanOnShutdownSelf = "skip orphan port cleanup"

var loadConfigWithSourceGuardFn func(string, string) (*Config, domain.ConfigFingerprint, error) = LoadConfigWithSourceGuard

func loadRuntimeConfig(path string) (*Config, error) {
	cfg, _, err := loadConfigWithSourceGuardFn(path, resolveSecondaryConfigPath(path))
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func resolveSecondaryConfigPath(primaryPath string) string {
	candidate := strings.TrimSpace(os.Getenv("RUNALL_SECONDARY_CONFIG"))
	if candidate == "" {
		return ""
	}
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func resolveOwnershipStorePath(configPath string) string {
	fromEnv := strings.TrimSpace(os.Getenv("RUNALL_OWNERSHIP_STORE"))
	if fromEnv != "" {
		return fromEnv
	}

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return filepath.Join(".runall", "ownership.json")
	}
	return filepath.Join(filepath.Dir(absConfigPath), ".runall", "ownership.json")
}

// previousRunAllShutdownResult describes how the previous UI-port occupant was retired.
type previousRunAllShutdownResult struct {
	// HadPrevious is true when at least one other PID was listening on the UI port.
	HadPrevious bool
	// GracefulShutdownSelf is true when /api/shutdown-self returned a success status.
	// In that mode managed services are intentionally left running and must NOT be
	// force-killed as "orphans" by cleanupOrphanManagedServices.
	GracefulShutdownSelf bool
}

var lookupUIPortListeners = listenerPIDs

func postUIShutdownSelfDefault(url string) (int, error) {
	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{Proxy: nil}, // never inherit shell HTTP(S)_PROXY
	}
	resp, reqErr := httpClient.Post(url, "application/json", nil)
	if reqErr != nil {
		return 0, reqErr
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

var postUIShutdownSelf = postUIShutdownSelfDefault

func skipRetireOccupiedUIPortDuringTest(portNum string) bool {
	return testing.Testing() && portNum == "9999"
}

// killPreviousRunAllProcess kills any process listening on the UI port before starting a new runAll instance.
// This ensures only one runAll instance is running at a time.
func killPreviousRunAllProcess(uiPort string) previousRunAllShutdownResult {
	result := previousRunAllShutdownResult{}
	port := strings.TrimSpace(uiPort)
	if port == "" {
		return result
	}
	// Extract port number from address like ":9999"
	portNum := port
	if strings.HasPrefix(portNum, ":") {
		portNum = strings.TrimPrefix(portNum, ":")
	}
	if skipRetireOccupiedUIPortDuringTest(portNum) {
		log.Printf("[runAll] skip retiring occupied UI port %s during go test (OPT-20260830-020)", portNum)
		return result
	}

	pids, err := lookupUIPortListeners(portNum)
	if err != nil {
		log.Printf("[runAll] check previous instance on port %s: %v", portNum, err)
		return result
	}
	if len(pids) == 0 {
		return result
	}

	// Get current PID to avoid killing ourselves
	currentPID := os.Getpid()

	for _, pid := range pids {
		if pid == currentPID {
			continue
		}
		result.HadPrevious = true
		log.Printf("[runAll] killing previous runAll instance (PID=%d) listening on port %s", pid, portNum)

		shutdownURL := fmt.Sprintf("http://localhost:%s/api/shutdown-self", portNum)
		log.Printf("[runAll] requesting graceful shutdown from previous instance: %s", shutdownURL)
		sub := retirePreviousRunAll(pid, &previousRunAllHooks{
			postShutdownSelf: func() (int, error) {
				return postUIShutdownSelf(shutdownURL)
			},
			alive:        func(p int) bool { return syscall.Kill(p, 0) == nil },
			kill:         func(p int, sig syscall.Signal) error { return syscall.Kill(p, sig) },
			gracefulWait: 3 * time.Second,
		})
		if sub.GracefulShutdownSelf {
			result.GracefulShutdownSelf = true
		}
	}
	return result
}

// shouldSkipOrphanCleanup reports whether still-listening managed service ports must
// NOT be force-killed as orphans: either the previous instance used /api/shutdown-self
// (services intentionally kept), or there was no prior UI listener at all
// (HadPrevious=false) — in which case the listeners are surviving services from a
// previous runAll that should be adopted, not SIGKILLed (OPT-20260820-008).
func shouldSkipOrphanCleanup(previous previousRunAllShutdownResult) bool {
	return previous.GracefulShutdownSelf || !previous.HadPrevious
}

// cleanupOrphanManagedServices waits for managed service ports to be released after
// a previous runAll instance is killed. This prevents orphan processes from the old
// runAll from causing EADDRINUSE when the new runAll starts its managed services.
//
// When skipOrphanKill is true (shutdown-self hot-replace, or no prior UI occupant),
// managed services are intentionally kept — do not treat their listeners as orphans.
func cleanupOrphanManagedServices(cfg *Config, previous previousRunAllShutdownResult, skipOrphanKill bool) {
	if cfg == nil {
		return
	}
	if skipOrphanKill {
		log.Printf("[runAll] %s: previous instance kept managed services (shutdown-self or no prior UI)", capabilitySkipOrphanOnShutdownSelf)
		return
	}
	services := cfg.Flatten()
	if len(services) == 0 {
		return
	}

	// Collect all unique service ports to check
	portsToCheck := make(map[string]struct{})
	for _, svc := range services {
		for _, port := range resolveServicePorts(&svc) {
			portsToCheck[port] = struct{}{}
		}
	}
	if len(portsToCheck) == 0 {
		return
	}

	log.Printf("[runAll] checking for orphan processes on %d managed service ports...", len(portsToCheck))

	deadline := time.After(15 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		remaining := make(map[string][]int)
		for port := range portsToCheck {
			pids, err := listenerPIDs(port)
			if err != nil {
				continue
			}
			if len(pids) > 0 {
				remaining[port] = pids
			}
		}

		if len(remaining) == 0 {
			log.Printf("[runAll] all managed service ports are free")
			return
		}

		select {
		case <-deadline:
			log.Printf("[runAll] timeout waiting for managed service ports, force-cleaning orphans")
			for port, pids := range remaining {
				for _, pid := range pids {
					log.Printf("[runAll] force-killing orphan PID=%d on port %s", pid, port)
					_ = syscall.Kill(pid, syscall.SIGKILL)
				}
			}
			// Give the kernel a moment to release ports
			time.Sleep(500 * time.Millisecond)
			return
		case <-ticker.C:
			// Continue waiting
		}
	}
}

// residualRunAllProcess is a same-binary runAll that is still alive but no longer
// owns the UI port. It keeps polling health checks and, if ever SIGTERM'd, would
// tear down the managed services it believes it owns (OPT-20260817-001).
type residualRunAllProcess struct {
	PID int
	Exe string
}

// findResidualRunAllProcesses scans /proc for processes running the same binary
// (resolved executable path equal to selfExePath) that are neither this process nor
// listening on the UI port. Daemon instances are excluded because they intentionally
// run without a UI port.
func findResidualRunAllProcesses(selfExePath, uiPort string, selfPID int) []residualRunAllProcess {
	if runtime.GOOS != "linux" {
		return nil
	}
	selfExe, err := filepath.EvalSymlinks(selfExePath)
	if err != nil {
		log.Printf("[runAll] resolve self executable %s: %v", selfExePath, err)
		selfExe = selfExePath
	}

	portNum := strings.TrimPrefix(strings.TrimSpace(uiPort), ":")
	portListeners := make(map[int]struct{})
	if pids, err := listenerPIDs(portNum); err == nil {
		for _, pid := range pids {
			portListeners[pid] = struct{}{}
		}
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		log.Printf("[runAll] scan /proc for residual processes: %v", err)
		return nil
	}

	var residuals []residualRunAllProcess
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		if pid == selfPID {
			continue
		}
		if _, ok := portListeners[pid]; ok {
			continue // owns the UI port → it is the real UI instance
		}
		exeLink, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil {
			continue
		}
		exeLink = strings.TrimSuffix(exeLink, " (deleted)")
		evalExe, err := filepath.EvalSymlinks(exeLink)
		if err != nil {
			evalExe = exeLink
		}
		if evalExe != selfExe {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue
		}
		if strings.Contains(string(cmdline), "-daemon") {
			continue // daemon mode runs without the UI port by design
		}
		residuals = append(residuals, residualRunAllProcess{PID: pid, Exe: evalExe})
	}
	return residuals
}

// cleanupResidualRunAllProcesses logs residual same-binary runAll processes and,
// when RUNALL_CLEANUP_RESIDUAL=1, reaps them with SIGKILL. SIGTERM is deliberately
// avoided: a residual's graceful shutdown path would stop the managed services it
// thinks it owns, which may be the exact services this instance just started.
func cleanupResidualRunAllProcesses(uiPort string) {
	selfExe, err := os.Executable()
	if err != nil {
		log.Printf("[runAll] resolve self executable for residual scan: %v", err)
		return
	}
	residuals := findResidualRunAllProcesses(selfExe, uiPort, os.Getpid())
	if len(residuals) == 0 {
		return
	}
	reap := os.Getenv("RUNALL_CLEANUP_RESIDUAL") == "1"
	for _, res := range residuals {
		if reap {
			log.Printf("[runAll] reaping residual runAll without UI (PID=%d exe=%s): SIGKILL (SIGTERM would stop its managed services)", res.PID, res.Exe)
			_ = syscall.Kill(res.PID, syscall.SIGKILL)
		} else {
			log.Printf("[runAll] WARNING: residual runAll without UI (PID=%d exe=%s) — it keeps polling health checks; set RUNALL_CLEANUP_RESIDUAL=1 to reap it with SIGKILL", res.PID, res.Exe)
		}
	}
}

func main() {
	command := flag.String("command", "run", "run|doctor|takeover|build-all|deploy-sync")
	configPath := flag.String("config", "conf/runAll.yaml", "Path to YAML configuration file")
	daemon := flag.Bool("daemon", false, "Start services and exit (no Web UI)")
	uiPort := flag.String("ui-port", ":9999", "Web UI listen address")
	serviceName := flag.String("service", "", "Service name for takeover command")
	sessionID := flag.String("session-id", "", "Actor session id for takeover command")
	flag.Parse()

	if strings.TrimSpace(*command) == "deploy-sync" {
		os.Exit(runDeploySyncMain(*configPath))
	}

	cfg, err := loadRuntimeConfig(*configPath)
	if err != nil {
		emitStructuredFatal("Config error", err)
	}

	store := NewStatusStore()
	runner, err := NewRunner(cfg, store)
	if err != nil {
		emitStructuredFatal("Setup error", err)
	}
	ownershipRepo := infrastructure.NewFileServiceOwnershipRepository(resolveOwnershipStorePath(*configPath))
	runner.ownershipRepo = ownershipRepo
	runner.ownershipGuard = domain.NewServiceOwnershipGuardService(ownershipRepo)

	absConfigPath, _ := filepath.Abs(*configPath)
	runner.SetConfigPath(absConfigPath)
	// Dev tool logs go to <project_root>/logs/ alongside service tee logs.
	projectRoot := filepath.Dir(filepath.Dir(absConfigPath))
	runner.devToolLogRecorder = infrastructure.NewFileDevToolLogRecorder(projectRoot)

	// Start periodic log collection status emission so each managed service's
	// log production is visible in the Grafana trace-log-explore dashboard.
	runner.StartLogCollectionHeartbeat(30 * time.Second)
	defer runner.StopLogCollectionHeartbeat()

	// Track why the run loop ends: an OS signal vs the /api/shutdown-self
	// hot-replace API. waitTerminationSignal uses signal.Notify — the only
	// mechanism that cannot kill the process: the Go runtime keeps INT/TERM
	// (_SigKill) unblocked on every thread it creates, so the former
	// rt_sigtimedwait design (OPT-20260828-004) raced the delivery thread's
	// dieFromSignal handler and could abort the orchestrator before
	// applyOrchestratorExitKeepPolicy ran (nightly flake 2026-09-01). Notify
	// consumes the signal in the runtime handler and forwards it here, so the
	// ADR-0035 keep-services-on-SIGTERM policy always runs.
	exitReason := newRunAllExitReason()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s, origin, waitErr := waitTerminationSignal(nil)
		if waitErr != nil {
			log.Printf("[runAll] waitTerminationSignal: %v", waitErr)
			applyOrchestratorExitKeepPolicy(runner, domain.ExitTriggerSignal)
			cancel()
			return
		}
		exitReason.record("signal", s.String())
		// ADR-0035: SIGTERM/SIGINT must not tear down managed services.
		applyOrchestratorExitKeepPolicy(runner, domain.ExitTriggerSignal)
		// OPT-20260820-006: log our own process coordinates so a silent SIGTERM
		// (cron/agent/hot-replace) can be attributed from ps alone. si_pid is not
		// available on the signal.Notify path (see waitTerminationSignal).
		pid := os.Getpid()
		pgid, pgidErr := syscall.Getpgid(pid)
		sid, sidErr := getSessionID(pid)
		log.Printf("[runAll] received termination signal %s — exiting orchestrator (pid=%d ppid=%d pgid=%d sid=%d pgidErr=%v sidErr=%v skipShutdownServices=%v)",
			formatTerminationSignalLog(s, origin), pid, os.Getppid(), pgid, sid, pgidErr, sidErr, runner.skipShutdownServices)
		cancel()
	}()

	switch strings.TrimSpace(*command) {
	case "doctor":
		os.Exit(RunDoctor(ctx, runner, os.Stdout))
	case "takeover":
		if strings.TrimSpace(*serviceName) == "" {
			log.Printf("[runAll] takeover requires -service")
			os.Exit(doctorExitPreflightFailed)
		}
		if strings.TrimSpace(*sessionID) == "" {
			log.Printf("[runAll] takeover requires explicit -session-id")
			os.Exit(doctorExitPreflightFailed)
		}
		if err := runner.TakeoverService(*serviceName, *sessionID); err != nil {
			log.Printf("[runAll] takeover failed: %v", err)
			os.Exit(1)
		}
		return
	case "build-all":
		result, buildErr := runner.BuildAll(ctx)
		if buildErr != nil {
			log.Printf("[runAll] build-all failed: %v", buildErr)
			os.Exit(1)
		}
		if result != nil && len(result.Failed) > 0 {
			log.Printf("[runAll] build-all partial failure: built=%d failed=%v skipped=%v",
				result.Built, result.Failed, result.Skipped)
			os.Exit(1)
		}
		if result != nil {
			log.Printf("[runAll] build-all ok: status=%s built=%d total=%d skipped=%d",
				result.Status, result.Built, result.Total, len(result.Skipped))
		}
		return
	case "run":
		if !*daemon {
			srv := bootstrapUIMode(ctx, store, runner, cfg, *uiPort, cancel, exitReason)
			defer srv.Close()
		}
	default:
		log.Printf("[runAll] unknown command %q, want run|doctor|takeover|build-all|deploy-sync", *command)
		os.Exit(doctorExitPreflightFailed)
	}

	if err := runner.Run(ctx, *daemon); err != nil {
		log.Printf("[runAll] Exiting due to error: %v", err)
		persistOrchestratorExitBestEffort(runner, "error", err.Error(), summarizeServiceLifecycle(store))
		os.Exit(1)
	}
	source, detail := exitReason.get()
	lifecycle := summarizeServiceLifecycle(store)
	log.Printf("[runAll] Exiting. trigger=%s detail=%q lifecycle={%s}",
		source, detail, lifecycle)
	// OPT-20260905-005: durable fingerprint under .runall/ survives console-log truncate.
	persistOrchestratorExitBestEffort(runner, source, detail, lifecycle)
}
