package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

func getBashPath() string {
	// Try exec.LookPath first
	if path, err := exec.LookPath("bash"); err == nil {
		if _, err := os.Stat(path); err == nil {
			log.Printf("[runAll] bash found via LookPath: %s", path)
			return path
		}
		log.Printf("[runAll] LookPath found bash at %s but Stat failed: %v", path, err)
	} else {
		log.Printf("[runAll] exec.LookPath(bash) failed: %v", err)
	}

	// Try common bash locations in order of preference
	bashPaths := []string{
		"/bin/bash",
		"/usr/bin/bash",
		"/usr/local/bin/bash",
		"/opt/homebrew/bin/bash", // macOS with Homebrew
	}

	for _, p := range bashPaths {
		if stat, err := os.Stat(p); err == nil {
			// Check if it's executable
			if stat.Mode()&0111 != 0 {
				log.Printf("[runAll] bash found at %s (executable)", p)
				return p
			}
			log.Printf("[runAll] bash at %s exists but not executable", p)
		}
	}

	// Final fallback - try to find bash using shell command
	// This is a last resort when the above methods fail
	log.Printf("[runAll] WARNING: could not find bash via normal methods, using /bin/bash as fallback")
	return "/bin/bash"
}

type Runner struct {
	cfg                          *Config
	cfgPath                      string // absolute path to the primary config file, used for relative script resolution
	cfgReloadMu                  sync.Mutex
	cfgReloadModTime             time.Time
	cfgReloadSize                int64
	store                        *StatusStore
	levels                       []ExecutionLevel
	processes                    map[string]*exec.Cmd
	mu                           sync.Mutex
	monitors                     map[string]context.CancelFunc
	monitorMu                    sync.Mutex
	logRepository                domain.ServiceLogRepository
	logStatsProvider             domain.ServiceLogStatsProvider
	fileLogSink                  domain.ServiceLogFileSink
	devToolLogRecorder           domain.DevToolLogRecorder
	ownershipRepo                domain.ServiceOwnershipRepository
	ownershipGuard               domain.ServiceOwnershipGuardService
	runtimePrereqProbeRepository domain.ServiceRuntimePrereqProbeRepository
	preflightFn                  func(context.Context, Service) error
	listenerPIDsFn               func(string) ([]int, error)
	stdioLogMu                   sync.Mutex
	stdioLogPaths                map[string]string // child stdout/stderr inherit these files (ADR-0035 SIGPIPE)
	skipShutdownServices         bool              // When true, shutdown only closes runAll itself, not managed services
	hotReplaceMode               bool              // 热替换（shutdown-self）启动：UI 启动后对 adopt-skip 且配置探针的服务自动补 start（OPT-20260812-048）
	collectionHeartbeatCancel    context.CancelFunc
	progressBroadcaster          *ProgressBroadcaster
	activeStartAllRunID          string
	activeStartAllRunIDMu        sync.RWMutex
	activeStopAllRunID           string
	activeStopAllRunIDMu         sync.RWMutex
	activeBuildAllRunID          string
	activeBuildAllRunIDMu        sync.RWMutex
	activeRestartAll             bool
	restartAllPhase              string // "canary" | ""
	activeRestartAllRunID        string
	activeRestartAllMu           sync.RWMutex
	preciseRestartActive         preciseRestartActive // 精准编译重启并发防护（active + runID）
	bulkOpActive                 bulkOpGuard          // 统一 bulk 互斥（OPT-20260807-052）
	dagBootInProgress            int32
	lastFailedHealthReconcile    time.Time
	lastFailedHealthReconcileMu  sync.Mutex
	portProbeCache               map[string]portProbeCacheEntry
	portProbeCacheMu             sync.RWMutex
	listeningPortsSnap           map[string]struct{}
	listeningPortsSnapExpires    time.Time
	listeningPortsSnapMu         sync.RWMutex
	traceShippingMu              sync.RWMutex
	traceShippingLast            *domain.TraceShippingReport
	internalAPIsSmokeMu          sync.RWMutex
	internalAPIsSmokeLast        *InternalAPIsSmokeReport
	zombieRestartCount           map[string]int
	zombieRestartMu              sync.Mutex
	execQueue                    execQueueState // 页面刷新后下发的执行队列（当前进度 + 排队）
	migrateDirResolver           domain.DataMigrateDirResolver
	migrateSQLLister             domain.LocalSQLLister
	migrateAppliedReader         domain.AppliedStepReader
	migratePendingCache          migratePendingCache
}

type portProbeCacheEntry struct {
	active  bool
	expires time.Time
}

// ErrStartSkippedAlreadyHealthy indicates start was skipped because the service is already running.
var ErrStartSkippedAlreadyHealthy = errors.New("start skipped: already running")

// SetConfigPath stores the absolute path to the primary configuration file,
// used for resolving relative paths to scripts and resources.
func (r *Runner) SetConfigPath(path string) {
	r.cfgPath = path
}

const (
	defaultOwnershipSessionID        = "runall-bootstrap"
	defaultOwnershipConfigRef        = "runtime-managed"
	failedHealthReconcileMinInterval = 10 * time.Second
	statusPollInterval               = 2 * time.Second
	portProbeCacheTTL                = statusPollInterval
	statusPollIntervalMs             = int(statusPollInterval / time.Millisecond)
)

type actorSessionContextKey struct{}

// CascadeFailure attaches partial cascade progress to an error for API responses.
type CascadeFailure struct {
	Err    error
	Report domain.CascadeExecutionReport
}

func (e *CascadeFailure) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *CascadeFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type noopRuntimePrereqProbeRepository struct{}

func (noopRuntimePrereqProbeRepository) Probe(service domain.ManagedService) error {
	_ = service
	return nil
}

type runnerRuntimeContextRepository struct {
	runner *Runner
}

func (r *runnerRuntimeContextRepository) FindByName(name string) (domain.ManagedService, error) {
	svc := r.runner.findService(name)
	if svc == nil {
		return domain.ManagedService{}, fmt.Errorf("service %q not found", name)
	}
	status := r.runner.store.Get(name)
	if status == nil {
		return domain.ManagedService{}, fmt.Errorf("service %q not found", name)
	}
	return domain.NewManagedService(svc.Name, status.Group, string(status.Status), svc.DependsOn)
}

func (r *runnerRuntimeContextRepository) Save(service domain.ManagedService) error {
	r.runner.store.Update(service.Name, Status(service.Status), "")
	return nil
}

func (r *runnerRuntimeContextRepository) ListByGroup(groupName string) ([]domain.ManagedService, error) {
	var result []domain.ManagedService
	for _, group := range r.runner.cfg.Groups {
		if group.Name != groupName {
			continue
		}
		for _, svc := range group.Services {
			status := r.runner.store.Get(svc.Name)
			if status == nil {
				continue
			}
			managed, err := domain.NewManagedService(svc.Name, group.Name, string(status.Status), svc.DependsOn)
			if err != nil {
				return nil, err
			}
			result = append(result, managed)
		}
	}
	return result, nil
}

func (r *runnerRuntimeContextRepository) ListAll() ([]domain.ManagedService, error) {
	services := r.runner.cfg.Flatten()
	result := make([]domain.ManagedService, 0, len(services))
	for _, svc := range services {
		status := r.runner.store.Get(svc.Name)
		if status == nil {
			continue
		}
		managed, err := domain.NewManagedService(svc.Name, status.Group, string(status.Status), svc.DependsOn)
		if err != nil {
			return nil, err
		}
		result = append(result, managed)
	}
	return result, nil
}

func NewRunner(cfg *Config, store *StatusStore) (*Runner, error) {
	cfg.fillDefaults()

	services := cfg.Flatten()
	names := make([]string, len(services))
	for i, svc := range services {
		names[i] = svc.Name
	}
	store.Init(names)

	// Populate command and URL in store for UI display
	for _, svc := range services {
		store.SetCommand(svc.Name, svc.Command)
		store.SetURL(svc.Name, svc.HealthCheck.DisplayEndpoint())
		healthPort := domain.ResolveHealthPort(svc.HealthCheck.URL)
		if healthPort == "" {
			healthPort = domain.ResolveTCPPort(svc.HealthCheck.TCP)
		}
		store.SetHealthPort(svc.Name, healthPort)
		store.SetCommandPort(svc.Name, domain.ResolveCommandPort(svc.Command))
	}
	for _, group := range cfg.Groups {
		for _, svc := range group.Services {
			store.SetGroup(svc.Name, group.Name)
		}
	}

	// Build dependency status references
	for _, svc := range services {
		deps := make([]DepStatus, len(svc.DependsOn))
		for i, depName := range svc.DependsOn {
			deps[i] = DepStatus{Name: depName, Status: StatusPending}
		}
		store.SetDependsOn(svc.Name, deps)
	}

	levels, err := BuildDAG(services)
	if err != nil {
		return nil, err
	}

	ownershipRepo := infrastructure.NewInMemoryServiceOwnershipRepository()

	logRepo := domain.ServiceLogRepository(infrastructure.NewInMemoryServiceLogRepository(infrastructure.DefaultServiceLogCapacity))
	var fileLogSink domain.ServiceLogFileSink
	if fileRoot := strings.TrimSpace(cfg.Logging.FileRoot); fileRoot != "" {
		if err := os.Setenv("RUNALL_LOG_ROOT", fileRoot); err != nil {
			log.Printf("[runAll] warning: set RUNALL_LOG_ROOT: %v", err)
		}
		fileSink, err := infrastructure.NewFileServiceLogSink(fileRoot)
		if err != nil {
			return nil, fmt.Errorf("file log sink: %w", err)
		}
		fileLogSink = fileSink
		logRepo = infrastructure.NewTeeServiceLogRepository(logRepo, fileSink)
		log.Printf(
			"[runAll] centralized logs: tee %s → Promtail (ai-monitor) → Loki %s → Grafana %s",
			fileRoot,
			strings.TrimSpace(cfg.Observability.LokiURL),
			strings.TrimSpace(cfg.Observability.GrafanaURL),
		)
	}

	// Register per-service log file paths (from conf_app or runAll.yaml overrides).
	if fileSink, ok := fileLogSink.(*infrastructure.FileServiceLogSink); ok {
		for _, svc := range cfg.Flatten() {
			logFile := resolveServiceLogFile(cfg, svc.Name, svc.LogFile)
			if logFile != "" {
				fileSink.SetServicePath(svc.Name, logFile)
			}
		}
	}

	// Wrap in metered repository to track per-service log collection statistics.
	meteredRepo := infrastructure.NewMeteredServiceLogRepository(logRepo)

	runner := &Runner{
		cfg:                          cfg,
		store:                        store,
		levels:                       levels,
		processes:                    make(map[string]*exec.Cmd),
		stdioLogPaths:                make(map[string]string),
		monitors:                     make(map[string]context.CancelFunc),
		logRepository:                meteredRepo,
		logStatsProvider:             meteredRepo,
		fileLogSink:                  fileLogSink,
		ownershipRepo:                ownershipRepo,
		ownershipGuard:               domain.NewServiceOwnershipGuardService(ownershipRepo),
		runtimePrereqProbeRepository: noopRuntimePrereqProbeRepository{},
		listenerPIDsFn:               listenerPIDs,
		progressBroadcaster:          NewProgressBroadcaster(),
		zombieRestartCount:           make(map[string]int),
	}
	seedPendingLifecycleLogs(runner, names)
	return runner, nil
}

func seedPendingLifecycleLogs(runner *Runner, names []string) {
	if runner == nil {
		return
	}
	for _, name := range names {
		runner.appendLifecycleLog(
			name,
			"default state: not started — no process launched yet (use 启动/全部启动)",
		)
	}
}

func withActorSessionID(ctx context.Context, actorSessionID string) context.Context {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return ctx
	}
	return context.WithValue(ctx, actorSessionContextKey{}, actor)
}

func actorSessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw := ctx.Value(actorSessionContextKey{})
	actor, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(actor)
}

// logDAGTopology prints the full DAG topology at startup for diagnosis of
// race conditions and dependency ordering issues. Groups are organizational
// labels and do NOT define execution boundaries — only per-service depends_on
// controls the DAG topology.
func (r *Runner) logDAGTopology() {
	totalSvcs := 0
	for _, level := range r.levels {
		totalSvcs += len(level.Services)
	}
	log.Printf("[runAll] DAG topology: %d levels, %d services total", len(r.levels), totalSvcs)
	for i, level := range r.levels {
		for _, node := range level.Services {
			svc := node.Service
			group := "?"
			if s := r.store.Get(svc.Name); s != nil && s.Group != "" {
				group = s.Group
			}
			deps := "none"
			if len(svc.DependsOn) > 0 {
				deps = strings.Join(svc.DependsOn, ", ")
			}
			log.Printf("[runAll]   L%d [%s] %s ← (%s)", i+1, group, svc.Name, deps)
		}
	}
}

func (r *Runner) Run(ctx context.Context, daemon bool) error {
	resilient := !daemon
	r.logDAGTopology()

	if daemon {
		r.setDAGBootInProgress(true)
		for _, level := range r.levels {
			if err := r.executeLevel(ctx, level, resilient); err != nil {
				r.setDAGBootInProgress(false)
				return err
			}
		}
		r.setDAGBootInProgress(false)
		r.logDAGBootFinished()

		// Start continuous health monitoring for all services.
		services := r.cfg.Flatten()
		for _, svc := range services {
			s := r.store.Get(svc.Name)
			if s != nil && s.Status == StatusHealthy {
				r.startMonitoring(ctx, svc)
			}
		}

		log.Println("[runAll] Daemon mode: exiting.")
		return nil
	}

	log.Println("[runAll] UI mode: services idle — use Web UI「全部启动」 or POST /api/start-all to start.")
	// Hot-replace (shutdown-self) leaves managed processes alive; adopt listeners so UI is not blank.
	adopted := r.adoptListeningServices(ctx)
	if adopted > 0 {
		log.Printf("[runAll] adopted %d already-listening service(s) into StatusStore", adopted)
	}
	// OPT-20260812-048：热替换后 adopt 只回填存活服务，替换前已 down 且配置了探针的服务
	// 不会自动重启；此处按 DAG 依赖序自动补 start，避免部署 runAll 后服务掉线需手工 start-all。
	if r.hotReplaceMode {
		if compensated := r.compensateHotReplaceDownServices(ctx); compensated > 0 {
			log.Printf("[runAll] hot-replace compensation: auto-started %d down service(s)", compensated)
		}
	}
	log.Println("[runAll] Running. Ctrl+C exits the orchestrator; managed services keep running (stop-all or RUNALL_SHUTDOWN_SERVICES=1 to tear down).")
	applyOrchestratorExitKeepPolicy(r, domain.ExitTriggerSignal)
	<-ctx.Done()
	log.Println("[runAll] Shutting down orchestrator...")
	r.stopAllMonitors()

	// Only shutdown managed services if skipShutdownServices is false
	if !r.skipShutdownServices {
		r.Shutdown()
	} else {
		log.Println("[runAll] Skipping managed services shutdown (orchestrator exit keeps services; ADR-0035)")
	}
	return nil
}

// SetSkipShutdownServices sets the flag to skip shutting down managed services when runAll exits.
func (r *Runner) SetSkipShutdownServices(skip bool) {
	r.skipShutdownServices = skip
}

func (r *Runner) executeLevel(ctx context.Context, level ExecutionLevel, resilient bool) error {
	levelCtx, levelCancel := context.WithCancel(ctx)
	defer levelCancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(level.Services))
	exitFailed := make(chan struct{})
	var exitOnce sync.Once

	abortPeers := func() {
		if resilient {
			return
		}
		exitOnce.Do(func() { close(exitFailed); levelCancel() })
	}

	for _, node := range level.Services {
		// Check if any dependency was skipped/failed
		skip := false
		for _, depName := range node.Service.DependsOn {
			depStatus := r.store.Get(depName)
			if depStatus.Status == StatusFailed || depStatus.Status == StatusSkipped {
				r.store.Update(node.Service.Name, StatusSkipped, fmt.Sprintf("dependency %s is %s", depName, depStatus.Status))
				log.Printf("[%s] SKIPPED: dependency %s is %s", node.Service.Name, depName, depStatus.Status)
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		wg.Add(1)
		go func(node *ServiceNode) {
			defer wg.Done()
			if !resilient {
				select {
				case <-exitFailed:
					return
				default:
				}
			}
			if err := r.runPreflight(levelCtx, node.Service); err != nil {
				current := r.store.Get(node.Service.Name)
				if current == nil || current.Status != StatusFailed {
					r.store.Update(node.Service.Name, StatusFailed, fmt.Sprintf("preflight failed: %v", err))
				}
				r.store.UpdateDependencyStatus(node.Service.Name, StatusFailed)
				if !resilient && node.Service.OnFailure == "exit" {
					errCh <- err
					abortPeers()
				}
				return
			}
			if err := r.waitForDependencyReadiness(levelCtx, node.Service); err != nil {
				r.store.Update(node.Service.Name, StatusSkipped, err.Error())
				log.Printf("[%s] SKIPPED: %v", node.Service.Name, err)
				return
			}
			if err := r.startAndCheck(levelCtx, node); err != nil {
				if !resilient && node.Service.OnFailure == "exit" {
					errCh <- err
					abortPeers()
				}
			}
		}(node)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var firstErr error
	for err := range errCh {
		if firstErr == nil {
			firstErr = err
		}
	}

	// Strict mode: any exit-type failure stops everything.
	if firstErr != nil {
		r.Shutdown()
		return firstErr
	}

	return nil
}

func (r *Runner) runPreflight(ctx context.Context, svc Service) error {
	if r.preflightFn != nil {
		return r.preflightFn(ctx, svc)
	}
	if err := r.preflightService(ctx, svc); err != nil {
		return err
	}
	return r.probeRuntimePrerequisite(svc)
}

func (r *Runner) probeRuntimePrerequisite(svc Service) error {
	if r.runtimePrereqProbeRepository == nil {
		return nil
	}

	status := string(StatusPending)
	if current := r.store.Get(svc.Name); current != nil {
		status = string(current.Status)
	}
	managedService, err := domain.NewManagedService(svc.Name, "", status, svc.DependsOn)
	if err != nil {
		return fmt.Errorf("[%s] build runtime prereq context: %w", svc.Name, err)
	}

	if err := r.runtimePrereqProbeRepository.Probe(managedService); err != nil {
		msg := fmt.Sprintf("%s: %v", domain.ServiceFailureCodeRuntimePrereq, err)
		r.store.RecordPreflightFailure(svc.Name, domain.ServiceFailureCodeRuntimePrereq, msg)
		return fmt.Errorf("[%s] %s", svc.Name, msg)
	}
	return nil
}

func (r *Runner) startAndCheck(ctx context.Context, node *ServiceNode) error {
	svc := node.Service

	// CAS guard: prevent concurrent launches from DAG boot and API /start-all.
	// Callers that already transitioned the service (startService → Starting,
	// restartService → Restarting) proceed directly without double-CAS.
	current := r.store.Get(svc.Name)
	alreadyTransitionedByCaller := startupAlreadyClaimed(current, node.forceFreshStart)
	if !alreadyTransitionedByCaller {
		if previousStatus, ok := r.transitionServiceToStarting(svc.Name); !ok {
			latest := r.store.Get(svc.Name)
			if latest != nil && (latest.Status == StatusStarting ||
				latest.Status == StatusRestarting) {
				log.Printf("[%s] another goroutine is starting this service, waiting...", svc.Name)
				r.appendLifecycleLog(svc.Name, "another goroutine is starting this service, waiting for completion")
				return r.waitForServiceStart(ctx, svc)
			}
			return fmt.Errorf("[%s] service is %s, cannot start", svc.Name, func() string {
				if latest != nil {
					return string(latest.Status)
				}
				return "unknown"
			}())
		} else {
			log.Printf("[%s] CAS %s → starting acquired", svc.Name, previousStatus)
		}
	} else {
		log.Printf("[%s] already in %s state (pre-transitioned by caller)", svc.Name, current.Status)
	}

	// Check if the service port is already being listened to before starting
	ports := resolveServicePorts(&svc)
	if len(ports) > 0 {
		for _, port := range ports {
			pids, err := r.listenerPIDsForPort(port)
			if err != nil {
				log.Printf("[%s] port check error on %s: %v", svc.Name, port, err)
				r.store.Update(svc.Name, StatusFailed,
					fmt.Sprintf("port check failed on %s: %v", port, err))
				r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
				return fmt.Errorf("[%s] port check error on %s: %w", svc.Name, port, err)
			}
			if len(pids) > 0 {
				log.Printf("[%s] port %s is already being listened to (PID=%v)", svc.Name, port, pids)

				if node.allowOverlapStart {
					node.overlapHadListeners = true
					log.Printf("[%s] canary overlap: port %s occupied by PID=%v, starting peer listener", svc.Name, port, pids)
					r.appendLifecycleLog(svc.Name, fmt.Sprintf("金丝雀：端口 %s 仍由 PID=%v 监听，重叠启动", port, pids))
					continue
				}

				if node.forceFreshStart {
					// restart 上下文（kill-first，OPT-20260810-005）：旧进程必须已死，
					// 任何存活监听者都是停止未完成的残留（含 crontab watchdog 在
					// 构建窗口抢拉的副本）。主动终止残留 → 等待端口释放 → 再启动
					// 新二进制；绝不「跳过启动」，也绝不在端口仍占用时强行启动
					// （后者会 EADDRINUSE，并把本已健康的残留实例误标 failed）。
					log.Printf("[%s] restart: port %s still occupied by PID=%v after stop, terminating residual listeners", svc.Name, port, pids)
					r.appendLifecycleLog(svc.Name, fmt.Sprintf("重启：端口 %s 停止后仍被 PID=%v 占用，终止残留监听", port, pids))
					if termErr := r.terminatePortListeners(port); termErr != nil {
						log.Printf("[%s] terminate residual listeners on port %s: %v", svc.Name, port, termErr)
					}
					if err := r.waitForPortFree(port, servicePortReleaseWait); err != nil {
						msg := fmt.Sprintf("port %s still occupied after terminate+wait (PID residual): %v", port, err)
						log.Printf("[%s] %s", svc.Name, msg)
						r.appendLifecycleLog(svc.Name, fmt.Sprintf("端口 %s %v 未释放，放弃启动", port, servicePortReleaseWait))
						r.store.RecordFailure(
							svc.Name,
							domain.ServiceLifecyclePhaseLaunch,
							domain.ServiceFailureCodeProcessExited,
							msg,
						)
						r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
						return fmt.Errorf("[%s] %s", svc.Name, msg)
					}
					continue
				}

				// If health check passes, the service is genuinely running — skip start.
				if err := checkProbe(ctx, svc.HealthCheck); err == nil {
					log.Printf("[%s] port %s listening and healthy (PID=%v), skipping start", svc.Name, port, pids)
					r.appendLifecycleLog(svc.Name, fmt.Sprintf("服务端口 %s 已被监听且健康 (PID=%v)，跳过启动", port, pids))
					discoveredPID := r.discoverRunningPIDFromListeners(&svc)
					if discoveredPID > 0 {
						r.store.SetPID(svc.Name, discoveredPID)
						r.establishServiceOwnership(svc, discoveredPID, actorSessionIDFromContext(ctx))
					}
					r.store.Update(svc.Name, StatusHealthy, "")
					r.probeReadiness(ctx, svc)
					r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
					return nil
				}

				// Health check failed — old process is likely shutting down or hung.
				// Terminate residual listeners, wait for release; still occupied → fail
				// (do not force-start into a busy port → EADDRINUSE).
				log.Printf("[%s] port %s occupied but unhealthy (PID=%v), terminating residual listeners", svc.Name, port, pids)
				r.appendLifecycleLog(svc.Name, fmt.Sprintf("端口 %s 被占用但健康检查失败 (PID=%v)，终止残留监听", port, pids))
				if termErr := r.terminatePortListeners(port); termErr != nil {
					log.Printf("[%s] terminate residual listeners on port %s: %v", svc.Name, port, termErr)
				}
				if err := r.waitForPortFree(port, 30*time.Second); err != nil {
					msg := fmt.Sprintf("port %s still occupied after terminate+wait: %v", port, err)
					log.Printf("[%s] %s", svc.Name, msg)
					r.appendLifecycleLog(svc.Name, fmt.Sprintf("端口 %s 30s 未释放，放弃启动", port))
					r.store.RecordFailure(
						svc.Name,
						domain.ServiceLifecyclePhaseLaunch,
						domain.ServiceFailureCodeProcessExited,
						msg,
					)
					r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
					return fmt.Errorf("[%s] %s", svc.Name, msg)
				}
				// Fall through to normal start below.
			}
		}
	}

	cmd := exec.Command(getBashPath(), "-c", svc.EffectiveStartCommand())
	cmd.SysProcAttr = managedServiceSysProcAttr()

	startDir, dirErr := startCommandDir(&svc)
	if dirErr != nil {
		r.store.RecordFailure(
			svc.Name,
			domain.ServiceLifecyclePhaseLaunch,
			domain.ServiceFailureCodeProcessExited,
			dirErr.Error(),
		)
		r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
		if svc.OnFailure == "exit" {
			return fmt.Errorf("[%s] failed to start: %w", svc.Name, dirErr)
		}
		log.Printf("[%s] failed to start, skipping: %v", svc.Name, dirErr)
		return nil
	}
	if startDir != "" {
		cmd.Dir = startDir
	}
	cmd.Env = r.buildServiceEnv(svc)
	if node.allowOverlapStart {
		cmd.Env = append(cmd.Env, runallCanaryOverlapEnv+"=1")
	}

	stdioPath, err := r.prepareManagedStdioLogPath(svc)
	if err != nil {
		r.store.RecordFailure(
			svc.Name,
			domain.ServiceLifecyclePhaseLaunch,
			domain.ServiceFailureCodeProcessExited,
			err.Error(),
		)
		r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
		if svc.OnFailure == "exit" {
			return fmt.Errorf("[%s] stdio log path: %w", svc.Name, err)
		}
		log.Printf("[%s] stdio log path error, skipping: %v", svc.Name, err)
		return nil
	}
	stdioFile, err := openManagedServiceStdioFile(stdioPath)
	if err != nil {
		r.store.RecordFailure(
			svc.Name,
			domain.ServiceLifecyclePhaseLaunch,
			domain.ServiceFailureCodeProcessExited,
			err.Error(),
		)
		r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
		if svc.OnFailure == "exit" {
			return fmt.Errorf("[%s] stdio log file: %w", svc.Name, err)
		}
		log.Printf("[%s] stdio log file error, skipping: %v", svc.Name, err)
		return nil
	}
	attachManagedServiceStdio(cmd, stdioFile)

	if err := cmd.Start(); err != nil {
		_ = closeParentStdioFile(stdioFile)
		r.store.RecordFailure(
			svc.Name,
			domain.ServiceLifecyclePhaseLaunch,
			domain.ServiceFailureCodeProcessExited,
			err.Error(),
		)
		r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
		if svc.OnFailure == "exit" {
			return fmt.Errorf("[%s] failed to start: %w", svc.Name, err)
		}
		log.Printf("[%s] failed to start, skipping: %v", svc.Name, err)
		return nil
	}
	if err := closeParentStdioFile(stdioFile); err != nil {
		log.Printf("[%s] close parent stdio log fd: %v", svc.Name, err)
	}

	r.mu.Lock()
	r.processes[svc.Name] = cmd
	r.mu.Unlock()
	r.store.SetPID(svc.Name, cmd.Process.Pid)

	return r.settleStartupHealth(ctx, svc, node, cmd)
}

func (r *Runner) probeReadiness(ctx context.Context, svc Service) {
	if !svc.HealthCheck.HasSplitProbe() {
		r.store.SetReadiness(svc.Name, ReadinessReady, "")
		return
	}
	if err := checkHealth(ctx, svc.HealthCheck.ReadinessProbeURL()); err != nil {
		r.store.SetReadiness(svc.Name, ReadinessDegraded, err.Error())
		return
	}
	r.store.SetReadiness(svc.Name, ReadinessReady, "")
}

func (r *Runner) waitForDependencyReadiness(ctx context.Context, svc Service) error {
	for _, depName := range svc.DependsOn {
		dep := r.findService(depName)
		if dep == nil || !dep.HealthCheck.HasSplitProbe() {
			continue
		}
		st := r.store.Get(depName)
		if st == nil || st.Status == StatusSkipped || st.Status == StatusFailed {
			continue
		}
		probe := dep.HealthCheck
		probe.URL = probe.ReadinessProbeURL()
		if err := waitHealthy(ctx, probe); err != nil {
			return fmt.Errorf("dependency %q readiness: %w", depName, err)
		}
		r.store.SetReadiness(depName, ReadinessReady, "")
	}
	return nil
}

func (r *Runner) establishServiceOwnership(svc Service, pid int, actorSessionID string) {
	if r.ownershipRepo == nil || pid <= 0 {
		return
	}
	owner := strings.TrimSpace(actorSessionID)
	if owner == "" {
		owner = defaultOwnershipSessionID
	}
	ownership, err := domain.NewServiceOwnership(
		svc.Name,
		owner,
		pid,
		defaultOwnershipConfigRef,
		svc.HealthCheck.DisplayEndpoint(),
		time.Now(),
	)
	if err != nil {
		log.Printf("[%s] establish ownership skipped: %v", svc.Name, err)
		return
	}
	if err := r.ownershipRepo.Save(ownership); err != nil {
		log.Printf("[%s] persist ownership skipped: %v", svc.Name, err)
	}
}

func waitHealthyWithLaunchCheck(ctx context.Context, process *os.Process, cfg HealthCheck, allowLaunchExit bool) error {
	interval := time.Duration(cfg.Backoff.Initial * float64(time.Second))
	maxInterval := time.Duration(cfg.Backoff.Max * float64(time.Second))
	deadline := time.Now().Add(time.Duration(cfg.Timeout) * time.Second)
	var lastCheckErr error

	for i := 0; i < cfg.Retries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}

		if !allowLaunchExit {
			if exited, exitCode := processExitInfo(process); exited {
				return formatLaunchProcessExitedError(exitCode, lastCheckErr)
			}
		}

		if time.Now().After(deadline) {
			if lastCheckErr != nil {
				return fmt.Errorf("health check timed out after %ds (last error: %v)", cfg.Timeout, lastCheckErr)
			}
			return fmt.Errorf("health check timed out after %ds", cfg.Timeout)
		}

		err := checkProbe(ctx, cfg)
		if err == nil {
			return nil
		}
		lastCheckErr = err
		if !allowLaunchExit {
			if exited, exitCode := processExitInfo(process); exited {
				return formatLaunchProcessExitedError(exitCode, lastCheckErr)
			}
		}

		interval = time.Duration(float64(interval) * cfg.Backoff.Multiplier)
		if interval > maxInterval {
			interval = maxInterval
		}
	}

	if lastCheckErr != nil {
		return fmt.Errorf("health check failed after %d retries: %w", cfg.Retries, lastCheckErr)
	}
	return fmt.Errorf("health check failed after %d retries", cfg.Retries)
}

func formatLaunchProcessExitedError(exitCode int, lastCheckErr error) error {
	if exitCode >= 0 && lastCheckErr != nil {
		return fmt.Errorf("launch process exited before readiness (exit=%d; last error: %v)", exitCode, lastCheckErr)
	}
	if exitCode >= 0 {
		return fmt.Errorf("launch process exited before readiness (exit=%d)", exitCode)
	}
	if lastCheckErr != nil {
		return fmt.Errorf("launch process exited before readiness (last error: %v)", lastCheckErr)
	}
	return fmt.Errorf("launch process exited before readiness")
}

const (
	startupFailureLogTailLines = 12
	startupFailureLogWait      = 200 * time.Millisecond
	startupFailureLogPoll      = 20 * time.Millisecond
	startupFailureLogMaxChars  = 900
)

// looksLikeStartupFatal reports whether a log line likely explains process exit
// (DB corruption, panic, JSON level=error, etc.). Used when stderr only has
// confload/config noise while the real fatal landed on stdout.
func looksLikeStartupFatal(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if lower == "" {
		return false
	}
	for _, needle := range []string{
		`"level":"error"`,
		`"level": "error"`,
		"fatal",
		"panic:",
		"traceback",
		"malformed",
		"syntaxerror",
		"taberror",
		"address already in use",
		"no such file",
		"permission denied",
		"database disk image",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func anyLooksLikeStartupFatal(lines []string) bool {
	for _, line := range lines {
		if looksLikeStartupFatal(line) {
			return true
		}
	}
	return false
}

// enrichStartupFailureWithRecentLogs appends recent service log lines (prefer stderr)
// so operators see TabError/SyntaxError text instead of only connection-refused noise.
// If stderr is present but looks like non-fatal noise (e.g. confload), prefer any-stream
// lines that look like startup fatals (stdout JSON error / malformed DB).
func (r *Runner) enrichStartupFailureWithRecentLogs(serviceName string, err error) error {
	if err == nil || r == nil || r.logRepository == nil {
		return err
	}
	msg := err.Error()
	if !strings.Contains(msg, "launch process exited before readiness") &&
		!strings.Contains(msg, "health check timed out") &&
		!strings.Contains(msg, "health check failed") {
		return err
	}
	if strings.Contains(msg, "recent stderr:") || strings.Contains(msg, "recent logs:") {
		return err
	}

	preferStderr := strings.Contains(msg, "launch process exited before readiness")
	if preferStderr {
		deadline := time.Now().Add(startupFailureLogWait)
		for {
			if lines := r.recentStreamMessages(serviceName, domain.StreamStderr, startupFailureLogTailLines); len(lines) > 0 {
				if anyLooksLikeStartupFatal(lines) {
					return fmt.Errorf("%w; recent stderr: %s", err, joinLogSnippet(lines, startupFailureLogMaxChars))
				}
				if fatalLines := r.recentFatalMessages(serviceName, startupFailureLogTailLines); len(fatalLines) > 0 {
					return fmt.Errorf("%w; recent logs: %s", err, joinLogSnippet(fatalLines, startupFailureLogMaxChars))
				}
				return fmt.Errorf("%w; recent stderr: %s", err, joinLogSnippet(lines, startupFailureLogMaxChars))
			}
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(startupFailureLogPoll)
		}
	}

	lines := r.recentStreamMessages(serviceName, domain.StreamStderr, startupFailureLogTailLines)
	label := "recent stderr"
	if len(lines) == 0 || !anyLooksLikeStartupFatal(lines) {
		if fatalLines := r.recentFatalMessages(serviceName, startupFailureLogTailLines); len(fatalLines) > 0 {
			lines = fatalLines
			label = "recent logs"
		} else if len(lines) == 0 {
			lines = r.recentStreamMessages(serviceName, "", startupFailureLogTailLines)
			label = "recent logs"
		}
	}
	if len(lines) == 0 {
		return err
	}
	return fmt.Errorf("%w; %s: %s", err, label, joinLogSnippet(lines, startupFailureLogMaxChars))
}

// recentFatalMessages returns recent log lines (any stream) that look like startup fatals,
// in chronological order (oldest → newest), capped at limit newest matches.
func (r *Runner) recentFatalMessages(serviceName string, limit int) []string {
	all := r.recentStreamMessages(serviceName, "", limit*4)
	if len(all) == 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for i := len(all) - 1; i >= 0; i-- {
		if looksLikeStartupFatal(all[i]) {
			out = append(out, all[i])
			if len(out) >= limit {
				break
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func (r *Runner) recentStreamMessages(serviceName, stream string, limit int) []string {
	if r == nil || limit <= 0 {
		return nil
	}
	if fileLines := r.recentStdioFileMessages(serviceName, stream, limit); len(fileLines) > 0 {
		return fileLines
	}
	if r.logRepository == nil {
		return nil
	}
	entries := r.logRepository.Tail(serviceName, limit*4)
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if stream != "" && entry.Stream != stream {
			continue
		}
		msg := strings.TrimSpace(entry.Message)
		if msg == "" {
			continue
		}
		out = append(out, msg)
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	// reverse to chronological order
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func joinLogSnippet(lines []string, maxChars int) string {
	snippet := strings.Join(lines, " | ")
	if maxChars <= 0 || len(snippet) <= maxChars {
		return snippet
	}
	if maxChars < 4 {
		return snippet[len(snippet)-maxChars:]
	}
	return "..." + snippet[len(snippet)-(maxChars-3):]
}

// maxZombieRestarts is the maximum number of automatic restarts for a failed
// service within a single runAll session to prevent restart loops. The name is
// kept for back-compat: since OPT-20260902-026 the guard also covers alive-but-
// unhealthy processes, not only zombies.
const maxZombieRestarts = 3

// tryAutoRestartFailedService restarts a service runAll itself launched that has
// failed its health check `UnhealthyThreshold` consecutive times, whether the
// tracked process has exited (zombie) or is still alive but no longer serving
// (e.g. precise-restart compiled a new binary but a stale process kept the port
// and died shortly after the swap). Bounded by a per-session cap so a genuinely
// broken binary cannot restart-loop; services without a runAll process handle
// (external / docker-compose managed) are left to the cron watchdog.
//
// Must be called while the service is already marked StatusFailed.
func (r *Runner) tryAutoRestartFailedService(svc Service) {
	r.zombieRestartMu.Lock()
	count := r.zombieRestartCount[svc.Name]
	if count >= maxZombieRestarts {
		r.zombieRestartMu.Unlock()
		log.Printf("[%s] auto-restart skipped: reached %d attempts this session", svc.Name, maxZombieRestarts)
		return
	}
	r.zombieRestartCount[svc.Name] = count + 1
	r.zombieRestartMu.Unlock()

	r.mu.Lock()
	cmd := r.processes[svc.Name]
	r.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		log.Printf("[%s] auto-restart skipped: no tracked process handle", svc.Name)
		return
	}

	exited, code := processExitInfo(cmd.Process)
	if exited {
		log.Printf("[%s] process exited (code=%d), auto-restart attempt %d/%d",
			svc.Name, code, count+1, maxZombieRestarts)
	} else {
		log.Printf("[%s] unhealthy (process alive, health check failing), auto-restart attempt %d/%d",
			svc.Name, count+1, maxZombieRestarts)
	}

	go func() {
		ctx := context.Background()
		if err := r.restartService(ctx, svc.Name); err != nil {
			log.Printf("[%s] auto-restart failed: %v", svc.Name, err)
			return
		}
		log.Printf("[%s] auto-restart succeeded", svc.Name)
		r.zombieRestartMu.Lock()
		r.zombieRestartCount[svc.Name] = 0
		r.zombieRestartMu.Unlock()
	}()
}

// processHasExited reports whether the launch process is no longer running.
// On Linux, an exited-but-unreaped child remains a zombie; Signal(0) still
// succeeds for zombies, so we non-blocking Wait4 first to detect/reap them.
func processHasExited(process *os.Process) bool {
	exited, _ := processExitInfo(process)
	return exited
}

func processExitInfo(process *os.Process) (exited bool, exitCode int) {
	if process == nil {
		return true, -1
	}
	var status syscall.WaitStatus
	wpid, err := syscall.Wait4(process.Pid, &status, syscall.WNOHANG, nil)
	if wpid == process.Pid {
		if status.Exited() {
			return true, status.ExitStatus()
		}
		if status.Signaled() {
			return true, 128 + int(status.Signal())
		}
		return true, -1
	}
	if err == syscall.ECHILD {
		return true, -1
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return true, -1
	}
	return false, 0
}

func classifyStartupFailure(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "", ""
	}
	msg := err.Error()
	if strings.Contains(msg, "launch process exited before readiness") {
		return domain.ServiceLifecyclePhaseLaunch, domain.ServiceFailureCodeProcessExited
	}
	if strings.Contains(msg, "unhealthy: HTTP") {
		return domain.ServiceLifecyclePhaseReadiness, domain.ServiceFailureCodeBadReadiness
	}
	return domain.ServiceLifecyclePhaseReadiness, domain.ServiceFailureCodeReadinessTimeout
}

func (r *Runner) Shutdown() {
	r.stopAllMonitors()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Shutdown in reverse order
	for i := len(r.levels) - 1; i >= 0; i-- {
		level := r.levels[i]
		var wg sync.WaitGroup
		for _, node := range level.Services {
			cmd, ok := r.processes[node.Service.Name]
			if !ok || cmd.Process == nil {
				continue
			}
			wg.Add(1)
			go func(name string, cmd *exec.Cmd) {
				defer wg.Done()
				log.Printf("[%s] sending SIGTERM", name)
				syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)

				done := make(chan struct{})
				go func() {
					cmd.Wait()
					close(done)
				}()
				select {
				case <-done:
					log.Printf("[%s] stopped", name)
				case <-time.After(5 * time.Second):
					log.Printf("[%s] did not stop, sending SIGKILL", name)
					syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
					cmd.Wait()
				}
			}(node.Service.Name, cmd)
		}
		wg.Wait()
	}
}

func (r *Runner) RestartService(ctx context.Context, name string) error {
	return r.restartService(ctx, name)
}

func (r *Runner) TakeoverService(name string, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required for explicit takeover")
	}

	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}
	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}
	existingOwnership, err := r.ownershipRepo.FindByServiceName(name)
	if err != nil {
		return err
	}
	pid := current.PID
	if pid <= 0 {
		pid = r.discoverRunningPIDFromListeners(svc)
	}
	usedResidualOwnershipForStoppedOrFailed := pid <= 0 &&
		(current.Status == StatusStopped || current.Status == StatusFailed) &&
		existingOwnership.ServiceName != "" && existingOwnership.PID > 0
	if usedResidualOwnershipForStoppedOrFailed {
		pid = existingOwnership.PID
	}
	if pid <= 0 {
		return fmt.Errorf("service %q has no running process to take over", name)
	}

	ownership, err := domain.NewServiceOwnership(
		name,
		actor,
		pid,
		"takeover",
		svc.HealthCheck.DisplayEndpoint(),
		time.Now(),
	)
	if err != nil {
		return err
	}
	if err := r.ownershipRepo.Save(ownership); err != nil {
		return err
	}
	if usedResidualOwnershipForStoppedOrFailed && current.Status == StatusFailed {
		r.store.Update(name, StatusStopped, "")
		r.store.UpdateDependencyStatus(name, StatusStopped)
	}
	return nil
}

func (r *Runner) discoverRunningPIDFromListeners(svc *Service) int {
	if svc == nil {
		return 0
	}
	lookup := r.listenerPIDsFn
	if lookup == nil {
		lookup = listenerPIDs
	}
	var candidates []int
	for _, port := range resolveServicePorts(svc) {
		pids, err := lookup(port)
		if err != nil {
			continue
		}
		for _, pid := range pids {
			if pid > 0 {
				candidates = append(candidates, pid)
			}
		}
	}
	if len(candidates) == 0 {
		return 0
	}
	sort.Ints(candidates)
	return candidates[0]
}

func (r *Runner) RestartServiceWithActor(ctx context.Context, name string, actorSessionID string) error {
	// Align with cascade start/stop: when the initiating actor is not the owner,
	// delegate to the registered owner session so UI/API restart works without
	// an explicit takeover (same semantics as resolveCascadeStepActor).
	effectiveActor := r.resolveCascadeStepActor(name, actorSessionID)
	if err := r.ownershipGuard.EnsureOperableBySession(name, effectiveActor); err != nil {
		return err
	}
	return r.restartService(withActorSessionID(ctx, effectiveActor), name)
}

func (r *Runner) StopService(ctx context.Context, name string) error {
	return r.stopService(ctx, name)
}

func (r *Runner) StopServiceWithActor(ctx context.Context, name string, actorSessionID string) error {
	// Same as restart/cascade: non-owner UI actors delegate to the registered owner.
	effectiveActor := r.resolveCascadeStepActor(name, actorSessionID)
	if err := r.ownershipGuard.EnsureOperableBySession(name, effectiveActor); err != nil {
		return err
	}
	return r.stopService(ctx, name)
}

func (r *Runner) stopService(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}

	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}
	if current.Status == StatusStopped && !r.probeActivePortListeners(svc) {
		r.appendLifecycleLog(name, "already stopped, skipping")
		return nil
	}

	r.appendLifecycleLog(name, fmt.Sprintf("stop requested (current status=%s)", current.Status))

	managedService, err := domain.NewManagedService(svc.Name, current.Group, string(current.Status), svc.DependsOn)
	if err != nil {
		r.appendLifecycleLog(name, fmt.Sprintf("stop failed: %v", err))
		return err
	}
	policyService := domain.NewServiceStopPolicyService(&runnerRuntimeContextRepository{runner: r})
	decision, err := policyService.EvaluateStop(name)
	if err != nil {
		r.appendLifecycleLog(name, fmt.Sprintf("stop failed: %v", err))
		return err
	}
	activeDependents := mergeDependentNames(decision.ActiveDependents, r.runningDependents(name))
	if err := managedService.CanStop(activeDependents); err != nil {
		r.appendLifecycleLog(name, fmt.Sprintf("stop blocked: %v", err))
		return err
	}

	return r.executeServiceStop(ctx, svc, name, "stop requested")
}

// executeServiceStop performs the common stop sequence shared by all stop variants:
// stop monitoring, run configured stop command, finalize process/port termination,
// update store status/PID/dependency status, and clean up ownership.
//
// logPrefix controls lifecycle logging: when non-empty, start/failure/success
// events are logged with the given prefix (e.g. "stop requested"). An empty
// prefix suppresses lifecycle logging (used by dev database clear operations
// where log noise is undesirable).
func (r *Runner) executeServiceStop(ctx context.Context, svc *Service, name string, logPrefix string) error {
	doLog := func(msg string) {
		if logPrefix != "" {
			r.appendLifecycleLog(name, msg)
		}
	}

	r.stopMonitoring(name)
	if err := r.runStopCommandIfConfigured(ctx, svc); err != nil {
		doLog(fmt.Sprintf("stop command failed: %v", err))
		return err
	}
	if err := r.finalizeServiceStop(ctx, svc); err != nil {
		doLog(fmt.Sprintf("stop finalization failed: %v", err))
		r.store.Update(name, StatusFailed, err.Error())
		r.store.UpdateDependencyStatus(name, StatusFailed)
		return err
	}

	doLog("stopped successfully")
	r.store.SetPID(name, 0)
	r.store.Update(name, StatusStopped, "")
	r.store.UpdateDependencyStatus(name, StatusStopped)
	if r.ownershipRepo != nil {
		if err := r.ownershipRepo.DeleteByServiceName(name); err != nil {
			cleanupErr := fmt.Errorf("ownership cleanup failed: %w", err)
			doLog(fmt.Sprintf("ownership cleanup failed: %v", cleanupErr))
			r.store.Update(name, StatusFailed, cleanupErr.Error())
			r.store.UpdateDependencyStatus(name, StatusFailed)
			return cleanupErr
		}
	}
	return nil
}

// forceStopService stops a service without the CanStop downstream-dependency check.
// Used by StopAll operations where every service is being shut down and the
// dependency ordering is already handled by the stop plan.
func (r *Runner) forceStopService(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}

	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}
	if current.Status == StatusStopped && !r.probeActivePortListeners(svc) {
		r.appendLifecycleLog(name, "already stopped, skipping")
		return nil
	}

	r.appendLifecycleLog(name, fmt.Sprintf("stop-all: stop requested (current status=%s)", current.Status))
	return r.executeServiceStop(ctx, svc, name, "stop-all: stop")
}

// stopServiceForDevDatabaseClear stops a service without CanStop/dependent policy.
// Used by database clear/init operations where all services are shut down
// regardless of dependencies.  Lifecycle logging is suppressed to avoid
// noise during dev tool operations.
func (r *Runner) stopServiceForDevDatabaseClear(ctx context.Context, name string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}
	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}
	if current.Status == StatusStopped && !r.probeActivePortListeners(svc) {
		return nil
	}

	return r.executeServiceStop(ctx, svc, name, "")
}

func (r *Runner) StartService(ctx context.Context, name string) error {
	r.ensureConfigHasServiceBestEffort(name)
	return r.startService(ctx, name)
}

func (r *Runner) StartServiceWithActor(ctx context.Context, name string, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	r.ensureConfigHasServiceBestEffort(name)
	// Same as restart/cascade: non-owner UI actors delegate to the registered owner.
	effectiveActor := r.resolveCascadeStepActor(name, actor)
	if err := r.ownershipGuard.EnsureOperableBySession(name, effectiveActor); err != nil {
		return err
	}
	return r.startService(withActorSessionID(ctx, effectiveActor), name)
}

func (r *Runner) cascadeOrchestration() *domain.ServiceCascadeOrchestrationService {
	return domain.NewServiceCascadeOrchestrationService(
		newConfigServiceTopologyRepository(r.cfg),
		&runnerRuntimeContextRepository{runner: r},
	)
}

func (r *Runner) StartServiceCascade(ctx context.Context, name string) error {
	return r.StartServiceCascadeWithActor(ctx, name, defaultOwnershipSessionID)
}

func (r *Runner) StartServiceCascadeWithActor(ctx context.Context, name string, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	plan, err := r.cascadeOrchestration().PlanStartCascade(name)
	if err != nil {
		return err
	}
	if len(plan.OrderedNames) == 0 {
		current := r.store.Get(name)
		if current != nil && current.Status == StatusHealthy {
			return nil
		}
		return fmt.Errorf("no stopped services to start in cascade for %q", name)
	}
	log.Printf("[cascade] start plan: %s", plan.String())
	r.logLifecyclePlanQueue(plan)
	return r.executeLifecyclePlan(ctx, plan, actor, func(stepCtx context.Context, serviceName, stepActor string) error {
		return r.StartServiceWithActor(stepCtx, serviceName, stepActor)
	})
}

func (r *Runner) StopServiceCascade(ctx context.Context, name string) error {
	return r.StopServiceCascadeWithActor(ctx, name, defaultOwnershipSessionID)
}

func (r *Runner) StopServiceCascadeWithActor(ctx context.Context, name string, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	plan, err := r.cascadeOrchestration().PlanStopCascade(name)
	if err != nil {
		return err
	}
	if len(plan.OrderedNames) == 0 {
		return nil
	}
	log.Printf("[cascade] stop plan: %s", plan.String())
	return r.executeLifecyclePlan(ctx, plan, actor, func(stepCtx context.Context, serviceName, stepActor string) error {
		return r.StopServiceWithActor(stepCtx, serviceName, stepActor)
	})
}

func (r *Runner) StartGroup(ctx context.Context, group string) error {
	return r.StartGroupWithActor(ctx, group, defaultOwnershipSessionID)
}

func (r *Runner) StartGroupWithActor(ctx context.Context, group string, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	if err := r.rejectManualStartDuringDAGBoot(); err != nil {
		return err
	}
	r.reloadRunAllConfigBestEffort()
	plan, err := r.cascadeOrchestration().PlanStartGroup(group)
	if err != nil {
		return err
	}
	if len(plan.OrderedNames) == 0 {
		return nil
	}
	log.Printf("[cascade] start group %q plan: %s", group, plan.String())
	levels, err := r.buildParallelStartLevels(plan.OrderedNames)
	if err != nil {
		return err
	}
	r.logParallelStartPlanPreview(plan.Operation, levels)
	return r.executeParallelStartPlan(ctx, plan.Operation, levels, actor)
}

func (r *Runner) StartAll(ctx context.Context) error {
	return r.StartAllWithActor(ctx, defaultOwnershipSessionID)
}

func (r *Runner) StartAllWithActor(ctx context.Context, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	// OPT-20260901-001: refuse bulk start when the registry still has unapplied
	// dataMigrate SQL (fresh/empty MySQL volume). The operator must first click
	//「初始化全部数据库」so the schema gap does not surface as per-service exit.
	if msg := r.bulkStartMigrationBlock(ctx); msg != "" {
		err := fmt.Errorf("start-all blocked: %s", msg)
		r.publishStartAllTerminalError(err)
		return err
	}
	if err := r.rejectManualStartDuringDAGBoot(); err != nil {
		r.publishStartAllTerminalError(err)
		return err
	}
	r.reloadRunAllConfigBestEffort()
	plan, err := r.cascadeOrchestration().PlanStartAll()
	if err != nil {
		r.publishStartAllTerminalError(err)
		return err
	}
	if len(plan.OrderedNames) == 0 {
		// All services already started or no services configured.
		if runID := r.GetActiveStartAllRunID(); runID != "" {
			log.Printf("[cascade] start all plan: empty (no startable services) run_id=%s", runID)
			if r.progressBroadcaster != nil {
				r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
					Total: 0, Started: 0, Failed: 0, Remaining: 0,
					Done: true, Phase: "done", Operation: "start",
				})
			}
			r.scheduleReleaseStartAllRun(runID)
		}
		return nil
	}
	log.Printf("[cascade] start all plan: %s", plan.String())
	levels, err := r.buildParallelStartLevels(plan.OrderedNames)
	if err != nil {
		r.publishStartAllTerminalError(err)
		return err
	}
	r.logParallelStartPlanPreview(plan.Operation, levels)
	return r.executeParallelStartPlan(ctx, plan.Operation, levels, actor)
}

// StopAll stops all services in reverse dependency order.
func (r *Runner) StopAll(ctx context.Context) error {
	return r.StopAllWithActor(ctx, defaultOwnershipSessionID)
}

// StopAllWithActor stops all services in reverse dependency order using the given actor session.
// Unlike start-all (which aborts on first failure because dependencies won't be ready),
// stop-all continues stopping remaining services even if one fails.
func (r *Runner) StopAllWithActor(ctx context.Context, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	plan, err := r.cascadeOrchestration().PlanStopAll()
	if err != nil {
		r.publishStopAllTerminalError(err)
		return err
	}
	if len(plan.OrderedNames) == 0 {
		if runID := r.GetActiveStopAllRunID(); runID != "" {
			log.Printf("[cascade] stop all plan: empty (no stoppable services) run_id=%s", runID)
			if r.progressBroadcaster != nil {
				r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
					Total: 0, Started: 0, Failed: 0, Remaining: 0,
					Done: true, Phase: "done", Operation: "stop",
				})
			}
			r.scheduleReleaseStopAllRun(runID)
		}
		return nil
	}
	log.Printf("[cascade] stop all plan: %s", plan.String())

	// Use pre-set runID (from SetActiveStopAllRunID) if available, otherwise generate one.
	r.activeStopAllRunIDMu.Lock()
	runID := r.activeStopAllRunID
	if runID == "" {
		runID = fmt.Sprintf("stop-all-%d", time.Now().UnixNano())
	}
	r.activeStopAllRunID = runID
	r.activeStopAllRunIDMu.Unlock()
	defer func() {
		r.activeStopAllRunIDMu.Lock()
		if r.activeStopAllRunID == runID {
			r.activeStopAllRunID = ""
		}
		r.activeStopAllRunIDMu.Unlock()
	}()

	total := len(plan.OrderedNames)
	var stopped, failed int
	var allErrors []string

	emit := func(current string) {
		remaining := total - stopped - failed
		if remaining < 0 {
			remaining = 0
		}
		lastErr := ""
		if len(allErrors) > 0 {
			lastErr = allErrors[len(allErrors)-1]
		}
		r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
			Total: total, Started: stopped, Failed: failed,
			Current: current, Remaining: remaining,
			Done: false, Error: lastErr,
			Errors: append([]string(nil), allErrors...),
			Phase:  "progress", Operation: "stop",
		})
	}

	// Initial event
	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total: total, Remaining: total,
		Phase: "starting", Operation: "stop",
	})

	for _, name := range plan.OrderedNames {
		select {
		case <-ctx.Done():
			r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
				Total: total, Started: stopped, Failed: failed,
				Remaining: total - stopped - failed,
				Done:      true, Phase: progressPhaseForContextErr(ctx),
				Error:     ctx.Err().Error(),
				Errors:    append([]string(nil), allErrors...),
				Operation: "stop",
			})
			r.progressBroadcaster.CloseRun(runID)
			return ctx.Err()
		default:
		}
		emit(name)
		// Use forceStopService to skip the per-service CanStop downstream-dependency
		// check.  The plan already orders services in reverse dependency order
		// (dependents first), and the intent of "stop all" is to shut down
		// everything regardless of dependency relationships.
		if err := r.forceStopService(ctx, name); err != nil {
			log.Printf("[cascade] stop-all: %s failed: %v", name, err)
			failed++
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", name, err))
		} else {
			stopped++
		}
		emit("")
	}

	// Final done event
	lastErr := ""
	if len(allErrors) > 0 {
		lastErr = allErrors[len(allErrors)-1]
	}
	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total: total, Started: stopped, Failed: failed,
		Remaining: 0, Done: true,
		Error:  lastErr,
		Errors: append([]string(nil), allErrors...),
		Phase:  "done", Operation: "stop",
	})

	go func() {
		time.Sleep(30 * time.Second)
		r.progressBroadcaster.CloseRun(runID)
	}()

	if len(allErrors) > 0 {
		return fmt.Errorf("stop-all: %d/%d services failed: %s", len(allErrors), total, strings.Join(allErrors, ", "))
	}
	return nil
}

// BuildAll builds all buildable services in dependency order.
// When a build-all runID is active (UI), progress events are published for SSE clients.
func (r *Runner) BuildAll(ctx context.Context) (*domain.BuildGroupResult, error) {
	if deployModeActive() {
		return r.buildAllFromSource(ctx)
	}
	allSvcs := r.cfg.Flatten()
	if len(allSvcs) == 0 {
		if runID := r.GetActiveBuildAllRunID(); runID != "" {
			r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
				Total: 0, Started: 0, Failed: 0, Remaining: 0,
				Done: true, Phase: "done", Operation: "build",
			})
			go func() {
				time.Sleep(30 * time.Second)
				r.progressBroadcaster.CloseRun(runID)
			}()
		}
		return nil, fmt.Errorf("no services configured")
	}

	svcPtrs := make([]*Service, len(allSvcs))
	for i := range allSvcs {
		svcPtrs[i] = &allSvcs[i]
	}
	ordered := r.orderByDependencyDepth(svcPtrs)

	var built int
	var failed []string
	var skipped []string
	var noBuild []string
	var buildable []*Service
	for _, svc := range ordered {
		if resolveBuildCommand(*svc) == "" {
			noBuild = append(noBuild, svc.Name)
			continue
		}
		buildable = append(buildable, svc)
	}

	r.activeBuildAllRunIDMu.Lock()
	runID := r.activeBuildAllRunID
	if runID == "" {
		runID = fmt.Sprintf("build-all-%d", time.Now().UnixNano())
		r.activeBuildAllRunID = runID
	}
	r.activeBuildAllRunIDMu.Unlock()
	defer func() {
		r.activeBuildAllRunIDMu.Lock()
		if r.activeBuildAllRunID == runID {
			r.activeBuildAllRunID = ""
		}
		r.activeBuildAllRunIDMu.Unlock()
	}()

	total := len(buildable)
	var failedCount int
	var allErrors []string

	emit := func(current string) {
		remaining := total - built - failedCount - len(skipped)
		if remaining < 0 {
			remaining = 0
		}
		lastErr := ""
		if len(allErrors) > 0 {
			lastErr = allErrors[len(allErrors)-1]
		}
		r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
			Total: total, Started: built, Failed: failedCount, Skipped: len(skipped),
			Current: current, Remaining: remaining,
			Done: false, Error: lastErr,
			Errors: append([]string(nil), allErrors...),
			Phase:  "progress", Operation: "build",
		})
	}

	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total: total, Remaining: total,
		Phase: "starting", Operation: "build",
	})

	// conf-sync once per BuildAll (not per service) — legacy debt fix TestBuildAll_SyncsMonorepoConfOnce
	if shouldSyncMonorepoConf() {
		if err := r.SyncMonorepoConf(ctx); err != nil {
			log.Printf("[build-all] conf-sync warning (continuing): %v", err)
		}
	} else {
		log.Printf("[build-all] skip SyncMonorepoConf in DEPLOY_MODE (ADR-0056)")
	}

	for _, svc := range buildable {
		select {
		case <-ctx.Done():
			// 中断：不清空精准编译登记（约束 42）。
			r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
				Total: total, Started: built, Failed: failedCount, Skipped: len(skipped),
				Remaining: total - built - failedCount - len(skipped),
				Done:      true, Phase: progressPhaseForContextErr(ctx),
				Error:     ctx.Err().Error(),
				Errors:    append([]string(nil), allErrors...),
				Operation: "build",
			})
			r.progressBroadcaster.CloseRun(runID)
			return nil, ctx.Err()
		default:
		}

		current := r.store.Get(svc.Name)
		if current == nil || !domain.IsTerminalBuildStatus(string(current.Status)) {
			skipped = append(skipped, svc.Name)
			emit("")
			continue
		}

		emit(svc.Name)
		if err := r.BuildService(ctx, svc.Name); err != nil {
			log.Printf("[build-all] %s build failed: %v", svc.Name, err)
			failed = append(failed, svc.Name)
			failedCount++
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", svc.Name, err))
		} else {
			built++
		}
		emit("")
	}

	// 正常完成（含部分失败）视为已消费精准编译登记；中断路径在上方提前 return，不会走到这里。
	if cerr := clearPreciseRestartRegistrationsAfterFullRebuild(r.cfgPath); cerr != nil {
		log.Printf("[build-all] clear precise-restart registrations: %v", cerr)
	} else {
		log.Printf("[build-all] precise-restart registrations cleared")
	}

	lastErr := ""
	if len(allErrors) > 0 {
		lastErr = allErrors[len(allErrors)-1]
	}
	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total: total, Started: built, Failed: failedCount, Skipped: len(skipped),
		Remaining: 0, Done: true,
		Error:  lastErr,
		Errors: append([]string(nil), allErrors...),
		Phase:  "done", Operation: "build",
	})
	go func() {
		time.Sleep(30 * time.Second)
		r.progressBroadcaster.CloseRun(runID)
	}()

	resultTotal := built + len(failed)
	status := domain.ComputeBuildGroupStatus(resultTotal, built, failed)
	result, err := domain.NewBuildGroupResult(status, resultTotal, built, failed, skipped, noBuild)
	if err != nil {
		return nil, fmt.Errorf("build all: %w", err)
	}
	return &result, nil
}

func (r *Runner) resolveCascadeStepActor(serviceName, initiatingActor string) string {
	initiating := strings.TrimSpace(initiatingActor)
	if initiating == "" || r.ownershipRepo == nil {
		return initiating
	}
	ownership, err := r.ownershipRepo.FindByServiceName(serviceName)
	if err != nil {
		return initiating
	}
	hasOwnership := ownership.ServiceName != ""
	return domain.NewCascadeStepActorResolver().Resolve(initiating, ownership, hasOwnership)
}

func (r *Runner) executeLifecyclePlan(
	ctx context.Context,
	plan domain.ServiceLifecyclePlan,
	actor string,
	stepFn func(context.Context, string, string) error,
) error {
	completed := make([]string, 0, len(plan.OrderedNames))
	for _, serviceName := range plan.OrderedNames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		stepActor := r.resolveCascadeStepActor(serviceName, actor)
		if err := stepFn(ctx, serviceName, stepActor); err != nil {
			if len(completed) == 0 {
				return err
			}
			report, reportErr := domain.NewCascadeExecutionReport(completed, serviceName)
			if reportErr != nil {
				return fmt.Errorf("%s cascade failed on %q: %w", plan.Operation, serviceName, err)
			}
			wrapped := fmt.Errorf("%s cascade failed on %q: %w", plan.Operation, serviceName, err)
			return &CascadeFailure{Err: wrapped, Report: report}
		}
		completed = append(completed, serviceName)
	}
	return nil
}

func (r *Runner) buildParallelStartLevels(orderedNames []string) ([][]string, error) {
	if len(orderedNames) == 0 {
		return nil, nil
	}

	nameSet := make(map[string]struct{}, len(orderedNames))
	for _, name := range orderedNames {
		nameSet[name] = struct{}{}
	}

	indegree := make(map[string]int, len(orderedNames))
	dependents := make(map[string][]string)
	for _, name := range orderedNames {
		svc := r.findService(name)
		if svc == nil {
			return nil, fmt.Errorf("service %q not found", name)
		}
		count := 0
		for _, dep := range svc.DependsOn {
			if _, inPlan := nameSet[dep]; inPlan {
				count++
				dependents[dep] = append(dependents[dep], name)
			}
		}
		indegree[name] = count
	}
	for name := range dependents {
		sort.Strings(dependents[name])
	}

	current := make([]string, 0)
	for _, name := range orderedNames {
		if indegree[name] == 0 {
			current = append(current, name)
		}
	}
	sort.Strings(current)

	levels := make([][]string, 0)
	seen := 0
	for len(current) > 0 {
		level := append([]string(nil), current...)
		levels = append(levels, level)
		seen += len(level)

		next := make([]string, 0)
		for _, name := range current {
			for _, child := range dependents[name] {
				indegree[child]--
				if indegree[child] == 0 {
					next = append(next, child)
				}
			}
		}
		sort.Strings(next)
		current = next
	}
	if seen != len(orderedNames) {
		return nil, fmt.Errorf("cycle detected in start plan")
	}
	return levels, nil
}

func (r *Runner) executeParallelStartPlan(ctx context.Context, operation string, levels [][]string, actor string) error {
	total := 0
	for _, level := range levels {
		total += len(level)
	}

	// Use pre-set runID (from SetActiveStartAllRunID) if available, otherwise generate one.
	r.activeStartAllRunIDMu.Lock()
	runID := r.activeStartAllRunID
	if runID == "" {
		runID = fmt.Sprintf("start-all-%d", time.Now().UnixNano())
	}
	r.activeStartAllRunID = runID
	r.activeStartAllRunIDMu.Unlock()
	defer func() {
		r.activeStartAllRunIDMu.Lock()
		if r.activeStartAllRunID == runID {
			r.activeStartAllRunID = ""
		}
		r.activeStartAllRunIDMu.Unlock()
	}()

	var started, failed, skipped int
	var allErrors []string
	var mu sync.Mutex

	emit := func(current string) {
		mu.Lock()
		remaining := total - started - failed - skipped
		if remaining < 0 {
			remaining = 0
		}
		lastErr := ""
		errs := append([]string(nil), allErrors...)
		if len(errs) > 0 {
			lastErr = errs[len(errs)-1]
		}
		ev := StartAllProgressEvent{
			Total:     total,
			Started:   started,
			Skipped:   skipped,
			Failed:    failed,
			Current:   current,
			Remaining: remaining,
			Done:      false,
			Error:     lastErr,
			Errors:    errs,
			Phase:     "progress",
			Operation: "start",
		}
		r.progressBroadcaster.Publish(runID, ev)
		mu.Unlock()
	}

	// Initial event
	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total:     total,
		Started:   0,
		Skipped:   0,
		Failed:    0,
		Current:   "",
		Remaining: total,
		Done:      false,
		Phase:     "starting",
		Operation: "start",
	})

	for levelIdx, level := range levels {
		select {
		case <-ctx.Done():
			r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
				Total: total, Started: started, Skipped: skipped, Failed: failed,
				Remaining: total - started - failed - skipped,
				Done:      true, Phase: progressPhaseForContextErr(ctx),
				Error:     ctx.Err().Error(),
				Errors:    append([]string(nil), allErrors...),
				Operation: "start",
			})
			r.progressBroadcaster.CloseRun(runID)
			return ctx.Err()
		default:
		}
		r.logParallelStartLevel(operation, levelIdx+1, len(levels), level)

		var wg sync.WaitGroup
		for _, serviceName := range level {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				emit(name)
				stepActor := r.resolveCascadeStepActor(name, actor)
				if err := r.StartServiceWithActor(ctx, name, stepActor); err != nil {
					mu.Lock()
					if errors.Is(err, ErrStartSkippedAlreadyHealthy) {
						skipped++
					} else {
						failed++
						errMsg := fmt.Sprintf("%s: %v", name, err)
						allErrors = append(allErrors, errMsg)
					}
					mu.Unlock()
					if !errors.Is(err, ErrStartSkippedAlreadyHealthy) {
						log.Printf("[cascade] parallel %s level %d/%d error: %q: %v", operation, levelIdx+1, len(levels), name, err)
					}
				} else {
					mu.Lock()
					started++
					mu.Unlock()
				}
				emit("")
			}(serviceName)
		}
		wg.Wait()
	}

	// Final done event
	lastErr := ""
	if len(allErrors) > 0 {
		lastErr = allErrors[len(allErrors)-1]
	}
	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total:     total,
		Started:   started,
		Skipped:   skipped,
		Failed:    failed,
		Current:   "",
		Remaining: 0,
		Done:      true,
		Error:     lastErr,
		Errors:    append([]string(nil), allErrors...),
		Phase:     "done",
		Operation: "start",
	})

	// Soft post-start health: internal API live smoke（不阻断 start-all）
	r.maybeRunInternalAPIsSmokeAfterStartAll(failed)

	// Keep the run alive for 30s so late-subscribing SSE clients can get the final event,
	// then clean up subscribers.
	go func() {
		time.Sleep(30 * time.Second)
		r.progressBroadcaster.CloseRun(runID)
	}()

	return nil
}

func (r *Runner) logParallelStartPlanPreview(operation string, levels [][]string) {
	totalLevels := len(levels)
	for levelIdx, level := range levels {
		peerCount := len(level)
		levelNote := ""
		if totalLevels == 1 && peerCount > 1 {
			levelNote = " (dependencies already healthy or absent from plan)"
		}
		for _, name := range level {
			if levelIdx == 0 {
				r.appendLifecycleLog(
					name,
					fmt.Sprintf("%s: DAG level 1/%d — starts now with %d peer(s) in parallel%s", operation, totalLevels, peerCount, levelNote),
				)
				continue
			}
			r.appendLifecycleLog(
				name,
				fmt.Sprintf("%s: DAG level %d/%d — starts when in-plan dependencies in level %d are healthy (%d peer(s) in parallel)", operation, levelIdx+1, totalLevels, levelIdx, peerCount),
			)
		}
	}
}

func (r *Runner) logParallelStartLevel(operation string, levelNum, levelTotal int, level []string) {
	for _, name := range level {
		r.appendLifecycleLog(name, fmt.Sprintf("%s: DAG level %d/%d starting now", operation, levelNum, levelTotal))
	}
}

func (r *Runner) transitionServiceToStarting(name string) (Status, bool) {
	for _, from := range []Status{StatusStopped, StatusFailed, StatusSkipped, StatusPending} {
		if r.store.CompareAndSwapStatus(name, from, StatusStarting) {
			return from, true
		}
	}
	return "", false
}

// claimStartForOperator CASes an idle — or mid-retry — service to Starting for
// an explicit single-service Start. Retrying is reclaimable the same way
// restartService reclaims it: the caller tears down the in-flight process and
// launches fresh. This is intentionally NOT part of transitionServiceToStarting /
// IsStartableServiceStatus so bulk/cascade planning never double-binds a service
// that another goroutine is already launching.
func (r *Runner) claimStartForOperator(name string) (Status, bool) {
	for _, from := range []Status{StatusStopped, StatusFailed, StatusSkipped, StatusPending, StatusRetrying} {
		if r.store.CompareAndSwapStatus(name, from, StatusStarting) {
			return from, true
		}
	}
	return "", false
}

// waitForServiceStart polls the service status until it resolves to healthy, failed,
// skipped, or stopped, or the context is cancelled. This is used when a goroutine
// loses the CAS race and must wait for the winning goroutine to finish launching.
func (r *Runner) waitForServiceStart(ctx context.Context, svc Service) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := 30 * time.Second
	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			r.store.Update(svc.Name, StatusFailed,
				fmt.Sprintf("timed out after %v waiting for another goroutine to finish starting", timeout))
			r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
			return fmt.Errorf("[%s] timed out waiting for service start", svc.Name)
		case <-ticker.C:
			current := r.store.Get(svc.Name)
			if current == nil {
				continue
			}
			switch current.Status {
			case StatusHealthy:
				log.Printf("[%s] service started by another goroutine (now healthy)", svc.Name)
				r.appendLifecycleLog(svc.Name, "service started by another goroutine (now healthy)")
				return nil
			case StatusFailed, StatusSkipped, StatusStopped:
				log.Printf("[%s] service start resolved by another goroutine (now %s)", svc.Name, current.Status)
				r.appendLifecycleLog(svc.Name,
					fmt.Sprintf("service start resolved by another goroutine (now %s)", current.Status))
				if current.Status == StatusFailed {
					return fmt.Errorf("[%s] service start failed: %s", svc.Name, current.Error)
				}
				return nil
			}
			// StatusStarting, StatusRestarting, StatusRetrying, or StatusBuilding — keep waiting
		}
	}
}

func (r *Runner) startService(ctx context.Context, name string) error {
	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}
	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}

	if skipped, err := r.tryResolveStartWithoutLaunch(ctx, svc, current); skipped || err != nil {
		return err
	}

	managedService, err := domain.NewManagedService(svc.Name, current.Group, string(current.Status), svc.DependsOn)
	if err != nil {
		return err
	}
	// Explicit single-service Start may also reclaim a stuck Retrying service
	// (same carve-out as restartService); the global CanStart / IsStartableServiceStatus
	// keep excluding Retrying so bulk/cascade planning never double-binds a launch.
	if !managedService.CanStart() && current.Status != StatusRetrying {
		return fmt.Errorf("service %q is %s, can only start idle services", name, current.Status)
	}

	previousStatus, ok := r.claimStartForOperator(name)
	if !ok {
		latest := r.store.Get(name)
		if latest == nil {
			return fmt.Errorf("service %q not found", name)
		}
		return fmt.Errorf("service %q is %s, can only start idle services", name, latest.Status)
	}
	r.appendLifecycleLog(name, fmt.Sprintf("start requested (previous status=%s)", previousStatus))
	if previousStatus != StatusStopped {
		r.stopMonitoring(name)
		_ = r.stopProcess(name)
		r.store.SetPID(name, 0)
	}

	if err := r.runPreflight(ctx, *svc); err != nil {
		r.appendLifecycleLog(name, fmt.Sprintf("preflight failed: %v", err))
		r.store.SetPID(name, 0)
		return err
	}
	if len(svc.DependsOn) > 0 {
		r.appendLifecycleLog(name, "waiting for dependency readiness probes...")
	}
	if err := r.waitForDependencyReadiness(ctx, *svc); err != nil {
		r.appendLifecycleLog(name, fmt.Sprintf("start aborted waiting for dependencies: %v", err))
		r.store.SetPID(name, 0)
		r.store.Update(name, previousStatus, err.Error())
		return err
	}

	node := &ServiceNode{Service: *svc}
	if err := r.startAndCheck(ctx, node); err != nil {
		r.stopMonitoring(name)
		_ = r.stopProcess(name)
		r.store.SetPID(name, 0)
		return err
	}
	if err := r.ensureHealthyAfterManualStart(name); err != nil {
		return err
	}

	r.startMonitoring(ctx, *svc)
	return nil
}

func (r *Runner) ensureHealthyAfterManualStart(name string) error {
	current := r.store.Get(name)
	if current == nil {
		return fmt.Errorf("service %q not found", name)
	}
	if current.Status == StatusHealthy {
		return nil
	}
	r.stopMonitoring(name)
	_ = r.stopProcess(name)
	r.store.SetPID(name, 0)
	if current.FailureCode != "" && current.Error != "" {
		return fmt.Errorf("service %q failed to start [%s]: %s", name, current.FailureCode, current.Error)
	}
	if current.Error != "" {
		return fmt.Errorf("service %q failed to start: %s", name, current.Error)
	}
	if current.FailureCode != "" {
		return fmt.Errorf("service %q failed to start [%s]", name, current.FailureCode)
	}
	return fmt.Errorf("service %q failed to start", name)
}

func (r *Runner) runningDependents(name string) []string {
	var running []string
	for _, svc := range r.cfg.Flatten() {
		if !dependsOnService(svc.DependsOn, name) {
			continue
		}
		if r.IsServiceRunning(svc.Name) {
			running = append(running, svc.Name)
			continue
		}
		// Preserve stop-policy semantics for failed dependents that still
		// carry a launch PID even when health/port probes are unreachable.
		st := r.store.Get(svc.Name)
		if st != nil && st.PID > 0 && st.Status == StatusFailed {
			running = append(running, svc.Name)
		}
	}
	return running
}

func dependsOnService(dependsOn []string, name string) bool {
	for _, dep := range dependsOn {
		if strings.TrimSpace(dep) == name {
			return true
		}
	}
	return false
}

func mergeDependentNames(parts ...[]string) []string {
	seen := make(map[string]struct{})
	for _, list := range parts {
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			seen[item] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for item := range seen {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func (r *Runner) StopGroup(ctx context.Context, group string) error {
	return r.stopGroup(ctx, group, func(stopCtx context.Context, serviceName string) error {
		return r.StopService(stopCtx, serviceName)
	})
}

func (r *Runner) StopGroupWithActor(ctx context.Context, group string, actorSessionID string) error {
	actor := strings.TrimSpace(actorSessionID)
	if actor == "" {
		return fmt.Errorf("actor session id is required")
	}
	return r.stopGroup(ctx, group, func(stopCtx context.Context, serviceName string) error {
		stepActor := r.resolveCascadeStepActor(serviceName, actor)
		return r.StopServiceWithActor(stopCtx, serviceName, stepActor)
	})
}

func (r *Runner) stopGroup(ctx context.Context, group string, stopFn func(context.Context, string) error) error {
	var groupServices []Service
	for _, g := range r.cfg.Groups {
		if g.Name == group {
			groupServices = g.Services
			break
		}
	}
	if groupServices == nil {
		return fmt.Errorf("group %q not found", group)
	}

	stopOrder, err := stopOrderForGroup(groupServices)
	if err != nil {
		return fmt.Errorf("group %q has invalid dependencies: %w", group, err)
	}

	var stopErrors []error
	for _, serviceName := range stopOrder {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := stopFn(ctx, serviceName); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("%s: %w", serviceName, err))
		}
	}
	if len(stopErrors) > 0 {
		return fmt.Errorf("stop group %q failed: %w", group, errors.Join(stopErrors...))
	}
	return nil
}

// BuildGroup builds all buildable services in a group, ordered by dependency depth.
// It continues on individual failures (best-effort) and returns a summary result.
// When a build-all runID is active (UI), progress events are published for SSE clients
// (shared with BuildAll so the same progress panel/cancel endpoints work).
func (r *Runner) BuildGroup(ctx context.Context, groupName string) (*domain.BuildGroupResult, error) {
	svcs := r.findServicesByGroup(groupName)
	if svcs == nil {
		if runID := r.GetActiveBuildAllRunID(); runID != "" {
			r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
				Done: true, Phase: "error", Error: fmt.Sprintf("group %q not found", groupName),
				Operation: "build",
			})
		}
		return nil, fmt.Errorf("group %q not found", groupName)
	}

	ordered := r.orderByDependencyDepth(svcs)

	var built int
	var builtNames []string
	var failed []string
	var skipped []string
	var noBuild []string
	var buildable []*Service
	for _, svc := range ordered {
		if resolveBuildCommand(*svc) == "" {
			noBuild = append(noBuild, svc.Name)
			continue
		}
		buildable = append(buildable, svc)
	}

	r.activeBuildAllRunIDMu.Lock()
	runID := r.activeBuildAllRunID
	if runID == "" {
		runID = fmt.Sprintf("build-group-%d", time.Now().UnixNano())
		r.activeBuildAllRunID = runID
	}
	r.activeBuildAllRunIDMu.Unlock()
	defer func() {
		r.activeBuildAllRunIDMu.Lock()
		if r.activeBuildAllRunID == runID {
			r.activeBuildAllRunID = ""
		}
		r.activeBuildAllRunIDMu.Unlock()
	}()

	total := len(buildable)
	var failedCount int
	var allErrors []string

	emit := func(current string) {
		remaining := total - built - failedCount - len(skipped)
		if remaining < 0 {
			remaining = 0
		}
		lastErr := ""
		if len(allErrors) > 0 {
			lastErr = allErrors[len(allErrors)-1]
		}
		r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
			Total: total, Started: built, Failed: failedCount, Skipped: len(skipped),
			Current: current, Remaining: remaining,
			Done: false, Error: lastErr,
			Errors: append([]string(nil), allErrors...),
			Phase:  "progress", Operation: "build",
		})
	}

	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total: total, Remaining: total,
		Phase: "starting", Operation: "build",
	})

	if shouldSyncMonorepoConf() {
		if err := r.SyncMonorepoConf(ctx); err != nil {
			log.Printf("[build-group] conf-sync warning (continuing): %v", err)
		}
	} else {
		log.Printf("[build-group] skip SyncMonorepoConf in DEPLOY_MODE (ADR-0056)")
	}

	for _, svc := range buildable {
		select {
		case <-ctx.Done():
			r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
				Total: total, Started: built, Failed: failedCount, Skipped: len(skipped),
				Remaining: total - built - failedCount - len(skipped),
				Done:      true, Phase: progressPhaseForContextErr(ctx),
				Error:     ctx.Err().Error(),
				Errors:    append([]string(nil), allErrors...),
				Operation: "build",
			})
			r.progressBroadcaster.CloseRun(runID)
			return nil, ctx.Err()
		default:
		}

		current := r.store.Get(svc.Name)
		if current == nil || !domain.IsTerminalBuildStatus(string(current.Status)) {
			skipped = append(skipped, svc.Name)
			emit("")
			continue
		}

		emit(svc.Name)
		if err := r.BuildService(ctx, svc.Name); err != nil {
			log.Printf("[build-group] %s build failed: %v", svc.Name, err)
			failed = append(failed, svc.Name)
			failedCount++
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", svc.Name, err))
		} else {
			built++
			builtNames = append(builtNames, svc.Name)
		}
		emit("")
	}

	// 分组编译成功后仅按组裁剪精准重启登记（OPT-20260811-036）：
	// 只移除本组已登记且编译成功的服务名，其它组/失败/并发登记不动。
	if len(builtNames) > 0 {
		if terr := trimRegistrationsAfterGroupBuild(preciseRestartFile(r.cfgPath), builtNames); terr != nil {
			log.Printf("[build-group] trim precise-restart registrations: %v", terr)
		}
	}

	lastErr := ""
	if len(allErrors) > 0 {
		lastErr = allErrors[len(allErrors)-1]
	}
	r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
		Total: total, Started: built, Failed: failedCount, Skipped: len(skipped),
		Remaining: 0, Done: true,
		Error:  lastErr,
		Errors: append([]string(nil), allErrors...),
		Phase:  "done", Operation: "build",
	})
	go func() {
		time.Sleep(30 * time.Second)
		r.progressBroadcaster.CloseRun(runID)
	}()

	resultTotal := built + len(failed)
	status := domain.ComputeBuildGroupStatus(resultTotal, built, failed)
	result, err := domain.NewBuildGroupResult(status, resultTotal, built, failed, skipped, noBuild)
	if err != nil {
		return nil, fmt.Errorf("build group %q: %w", groupName, err)
	}
	return &result, nil
}

// findServicesByGroup returns all services in the named group, or nil if not found.
func (r *Runner) findServicesByGroup(groupName string) []*Service {
	for gi := range r.cfg.Groups {
		if r.cfg.Groups[gi].Name == groupName {
			svcs := make([]*Service, len(r.cfg.Groups[gi].Services))
			for si := range r.cfg.Groups[gi].Services {
				svcs[si] = &r.cfg.Groups[gi].Services[si]
			}
			return svcs
		}
	}
	return nil
}

// orderByDependencyDepth sorts services so that services with fewer dependencies come first.
func (r *Runner) orderByDependencyDepth(svcs []*Service) []*Service {
	if len(svcs) <= 1 {
		return svcs
	}

	depth := make(map[string]int)
	var computeDepth func(name string, visited map[string]bool) int
	computeDepth = func(name string, visited map[string]bool) int {
		if d, ok := depth[name]; ok {
			return d
		}
		if visited[name] {
			return 0 // circular dependency, treat as depth 0
		}
		visited[name] = true
		maxDep := 0
		for _, svc := range svcs {
			if svc.Name == name {
				for _, dep := range svc.DependsOn {
					dd := computeDepth(dep, visited)
					if dd >= maxDep {
						maxDep = dd + 1
					}
				}
				break
			}
		}
		depth[name] = maxDep
		return maxDep
	}

	// Compute depth for all services
	for _, svc := range svcs {
		computeDepth(svc.Name, make(map[string]bool))
	}

	sorted := make([]*Service, len(svcs))
	copy(sorted, svcs)
	sort.SliceStable(sorted, func(i, j int) bool {
		return depth[sorted[i].Name] < depth[sorted[j].Name]
	})
	return sorted
}

func (r *Runner) BuildService(ctx context.Context, name string) error {
	svc := r.findService(name)
	if svc == nil {
		return fmt.Errorf("service %q not found", name)
	}
	buildCmd := resolveBuildCommand(*svc)
	if buildCmd == "" {
		return fmt.Errorf("service %q has no build command configured", name)
	}

	// Allow build from any status except "building" (prevents concurrent builds).
	// Compilation is a disk-only operation independent of runtime state.
	buildableStatuses := []Status{
		StatusHealthy, StatusFailed, StatusStopped,
		StatusPending, StatusStarting, StatusRetrying,
		StatusSkipped, StatusRestarting,
		Status(""),
	}
	previousStatus := StatusFailed
	swapped := false
	for _, from := range buildableStatuses {
		if r.store.CompareAndSwapStatus(name, from, StatusBuilding) {
			previousStatus = from
			swapped = true
			break
		}
	}
	if !swapped {
		current := r.store.Get(name)
		if current == nil {
			return fmt.Errorf("service %q not found", name)
		}
		if current.Status == StatusBuilding {
			return fmt.Errorf("service %q is already building", name)
		}
		return fmt.Errorf("service %q is %s, cannot build right now", name, current.Status)
	}

	log.Printf("[%s] build-only requested", name)

	// Sync monorepo conf fragments (e.g. git-service.yaml → frontend/vue)
	// so that vite build reads the latest config values.
	// Skip when BuildAll/BuildGroup already synced once for this bulk run.
	if r.GetActiveBuildAllRunID() == "" && shouldSyncMonorepoConf() {
		if err := r.SyncMonorepoConf(ctx); err != nil {
			log.Printf("[%s] conf-sync warning (continuing): %v", name, err)
		}
	}

	if err := r.runBuild(ctx, svc, buildCmd); err != nil {
		r.store.Update(name, r.statusAfterBuildInterrupt(name, StatusFailed), err.Error())
		return err
	}

	// Build-only does not start the process. If the service was previously
	// failed, restore to stopped (not failed) so a successful compile does not
	// keep a sticky failure error in the UI.
	restored := r.statusAfterBuildInterrupt(name, previousStatus)
	if restored == StatusFailed {
		restored = StatusStopped
	}
	r.store.Update(name, restored, "")
	return nil
}

// statusAfterBuildInterrupt returns the status to restore after a build completes
// or fails. If stop-all / manual stop set the service to stopped while the build
// goroutine was running, keep stopped instead of restoring a phantom healthy state.
func (r *Runner) statusAfterBuildInterrupt(name string, previousStatus Status) Status {
	if r == nil || r.store == nil {
		return previousStatus
	}
	current := r.store.Get(name)
	if current != nil && current.Status == StatusStopped {
		return StatusStopped
	}
	return previousStatus
}

func (r *Runner) findService(name string) *Service {
	for gi := range r.cfg.Groups {
		for si := range r.cfg.Groups[gi].Services {
			svc := &r.cfg.Groups[gi].Services[si]
			if svc.Name == name {
				return svc
			}
		}
	}
	return nil
}

func (r *Runner) configuredServiceNames() []string {
	if r == nil || r.cfg == nil {
		return nil
	}
	services := r.cfg.Flatten()
	names := make([]string, 0, len(services))
	for _, svc := range services {
		if strings.TrimSpace(svc.Name) != "" {
			names = append(names, svc.Name)
		}
	}
	return names
}

func (r *Runner) ClearAllObservability(ctx context.Context) domain.ObservabilityStackResetResult {
	if r.devToolLogRecorder != nil {
		r.devToolLogRecorder.Append(domain.ToolObservabilityClear, "开始清空 Grafana 可观测数据...")
	}
	resetter := infrastructure.NewScriptObservabilityStorageResetter(
		infrastructure.ResolveObservabilityResetScript(r.cfgPath),
		"",
	)
	if _, prodRoot := promtailLogRoot(r); !prodRoot {
		// Unit tests construct Runners with t.TempDir() FileRoot; executing the real
		// reset script there would stop/recreate the shared Loki/Tempo/Prometheus stack
		// and repoint aimonitor-promtail at an empty temp dir via runall-local-promtail.sh
		// (OPT-20260903-004). Keep the in-memory/file clear, skip the storage script.
		log.Printf("[runAll] observability clear: skip reset script for non-production log root (OPT-20260903-004)")
		resetter = nil
	}
	svc := domain.NewObservabilityStackResetService(r.logRepository, r.fileLogSink, resetter)
	result := svc.ClearAll(ctx, r.configuredServiceNames())
	if r.devToolLogRecorder != nil {
		if result.Status == "ok" {
			r.devToolLogRecorder.Append(domain.ToolObservabilityClear,
				fmt.Sprintf("清空完成: memory_cleared=%d files_truncated=%d loki=%s tempo=%s prometheus=%s",
					result.MemoryServicesCleared, result.FilesTruncated,
					result.LokiReset, result.TempoReset, result.PrometheusReset))
		} else {
			r.devToolLogRecorder.Append(domain.ToolObservabilityClear,
				fmt.Sprintf("清空完成: status=%s", result.Status))
		}
	}
	return result
}

// ClearServiceLogs truncates tee log files via FileServiceLogSink and clears in-memory
// buffers. It does not reset Loki/Promtail/Tempo. Prefer this (or POST /api/logs/clear-all)
// over `rm logs/*.log`, which orphans open FDs and breaks Promtail shipping.
func (r *Runner) ClearServiceLogs() domain.ObservabilityStackResetResult {
	if r == nil {
		return domain.ObservabilityStackResetResult{Status: "partial"}
	}
	svc := domain.NewObservabilityStackResetService(r.logRepository, r.fileLogSink, nil)
	return svc.ClearAll(context.Background(), r.configuredServiceNames())
}

func (r *Runner) hasTrackedProcess(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cmd, ok := r.processes[name]
	return ok && cmd != nil
}

// IsServiceRunning reports whether a service is actually up (for dev database clear/init gates).
// Store status alone is not enough: failed/skipped and healthy+degraded can linger after stop.
func (r *Runner) IsServiceRunning(name string) bool {
	if r.hasTrackedProcess(name) {
		return true
	}
	svc := r.findService(name)
	if svc != nil && r.probeActivePortListeners(svc) {
		return true
	}
	if r.store == nil {
		return false
	}
	st := r.store.Get(name)
	if st == nil || st.Status == StatusStopped {
		return false
	}
	if svc == nil {
		return st.Status != StatusStopped
	}
	// Read-only probe: unreachable means not running even if store still says failed/healthy.
	return r.isServiceReachable(svc)
}

func (r *Runner) isServiceReachable(svc *Service) bool {
	if svc == nil {
		return false
	}
	return checkProbe(context.Background(), svc.HealthCheck) == nil
}

func (r *Runner) isExcludedDevDatabaseService(name string, exclude []string) bool {
	for _, ex := range exclude {
		if ex == name {
			return true
		}
	}
	return false
}

func (r *Runner) RunningApplicationsExcept(exclude []string) []string {
	if r == nil || r.cfg == nil {
		return nil
	}
	var running []string
	for _, svc := range r.cfg.Flatten() {
		name := strings.TrimSpace(svc.Name)
		if name == "" || r.isExcludedDevDatabaseService(name, exclude) {
			continue
		}
		if r.IsServiceRunning(name) {
			running = append(running, name)
		}
	}
	sort.Strings(running)
	return running
}

// devDBStopPerServiceTimeout bounds one service stop during dev database
// clear/init so a hung compose/ssh stop cannot stall the whole phase silently.
const devDBStopPerServiceTimeout = 90 * time.Second

func (r *Runner) StopAllApplicationsExcept(ctx context.Context, exclude []string, onProgress domain.ProgressCallback) ([]string, []string) {
	if r == nil || r.cfg == nil {
		return nil, nil
	}
	notify := func(msg string) {
		if onProgress != nil {
			onProgress(msg)
		}
	}
	var toStop []Service
	for _, svc := range r.cfg.Flatten() {
		if r.isExcludedDevDatabaseService(svc.Name, exclude) {
			continue
		}
		toStop = append(toStop, svc)
	}
	stopOrder, err := stopOrderForGroup(toStop)
	if err != nil {
		log.Printf("[dev-db] stop order: %v", err)
		stopOrder = configuredNamesFromServices(toStop)
	}
	notify(fmt.Sprintf("正在停止 %d 个服务（保留 %v）...", len(toStop), exclude))

	stoppedSet := make(map[string]struct{})
	const maxPasses = 8
	for pass := 0; pass < maxPasses; pass++ {
		madeProgress := false
		for _, name := range stopOrder {
			if !r.IsServiceRunning(name) {
				continue
			}
			if err := r.stopServiceForDevDatabaseClearWithTimeout(ctx, name); err != nil {
				log.Printf("[dev-db] stop %s (pass %d): %v", name, pass+1, err)
				continue
			}
			stoppedSet[name] = struct{}{}
			notify(fmt.Sprintf("已停止服务: %s", name))
			madeProgress = true
		}
		if r.RunningApplicationsExcept(exclude) == nil {
			break
		}
		if !madeProgress {
			break
		}
	}

	stopped := make([]string, 0, len(stoppedSet))
	for name := range stoppedSet {
		stopped = append(stopped, name)
	}
	sort.Strings(stopped)
	return stopped, r.RunningApplicationsExcept(exclude)
}

// stopServiceForDevDatabaseClearWithTimeout bounds a single stop with
// devDBStopPerServiceTimeout and logs when the deadline is hit, so the
// clear UI keeps receiving progress instead of appearing frozen.
func (r *Runner) stopServiceForDevDatabaseClearWithTimeout(ctx context.Context, name string) error {
	stopCtx, cancel := context.WithTimeout(ctx, devDBStopPerServiceTimeout)
	defer cancel()
	err := r.stopServiceForDevDatabaseClear(stopCtx, name)
	if stopCtx.Err() == context.DeadlineExceeded {
		log.Printf("[dev-db] stop %s 超时（> %s），继续下一个服务", name, devDBStopPerServiceTimeout)
	}
	return err
}

func configuredNamesFromServices(services []Service) []string {
	names := make([]string, 0, len(services))
	for _, svc := range services {
		if strings.TrimSpace(svc.Name) != "" {
			names = append(names, svc.Name)
		}
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return names
}

func (r *Runner) TryAcquireDevDatabaseReset() bool {
	return tryAcquireDevDBResetFlag()
}

func (r *Runner) ReleaseDevDatabaseReset() {
	r.ClearActiveDevDBRun()
	releaseDevDBResetFlag()
}

// devDBLastStatusPath 返回 dev 数据库操作状态标记文件路径（<monorepoRoot>/.runall/<tool>.last_status，
// tool 即 domain.ToolDbInit "db-init" / ToolDbClear "db-clear"）。
// .runall/ 属运行时状态（gitignore）；无 root 时返回空串（单测/裸 Runner 场景不落盘）。
func (r *Runner) devDBLastStatusPath(tool string) string {
	if r == nil || tool == "" {
		return ""
	}
	// 与 infrastructure.ResolveMonorepoRoot 一致：MONOREPO_ROOT 环境变量优先（便于测试隔离）。
	if env := strings.TrimSpace(os.Getenv("MONOREPO_ROOT")); env != "" {
		return filepath.Join(env, ".runall", tool+".last_status")
	}
	root, err := r.monorepoRoot()
	if err != nil || root == "" {
		return ""
	}
	return filepath.Join(root, ".runall", tool+".last_status")
}

// RecordDevDBLastStatus 记录 dev 数据库操作的当前状态到 .runall/<tool>.last_status。
// 开始（running）、结束（ok/partial/blocked/panic）都会写；进程 OOM/panic 时停留在 running，
// 运维可据此判断清库/初始化是否异常中断（OPT-20260813-002 #4）。写失败仅告警，不阻断流程。
func (r *Runner) RecordDevDBLastStatus(tool, status, message string) {
	if r == nil {
		return
	}
	path := r.devDBLastStatusPath(tool)
	if path == "" {
		return
	}
	lines := []string{
		"tool=" + tool,
		"status=" + status,
		"ts=" + time.Now().Format(time.RFC3339),
	}
	if message != "" {
		lines = append(lines, "message="+message)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("[runner] record dev %s status: mkdir: %v", tool, err)
		return
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		log.Printf("[runner] record dev %s status: %v", tool, err)
	}
}

func (r *Runner) loadDevDatabaseContext(configPath string) (root string, entries []domain.RegisteredDatabase, scripts *infrastructure.BashScriptRunner, err error) {
	root, err = infrastructure.ResolveMonorepoRoot(configPath)
	if err != nil {
		return "", nil, nil, err
	}
	entries, err = infrastructure.LoadRegisteredDatabases(root)
	if err != nil {
		return root, nil, nil, err
	}
	scripts = &infrastructure.BashScriptRunner{MonorepoRoot: root}
	return root, entries, scripts, nil
}

func (r *Runner) ClearAllDatabases(ctx context.Context, configPath string) domain.DatabasePlatformClearResult {
	// Build a progress callback that writes to the dev tool log so the
	// frontend can poll /api/dev/logs?tool=db-clear for live step updates.
	onProgress := func(msg string) {
		if r.devToolLogRecorder != nil {
			r.devToolLogRecorder.Append(domain.ToolDbClear, msg)
		}
	}
	onProgress("开始清空全部数据库...")
	root, entries, scripts, err := r.loadDevDatabaseContext(configPath)
	if err != nil {
		onProgress(fmt.Sprintf("加载配置失败: %v", err))
		return domain.DatabasePlatformClearResult{Status: "partial"}
	}
	redisScript, kafkaScript, mysqlScript, err := infrastructure.ResolveDevDatabaseScripts(root)
	if err != nil {
		onProgress(fmt.Sprintf("解析数据库脚本失败: %v", err))
		return domain.DatabasePlatformClearResult{Status: "partial"}
	}
	// Only create SQLite remover if there are non-MySQL databases.
	// When all databases are MySQL-backed, pass nil so the clear service
	// skips the SQLite removal step entirely (no misleading log messages).
	var remover domain.SQLiteRemover
	if domain.HasNonMySQLDatabases(entries) {
		remover = &infrastructure.RegistrySQLiteRemover{MonorepoRoot: root}
	}

	// Ensure reserved infrastructure services (docker-redis, docker-kafka)
	// are running before attempting to flush/recreate them.  They may be
	// stopped if a Stop All was performed prior to the clear operation.
	for _, name := range domain.DevDatabaseClearExcludeServices {
		if r.IsServiceRunning(name) {
			continue
		}
		onProgress(fmt.Sprintf("预留服务 %s 未运行，正在启动...", name))
		if err := r.StartService(ctx, name); err != nil {
			onProgress(fmt.Sprintf("警告: 预留服务 %s 启动失败: %v", name, err))
		} else {
			onProgress(fmt.Sprintf("预留服务 %s 已就绪", name))
		}
	}

	svc := domain.NewDatabasePlatformClearService(
		r,
		remover,
		scripts,
		entries,
		redisScript,
		kafkaScript,
		mysqlScript,
		domain.DevDatabaseClearExcludeServices,
		onProgress,
	)
	result := svc.Clear(ctx)
	// Final summary for the dev log viewer
	onProgress(fmt.Sprintf("停止服务: %v", result.ServicesStopped))
	if len(result.ServicesStillRunning) > 0 {
		onProgress(fmt.Sprintf("仍在运行: %v", result.ServicesStillRunning))
	}
	if len(result.SQLiteRemoved) > 0 {
		onProgress(fmt.Sprintf("SQLite 已删除: %v", result.SQLiteRemoved))
	}
	onProgress(fmt.Sprintf("MySQL: %s Redis: %s Kafka: %s", result.MySQLReset, result.RedisReset, result.KafkaReset))
	onProgress(fmt.Sprintf("清空数据库完成: status=%s", result.Status))
	return result
}

func (r *Runner) InitAllDatabases(ctx context.Context, configPath string) domain.DatabasePlatformInitResult {
	// Build a progress callback that writes to the dev tool log so the
	// frontend can poll /api/dev/logs?tool=db-init for live step updates.
	onProgress := func(msg string) {
		if r.devToolLogRecorder != nil {
			r.devToolLogRecorder.Append(domain.ToolDbInit, msg)
		}
	}
	onProgress("开始初始化全部数据库...")
	_, entries, scripts, err := r.loadDevDatabaseContext(configPath)
	if err != nil {
		onProgress(fmt.Sprintf("加载配置失败: %v", err))
		return domain.DatabasePlatformInitResult{
			Status: "partial",
			Migrations: []domain.DatabaseStepResult{{
				Database: "registry",
				Status:   "failed",
				Message:  err.Error(),
			}},
		}
	}
	// saas init.sh 需要通过 API 调用 taskBill / taskTenantService /
	// taskProjectService 完成 seed 数据写入。如果这些服务未运行（例如在
	//「清空数据库」之后直接「初始化数据库」），init.sh 的 wait_for_service
	// 会因为服务不在而超时（60s）。此处确保前置服务已启动并就绪。
	prereqServices := []string{
		"task-bill",
		"task-tenant-service",
		"task-project-service",
	}
	for _, name := range prereqServices {
		if r.IsServiceRunning(name) {
			continue
		}
		onProgress(fmt.Sprintf("前置服务 %s 未运行，正在启动...", name))
		if err := r.StartService(ctx, name); err != nil {
			onProgress(fmt.Sprintf("警告: 前置服务 %s 启动失败: %v", name, err))
		} else {
			onProgress(fmt.Sprintf("前置服务 %s 已就绪", name))
		}
	}

	svc := domain.NewDatabasePlatformInitService(
		r,
		scripts,
		entries,
		domain.DevDatabaseClearExcludeServices,
		onProgress,
	)
	result := svc.Init(ctx)
	// Final summary for the dev log viewer
	for _, m := range result.Migrations {
		msg := fmt.Sprintf("migrate %s: %s", m.Database, m.Status)
		if m.Message != "" {
			msg += " - " + m.Message
		}
		onProgress(msg)
	}
	for _, i := range result.Inits {
		msg := fmt.Sprintf("init %s: %s", i.Database, i.Status)
		if i.Message != "" {
			msg += " - " + i.Message
		}
		onProgress(msg)
	}
	onProgress(fmt.Sprintf("初始化数据库完成: status=%s", result.Status))

	// After migrate/init, restore login-critical path so WeChat/email login
	// does not hit APISIX 502 (Connection refused to task-auth) during the
	// post-clear window. See docs/superpowers/specs/2026-08-11-wechat-login-apisix-502-design.md
	if _, failed, ensureErr := domain.EnsureLoginCriticalPath(ctx, r, onProgress); ensureErr != nil {
		onProgress(fmt.Sprintf("警告: 登录关键路径未就绪 failed=%v err=%v", failed, ensureErr))
		if result.Status == "ok" {
			result.Status = "partial"
		}
		result.Inits = append(result.Inits, domain.DatabaseStepResult{
			Database: "login-critical-path",
			Status:   "failed",
			Message:  ensureErr.Error(),
		})
	}

	return result
}

func (r *Runner) stopProcess(name string) bool {
	r.mu.Lock()
	cmd, ok := r.processes[name]
	if ok {
		delete(r.processes, name)
	}
	r.mu.Unlock()

	if !ok || cmd == nil || cmd.Process == nil {
		return false
	}

	log.Printf("[%s] restarting: sending SIGTERM", name)
	pgid := cmd.Process.Pid
	syscall.Kill(-pgid, syscall.SIGTERM)

	// 等待进程组消亡而非组长进程退出（OPT-20260810-005）：start_command 常为
	// bash 包装（如 `bash run.sh start ...`），组长 bash 收到 SIGTERM 立即退出，
	// cmd.Wait() 快速返回会让 5s SIGKILL 升级短路；优雅关闭中的服务进程继续
	// 监听端口，随后被 startAndCheck 判定「端口已监听且健康」而跳过启动新二进制。
	if waitProcessGroupExit(pgid, 5*time.Second) {
		log.Printf("[%s] stopped for restart", name)
		_ = cmd.Wait()
		reapZombies()
		return true
	}

	log.Printf("[%s] did not stop, sending SIGKILL", name)
	syscall.Kill(-pgid, syscall.SIGKILL)
	waitProcessGroupExit(pgid, 5*time.Second)
	_ = cmd.Wait()

	// Reap any remaining zombies (children of the killed process) to prevent
	// <defunct> accumulation. Wait4(-1, WNOHANG) collects any unreaped child.
	reapZombies()

	return true
}

// waitProcessGroupExit 轮询进程组是否已无存活成员（kill(-pgid, 0) 返回 ESRCH）。
// 组内僵尸进程也算成员，因此每轮先收割；EPERM（存在但无权限）视作存活。
func waitProcessGroupExit(pgid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		reapZombies()
		if err := syscall.Kill(-pgid, 0); err != nil {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// reapZombies collects any zombie children that lost their parent.
// Using Wait4 with pid=-1 and WNOHANG reaps all children without blocking.
func reapZombies() {
	for {
		var status syscall.WaitStatus
		wpid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if wpid <= 0 || err != nil {
			break
		}
	}
}

func (r *Runner) ensureServiceNotReachable(ctx context.Context, svc *Service) error {
	if svc == nil {
		return fmt.Errorf("service is required")
	}
	// Use the caller's ctx, not context.Background(): a slow health endpoint
	// must not outlive the precise-restart / stop-phase deadline (OPT-20260817-034).
	if checkProbe(ctx, svc.HealthCheck) != nil {
		return nil
	}

	ports := resolveServicePorts(svc)
	if len(ports) == 0 {
		return fmt.Errorf("service %q still reachable at %s after stop", svc.Name, svc.HealthCheck.DisplayEndpoint())
	}

	for _, port := range ports {
		pids, listErr := r.listenerPIDsForPort(port)
		if listErr != nil {
			log.Printf("[%s] list listeners on port %s failed: %v", svc.Name, port, listErr)
		}
		// Reachable but no visible PIDs: common when a privileged foreign process
		// (e.g. systemd redis-server as user "redis") still answers the health
		// endpoint. lsof without root cannot list it, and kill would be EPERM.
		// stop_command already ran; residual reachability is outside managed ownership.
		if len(pids) == 0 {
			log.Printf(
				"[%s] endpoint still reachable at %s after stop, but no visible listeners on port %s; treating managed stop as complete",
				svc.Name,
				svc.HealthCheck.DisplayEndpoint(),
				port,
			)
			return nil
		}
		if err := terminateListenersByPort(port); err != nil {
			log.Printf("[%s] stop fallback on port %s failed: %v", svc.Name, port, err)
		}
		if checkProbe(ctx, svc.HealthCheck) != nil {
			return nil
		}
	}

	return fmt.Errorf("service %q still reachable at %s after stop", svc.Name, svc.HealthCheck.DisplayEndpoint())
}

func (r *Runner) listenerPIDsForPort(port string) ([]int, error) {
	if r != nil && r.listenerPIDsFn != nil {
		return r.listenerPIDsFn(port)
	}
	return listenerPIDs(port)
}

func (r *Runner) hasActivePortListeners(svc *Service) bool {
	if svc == nil {
		return false
	}
	now := time.Now()
	r.portProbeCacheMu.RLock()
	if entry, ok := r.portProbeCache[svc.Name]; ok && now.Before(entry.expires) {
		r.portProbeCacheMu.RUnlock()
		return entry.active
	}
	r.portProbeCacheMu.RUnlock()

	active := serviceHasActivePortInSnapshot(svc, r.listeningTCPPortsSnapshot())

	r.portProbeCacheMu.Lock()
	if r.portProbeCache == nil {
		r.portProbeCache = make(map[string]portProbeCacheEntry)
	}
	r.portProbeCache[svc.Name] = portProbeCacheEntry{active: active, expires: now.Add(portProbeCacheTTL)}
	r.portProbeCacheMu.Unlock()
	return active
}

func (r *Runner) probeActivePortListeners(svc *Service) bool {
	if svc == nil {
		return false
	}
	probe := infrastructure.NewLsofPortListenerProbeRepository(r.listenerPIDsForPort)
	runtime := domain.NewServicePortRuntimeService(probe)
	active, port, err := runtime.HasActivePortListeners(resolveServicePorts(svc))
	if err != nil {
		log.Printf("[%s] port listener probe failed, treating as running: %v", svc.Name, err)
		return true
	}
	_ = port
	return active
}

func (r *Runner) finalizeServiceStop(ctx context.Context, svc *Service) error {
	if svc == nil {
		return fmt.Errorf("service is required")
	}
	hadTrackedProcess := r.stopProcess(svc.Name)
	if r.probeActivePortListeners(svc) {
		return r.ensureServicePortsReleased(ctx, svc)
	}
	if !hadTrackedProcess {
		return r.ensureServiceNotReachable(ctx, svc)
	}
	return nil
}

func (r *Runner) ensureServicePortsReleased(ctx context.Context, svc *Service) error {
	if svc == nil {
		return fmt.Errorf("service is required")
	}
	ports := resolveServicePorts(svc)
	if len(ports) == 0 {
		return r.ensureServiceNotReachable(ctx, svc)
	}
	escalation := domain.NewServicePortStopEscalationService(
		infrastructure.NewLsofPortListenerProbeRepository(r.listenerPIDsForPort),
		infrastructure.NewSyscallForeignProcessTerminationRepository(),
	)
	port, remaining, err := escalation.ReleasePorts(ports)
	if err != nil {
		return fmt.Errorf("[%s] release port listeners: %w", svc.Name, err)
	}
	if port != "" {
		return fmt.Errorf(
			"service %q still has listeners on port %s after stop (pid=%v)",
			svc.Name,
			port,
			remaining,
		)
	}
	return nil
}

func resolveServicePorts(svc *Service) []string {
	seen := map[string]struct{}{}
	var ports []string

	for _, candidate := range []string{
		domain.ResolveHealthPort(svc.HealthCheck.URL),
		domain.ResolveTCPPort(svc.HealthCheck.TCP),
		domain.ResolveCommandPort(svc.Command),
		domain.ResolveCommandPort(svc.EffectiveStartCommand()),
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		ports = append(ports, candidate)
	}

	return ports
}

func terminateListenersByPort(port string) error {
	pids, err := listenerPIDs(port)
	if err != nil {
		return err
	}
	return terminatePIDList(port, pids)
}

// terminatePortListeners 终止占用指定端口的监听进程（经 listenerPIDsFn 可注入）。
func (r *Runner) terminatePortListeners(port string) error {
	pids, err := r.listenerPIDsForPort(port)
	if err != nil {
		return err
	}
	return terminatePIDList(port, pids)
}

func terminatePIDList(port string, pids []int) error {
	if len(pids) == 0 {
		return nil
	}

	for _, pid := range pids {
		// 绝不终止自身进程：进程内监听者（如测试进程内 health server、守护进程
		// 自身 HTTP 端口）被 lsof 列为监听者时，杀掉它等于自杀（测试场景直接
		// 表现为 signal: terminated）。
		if pid == os.Getpid() {
			log.Printf("[terminateListenersByPort] skip own pid %d on port %s", pid, port)
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	time.Sleep(250 * time.Millisecond)

	remaining := make([]int, 0, len(pids))
	for _, pid := range pids {
		if err := syscall.Kill(pid, 0); err == nil {
			remaining = append(remaining, pid)
		}
	}
	for _, pid := range remaining {
		if pid == os.Getpid() {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}

	return nil
}

func listenerPIDs(port string) ([]int, error) {
	if strings.TrimSpace(port) == "" {
		return nil, nil
	}

	cmd := exec.Command("lsof", "-t", "-iTCP:"+port, "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("list listeners on port %s: %w", port, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	pids := make([]int, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, convErr := strconv.Atoi(line)
		if convErr != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

// waitForPortFree polls until no process is listening on the given port or timeout.
// Returns nil when the port is free, or an error on timeout.
func waitForPortFree(port string, timeout time.Duration) error {
	return waitForPortFreeWith(listenerPIDs, port, timeout)
}

func (r *Runner) waitForPortFree(port string, timeout time.Duration) error {
	if r == nil {
		return waitForPortFree(port, timeout)
	}
	return waitForPortFreeWith(r.listenerPIDsForPort, port, timeout)
}

func waitForPortFreeWith(listFn func(string) ([]int, error), port string, timeout time.Duration) error {
	if listFn == nil {
		listFn = listenerPIDs
	}
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		pids, err := listFn(port)
		if err != nil {
			return fmt.Errorf("waitForPortFree: %w", err)
		}
		if len(pids) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("waitForPortFree: port %s still occupied by PID=%v after %v", port, pids, timeout)
		}
		<-ticker.C
	}
}

func stopOrderForGroup(services []Service) ([]string, error) {
	if len(services) == 0 {
		return nil, nil
	}

	servicesByName := make(map[string]Service, len(services))
	for _, svc := range services {
		servicesByName[svc.Name] = svc
	}

	visitState := make(map[string]int, len(services))
	order := make([]string, 0, len(services))

	var visit func(string) error
	visit = func(name string) error {
		switch visitState[name] {
		case 1:
			return fmt.Errorf("cyclic dependency detected at service %q", name)
		case 2:
			return nil
		}

		visitState[name] = 1
		svc := servicesByName[name]
		for _, dep := range svc.DependsOn {
			if _, exists := servicesByName[dep]; !exists {
				continue
			}
			if err := visit(dep); err != nil {
				return err
			}
		}
		visitState[name] = 2
		order = append(order, name)
		return nil
	}

	for _, svc := range services {
		if err := visit(svc.Name); err != nil {
			return nil, err
		}
	}

	stopOrder := make([]string, 0, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		stopOrder = append(stopOrder, order[i])
	}
	return stopOrder, nil
}

// minBuildMemAvailableKB: 编译前期望的最小 MemAvailable（KiB）。本机 tmpfs+全栈
// 跑满时 vite/go 编译常因内存不足被 OOM Kill（exit 137 / signal: killed），
// 导致「全部重新编译」不可复现失败（OPT-20260812-041）。
const minBuildMemAvailableKB = int64(4 * 1024 * 1024)

// buildOOMRetryEnv: OOM 重试时降低编译并发/GC 压力（激进 GC + 限制并行）。
var buildOOMRetryEnv = map[string]string{
	"GOGC":       "40",
	"GOMAXPROCS": "2",
}

// isOOMExit reports whether a build wait error indicates the process was
// killed by the OOM killer（exit 137 = 128+SIGKILL，或 Go 报 signal: killed）。
func isOOMExit(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "signal: killed") || strings.Contains(s, "exit status 137")
}

// readMemAvailableKB returns MemAvailable from /proc/meminfo in KiB.
func readMemAvailableKB() (int64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemAvailable:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				return kb, nil
			}
		}
	}
	return 0, fmt.Errorf("MemAvailable not found in /proc/meminfo")
}

// waitForBuildMemory polls MemAvailable until it clears the threshold or times
// out（20s），给 OS 回收 page cache 的机会后再启动 OOM 高发编译。读不到则放行，
// 避免构建被该探针阻断。
func waitForBuildMemory(ctx context.Context) (availKB int64, ok bool) {
	availKB, err := readMemAvailableKB()
	if err != nil || availKB >= minBuildMemAvailableKB {
		return availKB, err == nil
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return availKB, false
		case <-time.After(2 * time.Second):
		}
		availKB, err = readMemAvailableKB()
		if err != nil || availKB >= minBuildMemAvailableKB {
			return availKB, err == nil
		}
	}
	return availKB, false
}

// mergeEnv returns env with overrides applied（替换已有键，追加新键）。
func mergeEnv(env []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return env
	}
	result := append([]string(nil), env...)
	for k, v := range overrides {
		prefix := k + "="
		replaced := false
		for i, e := range result {
			if strings.HasPrefix(e, prefix) {
				result[i] = prefix + v
				replaced = true
			}
		}
		if !replaced {
			result = append(result, prefix+v)
		}
	}
	return result
}

func (r *Runner) runBuild(ctx context.Context, svc *Service, buildCommand string) error {
	workDir := svc.WorkingDir
	if workDir == "" && r.cfg != nil {
		workDir = "."
	}
	if err := checkBuildDiskSpace(workDir); err != nil {
		return fmt.Errorf("[%s] %w", svc.Name, err)
	}

	// OPT-20260812-041: MemAvailable 过低时先等待回收，避免 vite/go 编译 OOM Kill
	avail, ok := waitForBuildMemory(ctx)
	if !ok {
		r.appendLifecycleLog(svc.Name, fmt.Sprintf(
			"编译前 MemAvailable≈%.1fGi 仍低于 4Gi 阈值，继续构建（若 OOM 将自动降并发重试）",
			float64(avail)/(1024*1024)))
	} else if avail < minBuildMemAvailableKB {
		r.appendLifecycleLog(svc.Name, fmt.Sprintf(
			"MemAvailable 已恢复到 %.1fGi，开始构建", float64(avail)/(1024*1024)))
	}

	if err := r.runBuildAttempt(ctx, svc, buildCommand, nil); err != nil {
		if !isOOMExit(err) {
			return err
		}
		r.appendLifecycleLog(svc.Name, "构建被 OOM Kill（signal: killed / exit 137），降并发自动重试一次")
		if _, ok := waitForBuildMemory(ctx); !ok {
			r.appendLifecycleLog(svc.Name, "重试前 MemAvailable 仍低于阈值，继续重试")
		}
		if retryErr := r.runBuildAttempt(ctx, svc, buildCommand, buildOOMRetryEnv); retryErr != nil {
			if isOOMExit(retryErr) {
				return fmt.Errorf("[%s] build failed (OOM, retry also killed): %w", svc.Name, retryErr)
			}
			return fmt.Errorf("[%s] build failed (retry after OOM): %w", svc.Name, retryErr)
		}
		return nil
	}
	return nil
}

// runBuildAttempt executes the build command once, streaming output through
// pipes we own. envOverrides are applied on top of the service environment
// （用于 OOM 降并发重试）。
func (r *Runner) runBuildAttempt(ctx context.Context, svc *Service, buildCommand string, envOverrides map[string]string) error {
	cmd := exec.CommandContext(ctx, getBashPath(), "-c", buildCommand)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if dir := lifecycleWorkDir(svc); dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = mergeEnv(r.buildServiceEnv(*svc), envOverrides)

	// Route build output through pipes we own instead of cmd.StdoutPipe/
	// StderrPipe: cmd.Wait() closes those read ends as soon as the process
	// exits, racing the streamOutput goroutines — under load the goroutines
	// may not be scheduled in time, losing the tail of the build log
	// ("output read error: read |0: file already closed"). Here we hold the
	// write ends, so we close them after Wait to signal EOF and then drain
	// the goroutines before returning.
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamOutput(stdoutR, svc.Name, domain.StreamStdout, r.logRepository)
	}()
	go func() {
		defer wg.Done()
		streamOutput(stderrR, svc.Name, domain.StreamStderr, r.logRepository)
	}()

	if err := cmd.Start(); err != nil {
		stdoutW.Close()
		stderrW.Close()
		wg.Wait()
		return fmt.Errorf("[%s] build failed to start: %w", svc.Name, err)
	}

	waitErr := cmd.Wait()
	stdoutW.Close()
	stderrW.Close()
	wg.Wait()

	if waitErr != nil {
		return fmt.Errorf("[%s] build failed: %w", svc.Name, waitErr)
	}

	log.Printf("[%s] build succeeded", svc.Name)
	return nil
}

func (r *Runner) appendLifecycleLog(serviceName, message string) {
	if r == nil || r.logRepository == nil {
		return
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		return
	}
	if !strings.HasPrefix(msg, "[runAll]") {
		msg = "[runAll] " + msg
	}
	// Lifecycle messages are runAll orchestration events (not service stderr).
	// Use stdout so the UI renders them as informational rather than error output.
	entry, err := domain.NewLogEntry(time.Now(), serviceName, domain.StreamStdout, msg)
	if err != nil {
		log.Printf("[%s] lifecycle log skipped: %v", serviceName, err)
		return
	}
	r.logRepository.Append(serviceName, entry)
}

// appendStructuredLog writes a JSON log entry that Promtail can parse into Loki labels
// (service, level, msg, trace_id). The entry flows through the same tee→Promtail→Loki
// pipeline as the services' own structured logs, making runAll-originated status entries
// visible in Grafana dashboards like trace-log-explore.
func (r *Runner) appendStructuredLog(serviceName, level, msg string, extra map[string]interface{}) {
	if r == nil || r.logRepository == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	traceID := ""
	if raw, ok := extra["trace_id"]; ok {
		if s, ok := raw.(string); ok {
			traceID = strings.TrimSpace(s)
		}
	}
	fields := []string{
		fmt.Sprintf(`"ts":"%s"`, now),
		fmt.Sprintf(`"level":"%s"`, level),
		fmt.Sprintf(`"msg":"%s"`, msg),
		fmt.Sprintf(`"service":"%s"`, serviceName),
		fmt.Sprintf(`"trace_id":"%s"`, traceID),
	}
	for k, v := range extra {
		if k == "trace_id" {
			continue
		}
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		fields = append(fields, fmt.Sprintf(`"%s":%s`, k, string(b)))
	}
	payload := "{" + strings.Join(fields, ",") + "}"
	entry, err := domain.NewLogEntry(time.Now(), serviceName, domain.StreamStdout, payload)
	if err != nil {
		log.Printf("[%s] structured log skipped: %v", serviceName, err)
		return
	}
	r.logRepository.Append(serviceName, entry)
}

// emitCollectionStatus writes a collection_status structured log entry for every
// configured service. These entries flow through Promtail→Loki and appear in
// Grafana dashboards, making each service visible even when it produces no logs.
func (r *Runner) emitCollectionStatus() {
	if r == nil || r.cfg == nil {
		return
	}
	for _, svc := range r.cfg.Flatten() {
		var stats domain.LogCollectionStats
		if r.logStatsProvider != nil {
			stats = r.logStatsProvider.Stats(svc.Name)
		}

		st := r.store.Get(svc.Name)
		svcStatus := "unknown"
		if st != nil {
			svcStatus = string(st.Status)
		}

		extra := map[string]interface{}{
			"total_lines":    stats.TotalLines,
			"total_bytes":    stats.TotalBytes,
			"service_status": svcStatus,
			"has_file_sink":  r.fileLogSink != nil,
			"collection_ts":  time.Now().UTC().Format(time.RFC3339),
		}
		if !stats.LastLogAt.IsZero() {
			extra["last_log_at"] = stats.LastLogAt.UTC().Format(time.RFC3339)
		}

		r.appendStructuredLog(svc.Name, "info", "collection_status", extra)
	}
}

// StartLogCollectionHeartbeat begins periodic emission of collection_status log
// entries at the given interval. These entries make each service's log collection
// status visible in Grafana dashboards.
func (r *Runner) StartLogCollectionHeartbeat(interval time.Duration) {
	if r == nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.collectionHeartbeatCancel = cancel

	log.Printf("[runAll] log collection heartbeat started (interval=%s)", interval)
	// Emit once immediately so services are visible on startup.
	r.emitCollectionStatus()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Printf("[runAll] log collection heartbeat stopped")
				return
			case <-ticker.C:
				r.emitCollectionStatus()
			}
		}
	}()
}

// StopLogCollectionHeartbeat stops the periodic collection_status emission.
func (r *Runner) StopLogCollectionHeartbeat() {
	if r == nil || r.collectionHeartbeatCancel == nil {
		return
	}
	r.collectionHeartbeatCancel()
	r.collectionHeartbeatCancel = nil
}

// LogStatsProvider returns the log statistics provider for API consumption.
func (r *Runner) LogStatsProvider() domain.ServiceLogStatsProvider {
	if r == nil {
		return nil
	}
	return r.logStatsProvider
}

func (r *Runner) logLifecyclePlanQueue(plan domain.ServiceLifecyclePlan) {
	total := len(plan.OrderedNames)
	if total == 0 {
		return
	}
	for i, name := range plan.OrderedNames {
		if i == 0 {
			r.appendLifecycleLog(name, fmt.Sprintf("%s cascade: step 1/%d starting now", plan.Operation, total))
			continue
		}
		r.appendLifecycleLog(
			name,
			fmt.Sprintf("%s cascade: queued step %d/%d (waiting for prior services)", plan.Operation, i+1, total),
		)
	}
}

func (r *Runner) logContains(name, substr string) bool {
	if r == nil || r.logRepository == nil || substr == "" {
		return false
	}
	for _, entry := range r.logRepository.Tail(name, 40) {
		if strings.Contains(entry.Message, substr) {
			return true
		}
	}
	return false
}

// localProbeVisible requires the probe in the in-memory tee AND on the file path
// Promtail tails. Memory-only visibility (e.g. writes to a deleted inode) is not enough.
func (r *Runner) localProbeVisible(name, probeID string) bool {
	if !r.logContains(name, probeID) {
		return false
	}
	if r == nil || r.fileLogSink == nil || r.cfg == nil {
		return true
	}
	confAppLogFile := ""
	for _, svc := range r.cfg.Flatten() {
		if svc.Name == name {
			confAppLogFile = svc.LogFile
			break
		}
	}
	path := resolveServiceLogFile(r.cfg, name, confAppLogFile)
	if strings.TrimSpace(path) == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), probeID)
}

func (r *Runner) bootStillInProgress() bool {
	if r == nil || r.store == nil {
		return false
	}
	for _, svc := range r.store.All() {
		switch svc.Status {
		case StatusStarting, StatusRetrying, StatusRestarting:
			return true
		}
	}
	return false
}

func (r *Runner) IsDAGBootInProgress() bool {
	if r == nil {
		return false
	}
	return atomic.LoadInt32(&r.dagBootInProgress) == 1
}

func (r *Runner) setDAGBootInProgress(active bool) {
	if r == nil {
		return
	}
	if active {
		atomic.StoreInt32(&r.dagBootInProgress, 1)
		return
	}
	atomic.StoreInt32(&r.dagBootInProgress, 0)
}

func (r *Runner) logDAGBootFinished() {
	if r == nil || r.store == nil {
		log.Println("[runAll] DAG boot finished.")
		return
	}
	var healthy, failed, skipped, other int
	for _, svc := range r.store.All() {
		switch svc.Status {
		case StatusHealthy:
			healthy++
		case StatusFailed:
			failed++
		case StatusSkipped:
			skipped++
		default:
			other++
		}
	}
	if failed > 0 || skipped > 0 || other > 0 {
		log.Printf(
			"[runAll] DAG boot finished (%d healthy, %d failed, %d skipped, %d other). Manual start-all available for retries.",
			healthy, failed, skipped, other,
		)
		return
	}
	log.Println("[runAll] DAG boot finished. All services healthy.")
}

func (r *Runner) rejectManualStartDuringDAGBoot() error {
	if r.IsDAGBootInProgress() {
		return fmt.Errorf("runAll 正在自动引导服务，请等待 DAG 引导完成后再试")
	}
	return nil
}

// tryResolveStartWithoutLaunch handles idempotent start when another goroutine already
// launched the service or the service is already healthy.
func (r *Runner) tryResolveStartWithoutLaunch(ctx context.Context, svc *Service, current *ServiceStatus) (bool, error) {
	if svc == nil || current == nil {
		return false, nil
	}
	switch current.Status {
	case StatusHealthy:
		if err := checkProbe(ctx, svc.HealthCheck); err != nil {
			return false, fmt.Errorf("service %q is %s, can only start idle services", svc.Name, current.Status)
		}
		r.appendLifecycleLog(svc.Name, "start skipped: already healthy")
		r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
		r.startMonitoring(ctx, *svc)
		return true, ErrStartSkippedAlreadyHealthy
	case StatusStarting, StatusRestarting:
		r.appendLifecycleLog(
			svc.Name,
			fmt.Sprintf("start waiting: launch already in progress (status=%s)", current.Status),
		)
		if err := r.waitForServiceStart(ctx, *svc); err != nil {
			return true, err
		}
		return true, ErrStartSkippedAlreadyHealthy
	// StatusRetrying intentionally falls through to the reclaim path below: a
	// stuck retry (canary overlap left the store mid-health-check) must be
	// reclaimable by an explicit Start, mirroring how restartService reclaims it.
	// Genuinely in-flight launches owned by another goroutine are protected by
	// the Starting/Restarting wait above plus the supersede guards in startAndCheck.
	default:
		return false, nil
	}
}

func (r *Runner) explainPendingCause(name string) string {
	st := r.store.Get(name)
	if st == nil {
		return "service not found in status store"
	}
	if st.Status != StatusPending {
		return fmt.Sprintf("status is %s (not the default not-started state)", st.Status)
	}
	if st.Error != "" {
		return fmt.Sprintf("not started with error: %s", st.Error)
	}
	if r.logContains(name, "start requested") {
		return "start API accepted but not started yet — check runAll terminal for async start errors"
	}
	if r.logContains(name, "cascade: queued") {
		return "queued in start-all/start-group — waiting for prior services in the chain to finish"
	}
	if r.logContains(name, "DAG level") && !r.logContains(name, "starting now") {
		return "queued in start-all/start-group — waiting for in-plan dependencies to finish starting"
	}
	if r.logContains(name, "cascade: step 1") {
		return "first in cascade plan — start should begin shortly; if stuck, check runAll terminal logs"
	}
	for _, dep := range st.DependsOn {
		if dep.Status != StatusHealthy {
			return fmt.Sprintf(
				"blocked by dependency %q (status=%s) — fix/start the dependency first, or use 全部启动 for ordered cascade",
				dep.Name,
				dep.Status,
			)
		}
	}
	if r.bootStillInProgress() {
		return "runAll is still booting earlier services — this one will auto-start when its DAG level is reached"
	}
	return "not started yet — click 启动 on this row or 全部启动"
}

func (r *Runner) refreshPendingDiagnostic(name string) {
	if r == nil || r.store == nil {
		return
	}
	st := r.store.Get(name)
	if st == nil {
		return
	}
	if st.Status != StatusPending && !(st.Status == StatusStarting && st.PID <= 0) {
		return
	}
	cause := r.explainPendingCause(name)
	if st.Status == StatusStarting && st.PID <= 0 {
		cause = fmt.Sprintf("starting without PID yet: %s", cause)
	}
	msg := "pending diagnostic: " + cause
	for _, entry := range r.logRepository.Tail(name, 8) {
		if strings.Contains(entry.Message, msg) {
			return
		}
	}
	r.appendLifecycleLog(name, msg)
}

func (r *Runner) lifecycleLogSnapshot(name string) []domain.LogEntry {
	if r == nil || r.store == nil {
		return nil
	}
	st := r.store.Get(name)
	if st == nil {
		entry, err := domain.NewLogEntry(time.Now(), name, domain.StreamStdout, "[runAll] service not found in status store")
		if err != nil {
			return nil
		}
		return []domain.LogEntry{entry}
	}

	messages := []string{}
	if st.Status != StatusPending {
		messages = append(messages, fmt.Sprintf("status=%s", st.Status))
	} else {
		messages = append(messages, "status=(not started)")
	}
	if st.Phase != "" {
		messages = append(messages, fmt.Sprintf("phase=%s", st.Phase))
	}
	if st.Readiness != "" {
		messages = append(messages, fmt.Sprintf("readiness=%s", st.Readiness))
	}
	if st.Error != "" {
		messages = append(messages, fmt.Sprintf("error: %s", st.Error))
	}
	if st.ReadinessError != "" {
		messages = append(messages, fmt.Sprintf("readiness_error: %s", st.ReadinessError))
	}
	if st.FailureCode != "" {
		messages = append(messages, fmt.Sprintf("failure_code=%s", st.FailureCode))
	}
	if len(st.DependsOn) > 0 {
		for _, dep := range st.DependsOn {
			messages = append(messages, fmt.Sprintf("dependency %s: %s", dep.Name, dep.Status))
		}
	}
	switch st.Status {
	case StatusPending:
		messages = append(messages, r.explainPendingCause(name))
	case StatusStarting, StatusRetrying, StatusRestarting:
		if st.PID <= 0 {
			messages = append(messages, "starting — waiting to launch process or for readiness probes")
		}
	}

	entries := make([]domain.LogEntry, 0, len(messages))
	now := time.Now()
	for _, message := range messages {
		entry, err := domain.NewLogEntry(now, name, domain.StreamStdout, "[runAll] "+message)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func streamOutput(reader io.Reader, name, stream string, repository domain.ServiceLogRepository) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if repository != nil {
			entry, err := domain.NewLogEntry(time.Now(), name, stream, line)
			if err != nil {
				log.Printf("[%s] log entry skipped: %v", name, err)
				continue
			}
			repository.Append(name, entry)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[%s] output read error: %v", name, err)
	}
}

func (r *Runner) startMonitoring(ctx context.Context, svc Service) {
	r.monitorMu.Lock()
	if _, exists := r.monitors[svc.Name]; exists {
		r.monitorMu.Unlock()
		return
	}
	monCtx, cancel := context.WithCancel(ctx)
	r.monitors[svc.Name] = cancel
	r.monitorMu.Unlock()

	interval := time.Duration(svc.HealthCheck.CheckInterval) * time.Second
	go r.runMonitor(monCtx, svc, interval)
}

func (r *Runner) stopMonitoring(name string) {
	r.monitorMu.Lock()
	defer r.monitorMu.Unlock()
	if cancel, ok := r.monitors[name]; ok {
		cancel()
		delete(r.monitors, name)
	}
}

func (r *Runner) stopAllMonitors() {
	r.monitorMu.Lock()
	defer r.monitorMu.Unlock()
	for name, cancel := range r.monitors {
		cancel()
		delete(r.monitors, name)
	}
}

func (r *Runner) removeMonitor(name string) {
	r.monitorMu.Lock()
	defer r.monitorMu.Unlock()
	delete(r.monitors, name)
}

// reconcileFailedServiceHealthThrottled limits health re-probes so /api/status stays responsive.
func (r *Runner) reconcileFailedServiceHealthThrottled(ctx context.Context) {
	if r == nil {
		return
	}
	now := time.Now()
	r.lastFailedHealthReconcileMu.Lock()
	if !r.lastFailedHealthReconcile.IsZero() && now.Sub(r.lastFailedHealthReconcile) < failedHealthReconcileMinInterval {
		r.lastFailedHealthReconcileMu.Unlock()
		return
	}
	r.lastFailedHealthReconcile = now
	r.lastFailedHealthReconcileMu.Unlock()
	r.reconcileFailedServiceHealth(ctx)
}

// reconcileFailedServiceHealth re-probes services marked failed. When the health
// endpoint is reachable again (e.g. Docker stack restarted manually), status is
// restored so the UI refresh reflects recovery without a full runAll restart.
func (r *Runner) reconcileFailedServiceHealth(ctx context.Context) {
	if r == nil || r.cfg == nil {
		return
	}
	var recoveredNames []string
	for _, svc := range r.cfg.Flatten() {
		st := r.store.Get(svc.Name)
		if st == nil || st.Status != StatusFailed {
			continue
		}
		if !svc.HealthCheck.UsesTCP() && !svc.HealthCheck.UsesExec() && strings.TrimSpace(svc.HealthCheck.URL) == "" {
			continue
		}
		if svc.HealthCheck.UsesTCP() && strings.TrimSpace(svc.HealthCheck.TCP) == "" {
			continue
		}
		if svc.HealthCheck.UsesExec() && strings.TrimSpace(svc.HealthCheck.Exec) == "" {
			continue
		}
		now := time.Now()
		if svc.HealthCheck.HasSplitProbe() {
			if err := checkHealth(ctx, svc.HealthCheck.StartupProbeURL()); err != nil {
				r.store.SetLastChecked(svc.Name, now)
				continue
			}
			r.store.Update(svc.Name, StatusHealthy, "")
			r.probeReadiness(ctx, svc)
			r.store.SetLastChecked(svc.Name, now)
			r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
			log.Printf("[%s] recovered (liveness ok on status reconcile)", svc.Name)
			r.startMonitoring(context.Background(), svc)
			recoveredNames = append(recoveredNames, svc.Name)
			continue
		}
		if err := checkProbe(ctx, svc.HealthCheck); err != nil {
			r.store.SetLastChecked(svc.Name, now)
			continue
		}
		r.store.Update(svc.Name, StatusHealthy, "")
		r.store.SetLastChecked(svc.Name, now)
		r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
		log.Printf("[%s] recovered (health check ok on status reconcile)", svc.Name)
		r.startMonitoring(context.Background(), svc)
		recoveredNames = append(recoveredNames, svc.Name)
	}
	// Wake up dependents that were skipped/failed waiting for recovered services.
	for _, name := range recoveredNames {
		r.restartStoppedDependents(ctx, name)
	}
}

// restartStoppedDependents finds all services that depend on the given service
// and are stuck in a stopped/skipped/failed state due to dependency failures,
// then restarts them. This is the "wake-up" mechanism: when an upstream dependency
// recovers from failure, downstream services that gave up waiting are
// automatically restarted instead of staying permanently stopped.
func (r *Runner) restartStoppedDependents(ctx context.Context, dependencyName string) {
	if r == nil || r.cfg == nil {
		return
	}
	for _, svc := range r.cfg.Flatten() {
		if !dependsOnService(svc.DependsOn, dependencyName) {
			continue
		}
		st := r.store.Get(svc.Name)
		if st == nil {
			continue
		}
		// Only restart services that were skipped/failed/stopped — healthy or
		// already-starting services don't need intervention.
		if !domain.IsStartableServiceStatus(string(st.Status)) {
			continue
		}
		log.Printf("[%s] dependency %q recovered, restarting", svc.Name, dependencyName)
		r.appendLifecycleLog(svc.Name,
			fmt.Sprintf("dependency %q recovered — auto-restarting", dependencyName))
		// Use a fresh context so the restart isn't tied to the monitor's lifecycle.
		if err := r.startService(context.Background(), svc.Name); err != nil {
			log.Printf("[%s] auto-restart after dependency recovery failed: %v", svc.Name, err)
			r.appendLifecycleLog(svc.Name,
				fmt.Sprintf("auto-restart failed: %v", err))
		}
	}
}

func (r *Runner) runMonitor(ctx context.Context, svc Service, interval time.Duration) {
	defer r.removeMonitor(svc.Name)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	consecutive := 0
	threshold := svc.HealthCheck.UnhealthyThreshold
	if threshold < 1 {
		threshold = 1
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			r.store.SetLastChecked(svc.Name, now)

			if svc.HealthCheck.HasSplitProbe() {
				liveErr := checkHealth(ctx, svc.HealthCheck.StartupProbeURL())
				if liveErr != nil {
					consecutive++
					log.Printf("[%s] liveness check failed (%d/%d): %v", svc.Name, consecutive, threshold, liveErr)
					if consecutive >= threshold {
						if st := r.store.Get(svc.Name); st == nil || st.Status != StatusFailed {
							r.store.Update(svc.Name, StatusFailed, liveErr.Error())
							log.Printf("[%s] marked failed after %d consecutive liveness failures", svc.Name, consecutive)
							r.tryAutoRestartFailedService(svc)
						}
						consecutive = threshold
					}
					continue
				}
				consecutive = 0
				if st := r.store.Get(svc.Name); st != nil && st.Status == StatusFailed {
					r.store.Update(svc.Name, StatusHealthy, "")
					r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
					log.Printf("[%s] recovered (liveness check ok)", svc.Name)
					// Wake up dependents that were skipped waiting for this service.
					go r.restartStoppedDependents(context.Background(), svc.Name)
				}
				if readyErr := checkHealth(ctx, svc.HealthCheck.ReadinessProbeURL()); readyErr != nil {
					r.store.SetReadiness(svc.Name, ReadinessDegraded, readyErr.Error())
					log.Printf("[%s] readiness degraded: %v", svc.Name, readyErr)
				} else {
					r.store.SetReadiness(svc.Name, ReadinessReady, "")
				}
				continue
			}

			err := checkProbe(ctx, svc.HealthCheck)
			if err != nil {
				consecutive++
				log.Printf("[%s] health check failed (%d/%d): %v", svc.Name, consecutive, threshold, err)
				if consecutive >= threshold {
					if st := r.store.Get(svc.Name); st == nil || st.Status != StatusFailed {
						r.store.Update(svc.Name, StatusFailed, err.Error())
						log.Printf("[%s] marked failed after %d consecutive failures", svc.Name, consecutive)
						// Auto-restart exited (zombie) or alive-but-unhealthy process (OPT-20260902-026):
						// do not wait up to 5 minutes for the external cron watchdog.
						r.tryAutoRestartFailedService(svc)
					}
					consecutive = threshold
				}
				continue
			}
			consecutive = 0
			if st := r.store.Get(svc.Name); st != nil && st.Status == StatusFailed {
				r.store.Update(svc.Name, StatusHealthy, "")
				r.store.UpdateDependencyStatus(svc.Name, StatusHealthy)
				log.Printf("[%s] recovered (health check ok)", svc.Name)
			}
		}
	}
}

// SubscribeStartAllProgress subscribes to progress events for the active start-all run.
// Returns a channel and the current runID. If no start-all is active, returns nil channel.
func (r *Runner) SubscribeStartAllProgress() (chan StartAllProgressEvent, string) {
	r.activeStartAllRunIDMu.RLock()
	runID := r.activeStartAllRunID
	r.activeStartAllRunIDMu.RUnlock()
	if runID == "" {
		return nil, ""
	}
	ch := r.progressBroadcaster.Subscribe(runID)
	return ch, runID
}

// GetActiveStartAllRunID returns the currently active start-all run ID, or empty string.
func (r *Runner) GetActiveStartAllRunID() string {
	r.activeStartAllRunIDMu.RLock()
	defer r.activeStartAllRunIDMu.RUnlock()
	return r.activeStartAllRunID
}

// SetActiveStartAllRunID generates a new runID, stores it, and returns it.
// The executeParallelStartPlan will use this pre-set runID if available.
func (r *Runner) SetActiveStartAllRunID() string {
	runID := fmt.Sprintf("start-all-%d", time.Now().UnixNano())
	r.activeStartAllRunIDMu.Lock()
	r.activeStartAllRunID = runID
	r.activeStartAllRunIDMu.Unlock()
	return runID
}

// bulkProgressRetainAfterTerminal keeps a terminal bulk snapshot long enough for
// late SSE subscribers, then drops the active runID so status polling cannot
// replay the same done/error event forever (toast storm).
var bulkProgressRetainAfterTerminal = 30 * time.Second

func (r *Runner) publishStartAllTerminalError(err error) {
	if r == nil || err == nil {
		return
	}
	runID := r.GetActiveStartAllRunID()
	if runID == "" {
		return
	}
	log.Printf("[cascade] start all terminal error run_id=%s err=%v", runID, err)
	if r.progressBroadcaster != nil {
		r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
			Done: true, Phase: "error", Error: err.Error(), Operation: "start",
		})
	}
	r.scheduleReleaseStartAllRun(runID)
}

func (r *Runner) publishStopAllTerminalError(err error) {
	if r == nil || err == nil {
		return
	}
	runID := r.GetActiveStopAllRunID()
	if runID == "" {
		return
	}
	log.Printf("[cascade] stop all terminal error run_id=%s err=%v", runID, err)
	if r.progressBroadcaster != nil {
		r.progressBroadcaster.Publish(runID, StartAllProgressEvent{
			Done: true, Phase: "error", Error: err.Error(), Operation: "stop",
		})
	}
	r.scheduleReleaseStopAllRun(runID)
}

func (r *Runner) scheduleReleaseStartAllRun(runID string) {
	r.scheduleReleaseBulkRun(runID, r.clearActiveStartAllRunIDIf)
}

func (r *Runner) scheduleReleaseStopAllRun(runID string) {
	r.scheduleReleaseBulkRun(runID, r.clearActiveStopAllRunIDIf)
}

func (r *Runner) clearActiveStartAllRunIDIf(runID string) {
	if r == nil || runID == "" {
		return
	}
	r.activeStartAllRunIDMu.Lock()
	if r.activeStartAllRunID == runID {
		r.activeStartAllRunID = ""
	}
	r.activeStartAllRunIDMu.Unlock()
}

func (r *Runner) clearActiveStopAllRunIDIf(runID string) {
	if r == nil || runID == "" {
		return
	}
	r.activeStopAllRunIDMu.Lock()
	if r.activeStopAllRunID == runID {
		r.activeStopAllRunID = ""
	}
	r.activeStopAllRunIDMu.Unlock()
}

func (r *Runner) scheduleReleaseBulkRun(runID string, clear func(string)) {
	if r == nil || runID == "" {
		return
	}
	delay := bulkProgressRetainAfterTerminal
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		if clear != nil {
			clear(runID)
		}
		if r.progressBroadcaster != nil {
			r.progressBroadcaster.CloseRun(runID)
		}
	}()
}

// GetProgressBroadcaster returns the underlying progress broadcaster (for SSE unsubscribe).
func (r *Runner) GetProgressBroadcaster() *ProgressBroadcaster {
	return r.progressBroadcaster
}

// SubscribeStopAllProgress subscribes to progress events for the active stop-all run.
func (r *Runner) SubscribeStopAllProgress() (chan StartAllProgressEvent, string) {
	r.activeStopAllRunIDMu.RLock()
	runID := r.activeStopAllRunID
	r.activeStopAllRunIDMu.RUnlock()
	if runID == "" {
		return nil, ""
	}
	ch := r.progressBroadcaster.Subscribe(runID)
	return ch, runID
}

// GetActiveStopAllRunID returns the currently active stop-all run ID, or empty string.
func (r *Runner) GetActiveStopAllRunID() string {
	r.activeStopAllRunIDMu.RLock()
	defer r.activeStopAllRunIDMu.RUnlock()
	return r.activeStopAllRunID
}

// SetActiveStopAllRunID generates a new runID for stop-all, stores it, and returns it.
func (r *Runner) SetActiveStopAllRunID() string {
	runID := fmt.Sprintf("stop-all-%d", time.Now().UnixNano())
	r.activeStopAllRunIDMu.Lock()
	r.activeStopAllRunID = runID
	r.activeStopAllRunIDMu.Unlock()
	return runID
}

// SubscribeBuildAllProgress subscribes to progress events for the active build-all run.
func (r *Runner) SubscribeBuildAllProgress() (chan StartAllProgressEvent, string) {
	r.activeBuildAllRunIDMu.RLock()
	runID := r.activeBuildAllRunID
	r.activeBuildAllRunIDMu.RUnlock()
	if runID == "" {
		return nil, ""
	}
	ch := r.progressBroadcaster.Subscribe(runID)
	return ch, runID
}

// GetActiveBuildAllRunID returns the currently active build-all run ID, or empty string.
func (r *Runner) GetActiveBuildAllRunID() string {
	r.activeBuildAllRunIDMu.RLock()
	defer r.activeBuildAllRunIDMu.RUnlock()
	return r.activeBuildAllRunID
}

// SetActiveBuildAllRunID generates a new runID for build-all, stores it, and returns it.
// Prefer TryBeginBuildAllRun for API handlers that must reject concurrent bulk builds.
func (r *Runner) SetActiveBuildAllRunID() string {
	runID := fmt.Sprintf("build-all-%d", time.Now().UnixNano())
	r.activeBuildAllRunIDMu.Lock()
	r.activeBuildAllRunID = runID
	r.activeBuildAllRunIDMu.Unlock()
	return runID
}

// TryBeginBuildAllRun atomically starts a bulk-build run if none is active.
// Returns ("", false) when a global or group rebuild is already in progress,
// or when another bulk op (start/stop/restart/precise) holds the bulk mutex
// (OPT-20260807-052).
func (r *Runner) TryBeginBuildAllRun() (string, bool) {
	if !r.TryBeginBulk("build-all") {
		return "", false
	}
	r.activeBuildAllRunIDMu.Lock()
	defer r.activeBuildAllRunIDMu.Unlock()
	if r.activeBuildAllRunID != "" {
		r.EndBulk("build-all")
		return "", false
	}
	runID := fmt.Sprintf("build-all-%d", time.Now().UnixNano())
	r.activeBuildAllRunID = runID
	return runID, true
}

// ActiveBulkProgressSnapshot describes an in-flight bulk lifecycle operation
// (start-all / stop-all / build-all/group / restart-all) for UI resume after page refresh.
type ActiveBulkProgressSnapshot struct {
	Kind      string                 `json:"kind"` // "start-all", "stop-all", "build-all", "restart-all", "precise-restart", "init-db", "clear-db"
	RunID     string                 `json:"run_id"`
	Operation string                 `json:"operation,omitempty"`
	Label     string                 `json:"label,omitempty"`
	Event     *StartAllProgressEvent `json:"event,omitempty"`
}

// ActiveBulkProgress returns the first active bulk progress snapshot, if any.
// Priority: build-all > restart-all > stop-all > start-all.
func (r *Runner) ActiveBulkProgress() *ActiveBulkProgressSnapshot {
	if r == nil || r.progressBroadcaster == nil {
		return nil
	}
	if r.IsRestartAllActive() {
		runID := r.GetActiveRestartAllRunID()
		if runID != "" {
			snap := &ActiveBulkProgressSnapshot{Kind: "restart-all", RunID: runID, Operation: "restart", Label: bulkOpLabel("restart-all")}
			if ev, ok := r.progressBroadcaster.Latest(runID); ok {
				cp := ev
				snap.Event = &cp
				if ev.Operation != "" {
					snap.Operation = ev.Operation
				}
			}
			return snap
		}
	}
	if r.IsPreciseRestartActive() {
		if runID := r.GetActivePreciseRestartRunID(); runID != "" {
			snap := &ActiveBulkProgressSnapshot{Kind: "precise-restart", RunID: runID, Operation: "restart", Label: bulkOpLabel("precise-restart")}
			if ev, ok := r.progressBroadcaster.Latest(runID); ok {
				cp := ev
				snap.Event = &cp
				if ev.Operation != "" {
					snap.Operation = ev.Operation
				}
			}
			return snap
		}
	}
	if op, runID := r.GetActiveDevDBRun(); runID != "" {
		snap := &ActiveBulkProgressSnapshot{Kind: op, RunID: runID, Operation: op, Label: bulkOpLabel(op)}
		if ev, ok := r.progressBroadcaster.Latest(runID); ok {
			cp := ev
			snap.Event = &cp
			if ev.Operation != "" {
				snap.Operation = ev.Operation
			}
		}
		return snap
	}
	type candidate struct {
		kind  string
		runID string
	}
	cands := []candidate{
		{"build-all", r.GetActiveBuildAllRunID()},
		{"stop-all", r.GetActiveStopAllRunID()},
		{"start-all", r.GetActiveStartAllRunID()},
	}
	for _, c := range cands {
		if c.runID == "" {
			continue
		}
		snap := &ActiveBulkProgressSnapshot{Kind: c.kind, RunID: c.runID, Label: bulkOpLabel(c.kind)}
		if ev, ok := r.progressBroadcaster.Latest(c.runID); ok {
			cp := ev
			snap.Event = &cp
			snap.Operation = ev.Operation
		} else {
			switch c.kind {
			case "build-all":
				snap.Operation = "build"
			case "stop-all":
				snap.Operation = "stop"
			default:
				snap.Operation = "start"
			}
		}
		return snap
	}
	return nil
}

// GenerateSingleRunID creates a unique runID for a single-service operation.
func (r *Runner) GenerateSingleRunID(op, name string) string {
	return fmt.Sprintf("single-%s-%s-%d", op, name, time.Now().UnixNano())
}

// PublishProgress publishes a progress event to a specific runID.
func (r *Runner) PublishProgress(runID string, ev StartAllProgressEvent) {
	r.progressBroadcaster.Publish(runID, ev)
}

// SubscribeProgress subscribes to a specific runID and returns the channel.
func (r *Runner) SubscribeProgress(runID string) chan StartAllProgressEvent {
	return r.progressBroadcaster.Subscribe(runID)
}

// CloseProgressRun schedules cleanup of a progress run after a delay.
func (r *Runner) CloseProgressRun(runID string, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		r.progressBroadcaster.CloseRun(runID)
	}()
}
