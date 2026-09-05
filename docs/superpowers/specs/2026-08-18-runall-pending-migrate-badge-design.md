# 设计：9999「初始化全部数据库」标注未 migrate

- **日期**: 2026-08-18
- **作者**: cursor
- **迭代**: runall-pending-migrate-badge
- **状态**: accepted（goal-mode 自动采纳）
- **意图**: `docs/intents/platform/runall_pending_migrate_badge.intent.md`
- **python_api_approval**: n/a（零新增 Python HTTP；落 runAll Go）
- **架构**: **不更新** ArchiMate。无新服务/无新所有权表/无新 MQ。runAll 已是 `InitAllDatabases` 编排器，本次只增加对既有 `data_migrate_log` 的运维只读 diff。

---

## 对当前架构的理解

- current：`enterprise-landscape` / `application-integration` **v85**
- 应用层：runAll :9999 Dev UI 已有 `POST /api/dev/init-databases` → `InitAllDatabases` → `db/registry.yaml` → `apply_datamigrate.sh`
- 追踪表：各库 `data_migrate_log.step_key` = SQL 文件名（与 apply 脚本一致）
- 已有离线巡检：`db/scripts/ci/check_data_migrate_applied.py`（语义 SSOT：local−applied = missing）

本次不新增业务组件，只让运维在按钮上看到巡检结果。

## 🕸️ Code Review Graph 分析

- Step 0：`code-review-graph update --brief` 成功（20 files / 0 nodes）。
- `InitAllDatabases` 调用链：`ui.go` POST handler → `Runner.InitAllDatabases` → `domain.DatabasePlatformInitService.Init`。
- 新符号与 Init 并列：只读 `MigratePendingStatus`，不改写路径。
- `.codegraph/` 索引缺失；未跑 `codegraph_explore`（CLI 有 `codegraph`，无 MCP）。

## 问题

按钮「初始化全部数据库（开发）」看不出哪些库/文件尚未应用。开发加了 `dataMigrate/*.sql` 后容易忘记点初始化，表现为缺列/缺表。

## 方案（采纳）

**按钮旁徽章 + 只读 API**，语义对齐 `check_data_migrate_applied.py`。

| 项 | 决策 |
|---|---|
| API | `GET /api/dev/migrate-status`（无 confirm；不走 2s `/api/status`） |
| 比较 | 各 registry MySQL：`dataMigrate/*.sql` 文件名 vs `data_migrate_log.step_key` |
| missing | 计入「未 migrate」，按钮琥珀高亮 + 数量 |
| stale | 仅 tooltip，不计入数量 |
| 表不存在 | 视为 applied=[] → 全部 local 为 pending（清库后正是要标的） |
| 连不上 | `unreachable`，单独计数，不误报 pending |
| 非 MySQL | `skipped` |
| UI 拉取 | 首屏一次 + init/clear 完成后；**禁止**挂到 2s `refresh()` |
| 服务端缓存 | 15s，避免连点/回页打满 mysql |

### 拒绝的方案

| 方案 | 拒绝原因 |
|---|---|
| 塞进 `/api/status` 2s 轮询 | 每次刷新打全部库；违反新周期工作不扩轮询 |
| 改按钮文案替代徽章 | 破坏现有 `ui_test.go` 文案锚点；徽章与精准重启标签一致 |
| 各业务服务暴露 status | 过重；权威入口已是 9999 |
| 调用 Python 巡检脚本 | HTTP 路径禁止依赖 Python 子进程作为契约 |

### API 契约

```json
{
  "pending_count": 3,
  "unreachable_count": 0,
  "databases": [
    {
      "key": "task-bill",
      "database": "task_bill",
      "status": "pending",
      "missing": ["046_foo.sql"],
      "stale": [],
      "local_count": 46,
      "applied_count": 45
    }
  ]
}
```

`status`: `ok` | `pending` | `unreachable` | `skipped`。错误体与现网 runAll 一致：`{ "error": "..." }`。

### UI

- 保持按钮可见文本「初始化全部数据库（开发）」
- 旁路 `#dev-init-db-pending-label`：有 missing 时显示 `N 未 migrate`
- `button.has-pending-migrate` 琥珀描边；`title` 列出 `库: 文件…`
- 徽章在按钮外，点击不触发初始化

### 数据访问边界

runAll 已通过 `migrate.sh` 写各库 `data_migrate_log`。本次只读该追踪表，**禁止**读业务表。查询用 mysql CLI / docker exec（与 `apply_datamigrate.sh` 同通道），不引入业务 ORM。

### 事件

纯查询，无领域事件。

### 可观测性

`[runall] migrate-status pending=%d unreachable=%d`；禁止日志含 MySQL 密码/DSN 口令。
