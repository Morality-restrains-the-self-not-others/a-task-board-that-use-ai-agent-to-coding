//go:build linux

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// waitTerminationSignal blocks until SIGINT or SIGTERM and returns it.
//
// This is signal.Notify-based, not rt_sigtimedwait. The original design used
// rt_sigtimedwait (Sigwaitinfo) to capture the sender si_pid (OPT-20260828-004),
// but that is fundamentally racy on Linux: the Go runtime deliberately keeps
// these _SigKill signals unblocked on every thread it creates
// (runtime.blockableSig refuses to block them), so a process-directed kill(2)
// is delivered to one of those threads. Its runtime handler then races the
// sigwait threads — and when it wins, dieFromSignal kills the whole orchestrator
// before it can run applyOrchestratorExitKeepPolicy. That was the nightly flake
// 2026-09-01 (helper subprocess died "signal: terminated").
//
// signal.Notify consumes the signal in the runtime handler and forwards it here,
// so the process can never die from the default _SigKill action. The cost is
// that si_pid is not available; callers log the signal name plus process
// coordinates instead (formatTerminationSignalLog omits the sender fields when
// the origin is unknown).
//
// stop closes once the caller is shutting down for another reason; the losing
// waiter must exit instead of blocking forever. nil means wait forever.
func waitTerminationSignal(stop <-chan struct{}) (os.Signal, signalOrigin, error) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)
	select {
	case s := <-ch:
		return s, signalOrigin{}, nil
	case <-stop:
		return nil, signalOrigin{}, errTerminationWaitStopped
	}
}
