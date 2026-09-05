# Intent: token-init 路径携带 tenant/workspace/task + 内网 loopback

## 背景

start-vm-auto 经 Django `_issue_token_via_go` 调 CRED 时，公网 credential 子域 + CRED 仅绑 127.0.0.1 导致 nginx 502。

## 需求（2026-07-12）

1. 修复 `start-vm-auto` → token init 502，使签发走通。
2. `/v1/token/init` 将 `tenantId`、`workspaceId`、`taskId` 放入路径。

## 契约

| 项 | 值 |
|---|---|
| Method | `POST` |
| Path | `/v1/token/init/tenant/{tenantId}/workspace/{workspaceId}/task/{taskId}/comment/{commentId}`（ADR-0005；旧 9 段须 query/body `comment_id`） |
| Body | 可空 JSON `{}` 或 `{"comment_id":"..."}`（新路径以路径为准） |
| 调用方 base | `http://127.0.0.1:8015`（Django `taskCredentialServiceBase` / TCG `credentialService.url`） |
| 成功 | `200` + `access_token` / `refresh_token` / `expires_at` |
| 路径非法 | `404` |



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：token-init URL/绑定范围配置，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: token-init 路径携带 tenant/workspace/task + 内网 loopback | — | — | — | — | token-init URL/绑定范围配置，无新增业务事件 |
## 变更记录

| 日期 | 相对旧版 | 原因 |
|---|---|---|
| 2026-07-12 | 旧：`POST /v1/token/init` + JSON body scope；Django base 为公网 credential 子域 | 502 根因修复 + 路径携带 scope |
