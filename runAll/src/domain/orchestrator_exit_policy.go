package domain

import "strings"

// OrchestratorExitTrigger is why the runAll process is leaving the run loop.
type OrchestratorExitTrigger string

const (
	ExitTriggerSignal       OrchestratorExitTrigger = "signal"
	ExitTriggerShutdownSelf OrchestratorExitTrigger = "shutdown-self"
	ExitTriggerListenerLost OrchestratorExitTrigger = "ui-listener-lost"
)

const shutdownServicesEnvEnabled = "1"

// KeepManagedServicesOnOrchestratorExit reports whether managed processes must
// survive the orchestrator process exiting.
//
// Default is keep. Only an explicit RUNALL_SHUTDOWN_SERVICES=1 combined with a
// termination signal restores the legacy "Ctrl-C tears down the stack" behavior.
// shutdown-self and UI listener loss always keep services.
func KeepManagedServicesOnOrchestratorExit(trigger OrchestratorExitTrigger, shutdownServicesEnv string) bool {
	switch trigger {
	case ExitTriggerShutdownSelf, ExitTriggerListenerLost:
		return true
	case ExitTriggerSignal:
		return strings.TrimSpace(shutdownServicesEnv) != shutdownServicesEnvEnabled
	default:
		return true
	}
}
