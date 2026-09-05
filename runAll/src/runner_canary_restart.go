package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"syscall"
	"time"

	"runAll/src/domain"
)

// runallCanaryOverlapEnv tells start recipes (taskEvents run.sh) to start a
// SO_REUSEPORT peer instead of exclusive-bind / pidfile short-circuit.
const runallCanaryOverlapEnv = "RUNALL_CANARY_OVERLAP"

// canaryPeerReadyTimeout waits for the new process group to appear on the
// listen port after overlap start. Override in tests.
var canaryPeerReadyTimeout = 15 * time.Second

// canaryDrainTimeout is SIGTERM wait for the old process group (ADR-0058).
var canaryDrainTimeout = 25 * time.Second

func (r *Runner) trackedCmd(name string) *exec.Cmd {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.processes[name]
}

func pidInProcessGroup(pid, pgid int) bool {
	if pid <= 0 || pgid <= 0 {
		return false
	}
	if pid == pgid {
		return true
	}
	g, err := syscall.Getpgid(pid)
	return err == nil && g == pgid
}

func (r *Runner) waitCanaryPeerReady(ctx context.Context, svc *Service, newPgid int) error {
	ports := resolveServicePorts(svc)
	if len(ports) == 0 || newPgid <= 0 {
		return nil
	}
	deadline := time.Now().Add(canaryPeerReadyTimeout)
	var last []int
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		for _, port := range ports {
			pids, err := r.listenerPIDsForPort(port)
			if err != nil {
				return err
			}
			last = pids
			for _, pid := range pids {
				if pidInProcessGroup(pid, newPgid) {
					log.Printf("[%s] canary overlap: peer listening pid=%d pgid=%d port=%s", svc.Name, pid, newPgid, port)
					return nil
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for process group %d on ports %v (listeners=%v)", newPgid, ports, last)
}

func (r *Runner) drainOldProcess(name string, oldCmd *exec.Cmd) {
	if oldCmd == nil || oldCmd.Process == nil {
		return
	}
	pgid := oldCmd.Process.Pid
	log.Printf("[%s] canary drain: SIGTERM old pgid=%d", name, pgid)
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	if waitProcessGroupExit(pgid, canaryDrainTimeout) {
		_ = oldCmd.Wait()
		reapZombies()
		log.Printf("[%s] canary drain: old process group exited", name)
		return
	}
	log.Printf("[%s] canary drain: SIGKILL old pgid=%d", name, pgid)
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	waitProcessGroupExit(pgid, 5*time.Second)
	_ = oldCmd.Wait()
	reapZombies()
}

func (r *Runner) killTrackedCmd(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid := cmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	waitProcessGroupExit(pgid, 2*time.Second)
	_ = cmd.Wait()
	reapZombies()
}

func (r *Runner) restoreTrackedCmd(name string, oldCmd *exec.Cmd, failedNew *exec.Cmd) {
	r.mu.Lock()
	r.processes[name] = oldCmd
	r.mu.Unlock()
	r.killTrackedCmd(failedNew)
	if oldCmd != nil && oldCmd.Process != nil {
		r.store.SetPID(name, oldCmd.Process.Pid)
	}
}

func (r *Runner) recordCanaryPeerFailure(name string, peerErr error) {
	if r == nil || r.store == nil || peerErr == nil {
		return
	}
	msg := peerErr.Error()
	phase := domain.ServiceLifecyclePhaseReadiness
	code := domain.ServiceFailureCodeReadinessTimeout
	for _, from := range []Status{StatusRetrying, StatusStarting, StatusRestarting, StatusBuilding} {
		if r.store.RecordFailureIfStatus(name, from, phase, code, msg) {
			return
		}
	}
}
