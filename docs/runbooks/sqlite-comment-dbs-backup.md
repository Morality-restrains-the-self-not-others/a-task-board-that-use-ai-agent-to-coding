# Runbook：评论双库（MySQL）备份与成对恢复

> ⚠️ **本文件已随存储引擎 MySQL 化更新**（2026-08-24，docs-cleanup）：历史内容为 SQLite 时代（嵌入式 SQLite 单写者粘滞），现运行环境为 MySQL，文件名沿用历史命名。`db/registry.yaml` 中 `path` 字段（legacy SQLite）仅用于清理，**不再有 .db 文件可备份**。

- **适用**：`taskTaskService`（MySQL 库 `task_task`，任务元数据 + 人类评论 `comments`）+ `taskAIComment`（MySQL 库 `task_ai_comment`，AI 指令评论 + 容器 Agent 评论）
- **相关设计**：`docs/superpowers/specs/2026-07-15-task-comments-sql-vs-nosql-scale-design.md` §13 R5/R6
- **连接参数**（SSOT）：`db/registry.yaml`（mysql 块：host/port/user/password；`database` 字段为库名）

## 背景

人类评论与任务元数据在 **task_task** 库；AI 指令评论与容器 Agent 评论在 **task_ai_comment** 库。两库为**独立 MySQL 库**（同一实例），沿用「评论/任务双库分库」模式（单库单表单 owner 约束 19 号）；SQLite 时代的「嵌入式单写者粘滞」约束已随 MySQL 化消除 —— 多副本并发读由 MySQL 实例承担，写由 InnoDB 行锁仲裁，**不再需要先停旧实例再启新实例**。

## 备份策略

### 频率

| 库 | MySQL database | 建议 |
|----|----------------|------|
| 任务 + 人类评论 | `task_task` | 至少每日；上线初期可每 6h |
| AI + Agent 评论 | `task_ai_comment` | 与人类评论**同周期** |

###  procedure（单库）

```bash
# 连接参数取自 db/registry.yaml（示例默认值，生产按实际覆盖）
MYSQL_HOST=127.0.0.1; MYSQL_PORT=3306; MYSQL_USER=taskapp; MYSQL_PASS=taskapp123
TS=$(date +%Y%m%d_%H%M%S)

# 单事务一致性备份（.backup API 的 MySQL 等价物）
mysqldump -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" \
  --single-transaction --routines --triggers \
  task_task > "/backup/task_task_${TS}.sql"

mysqldump -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" \
  --single-transaction --routines --triggers \
  task_ai_comment > "/backup/task_ai_comment_${TS}.sql"
```

> 也可用 `mysqlpump` / Percona XtraBackup 物理备份（大库时更快）；逻辑备份为基线，物理备份可叠加 binlog 做 PITR（`--single-transaction` 与 binlog 配合可精确到点恢复）。

### 双库成对备份

- **同一备份批次**内依次备份 `task_task` 与 `task_ai_comment`，记录**同一时间戳**与 Git/部署版本。
- 恢复演练须**同时恢复两库**，否则任务详情可能出现「有人类无 AI」或反之的 transient 不一致。
- 备份产物存 off-host（对象存储/异地），保留 ≥7 天；加密与访问审计按平台基线。

## 恢复演练（季度）

1. 在隔离环境恢复**成对**备份文件：

```bash
mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" task_task < /backup/task_task_${TS}.sql
mysql -h "$MYSQL_HOST" -P "$MYSQL_PORT" -u "$MYSQL_USER" -p"$MYSQL_PASS" task_ai_comment < /backup/task_ai_comment_${TS}.sql
```

2. 启动 `taskTaskService` + `taskAIComment`，`GET /api/health` 均 200。
3. 抽一条有评论的任务：`GET .../todos/{id}/` 含 `comments`、`ai_comments`、`container_agent_comments`。
4. 记录 RTO/RPO 与演练日期。

## 监控与告警

- MySQL 实例磁盘使用率 / 慢查询日志（两库所在实例）
- 评论列表 payload 字节（Phase 0 指标，可选）
- MySQL 连接池 `WaitTimeout` / 锁等待（`SHOW ENGINE INNODB STATUS` 或 Performance Schema 监控）

## 何时迁 Postgres

见评论规模设计 §5 Phase 3：**MySQL 实例容量或 HA 硬需求**达到时评估迁共享 Postgres（同 schema 换引擎），**非** Mongo。SQLite 单写者限制已不存在，当前 MySQL 形态即该设计「SQLite 双库首发形态」的工程落地。

## 联系人 / 变更

- 表 ownership：`db/table_ownership.yaml`
- 库注册 / DSN：`db/registry.yaml`
- 空壳表清理：`taskTaskService` 启动 migration `DROP TABLE IF EXISTS ai_task_comments`（仅当表空）
