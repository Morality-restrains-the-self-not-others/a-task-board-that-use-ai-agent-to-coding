# 价值流：任务与项目内容历史版本

- 日期：2026-08-30
- 设计：`docs/superpowers/specs/2026-08-30-task-project-entity-revisions-design.md`

Mapping the approved design into a value stream.

## Related Value Streams

- `task-management` / `todo-crud`：任务创建与 PATCH
- `project-workspace`：项目创建与更新
- 本增量不改进度流转、OAuth、启机

## 用户价值

改过标题/正文之后，仍能打开当时那一版内容。

## 增量（单一 MVP）

1. 创建与内容变更写入快照表
2. GET 列表/详情
3. 任务详情 + 项目详情查阅 UI
4. 领域事件 + observe 消费者

无第二期（还原、diff、工作空间/评论历史）。

## 步骤与测试点

| 步骤 | 测试文件 | 测试点 |
|------|----------|--------|
| 建帖写 v1 | task_revision_store_test.go | version=1 |
| 改标题写 v2 | task_revision_update_test.go | 未变字段不写 |
| GET 列表权限 | task_revision_handlers_test.go | 403/404/分页 |
| 项目对称 | project_revision_*_test.go | 同构 |
| 事件 | events / consumer replay | revision_id 幂等 |
| FE 面板 | EntityRevisionPanel.test.js | 空态/列表/详情/traceId |

## YAML

`conf/value-stream.yaml` 增加 `task-entity-revision`、`project-entity-revision`。
