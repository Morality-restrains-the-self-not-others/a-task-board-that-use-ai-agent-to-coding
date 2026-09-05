package dbload

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestOpenTestMySQLCleanupPanicsOnDropFailure(t *testing.T) {
	origFn := runMySQLExecFn
	defer func() { runMySQLExecFn = origFn }()
	runMySQLExecFn = func(dsn, sql string) error {
		if strings.Contains(sql, "DROP DATABASE") {
			return errors.New("simulated drop failure")
		}
		return nil
	}

	cleanup := makeTestDBCleanup("user:pass@tcp(127.0.0.1:3306)/base", "task_auth_test_abcdef12")
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("DROP failure must panic so leftover *_test_* databases fail the test")
		}
		msg := fmt.Sprint(rec)
		if !strings.Contains(msg, "task_auth_test_abcdef12") {
			t.Fatalf("panic must name the database, got %q", msg)
		}
		if !strings.Contains(msg, "simulated drop failure") {
			t.Fatalf("panic must include drop error, got %q", msg)
		}
	}()
	cleanup()
}

func TestOpenTestMySQLCleanupSilentOnDropSuccess(t *testing.T) {
	origFn := runMySQLExecFn
	defer func() { runMySQLExecFn = origFn }()
	runMySQLExecFn = func(dsn, sql string) error {
		return nil
	}

	var buf bytes.Buffer
	oldOut := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(oldOut)
		log.SetFlags(oldFlags)
	}()

	cleanup := makeTestDBCleanup("user:pass@tcp(127.0.0.1:3306)/base", "task_auth_test_abcdef12")
	cleanup()

	if buf.Len() != 0 {
		t.Fatalf("expected no log on success, got %q", buf.String())
	}
}
