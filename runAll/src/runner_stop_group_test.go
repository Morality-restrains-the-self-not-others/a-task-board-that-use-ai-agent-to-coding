package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunner_StopGroup_ContinuesAfterFailure(t *testing.T) {
	dir := t.TempDir()
	failDir := filepath.Join(dir, "fail")
	okDir := filepath.Join(dir, "ok")
	for _, d := range []struct {
		path, script string
	}{
		{failDir, "#!/usr/bin/env bash\nexit 1\n"},
		{okDir, "#!/usr/bin/env bash\ntouch stopped.marker\n"},
	} {
		if err := os.MkdirAll(d.path, 0o755); err != nil {
			t.Fatal(err)
		}
		runSh := filepath.Join(d.path, "run.sh")
		if err := os.WriteFile(runSh, []byte(d.script), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	store := NewStatusStore()
	runner, err := NewRunner(&Config{
		Version: "1",
		Groups: []Group{{
			Name: "g",
			Services: []Service{
				{
					Name:         "svc-fail",
					StartCommand: "bash run.sh",
					StopCommand:  "bash run.sh",
					WorkingDir:   failDir,
					HealthCheck:  HealthCheck{URL: "http://127.0.0.1:1/", Timeout: 1, Retries: 1},
				},
				{
					Name:         "svc-ok",
					StartCommand: "bash run.sh",
					StopCommand:  "bash run.sh",
					WorkingDir:   okDir,
					HealthCheck:  HealthCheck{URL: "http://127.0.0.1:1/", Timeout: 1, Retries: 1},
				},
			},
		}},
	}, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	store.Update("svc-fail", StatusHealthy, "")
	store.Update("svc-ok", StatusHealthy, "")

	err = runner.StopGroup(context.Background(), "g")
	if err == nil {
		t.Fatal("expected aggregated stop group error")
	}
	if _, statErr := os.Stat(filepath.Join(okDir, "stopped.marker")); statErr != nil {
		t.Fatalf("svc-ok stop_command should have run: %v", statErr)
	}
	status := store.Get("svc-ok")
	if status == nil || status.Status != StatusStopped {
		t.Fatalf("svc-ok status = %#v, want stopped", status)
	}
}
