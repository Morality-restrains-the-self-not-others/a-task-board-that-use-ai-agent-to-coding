package main

import (
	"errors"
	"syscall"
)

// isRetryableTerminationWaitError reports whether waitTerminationSignal can
// safely retry. rt_sigtimedwait returns EINTR when a non-waited signal (for
// example SIGCHLD or a Go runtime signal) interrupts the syscall. Treating
// that as a fatal orchestrator exit takes down :9999 and, with stdout pipes,
// used to SIGPIPE the whole managed stack.
func isRetryableTerminationWaitError(err error) bool {
	return errors.Is(err, syscall.EINTR)
}

// errTerminationWaitStopped is returned by waitTerminationSignal when the stop
// channel closes before a signal arrives — i.e. the caller is already shutting
// down for another reason. Callers must not treat it as an orchestrator-fatal
// error.
var errTerminationWaitStopped = errors.New("termination signal wait stopped")
