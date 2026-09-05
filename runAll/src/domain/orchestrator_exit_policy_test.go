package domain

import "testing"

func TestKeepManagedServicesOnOrchestratorExit_DefaultSignalKeeps(t *testing.T) {
	if !KeepManagedServicesOnOrchestratorExit(ExitTriggerSignal, "") {
		t.Fatal("default signal exit must keep managed services")
	}
	if !KeepManagedServicesOnOrchestratorExit(ExitTriggerSignal, "0") {
		t.Fatal("non-1 env must keep managed services")
	}
}

func TestKeepManagedServicesOnOrchestratorExit_DebugEnvStopsOnSignal(t *testing.T) {
	if KeepManagedServicesOnOrchestratorExit(ExitTriggerSignal, "1") {
		t.Fatal("RUNALL_SHUTDOWN_SERVICES=1 + signal must stop the stack")
	}
}

func TestKeepManagedServicesOnOrchestratorExit_ShutdownSelfAlwaysKeeps(t *testing.T) {
	if !KeepManagedServicesOnOrchestratorExit(ExitTriggerShutdownSelf, "1") {
		t.Fatal("shutdown-self must keep services even when debug env is set")
	}
	if !KeepManagedServicesOnOrchestratorExit(ExitTriggerListenerLost, "1") {
		t.Fatal("listener-lost must keep services even when debug env is set")
	}
}

func TestKeepManagedServicesOnOrchestratorExit_UnknownTriggerKeeps(t *testing.T) {
	if !KeepManagedServicesOnOrchestratorExit(OrchestratorExitTrigger("panic"), "") {
		t.Fatal("unknown trigger must keep services")
	}
}
