package domain

import (
	"context"
	"fmt"
	"sort"
)

// DatabaseStepResult is the outcome of one migrate or init script.
type DatabaseStepResult struct {
	Database string `json:"database"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// DatabasePlatformClearResult aggregates a dev-only clear (wipe) operation.
type DatabasePlatformClearResult struct {
	Status               string   `json:"status"`
	ServicesStopped      []string `json:"services_stopped"`
	ServicesStillRunning []string `json:"services_still_running,omitempty"`
	SQLiteRemoved        []string `json:"sqlite_removed"`
	MySQLReset           string   `json:"mysql_reset"`
	RedisReset           string   `json:"redis_reset"`
	KafkaReset           string   `json:"kafka_reset"`
}

// DatabasePlatformInitResult aggregates migrate + init scripts only.
type DatabasePlatformInitResult struct {
	Status          string               `json:"status"`
	BlockedServices []string             `json:"blocked_services,omitempty"`
	Migrations      []DatabaseStepResult `json:"migrations"`
	Inits           []DatabaseStepResult `json:"inits"`
}

// RegisteredDatabase is one entry from db/registry.yaml.
type RegisteredDatabase struct {
	Key           string
	Driver        string // "mysql" or "sqlite" (or "" = legacy sqlite)
	Database      string // MySQL database name
	Path          string
	Owner         string
	Order         int
	MigrateScript string
	InitScript    string
}

// ServiceStopper stops runAll-managed services before dev database clear/init.
type ServiceStopper interface {
	StopService(ctx context.Context, name string) error
	IsServiceRunning(name string) bool
	// StopAllApplicationsExcept stops non-excluded services (dependents-first, dev-only force stop).
	// onProgress receives a message per successfully stopped service so the dev
	// database-clear UI shows live progress instead of a long silent pause.
	StopAllApplicationsExcept(ctx context.Context, exclude []string, onProgress ProgressCallback) (stopped []string, stillRunning []string)
	// RunningApplicationsExcept lists non-excluded services that are not stopped.
	RunningApplicationsExcept(exclude []string) []string
}

// SQLiteRemover deletes sqlite files for registry keys.
type SQLiteRemover interface {
	RemoveAll(ctx context.Context, entries []RegisteredDatabase) ([]string, error)
}

// ScriptRunner runs shell scripts relative to monorepo root.
type ScriptRunner interface {
	RunScript(ctx context.Context, scriptPath string) error
}

// ProgressCallback reports step-level progress during dev database operations.
// The message should be a human-readable description of the current step,
// e.g. "正在 migrate: task_auth" or "正在重置 MySQL 数据库...".
type ProgressCallback func(message string)

// HasNonMySQLDatabases returns true if any registered database entry has a driver
// other than "mysql" (i.e., "sqlite" or ""), meaning legacy SQLite files may exist
// and should be cleaned up during a dev database clear operation.
func HasNonMySQLDatabases(entries []RegisteredDatabase) bool {
	for _, e := range entries {
		if e.Driver != "mysql" {
			return true
		}
	}
	return false
}

// countSQLiteEntries returns the number of entries that are not MySQL-backed
// (which implies they may have SQLite files to remove).
func countSQLiteEntries(entries []RegisteredDatabase) int {
	n := 0
	for _, e := range entries {
		if e.Driver != "mysql" {
			n++
		}
	}
	return n
}

type databasePlatformBase struct {
	stopper     ServiceStopper
	remover     SQLiteRemover
	scripts     ScriptRunner
	entries     []RegisteredDatabase
	redisScript string
	kafkaScript string
	mysqlScript string
	stopExclude []string
	onProgress  ProgressCallback
}

func newDatabasePlatformBase(
	stopper ServiceStopper,
	remover SQLiteRemover,
	scripts ScriptRunner,
	entries []RegisteredDatabase,
	redisScript, kafkaScript, mysqlScript string,
	stopExclude []string,
	onProgress ProgressCallback,
) databasePlatformBase {
	sorted := append([]RegisteredDatabase(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Order == sorted[j].Order {
			return sorted[i].Key < sorted[j].Key
		}
		return sorted[i].Order < sorted[j].Order
	})
	return databasePlatformBase{
		stopper:     stopper,
		remover:     remover,
		scripts:     scripts,
		entries:     sorted,
		redisScript: redisScript,
		kafkaScript: kafkaScript,
		mysqlScript: mysqlScript,
		stopExclude: append([]string(nil), stopExclude...),
		onProgress:  onProgress,
	}
}

// DatabasePlatformClearService wipes SQLite, Redis, and Kafka topics (no migrate/init).
type DatabasePlatformClearService struct {
	databasePlatformBase
}

func NewDatabasePlatformClearService(
	stopper ServiceStopper,
	remover SQLiteRemover,
	scripts ScriptRunner,
	entries []RegisteredDatabase,
	redisScript, kafkaScript, mysqlScript string,
	stopExclude []string,
	onProgress ProgressCallback,
) *DatabasePlatformClearService {
	return &DatabasePlatformClearService{
		databasePlatformBase: newDatabasePlatformBase(
			stopper, remover, scripts, entries, redisScript, kafkaScript, mysqlScript, stopExclude, onProgress,
		),
	}
}

func (s *DatabasePlatformClearService) Clear(ctx context.Context) DatabasePlatformClearResult {
	result := DatabasePlatformClearResult{
		Status:     "ok",
		MySQLReset: "skipped",
		RedisReset: "skipped",
		KafkaReset: "skipped",
	}
	if s == nil {
		result.Status = "partial"
		return result
	}

	notify := func(msg string) {
		if s.onProgress != nil {
			s.onProgress(msg)
		}
	}

	notify(fmt.Sprintf("正在停止所有应用服务（保留 %v）...", s.stopExclude))
	result.ServicesStopped, result.ServicesStillRunning = s.stopAllApplications(ctx)
	if len(result.ServicesStillRunning) > 0 {
		notify(fmt.Sprintf("警告: %d 个服务仍在运行，中止操作", len(result.ServicesStillRunning)))
		result.Status = "partial"
		return result
	}
	notify(fmt.Sprintf("已停止 %d 个服务", len(result.ServicesStopped)))

	if s.remover != nil {
		sqliteCount := countSQLiteEntries(s.entries)
		notify(fmt.Sprintf("正在清除 %d 个 SQLite 数据库文件...", sqliteCount))
		removed, err := s.remover.RemoveAll(ctx, s.entries)
		result.SQLiteRemoved = removed
		if err != nil {
			notify(fmt.Sprintf("SQLite 清除失败: %v", err))
			result.Status = "partial"
			return result
		}
		notify(fmt.Sprintf("已清除 %d 个 SQLite 数据库", len(removed)))
	}

	if s.scripts != nil && s.mysqlScript != "" {
		notify("正在重置 MySQL 数据库...")
		if err := s.scripts.RunScript(ctx, s.mysqlScript); err != nil {
			notify(fmt.Sprintf("MySQL 重置失败: %v", err))
			result.MySQLReset = "failed"
			result.Status = "partial"
			return result
		}
		result.MySQLReset = "ok"
		notify("MySQL 重置完成")
	}

	if s.scripts != nil && s.redisScript != "" {
		notify("正在重置 Redis...")
		if err := s.scripts.RunScript(ctx, s.redisScript); err != nil {
			notify(fmt.Sprintf("Redis 重置失败: %v", err))
			result.RedisReset = "failed"
			result.Status = "partial"
			// Continue to Kafka — Redis failure should not block independent
			// infrastructure reset steps.
		} else {
			result.RedisReset = "ok"
			notify("Redis 重置完成")
		}
	}

	if s.scripts != nil && s.kafkaScript != "" {
		notify("正在重建 Kafka topics...")
		if err := s.scripts.RunScript(ctx, s.kafkaScript); err != nil {
			notify(fmt.Sprintf("Kafka 重建失败: %v", err))
			result.KafkaReset = "failed"
			result.Status = "partial"
			// Don't return early — downstream callers always inspect the
			// result status field.  Let all steps run so the report is
			// comprehensive.
		} else {
			result.KafkaReset = "ok"
			notify("Kafka topics 重建完成")
		}
	}

	notify("清空数据库全部完成")
	return result
}

// DatabasePlatformInitService runs migrate + init scripts (does not delete databases).
type DatabasePlatformInitService struct {
	databasePlatformBase
}

func NewDatabasePlatformInitService(
	stopper ServiceStopper,
	scripts ScriptRunner,
	entries []RegisteredDatabase,
	stopExclude []string,
	onProgress ProgressCallback,
) *DatabasePlatformInitService {
	return &DatabasePlatformInitService{
		databasePlatformBase: newDatabasePlatformBase(
			stopper, nil, scripts, entries, "", "", "", stopExclude, onProgress,
		),
	}
}

func (s *DatabasePlatformInitService) Init(ctx context.Context) DatabasePlatformInitResult {
	result := DatabasePlatformInitResult{Status: "ok"}
	if s == nil {
		result.Status = "partial"
		return result
	}

	notify := func(msg string) {
		if s.onProgress != nil {
			s.onProgress(msg)
		}
	}

	notify("正在检查运行中的服务...")
	if blocked := s.runningApplications(); len(blocked) > 0 {
		notify(fmt.Sprintf("注意: %d 个服务仍在运行 (%v)，init 脚本可按需使用", len(blocked), blocked))
		// Init scripts (e.g. saas/init.sh) may need Go services for
		// API-driven seed operations. Warn instead of blocking so
		// init scripts can poll and wait for services they depend on.
	}

	for _, entry := range s.entries {
		if s.scripts == nil || entry.MigrateScript == "" {
			continue
		}
		notify(fmt.Sprintf("正在 migrate: %s", entry.Key))
		step := DatabaseStepResult{Database: entry.Key, Status: "ok"}
		if err := s.scripts.RunScript(ctx, entry.MigrateScript); err != nil {
			notify(fmt.Sprintf("migrate %s 失败: %v", entry.Key, err))
			step.Status = "failed"
			step.Message = err.Error()
			result.Migrations = append(result.Migrations, step)
			result.Status = "partial"
			return result
		}
		result.Migrations = append(result.Migrations, step)
	}

	for _, entry := range s.entries {
		if s.scripts == nil || entry.InitScript == "" {
			continue
		}
		notify(fmt.Sprintf("正在 init: %s", entry.Key))
		step := DatabaseStepResult{Database: entry.Key, Status: "ok"}
		if err := s.scripts.RunScript(ctx, entry.InitScript); err != nil {
			notify(fmt.Sprintf("init %s 失败: %v", entry.Key, err))
			step.Status = "failed"
			step.Message = err.Error()
			result.Inits = append(result.Inits, step)
			result.Status = "partial"
			// Continue with remaining init scripts — one failure
			// should not block other independent database inits.
			continue
		}
		result.Inits = append(result.Inits, step)
	}

	notify(fmt.Sprintf("初始化完成: %d migrate + %d init", len(result.Migrations), len(result.Inits)))
	return result
}

func (s *databasePlatformBase) stopAllApplications(ctx context.Context) (stopped []string, stillRunning []string) {
	if s.stopper == nil {
		return nil, nil
	}
	return s.stopper.StopAllApplicationsExcept(ctx, s.stopExclude, s.onProgress)
}

func (s *databasePlatformBase) runningApplications() []string {
	if s.stopper == nil {
		return nil
	}
	return s.stopper.RunningApplicationsExcept(s.stopExclude)
}
