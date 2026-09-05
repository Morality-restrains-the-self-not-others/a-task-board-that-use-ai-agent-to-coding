package tracelog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
)

func TestFatal_WritesJSONLevelError(t *testing.T) {
	if os.Getenv("TRACELOG_FATAL_HELPER") == "1" {
		Fatal("probe-service", "Config error", errors.New("boom"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_WritesJSONLevelError")
	cmd.Env = append(os.Environ(), "TRACELOG_FATAL_HELPER=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("exit=%v stderr=%q", err, stderr.String())
	}
	line := bytes.TrimSpace(stdout.Bytes())
	if len(line) == 0 {
		t.Fatalf("empty stdout; stderr=%q", stderr.String())
	}
	var payload map[string]string
	if err := json.Unmarshal(line, &payload); err != nil {
		t.Fatalf("json: %v line=%q", err, line)
	}
	if payload["level"] != "error" {
		t.Fatalf("level=%q", payload["level"])
	}
	if payload["service"] != "probe-service" {
		t.Fatalf("service=%q", payload["service"])
	}
	if payload["msg"] == "" || !bytes.Contains([]byte(payload["msg"]), []byte("boom")) {
		t.Fatalf("msg=%q", payload["msg"])
	}
	if _, ok := payload["trace_id"]; !ok {
		t.Fatalf("trace_id required: %+v", payload)
	}
	_ = io.Discard
}
