# dataMigrate — 数据迁移与初始化脚本

本目录存放各服务的 Schema 迁移脚本（DDL）和数据初始化脚本（seed / bootstrap），
按服务分目录，编号排序执行。

## 目录规范

```
dataMigrate/
  <service-name>/
    NNN_<描述>.<ext>   # 三位数字编号 + 描述名
```

- Schema DDL 优先（001-009）
- Data seed / bootstrap 在后（010+）

## 防重复机制（三层保障）

每个脚本通过 `data_migrate_log` 追踪表实现幂等执行：

```sql
CREATE TABLE IF NOT EXISTS data_migrate_log (
    step_key VARCHAR(255) PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

三层保障（缺一不可）：
- **L1 追踪表** — `data_migrate_log` 记录已执行的 step_key
- **L2 稳定存在性检查** — 用不可变的业务条件判断数据是否已存在
- **L3 SQL 幂等** — `CREATE TABLE IF NOT EXISTS` / `INSERT IGNORE`

## Go 服务参考实现

新增 Go 服务时，**不要**在 `openDB` / 默认 `main` 调用迁移。迁移由 9999 / `migrate.sh` 执行。可选提供显式 CLI：

```go
// go run ./src migrate  — 仅运维/ migrate.sh 调用
func main() {
    if len(os.Args) > 1 && os.Args[1] == "migrate" {
        _ = runDataMigrateFromDir(dsn, repoRoot)
        return
    }
    // server: openDB only — no migrate
}
```

禁止编写 `runMigrations` 函数。历史参考实现（**已废弃：勿在业务启动路径复制**）：

```go
// DEPRECATED as startup hook — kept only for migrate CLI / tests.
func runDataMigrate(repoRoot string) error {
    migDir := filepath.Join(repoRoot, "dataMigrate", "<service>")
    // ... 读取 .sql，事务执行，写入 data_migrate_log
    return nil
}
```

## 统一迁移入口

`http://<host>:9999/`（例：`http://10.2.150.68:9999/`）「初始化全部数据库」通过以下链路自动覆盖所有 dataMigrate SQL：

```
9999 UI → POST /api/dev/init-databases → InitAllDatabases()
  → db/registry.yaml（数据库注册表）
  → 逐库: db/<db>/migrate.sh
  → db/scripts/apply_datamigrate.sh <db_name> <dataMigrate_dir>
  → 逐文件幂等执行 SQL → data_migrate_log 追踪
```

**新增 SQL 文件生效方式**：放入 `dataMigrate/<service>/` → 在 **9999** 执行「初始化全部数据库」（或对该库跑 `migrate.sh`）→ **再**启动业务进程。  
**业务进程启动不再自动迁移**（元规则：`.ai/01_project_constraints/40_app_process_independent_of_db_migrate.md`）。

## DELIMITER / 存储过程双路径

含 `CREATE PROCEDURE` 的迁移若体内有 `;`，mysql CLI（`apply_datamigrate.sh`）**必须**使用 `DELIMITER // ... END //`。

Go `runDataMigrate*`（`database/sql` COM_QUERY）**不识别** `DELIMITER`：taskAuth / taskBill 在 `Exec` 前通过 `stripMySQLClientMeta` 去掉 `DELIMITER` 行并把行尾 `//` 改回 `;`。

新增同类脚本时：SQL 文件保留 `DELIMITER`（CLI 可用）；确保对应 Go 执行路径调用了 `stripMySQLClientMeta`（或等价预处理）。

## 独立执行

```bash
# 单个服务
bash db/scripts/apply_datamigrate.sh task_auth dataMigrate/taskAuth

# 所有服务（通过 9999 API）
curl -X POST 'http://127.0.0.1:9999/api/dev/init-databases?confirm=INIT_ALL'
```

## 服务清单

| 服务目录 | 文件数 | runMigrations | migrate.sh |
|----------|--------|--------------|------------|
| `taskAuth/` | 18 SQL + 1 py | ~~已重命名~~ → `runDataMigrateFromDir` | ✅ `apply_datamigrate.sh` |
| `taskBill/` | 32 SQL + 1 py + 3 md | ~~已重命名~~ → `runDataMigrateFromDir` | ✅ `apply_datamigrate.sh` |
| `taskReferral/` | 3 SQL | ~~已重命名~~ → `runDataMigrateFromDir` | ✅ `apply_datamigrate.sh` |
| `taskCloudService/` | 10 SQL | ~~已删除~~ → `runDataMigrate` only | ✅ `apply_datamigrate.sh` |
| `taskProjectService/` | 8 SQL | ~~已删除~~ → `runDataMigrate` only | ✅ `apply_datamigrate.sh` |
| `taskTaskService/` | 7 SQL + 3 skip | ~~已删除~~ → `runDataMigrate` only | ✅ `apply_datamigrate.sh` |
| `taskTenantService/` | 3 SQL | ~~已删除~~ → `runDataMigrate` only | ✅ `apply_datamigrate.sh` |
| `taskAIComment/` | 4 SQL | ~~已删除~~ → `runDataMigrate` only | ✅ `apply_datamigrate.sh` |
| `taskAiProvider/` | 2 SQL | (infrastructure 包管理) | ✅ `apply_datamigrate.sh` |
| `taskGitOauth/` | 2 SQL | (EnsureSchema 管理) | ✅ `apply_datamigrate.sh` |
| `taskCredentialService/` | 1 SQL | (infrastructure 包管理) | ✅ `apply_datamigrate.sh` |
| `taskBudget/` | 1 SQL | (budget_db.go 管理) | ✅ `apply_datamigrate.sh` |

**图例**: ✅=合规, ~~已删除~~=v2 清理完成, ~~已重命名~~=v2 重命名完成

## Code Review 门禁

- 新增 DDL 必须首日放入 `dataMigrate/<service>/` 目录
- PR 中出现 `func runMigrations` → 立即拒绝
- `bash dataMigrate/check_inline_ddl.sh` — CI 检测内嵌 CREATE TABLE
- `grep -rn 'func runMigrations'` — CI 检测残留 runMigrations 函数

## License

本仓库以 GNU Affero General Public License v3.0 授权，见 [LICENSE](./LICENSE)。不附带 AGPL 义务的专有许可见 [COMMERCIAL.md](./COMMERCIAL.md)。
