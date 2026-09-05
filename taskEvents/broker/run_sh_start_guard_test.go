package broker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunShStartIntentDoesNotCompile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "run.sh"))
	if err != nil {
		t.Fatalf("read run.sh: %v", err)
	}
	src := string(data)
	start := extractBashFunction(src, "start_intent")
	if start == "" {
		t.Fatal("start_intent not found in run.sh")
	}
	if strings.Contains(start, "build_intent") {
		t.Fatal("start_intent must not call build_intent (ADR-0027)")
	}
	if strings.Contains(start, "go build") {
		t.Fatal("start_intent must not invoke go build (ADR-0027)")
	}
	if !strings.Contains(start, "missing binary") {
		t.Fatal("start_intent must fail clearly when the last-good binary is missing")
	}
	if !strings.Contains(start, "RUNALL_CANARY_OVERLAP") {
		t.Fatal("start_intent must honor RUNALL_CANARY_OVERLAP so canary peers are not blocked by pidfile/exclusive preflight")
	}

	build := extractBashFunction(src, "build_intent")
	if build == "" {
		t.Fatal("build_intent not found in run.sh")
	}
	if !strings.Contains(build, ".new.$$") {
		t.Fatal("build_intent must compile to a temp file then mv (atomic last-good)")
	}
	if !strings.Contains(build, "go build") {
		t.Fatal("build_intent must still invoke go build")
	}
}

func extractBashFunction(src, name string) string {
	needle := name + "() {"
	idx := strings.Index(src, needle)
	if idx < 0 {
		return ""
	}
	rest := src[idx:]
	depth := 0
	for i, c := range rest {
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return rest[:i+1]
			}
		}
	}
	return rest
}
