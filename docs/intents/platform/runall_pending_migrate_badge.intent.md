# Intent: 9999 初始化按钮标注未 migrate

## 背景与目标

http://10.2.150.68:9999/ 的「初始化全部数据库（开发）」无法显示哪些库的 `dataMigrate` SQL 尚未写入 `data_migrate_log`。目标：有缺口时在按钮旁标注数量与文件清单。

## 范围与边界

- **范围内**：runAll `GET /api/dev/migrate-status`；Dev UI 徽章/高亮/tooltip；首屏与 init/clear 后刷新。
- **范围外**：不改 migrate 执行器；不把巡检塞进 2s `/api/status`；不读业务表；不新增 Python HTTP。

## 约束与风险

- 只读 `data_migrate_log`；SQL 参数化/固定语句，库名来自 registry。
- 不可达不得伪装成 pending。
- 禁止 2s 轮询打 MySQL。

## 验收标准

1. 存在 missing SQL 时，`#dev-init-databases` 带 `has-pending-migrate`，旁路标签含 `未 migrate` 与数量。
2. 全部已应用时无该高亮、标签为空。
3. tooltip 按库列出 missing 文件名。
4. 表不存在视为全部未应用。
5. MySQL 不可达计 `unreachable`，不增加 `pending_count`。

## 实施计划

1. domain：diff + report
2. infrastructure：列目录 + mysql 读 step_key
3. GET handler + UI 徽章
4. 单测锁定语义

## 业务意图 → 事件对照

> 运维只读巡检，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: 9999 初始化按钮标注未 migrate | — | — | — | — | 纯查询/Dev UI 标注，无系统事实变更 |

## 变更记录

- 2026-08-18：新增。
