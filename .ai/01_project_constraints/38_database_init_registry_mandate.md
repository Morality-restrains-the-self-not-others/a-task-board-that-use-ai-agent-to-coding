# 新服务数据库：必须注册到 db/registry.yaml 并接入统一初始化流程

## 硬约束

**任何新建服务如果需要持久化数据库（MySQL），必须同步完成以下三项，确保 runAll「初始化全部数据库」功能覆盖该服务：**

### 1. `db/registry.yaml` — 注册数据库条目

在 `databases:` 下添加新条目：

```yaml
  <db-key>:
    driver: mysql
    database: <mysql_database_name>      # MySQL 库名
    owner: <runall-service-name>        # runAll 中的服务名（对应 conf/runAll.yaml）
    order: <NN>                          # 初始化顺序（整数，越小越先执行）
    migrate_script: db/<script-dir>/migrate.sh
    init_script: db/<script-dir>/init.sh
    description: <一句话描述>
```

**字段约束：**
- `driver`: 必须为 `mysql`（项目已全面迁移到 MySQL，禁止新增 SQLite 数据库）
- `owner`: 必须与 `conf/runAll.yaml` 中的 `name` 字段一致
- `order`: 遵守依赖拓扑顺序（如 task-auth(9) 先于 saas(10)）
- `migrate_script` / `init_script`: 路径相对于 monorepo 根

### 2. `db/<script-dir>/migrate.sh` — Schema 迁移脚本

创建数据库表结构（DDL）。必须是 **幂等** 的（`CREATE TABLE IF NOT EXISTS`）。

三种标准模式任选其一：

**模式 A — Go 服务（推荐）：**
```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT/<service-dir>"
go run ./src migrate
```
前提：Go 服务的 `main.go` 支持 `go run ./src migrate` 子命令，该子命令执行 `openDB()` 后 `return`（不启动 HTTP server）。

**模式 B — Django 服务：**
```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT/<django-project-dir>"
PYTHON="${PYTHON:-python3}"
"$PYTHON" manage.py migrate --noinput
```

**模式 C — 纯 SQL 执行：**
```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
python3 "$ROOT/db/scripts/mysql_exec.py" <db-key> "$ROOT/dataMigrate/<service>"
```
适用于 migration 完全由 `dataMigrate/` 目录下的 `.sql` 文件管理的服务。

### 3. `db/<script-dir>/init.sh` — 数据初始化脚本

Seed 数据 / bootstrap 操作。同样必须幂等。

```bash
#!/usr/bin/env bash
set -euo pipefail
# 无 seed 数据时可为 no-op
echo "[<db-key>] init.sh: no-op (seed data handled by dataMigrate)"
exit 0
```

### 4. MySQL 数据库创建 — `dockerInfra/mysql/init/01-create-databases.sql`

在 MySQL 初始化 SQL 中添加：
```sql
CREATE DATABASE IF NOT EXISTS <mysql_database_name> CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 5. `db/load/registry.go` — 更新 DSN 环境变量映射

在 `mysqlDSNEnvByKey` map 中添加新条目（如需自定义 DSN）：
```go
"<db-key>": "<DB_KEY>_MYSQL_DSN",
```

---

## 架构说明

```
http://10.2.150.68:9999/ → 「初始化全部数据库」按钮
  → POST /api/dev/init-databases?confirm=INIT_ALL
    → runner.InitAllDatabases()
      → loadDevDatabaseContext()
        → LoadRegisteredDatabases()  // 读 db/registry.yaml
      → DatabasePlatformInitService.Init()
        → 检查是否有服务在运行（有则 block）
        → 按 order 排序后逐个执行:
          for each entry:
            run migrate_script  → 建表 (DDL)
            run init_script     → seed 数据
```

**关键设计原则：**
- 初始化流程**不启动任何服务**，脚本必须在 Shell 层面独立完成
- `migrate_script` 和 `init_script` 由 `BashScriptRunner` 在 monorepo 根目录执行
- 初始化前会检查是否有服务在运行，有则返回 `"blocked"` 状态
- 脚本内部通过 `ROOT="$(cd "$(dirname "$0")/../.." && pwd)"` 定位 monorepo 根

---

## Go 服务 migrate 子命令规范

新建 Go 服务时，`main.go` 必须包含标准 migrate 子命令：

```go
func main() {
    repoRoot, err := findMonorepoRoot()
    if err != nil {
        log.Fatalf("[service] monorepo root: %v", err)
    }
    loadConfig(repoRoot)

    // 标准 migrate 子命令 — 执行迁移后退出，不启动 HTTP server
    if len(os.Args) > 1 && os.Args[1] == "migrate" {
        if err := openDB(cfg.MySQLDSN); err != nil {
            log.Fatalf("[service] migration failed: %v", err)
        }
        log.Println("[service] migration complete")
        return  // 或 os.Exit(0)
    }

    // 正常启动流程...
    if err := openDB(cfg.MySQLDSN); err != nil {
        log.Fatalf("[service] db: %v", err)
    }
    // ... HTTP server
}
```

`openDB()` 必须包含完整的迁移逻辑：
```go
func openDB(dsn string) error {
    db, err = sql.Open("mysql", dsn)
    // pool settings...
    if err := db.Ping(); err != nil { return err }
    if err := runMigrations(); err != nil { return err }     // DDL
    if err := runDataMigrate(repoRoot()); err != nil { ... } // dataMigrate SQL
    return nil
}
```

---

## 检查清单

新建服务/数据库时逐项确认：

- [ ] `db/registry.yaml` 已添加数据库条目（`driver: mysql`, `migrate_script`, `init_script`）
- [ ] `db/<script-dir>/migrate.sh` 存在、可执行、幂等
- [ ] `db/<script-dir>/init.sh` 存在、可执行、幂等
- [ ] migrate.sh 使用的子命令/脚本**不启动 HTTP server**（Go migrate 子命令执行后 `return`）
- [ ] `dockerInfra/mysql/init/01-create-databases.sql` 已添加 `CREATE DATABASE IF NOT EXISTS`
- [ ] `db/load/registry.go` 的 `mysqlDSNEnvByKey` 已添加新条目
- [ ] Go 服务 `main.go` 包含标准 `migrate` 子命令
- [ ] Go 服务 `openDB()` **不**调用 `runDataMigrate`（业务进程与迁移解耦）
- [ ] 显式 `migrate` CLI 或 `migrate.sh` → `apply_datamigrate.sh` 覆盖本库
- [ ] `python3 dataMigrate/check_no_startup_migrate.py` 通过
- [ ] 所有 `dataMigrate/<service>/*.sql` 文件使用 MySQL 语法（非 SQLite）
- [ ] 在 runAll 面板点击「初始化全部数据库」验证通过

---

## 动机

2026-07-27 审计发现：`db/registry.yaml` 中 13 个数据库仅 6 个有 migrate/init 脚本，
其余 7 个依赖 Go 服务启动时自动迁移。这导致：

1. 「初始化全部数据库」功能只覆盖 46% 的数据库
2. 新服务开发者不知道需要注册到 registry.yaml
3. 数据库初始化与服务启动耦合，无法独立验证 schema 完整性

此规则确保所有数据库的初始化完全由 runAll 统一管理，实现零耦合离线初始化。

## 相关文件

- `db/registry.yaml` — 数据库注册表（权威来源）
- `db/scripts/mysql_exec.py` — MySQL 连接解析 + SQL 执行共享工具
- `db/load/registry.go` — registry.yaml 的 Go 解析器（`LoadDatabaseEntries`, `ResolveMySQLDSN`）
- `db/load/registry_entries.go` — 类型定义
- `runAll/src/domain/database_platform_reset.go` — 初始化/Clear 核心领域逻辑
- `runAll/src/ui.go` — HTTP API 端点注册（`/api/dev/init-databases`）
- `runAll/src/runner.go` — `InitAllDatabases()` 实现
- `runAll/src/infrastructure/database_platform_reset.go` — `LoadRegisteredDatabases`, `BashScriptRunner`
- `dockerInfra/mysql/init/01-create-databases.sql` — MySQL 容器首次启动建库脚本
