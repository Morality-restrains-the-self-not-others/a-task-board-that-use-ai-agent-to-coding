package main

import (
	"log"
	"os"

	"runAll/src/domain"
)

func applyOrchestratorExitKeepPolicy(runner *Runner, trigger domain.OrchestratorExitTrigger) {
	if runner == nil {
		return
	}
	if domain.KeepManagedServicesOnOrchestratorExit(trigger, os.Getenv("RUNALL_SHUTDOWN_SERVICES")) {
		runner.SetSkipShutdownServices(true)
		log.Printf("[runAll] orchestrator_exit_keeps_services trigger=%s", trigger)
	}
}
