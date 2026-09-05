package main

import (
	"context"
	"log"
	"net/http"
)

// bootstrapUIMode kills the previous UI listener, adopts managed services after
// shutdown-self, reloads exec_queue.json, then starts the status UI.
func bootstrapUIMode(ctx context.Context, store *StatusStore, runner *Runner, cfg *Config, uiPort string, cancel context.CancelFunc, exitReason *runAllExitReason) *http.Server {
	previous := killPreviousRunAllProcess(uiPort)
	// OPT-20260820-008: 无旧 UI listener（HadPrevious=false）时，仍在监听的托管端口属于
	// 上次 runAll 留下的存活服务，应走 adopt 而非当孤儿 SIGKILL（否则整栈被误杀，
	// Status UI 长期起不来）。
	skipOrphan := shouldSkipOrphanCleanup(previous)
	cleanupOrphanManagedServices(cfg, previous, skipOrphan)
	cleanupResidualRunAllProcesses(uiPort)
	if skipOrphan {
		if n := runner.adoptListeningServices(ctx); n > 0 {
			log.Printf("[runAll] pre-UI adopted %d listening service(s)", n)
		}
		if previous.GracefulShutdownSelf {
			// 热替换补偿标记：UI 启动后对 adopt-skip 且配置探针的服务自动补 start
			// （OPT-20260812-048），替代人工 start-all。
			runner.hotReplaceMode = true
		}
	}
	// After kill/adopt so the previous process has flushed exec_queue.json.
	runner.LoadExecQueueFromDisk()
	srv := startUIServer(store, runner, uiPort, cancel, exitReason)
	log.Printf("[runAll] Web UI: http://localhost%s", uiPort)
	return srv
}
