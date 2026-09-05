# 实施计划：任务与项目内容历史版本

- 日期：2026-08-30
- 设计 / 权限 / 价值流 / NFR / DDD 已就绪
- 垂直切片；每切片红→绿→提交

## Slice A — 领域纯函数

- [ ] `taskTaskService/domain/task_revision.go` + `_test.go`：Diff / NextVersion
- [ ] `taskProjectService/domain/project_revision.go` + `_test.go`

验证：`go test ./domain -count=1`

## Slice B — DDL

- [ ] `dataMigrate/taskTaskService/019_task_revision.sql`
- [ ] `dataMigrate/taskProjectService/014_project_revision.sql`
- [ ] utf8mb4 + 月分区 + 无 AUTO_INCREMENT

验证：`bash dataMigrate/check_cold_hot_separation.sh` 相关路径；SQL 语法

## Slice C — 任务存储 + 创建/更新挂钩

- [ ] store Insert/List/Get
- [ ] create + update 同事务
- [ ] 单测 T1–T3、T7

验证：`go test ./src -count=1 -run Revision`

## Slice D — 任务 GET API

- [ ] 路由 revisions 列表/详情
- [ ] 403/404/分页单测 T4–T6、T17

## Slice E — 任务事件

- [ ] eventTopicMap + publish after commit
- [ ] taskEvents intent 18072 `task_revision_recorded/1_observe`
- [ ] 重放测例 T8–T9
- [ ] conf/events + runAll health 同端口

## Slice F — 项目对称 C–E

- [ ] store + create/update + GET + 事件 18073 T10–T16

## Slice G — FE

- [ ] `EntityRevisionPanel.vue` + composable + tests F1–F7
- [ ] 挂到 TaskDetail 标题行与 ProjectDetail
- [ ] `Anti-Replay-OK`；错误 `data-traceId`

## Slice H — RBAC member + swagger 路由登记

- [ ] dataMigrate taskAuth member 行
- [ ] `db/api_route_ownership.yaml` 如适用

## 事件任务（强制）

- [ ] 契约：TASK_REVISION_RECORDED / PROJECT_REVISION_RECORDED
- [ ] publish 于应用服务
- [ ] Kafka adapter
- [ ] 消费者 1_observe + 不同 revision_id 不塌缩

## 完成定义

- 意图验收 T1–T17、F1–F7 绿
- 架构 v120 待 ship 时标 current
- 精准重启登记 taskTaskService、taskProjectService、taskEvents、taskFE
