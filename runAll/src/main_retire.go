package main

import (
	"log"
	"syscall"
	"time"
)

// previousRunAllHooks abstracts the pieces retirePreviousRunAll needs so the
// graceful-shutdown ordering (OPT-20260820-006) can be pinned by unit tests
// without touching real processes.
type previousRunAllHooks struct {
	// postShutdownSelf issues the /api/shutdown-self POST and returns the HTTP
	// status, or an error when the previous instance cannot be reached.
	postShutdownSelf func() (int, error)
	// alive reports whether pid still exists as a process.
	alive func(pid int) bool
	// kill sends a signal to pid.
	kill func(pid int, sig syscall.Signal) error
	// gracefulWait is how long to wait for the old instance to exit on its own
	// after a successful shutdown-self before falling back to a signal.
	gracefulWait time.Duration
}

// retirePreviousRunAll retires one previous runAll instance. Order matters
// (OPT-20260820-006): when /api/shutdown-self acknowledges with 2xx the old
// instance is already exiting with managed services kept alive. A SIGTERM on
// a *legacy* binary (pre-ADR-0035) would still Shutdown() the stack, so we
// wait for a voluntary exit and only fall back to a signal when the POST
// failed/non-2xx or the exit never happens.
func retirePreviousRunAll(pid int, h *previousRunAllHooks) previousRunAllShutdownResult {
	result := previousRunAllShutdownResult{HadPrevious: true}
	if h == nil || h.postShutdownSelf == nil {
		return result
	}

	status, err := h.postShutdownSelf()
	graceful := false
	if err != nil {
		log.Printf("[runAll] previous instance not responding to shutdown-self: %v", err)
	} else {
		log.Printf("[runAll] previous instance shutdown-self returned %d", status)
		if status >= 200 && status < 300 {
			result.GracefulShutdownSelf = true
			graceful = true
		}
	}

	if graceful {
		if pollProcessExit(pid, h.alive, h.gracefulWait) {
			log.Printf("[runAll] previous instance (PID=%d) exited via shutdown-self (managed services kept)", pid)
			return result
		}
		log.Printf("[runAll] previous instance (PID=%d) did not exit within %s of shutdown-self, sending SIGTERM", pid, h.gracefulWait)
	}

	if err := h.kill(pid, syscall.SIGTERM); err != nil {
		log.Printf("[runAll] SIGTERM to PID %d failed: %v", pid, err)
		return result
	}
	if pollProcessExit(pid, h.alive, h.gracefulWait) {
		log.Printf("[runAll] previous instance (PID=%d) stopped gracefully", pid)
		return result
	}
	log.Printf("[runAll] previous instance (PID=%d) did not stop, sending SIGKILL", pid)
	_ = h.kill(pid, syscall.SIGKILL)
	return result
}

// getSessionID returns the session ID (sid) of pid. syscall has no Getsid on
// Linux, so it goes through the raw SYS_GETSID call (OPT-20260820-006).
func getSessionID(pid int) (int, error) {
	sid, _, errno := syscall.Syscall(syscall.SYS_GETSID, uintptr(pid), 0, 0)
	if errno != 0 {
		return 0, errno
	}
	return int(sid), nil
}

// pollProcessExit polls alive until the process disappears or timeout elapses.
func pollProcessExit(pid int, alive func(int) bool, timeout time.Duration) bool {
	if alive == nil {
		return false
	}
	deadline := time.After(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if !alive(pid) {
			return true
		}
		select {
		case <-deadline:
			return false
		case <-ticker.C:
		}
	}
}
