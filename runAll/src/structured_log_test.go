package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestEmitStructuredFatal_WritesJSONLevelError(t *testing.T) {
	if os.Getenv("RUNALL_FATAL_HELPER") == "1" {
		emitStructuredFatal("Config error", errors.New("boom"))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestEmitStructuredFatal_WritesJSONLevelError")
	cmd.Env = append(os.Environ(), "RUNALL_FATAL_HELPER=1")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1")
	}
	var payload map[string]string
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &payload); err != nil {
		t.Fatalf("json=%q: %v", stdout.String(), err)
	}
	if payload["level"] != "error" || payload["service"] != "runall" {
		t.Fatalf("payload=%+v", payload)
	}
}
