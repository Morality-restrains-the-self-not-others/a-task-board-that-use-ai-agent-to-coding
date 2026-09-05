package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	dbload "dbload"

	"runAll/src/domain"
)

// RegistrySQLiteRemover deletes sqlite files via db/load.
type RegistrySQLiteRemover struct {
	MonorepoRoot string
}

func (r *RegistrySQLiteRemover) RemoveAll(ctx context.Context, entries []domain.RegisteredDatabase) ([]string, error) {
	_ = ctx
	root := r.MonorepoRoot
	if root == "" {
		var err error
		root, err = dbload.FindMonorepoRoot("")
		if err != nil {
			return nil, err
		}
	}
	dbEntries := make([]dbload.DatabaseEntry, 0, len(entries))
	for _, e := range entries {
		dbEntries = append(dbEntries, dbload.DatabaseEntry{Key: e.Key})
	}
	return dbload.RemoveSQLiteFiles(root, dbEntries)
}

// BashScriptRunner executes bash scripts from monorepo root.
type BashScriptRunner struct {
	MonorepoRoot string
	Runner       func(ctx context.Context, dir, script string) error
}

func (r *BashScriptRunner) RunScript(ctx context.Context, scriptPath string) error {
	root := r.MonorepoRoot
	if root == "" {
		var err error
		root, err = dbload.FindMonorepoRoot("")
		if err != nil {
			return err
		}
	}
	abs := scriptPath
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, scriptPath)
	}
	abs = filepath.Clean(abs)
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("script not found: %s: %w", abs, err)
	}
	runner := r.Runner
	if runner == nil {
		runner = runBashScript
	}
	return runner(ctx, root, abs)
}

func runBashScript(ctx context.Context, workDir, script string) error {
	bashPath := "/usr/bin/bash"
	for _, p := range []string{"/bin/bash", "/usr/bin/bash"} {
		if _, err := os.Stat(p); err == nil {
			bashPath = p
			break
		}
	}
	if path, err := exec.LookPath("bash"); err == nil {
		if _, err := os.Stat(path); err == nil {
			bashPath = path
		}
	}
	cmd := exec.CommandContext(ctx, bashPath, script)
	cmd.Dir = workDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s: %s", filepath.Base(script), msg)
	}
	return nil
}

func LoadRegisteredDatabases(monorepoRoot string) ([]domain.RegisteredDatabase, error) {
	entries, err := dbload.LoadDatabaseEntries(monorepoRoot)
	if err != nil {
		return nil, err
	}
	out := make([]domain.RegisteredDatabase, 0, len(entries))
	for _, e := range entries {
		out = append(out, domain.RegisteredDatabase{
			Key:           e.Key,
			Driver:        e.Driver,
			Database:      e.Database,
			Path:          e.Path,
			Owner:         e.Owner,
			Order:         e.Order,
			MigrateScript: e.MigrateScript,
			InitScript:    e.InitScript,
		})
	}
	return out, nil
}

func DefaultOwnerToRunAllService() map[string]string {
	return map[string]string{
		"saas-backend": "saas-backend",
		"task-auth":    "task-auth",
		"task-bill":    "task-bill",
		"git-oauth":    "git-oauth",
		"ai-provider":  "ai-provider",
	}
}

func ResolveMonorepoRoot(configPath string) (string, error) {
	if env := strings.TrimSpace(os.Getenv("MONOREPO_ROOT")); env != "" {
		return filepath.Clean(env), nil
	}
	start := ""
	if configPath != "" {
		start = filepath.Dir(configPath)
	}
	return dbload.FindMonorepoRoot(start)
}

func ResolveDevDatabaseScripts(monorepoRoot string) (redisScript, kafkaScript, mysqlScript string, err error) {
	if monorepoRoot == "" {
		monorepoRoot, err = dbload.FindMonorepoRoot("")
		if err != nil {
			return "", "", "", err
		}
	}
	redisScript = filepath.Join(monorepoRoot, "db", "_infra", "redis-flush.sh")
	kafkaScript = filepath.Join(monorepoRoot, "db", "_infra", "kafka-recreate.sh")
	mysqlScript = filepath.Join(monorepoRoot, "db", "_infra", "mysql-reset.sh")
	return redisScript, kafkaScript, mysqlScript, nil
}
