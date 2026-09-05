package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunShStartMissingPortFailsLoudly verifies that when an intent's
// conf/events/domain-events/<event>/config.yaml lacks intents.<intent>.port,
// `run.sh start` fails with a non-zero exit and a clear stderr message —
// instead of silently not listening, which only surfaces as a runAll
// READINESS_TIMEOUT after a long wait.
func TestRunShStartMissingPortFailsLoudly(t *testing.T) {
	root := findRepoRoot(t)
	runSh := filepath.Join(root, "run.sh")

	// Fake monorepo root containing ONLY the broken config, so the real
	// conf/events/... is never consulted.
	mono := t.TempDir()
	confDir := filepath.Join(mono, "conf", "events", "domain-events", "member_joined")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Intent block exists but intentionally has NO port key.
	broken := "intents:\n  1_create_default_git_identity:\n    name: create-default-git-identity\n"
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", runSh, "start", "member_joined/1_create_default_git_identity")
	cmd.Env = append(os.Environ(), "MONOREPO_ROOT="+mono)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected run.sh start to fail for missing port; got success\noutput:\n%s", out)
	}
	msg := string(out)
	if !strings.Contains(msg, "has no port") {
		t.Fatalf("stderr should name the missing port clearly; got:\n%s", msg)
	}
	if !strings.Contains(msg, "1_create_default_git_identity") {
		t.Fatalf("stderr should identify the failing intent; got:\n%s", msg)
	}
}
