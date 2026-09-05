package infrastructure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestScriptObservabilityStorageResetter_Success(t *testing.T) {
	script := filepath.Join(t.TempDir(), "reset.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	resetter := &ScriptObservabilityStorageResetter{
		ScriptPath: script,
		Runner: func(context.Context, string, string) (string, error) {
			return "", nil
		},
	}
	outcome, err := resetter.Reset(context.Background())
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if !outcome.AllOK() {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestScriptObservabilityStorageResetter_Failure(t *testing.T) {
	script := filepath.Join(t.TempDir(), "reset.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	resetter := &ScriptObservabilityStorageResetter{
		ScriptPath: script,
		Runner: func(context.Context, string, string) (string, error) {
			return "docker missing", errors.New("exit 1")
		},
	}
	outcome, err := resetter.Reset(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if outcome.LokiReset != "error: docker missing" {
		t.Fatalf("loki_reset = %q", outcome.LokiReset)
	}
}
