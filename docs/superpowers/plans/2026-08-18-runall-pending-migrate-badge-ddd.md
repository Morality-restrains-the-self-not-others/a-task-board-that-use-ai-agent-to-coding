# DDD — 9999 未 migrate 标注

- **日期**: 2026-08-18
- **限界上下文**: runAll 运维编排（已有 `database_platform_reset.go`）
- **不新建 BC**

## 值对象

- `MigrateDatabaseStatus`：单库巡检结果（key/database/status/missing/stale/counts/error）
- `MigratePendingReport`：pending_count + unreachable_count + []status

## 领域服务

- `DiffStepKeys(local, applied)` — 纯集合差
- `BuildMigratePendingReport(entries, resolveDir, listSQL, listApplied)` — 编排比对；MySQL only

## 端口

```
AppliedStepReader.ListApplied(ctx, databaseName) ([]string, error)
LocalSQLLister.ListSQL(ctx, absDir) ([]string, error)
MigrateScriptDirResolver.Resolve(monorepoRoot, migrateScriptPath) (absDir string, ok bool)
```

表不存在：适配器返回 `([], nil)` 而非 error。连接失败：error → unreachable。

## 事件

无。纯查询例外已写意图表。

## 依赖方向

domain 不 import infrastructure / database/sql / os/exec。
