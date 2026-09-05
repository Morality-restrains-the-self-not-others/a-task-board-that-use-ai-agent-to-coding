# 测试意图：任务与项目主要属性历史版本

对应：`task_project_entity_revisions.intent.md`

| ID | 场景 | 步骤 | 期望 |
|----|------|------|------|
| T1 | 创建任务写 v1 | POST todos 后查 revisions | 1 行 version_num=1，字段一致 |
| T2 | 无内容变更 | PATCH progress_column_id | revisions 仍 1 行 |
| T3 | 改标题 | PATCH title | 2 行；最新 version=2；changed_fields 含 title |
| T4 | 无权限 | 无 workspace 访问 GET list | 403 |
| T5 | IDOR revision | GET 其它 task 的 revisionId | 404 |
| T6 | 分页 | 插入 >limit 行 | total 正确，不跨 task |
| T7 | 同事务失败 | 插入 revision 失败 | 热表 title 未改 |
| T8 | 事件 | 改 description | 发布 TASK_REVISION_RECORDED，key=task_id，payload 无 description 全文 |
| T9 | 消费重放 | 同一 revision_id 两次 | 第二次 Ack 空操作 |
| T10–T16 | 项目对称 | 创建/改名/未变/403/404/分页/事件 | 与 T1–T8 同构 |
| T17 | 列表排序 | 多版本 | version_num DESC |

可执行：`taskTaskService/src/task_revision_*_test.go`；`taskProjectService/src/project_revision_*_test.go`；`taskEvents/.../taskrevisionrecorded`；`taskEvents/.../projectrevisionrecorded`。
