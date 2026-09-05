package main

import (
	"testing"

	"runAll/src/domain"
)

func TestApplyOrchestratorExitKeepPolicy_SetsSkip(t *testing.T) {
	runner, err := NewRunner(&Config{Version: "1"}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	t.Setenv("RUNALL_SHUTDOWN_SERVICES", "")
	applyOrchestratorExitKeepPolicy(runner, domain.ExitTriggerSignal)
	if !runner.skipShutdownServices {
		t.Fatal("default signal policy must set skipShutdownServices")
	}
}

func TestApplyOrchestratorExitKeepPolicy_DebugEnvLeavesSkipFalse(t *testing.T) {
	runner, err := NewRunner(&Config{Version: "1"}, NewStatusStore())
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	t.Setenv("RUNALL_SHUTDOWN_SERVICES", "1")
	applyOrchestratorExitKeepPolicy(runner, domain.ExitTriggerSignal)
	if runner.skipShutdownServices {
		t.Fatal("RUNALL_SHUTDOWN_SERVICES=1 must not skip stack shutdown on signal")
	}
}

func TestApplyOrchestratorExitKeepPolicy_NilRunner(t *testing.T) {
	applyOrchestratorExitKeepPolicy(nil, domain.ExitTriggerSignal)
}
