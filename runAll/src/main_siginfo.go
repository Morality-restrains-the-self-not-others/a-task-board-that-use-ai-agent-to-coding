package main

import (
	"fmt"
	"os"
)

// signalOrigin is the kill(2) sender identity. It was captured via
// rt_sigtimedwait (Sigwaitinfo) for OPT-20260828-004, but that mechanism races
// the Go runtime's unblocked-thread delivery and could kill the orchestrator
// outright, so termination waits now use signal.Notify and the origin is left
// empty (si_pid unavailable on that path). The type and formatters are kept so
// a future sender-identity mechanism can reuse the same log format.
type signalOrigin struct {
	Pid  int
	Uid  uint32
	Code int32
}

func formatSignalOriginFields(origin signalOrigin) string {
	return fmt.Sprintf("si_pid=%d si_uid=%d si_code=%d", origin.Pid, origin.Uid, origin.Code)
}

// formatTerminationSignalLog renders the termination log payload. When the
// sender identity is unknown (origin.Pid == 0, the signal.Notify path), only the
// signal name is printed — "si_pid=0" would falsely imply a sender was captured.
func formatTerminationSignalLog(sig os.Signal, origin signalOrigin) string {
	if origin.Pid == 0 {
		return sig.String()
	}
	return fmt.Sprintf("%s %s", sig, formatSignalOriginFields(origin))
}
