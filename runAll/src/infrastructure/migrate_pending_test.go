package infrastructure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemDataMigrateDirResolver_parsesQuotedRoot(t *testing.T) {
	root := t.TempDir()
	migDir := filepath.Join(root, "dataMigrate", "taskBill")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
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
	dir, ok := FilesystemDataMigrateDirResolver{}.Resolve(root, "db/task-bill/migrate.sh")
	if !ok || dir != migDir {
		t.Fatalf("dir=%q ok=%v want %q", dir, ok, migDir)
	}
}

func TestFilesystemSQLLister_listsSQLOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001_a.sql"), []byte("--\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	names, err := FilesystemSQLLister{}.ListSQL(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "001_a.sql" {
		t.Fatalf("names=%v", names)
	}
}

func TestMySQLAppliedStepReader_missingTableIsEmpty(t *testing.T) {
	r := &MySQLAppliedStepReader{
		Query: func(context.Context, string, string) ([]string, error) {
			return nil, errors.New("ERROR 1146 (42S02): Table 'task_auth.data_migrate_log' doesn't exist")
		},
	}
	rows, err := r.ListApplied(context.Background(), "task_auth")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows=%v", rows)
	}
}

func TestMySQLAppliedStepReader_connectionErrorSurfaces(t *testing.T) {
	r := &MySQLAppliedStepReader{
		Query: func(context.Context, string, string) ([]string, error) {
			return nil, errors.New("Can't connect to MySQL server")
		},
	}
	_, err := r.ListApplied(context.Background(), "task_auth")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIsAbsentMigrateLog(t *testing.T) {
	if !isAbsentMigrateLog(errors.New("Unknown database 'task_bill'")) {
		t.Fatal("unknown database should be absent")
	}
	if isAbsentMigrateLog(errors.New("Can't connect")) {
		t.Fatal("connect error is not absent table")
	}
}
