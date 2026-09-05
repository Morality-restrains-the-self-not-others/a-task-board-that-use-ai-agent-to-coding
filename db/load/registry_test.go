package dbload_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dbload"
)

func TestResolveDatabasePathFromRegistry(t *testing.T) {
	root := t.TempDir()
	registryDir := filepath.Join(root, "db")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := `version: "1"
databases:
  task-auth:
    path: db/task-auth/auth.sqlite3
`
	if err := os.WriteFile(filepath.Join(registryDir, "registry.yaml"), []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}
	authDir := filepath.Join(root, "db", "task-auth")
	if err := os.MkdirAll(authDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := dbload.ResolveDatabasePath("task-auth", root, false)
	if err != nil {
		t.Fatalf("ResolveDatabasePath: %v", err)
	}
	want := filepath.Join(root, "db", "task-auth", "auth.sqlite3")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveMySQLDSNUsesConfLocalPasswordOverlay(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "db"), 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := `version: "2"
mysql:
  host: 127.0.0.1
  port: 3306
  user: taskapp
  password: ""
databases:
  task-auth:
    driver: mysql
    database: task_auth
`
	if err := os.WriteFile(filepath.Join(root, "db", "registry.yaml"), []byte(tracked), 0o644); err != nil {
		t.Fatal(err)
	}
	overlayDir := filepath.Join(root, "conf-local", "db")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlayDir, "registry.yaml"), []byte("mysql:\n  password: overlay-secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := dbload.ResolveMySQLDSN("task-auth", root)
	if err != nil {
		t.Fatalf("ResolveMySQLDSN: %v", err)
	}
	if !strings.Contains(got, "overlay-secret") {
		t.Fatalf("dsn=%q want overlay password", got)
	}
	if strings.Contains(got, "taskapp123") {
		t.Fatalf("dsn must not keep a tracked sample password: %q", got)
	}
}

func TestResolveMySQLDSNUsesMYSQLPasswordEnvWhenOverlayEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "db"), 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := `version: "2"
mysql:
  host: 10.1.2.3
  port: 3306
  user: taskapp
  password: ""
databases:
  task-bill:
    driver: mysql
    database: task_bill
`
	if err := os.WriteFile(filepath.Join(root, "db", "registry.yaml"), []byte(tracked), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYSQL_PASSWORD", "env-secret")
	got, err := dbload.ResolveMySQLDSN("task-bill", root)
	if err != nil {
		t.Fatalf("ResolveMySQLDSN: %v", err)
	}
	if !strings.Contains(got, "env-secret") {
		t.Fatalf("dsn=%q want MYSQL_PASSWORD", got)
	}
}

func TestResolveDatabasePathEnvOverride(t *testing.T) {
	root := t.TempDir()
	custom := filepath.Join(root, "custom.sqlite3")
	t.Setenv("TASKAUTH_DATABASE_PATH", custom)

	got, err := dbload.ResolveDatabasePath("task-auth", root, false)
	if err != nil {
		t.Fatalf("ResolveDatabasePath: %v", err)
	}
	if got != custom {
		t.Fatalf("got %q want %q", got, custom)
	}
}
