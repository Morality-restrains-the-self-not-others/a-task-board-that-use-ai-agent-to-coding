//go:build !linux

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// waitTerminationSignal blocks until SIGINT or SIGTERM and returns it.
//
// signal.Notify is the reliable mechanism on every platform: the runtime keeps
// _SigKill signals (SIGINT/SIGTERM) unblocked on the threads it creates, so a
// raw sigwait would race the delivery thread's dieFromSignal handler and could
// kill the process. Notify consumes the signal in the runtime handler and
// forwards it here, so callers always reach the graceful keep-policy exit.
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
