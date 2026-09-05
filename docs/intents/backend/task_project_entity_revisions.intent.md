# 意图：任务与项目主要属性历史版本

- 日期：2026-08-30
- 设计：`docs/superpowers/specs/2026-08-30-task-project-entity-revisions-design.md`
- ADR：ADR-0051

## 背景与目标

任务/项目标题与正文只留最新值。目标：主要属性实际变更时写入不可变快照，授权用户可列表与查阅。

## 范围与边界

- 范围内：`task_revision`、`project_revision`；创建写 v1；PATCH 在 title/description 或 name/description 变化时写下一版；GET 列表/详情；事件 `TASK_REVISION_RECORDED` / `PROJECT_REVISION_RECORDED`。
- 范围外：进度/完成/指派/镜像/仓库/标签；一键还原；评论与工作空间；存量回填。

## 约束与风险

- DDL 仅 `dataMigrate/` + 9999；月分区；Snowflake PK。
- 与实体写同事务，fail-closed。
- 列表/详情权限与 GET 实体相同；禁止跨租户/跨实体读 revision。
- 事件 payload 不含正文；消费幂等键 `revision_id`。

## 验收标准

1. 新建任务 → 一行 `task_revision` version_num=1，title/description 与帖一致。
2. PATCH 只改进度 → 不新增 revision、不发事件。
3. PATCH 改 title → 新 revision，changed_fields 含 `title`，热表为新标题。
4. 无工作空间访问 → GET revisions 403；错误节点可带 trace。
5. 用其它任务的 revisionId 查本任务 → 404。
6. 项目 name/description 对称满足 1–5。
7. 插入失败则实体更新回滚。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 创建任务记下正文 | TASK_REVISION_RECORDED | handleCreateTask | taskEvents 1_observe | — |
| 任务标题/正文变更 | TASK_REVISION_RECORDED | handleUpdateTask | 同上 | 未变则不发 |
| 创建项目记下正文 | PROJECT_REVISION_RECORDED | handleCreateProject | 同上 | — |
| 项目名称/正文变更 | PROJECT_REVISION_RECORDED | handleUpdateProject | 同上 | 未变则不发 |
| 查阅版本 | — | GET | — | 纯查询 |

## 实施计划

1. dataMigrate 两表 + 分区。
2. 创建/更新路径同事务插入 + 单测。
3. GET 列表/详情 + IDOR 单测。
4. 事件契约、intent 18072/18073、重放测例。
5. FE 面板 + RBAC member 登记。
