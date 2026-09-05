# 功能意图：容器执行全链路绕过 Django

- **日期**: 2026-07-10
- **状态**: ✅ Phase A–D + E（auth-context / identity_id / mock-run 加深 / open-runtime-session / Django 死路由清理）shipped
- **设计文档**: `docs/superpowers/specs/2026-07-10-container-exec-bypass-django-design.md`

## 意图

容器执行请求（提交指令、job 生命周期、层/文件读、job 轮询）的**整条热路径**不得经过 Django（saas-backend），包括公网与 internal 回调。鉴权走 taskAuth，目标解析走 taskCloudService（+ Credential），出站由 taskContainerGateway 转发 onlineServiceJS。冷路径 job-stream publish、git-push（含 auth-context / identity_id）、mock-run（含 installed_image_id / env-defaults）、open-runtime-session 亦迁出 Django。

## 为何

1. runAll saas-backend 日志出现容器执行相关记录，说明仍有热路径打到 Django。
2. 运行时：`task-container-gateway` 每次请求 `django_validate`；Django 不可达 → 502（30s），执行不可用。
3. 公网已 410 迁出，但 internal 双跳未收口，与「Go 化容器栈」目标不一致。

## 验收（产品视角）

- 在已启动容器上发送指令 / 查看执行日志时，saas-backend 日志不再出现 `validate-session` / `resolve-container-target`。
- 临时停止 saas-backend 后，上述操作仍可用。
- 未登录、跨租户、未注册地址等错误语义与现网一致。
- job-stream / git-push（prepare|complete|auth-context|identity_id）/ mock-run（image 或 installed_image_id）/ open-runtime-session 不再打 Django internal。



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：热路径路由绕过 Django（HTTP/Gateway），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 功能意图：容器执行全链路绕过 Django | — | — | — | — | 热路径路由绕过 Django（HTTP/Gateway），无新增业务事件 |
## 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-10 | 初版意图，对齐 v13 设计 |
| 2026-07-10 | Phase A–D + sessionid 最小切片落地 |
| 2026-07-10 | Phase E：auth-context / identity_id / mock-run 加深 / open-runtime-session；Django 旧 tcg internal 路由删除（保留 clear-reachability / relay-workflow） |
| 2026-07-10 | Phase F：Django tcg internal 全清；WorkflowUpdate NO-OP；GitLab readiness；AI env agent；djangoInternalApiBase 删除；commit/push |
