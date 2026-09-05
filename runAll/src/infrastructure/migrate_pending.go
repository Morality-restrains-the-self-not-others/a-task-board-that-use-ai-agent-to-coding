package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	dbload "dbload"
)

var dataMigrateDirRe = regexp.MustCompile(`["']\$ROOT/dataMigrate/([A-Za-z0-9_]+)["']`)

// FilesystemDataMigrateDirResolver reads migrate.sh and maps to dataMigrate/<dir>.
type FilesystemDataMigrateDirResolver struct{}

func (FilesystemDataMigrateDirResolver) Resolve(monorepoRoot, migrateScript string) (string, bool) {
	if strings.TrimSpace(migrateScript) == "" || strings.TrimSpace(monorepoRoot) == "" {
		return "", false
	}
	scriptPath := migrateScript
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(monorepoRoot, migrateScript)
	}
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", false
	}
	m := dataMigrateDirRe.FindSubmatch(raw)
	if m == nil {
		return "", false
	}
	dir := filepath.Join(monorepoRoot, "dataMigrate", string(m[1]))
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", false
	}
	return dir, true
}

// FilesystemSQLLister lists *.sql basenames in a directory (non-recursive).
type FilesystemSQLLister struct{}

func (FilesystemSQLLister) ListSQL(_ context.Context, absDir string) ([]string, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			names = append(names, name)
		}
	}
	return names, nil
}

// MySQLQueryFunc runs a mysql -sN query against one database and returns stdout lines.
type MySQLQueryFunc func(ctx context.Context, databaseName, sql string) ([]string, error)

// MySQLAppliedStepReader reads data_migrate_log.step_key via mysql CLI / docker exec.
type MySQLAppliedStepReader struct {
	MonorepoRoot string
	Query        MySQLQueryFunc
}

func NewMySQLAppliedStepReader(monorepoRoot string) *MySQLAppliedStepReader {
	r := &MySQLAppliedStepReader{MonorepoRoot: monorepoRoot}
	r.Query = r.defaultQuery
	return r
}

func (r *MySQLAppliedStepReader) ListApplied(ctx context.Context, databaseName string) ([]string, error) {
	if r == nil || r.Query == nil {
		return nil, fmt.Errorf("mysql query not configured")
	}
	rows, err := r.Query(ctx, databaseName, "SELECT step_key FROM data_migrate_log ORDER BY step_key")
	if err != nil {
		if isAbsentMigrateLog(err) {
			return []string{}, nil
		}
		return nil, err
	}
	return rows, nil
}

func isAbsentMigrateLog(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "unknown database") ||
		strings.Contains(msg, "1146") ||
		strings.Contains(msg, "1049")
}

func (r *MySQLAppliedStepReader) defaultQuery(ctx context.Context, databaseName, sql string) ([]string, error) {
	databaseName = strings.TrimSpace(databaseName)
	if databaseName == "" {
		return nil, fmt.Errorf("database name required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cfg, err := dbload.ResolveMySQLConfig(r.MonorepoRoot)
	if err != nil {
		return nil, err
	}
	args, useDocker := mysqlClientArgs(cfg.User, cfg.Password, databaseName, sql)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if useDocker {
			return nil, fmt.Errorf("mysql docker query %s: %s", databaseName, msg)
		}
		return nil, fmt.Errorf("mysql query %s: %s", databaseName, msg)
	}
	return splitNonEmptyLines(stdout.String()), nil
}

func mysqlClientArgs(user, pass, databaseName, sql string) ([]string, bool) {
	container := detectMySQLContainer()
	if container != "" {
		return []string{
			"docker", "exec", "-i", container,
			"mysql", "--default-character-set=utf8mb4",
			"-u", user, "-p" + pass, "-sN", "-e", sql, databaseName,
		}, true
	}
	return []string{
		"mysql", "--default-character-set=utf8mb4",
		"-h", "127.0.0.1", "-P", "3306",
		"-u", user, "-p" + pass, "-sN", "-e", sql, databaseName,
	}, false
}

func detectMySQLContainer() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=docker-mysql", "--format", "{{.ID}}")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		return ""
	}
	if i := strings.IndexByte(id, '\n'); i >= 0 {
		id = id[:i]
	}
	return strings.TrimSpace(id)
}

func splitNonEmptyLines(s string) []string {
	var rows []string
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			rows = append(rows, ln)
		}
	}
	return rows
}
