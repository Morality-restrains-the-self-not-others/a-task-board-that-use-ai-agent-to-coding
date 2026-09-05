package dbload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var envByKey = map[string]string{
	"task-auth":       "TASKAUTH_DATABASE_PATH",
	"task-bill":       "TASKBILL_DATABASE_PATH",
	"task-budget":     "TASK_BUDGET_DATABASE_PATH",
	"git-oauth":       "GITOAUTH_DATABASE_PATH",
	"email":           "EMAIL_DATABASE_PATH",
	"ai-provider":     "AI_PROVIDER_DATABASE_PATH",
	"task-project":    "TASK_PROJECT_DATABASE_PATH",
	"task-task":       "TASK_TASK_DATABASE_PATH",
	"task-cloud":      "TASK_CLOUD_DATABASE_PATH",
	"task-ai-comment": "TASK_AI_COMMENT_DATABASE_PATH",
	"task-tenant":     "TASK_TENANT_DATABASE_PATH",
	"task-referral":   "TASK_REFERRAL_DATABASE_PATH",
	"container":       "CONTAINER_DATABASE_PATH",
}

// MySQLConfig holds the shared MySQL connection parameters.
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// RegistryDatabase holds per-database configuration from registry.yaml.
type RegistryDatabase struct {
	Driver   string `yaml:"driver"`
	Database string `yaml:"database"`
	Path     string `yaml:"path"` // legacy SQLite path (relative to monorepo root)
	Owner    string `yaml:"owner"`
	Order    int    `yaml:"order"`
}

type registryFile struct {
	Version   string                      `yaml:"version"`
	MySQL     *MySQLConfig                `yaml:"mysql"`
	Databases map[string]RegistryDatabase `yaml:"databases"`
}

// key → env override mapping for MySQL DSN
var mysqlDSNEnvByKey = map[string]string{
	"task-auth":       "TASKAUTH_MYSQL_DSN",
	"task-bill":       "TASKBILL_MYSQL_DSN",
	"task-budget":     "TASK_BUDGET_MYSQL_DSN",
	"git-oauth":       "GITOAUTH_MYSQL_DSN",
	"ai-provider":     "AI_PROVIDER_MYSQL_DSN",
	"task-project":    "TASK_PROJECT_MYSQL_DSN",
	"task-task":       "TASK_TASK_MYSQL_DSN",
	"task-cloud":      "TASK_CLOUD_MYSQL_DSN",
	"task-ai-comment": "TASK_AI_COMMENT_MYSQL_DSN",
	"container":       "CONTAINER_MYSQL_DSN",
	"task-tenant":     "TASK_TENANT_MYSQL_DSN",
	"task-referral":   "TASK_REFERRAL_MYSQL_DSN",
}

func FindMonorepoRoot(start string) (string, error) {
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "db", "registry.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("db/registry.yaml not found from %s", start)
}

func loadRegistry(monorepoRoot string) (*registryFile, error) {
	path := filepath.Join(monorepoRoot, "db", "registry.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc registryFile
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if err := applyMySQLPasswordOverlay(monorepoRoot, &doc.MySQL); err != nil {
		return nil, err
	}
	return &doc, nil
}

func applyMySQLPasswordOverlay(monorepoRoot string, mysql **MySQLConfig) error {
	if mysql == nil {
		return fmt.Errorf("mysql config pointer required")
	}
	overlayPath := filepath.Join(monorepoRoot, "conf-local", "db", "registry.yaml")
	raw, err := os.ReadFile(overlayPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		var extra struct {
			MySQL *MySQLConfig `yaml:"mysql"`
		}
		if err := yaml.Unmarshal(raw, &extra); err != nil {
			return fmt.Errorf("conf-local/db/registry.yaml: %w", err)
		}
		if extra.MySQL != nil && strings.TrimSpace(extra.MySQL.Password) != "" {
			if *mysql == nil {
				*mysql = extra.MySQL
			} else {
				(*mysql).Password = extra.MySQL.Password
			}
		}
	}
	if *mysql != nil && strings.TrimSpace((*mysql).Password) == "" {
		if pw := strings.TrimSpace(os.Getenv("MYSQL_PASSWORD")); pw != "" {
			(*mysql).Password = pw
		}
	}
	return nil
}

// ResolveDatabasePath resolves the SQLite file path for a given database key.
// Prefer ResolveMySQLDSN for MySQL-backed services; this function is retained for
// backward compatibility with legacy SQLite file paths.
func ResolveDatabasePath(key, monorepoRoot string, mkdir bool) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("database key required")
	}
	if envName, ok := envByKey[key]; ok {
		if override := strings.TrimSpace(os.Getenv(envName)); override != "" {
			p := override
			if !filepath.IsAbs(p) {
				p = filepath.Join(monorepoRoot, p)
			}
			if mkdir {
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					return "", err
				}
			}
			return filepath.Clean(p), nil
		}
	}
	if monorepoRoot == "" {
		var err error
		monorepoRoot, err = FindMonorepoRoot("")
		if err != nil {
			return "", err
		}
	}
	doc, err := loadRegistry(monorepoRoot)
	if err != nil {
		return "", err
	}
	entry, ok := doc.Databases[key]
	if !ok {
		return "", fmt.Errorf("registry.yaml missing database key: %s", key)
	}
	// If the entry has a legacy path field, use it
	if entry.Path != "" {
		abs := filepath.Join(monorepoRoot, entry.Path)
		if mkdir {
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return "", err
			}
		}
		return filepath.Clean(abs), nil
	}
	return "", fmt.Errorf("registry.yaml database key %s has no path (use MySQL DSN instead)", key)
}

// ResolveMySQLDSN returns a MySQL DSN string for the given database key.
// Resolution order:
//  1. Per-key env override (e.g. TASKAUTH_MYSQL_DSN)
//  2. Global MYSQL_DSN env override (shared DSN template; {database} replaced)
//  3. registry.yaml mysql block + per-database entry
func ResolveMySQLDSN(key, monorepoRoot string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("database key required")
	}

	// 1. Per-key env override
	if envName, ok := mysqlDSNEnvByKey[key]; ok {
		if dsn := strings.TrimSpace(os.Getenv(envName)); dsn != "" {
			return dsn, nil
		}
	}

	// 2. Global MYSQL_DSN template
	if globalDSN := strings.TrimSpace(os.Getenv("MYSQL_DSN")); globalDSN != "" {
		return globalDSN, nil
	}

	// 3. registry.yaml
	if monorepoRoot == "" {
		var err error
		monorepoRoot, err = FindMonorepoRoot("")
		if err != nil {
			return "", err
		}
	}
	doc, err := loadRegistry(monorepoRoot)
	if err != nil {
		return "", err
	}
	entry, ok := doc.Databases[key]
	if !ok {
		return "", fmt.Errorf("registry.yaml missing database key: %s", key)
	}

	mysql := doc.MySQL
	if mysql == nil {
		return "", fmt.Errorf("registry.yaml has no mysql block; cannot resolve MySQL DSN for %s", key)
	}

	dbName := entry.Database
	if dbName == "" {
		dbName = key
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true",
		mysql.User, mysql.Password, mysql.Host, mysql.Port, dbName)
	return dsn, nil
}

// ResolveMySQLConfig returns the shared MySQL connection config from registry.yaml.
func ResolveMySQLConfig(monorepoRoot string) (*MySQLConfig, error) {
	if monorepoRoot == "" {
		var err error
		monorepoRoot, err = FindMonorepoRoot("")
		if err != nil {
			return nil, err
		}
	}
	doc, err := loadRegistry(monorepoRoot)
	if err != nil {
		return nil, err
	}
	if doc.MySQL == nil {
		return nil, fmt.Errorf("registry.yaml has no mysql configuration block")
	}
	return doc.MySQL, nil
}

// GetDatabaseName returns the MySQL database name for a given key.
func GetDatabaseName(key, monorepoRoot string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("database key required")
	}
	if monorepoRoot == "" {
		var err error
		monorepoRoot, err = FindMonorepoRoot("")
		if err != nil {
			return "", err
		}
	}
	doc, err := loadRegistry(monorepoRoot)
	if err != nil {
		return "", err
	}
	entry, ok := doc.Databases[key]
	if !ok {
		return "", fmt.Errorf("registry.yaml missing database key: %s", key)
	}
	if entry.Database != "" {
		return entry.Database, nil
	}
	return key, nil
}
