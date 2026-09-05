package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

func writeMiniMigrateRepo(t *testing.T, root string) {
	t.Helper()
	mig := filepath.Join(root, "dataMigrate", "taskBill")
	if err := os.MkdirAll(mig, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mig, "001_schema.sql"), []byte("--\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mig, "002_new.sql"), []byte("--\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "db", "task-bill", "migrate.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "exec bash \"$ROOT/db/scripts/apply_datamigrate.sh\" \"task_bill\" \"$ROOT/dataMigrate/taskBill\"\n"
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := []byte(`version: "2"
mysql:
  host: 127.0.0.1
  port: 3306
  user: taskapp
  password: unused
databases:
  task-bill:
    driver: mysql
    database: task_bill
    owner: task-bill
    order: 1
    migrate_script: db/task-bill/migrate.sh
`)
	if err := os.WriteFile(filepath.Join(root, "db", "registry.yaml"), reg, 0o644); err != nil {
		t.Fatal(err)
	}
}

type staticAppliedReader []string

func (s staticAppliedReader) ListApplied(context.Context, string) ([]string, error) {
	return append([]string(nil), s...), nil
}

func TestMigrateStatusAPI_reportsMissingSQL(t *testing.T) {
	runner, root := newTestRunnerForDevDB(t)
	writeMiniMigrateRepo(t, root)
	runner.migrateAppliedReader = staticAppliedReader{"001_schema.sql"}
	runner.migrateDirResolver = infrastructure.FilesystemDataMigrateDirResolver{}
	runner.migrateSQLLister = infrastructure.FilesystemSQLLister{}

	mux := http.NewServeMux()
	registerUIHandlers(mux, NewStatusStore(), runner, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/dev/migrate-status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var rep domain.MigratePendingReport
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.PendingCount != 1 {
		t.Fatalf("pending=%d body=%s", rep.PendingCount, rec.Body.String())
	}
	if len(rep.Databases) != 1 || rep.Databases[0].Status != domain.MigrateStatusPending {
		t.Fatalf("databases=%+v", rep.Databases)
	}
	if len(rep.Databases[0].Missing) != 1 || rep.Databases[0].Missing[0] != "002_new.sql" {
		t.Fatalf("missing=%v", rep.Databases[0].Missing)
	}
}

func TestMigrateStatusAPI_methodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	registerUIHandlers(mux, NewStatusStore(), nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/dev/migrate-status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestMigrateStatusAPI_nilRunner(t *testing.T) {
	mux := http.NewServeMux()
	registerUIHandlers(mux, NewStatusStore(), nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/dev/migrate-status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		// writeJSONError typically 400
		if rec.Code < 400 {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
	if !strings.Contains(rec.Body.String(), "runner is required") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
