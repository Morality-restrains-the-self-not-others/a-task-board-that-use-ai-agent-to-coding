package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

// OPT-20260818-040 回归：migrate-status 依赖 RegisteredDatabase.Database
// （MySQL 库名）做按库并发查询。该字段曾漏提交导致 HEAD 上 src/domain 编译
// 断裂（e.Database undefined）；此测试在字段缺失时无法编译/通过，保证
// LoadRegisteredDatabases 始终填充 Database。
func TestLoadRegisteredDatabasesPopulatesDatabaseField(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "db", "registry.yaml")
	if err := os.MkdirAll(filepath.Dir(registry), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registry, []byte(`version: "1"
databases:
  task-bill:
    driver: mysql
    database: task_bill
    migrate_script: db/task-bill/migrate.sh
  task-auth:
    driver: mysql
    database: task_auth
    migrate_script: db/task-auth/migrate.sh
`), 0o644); err != nil {
		t.Fatal(err)
	}

	rows, err := LoadRegisteredDatabases(root)
	if err != nil {
		t.Fatalf("LoadRegisteredDatabases: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len=%d want 2", len(rows))
	}
	byKey := map[string]string{}
	for _, r := range rows {
		byKey[r.Key] = r.Database
	}
	if byKey["task-bill"] != "task_bill" {
		t.Fatalf("task-bill database=%q want task_bill (missing field regression)", byKey["task-bill"])
	}
	if byKey["task-auth"] != "task_auth" {
		t.Fatalf("task-auth database=%q want task_auth", byKey["task-auth"])
	}
}
