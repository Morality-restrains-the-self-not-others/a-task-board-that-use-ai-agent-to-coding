package go_relaytotrae_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "src", "main.go")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go_relayToTrae module root not found")
		}
		dir = parent
	}
}

func TestBuildScriptProducesBinary(t *testing.T) {
	root := moduleRoot(t)
	bin := filepath.Join(root, "bin", "go_relayToTrae")
	_ = os.Remove(bin)

	cmd := exec.Command("bash", "build.sh")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build.sh failed: %v\n%s", err, out)
	}

	info, err := os.Stat(bin)
	if err != nil {
		t.Fatalf("binary missing: %v\nbuild output:\n%s", err, out)
	}
	if info.IsDir() || info.Size() == 0 {
		t.Fatal("bin/go_relayToTrae missing or empty")
	}
}
