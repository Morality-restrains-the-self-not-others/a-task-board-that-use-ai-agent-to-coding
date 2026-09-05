# Design: translate-branch-title 纯 Go（去掉 Django 委托）

- **日期**: 2026-07-20
- **状态**: approved（goal-mode 自动采用）
- **落点**: 扩展 `taskProjectService`（不新建服务）

## 问题

公网 `POST /api/tenant/{tid}/projects/translate-branch-title/` 已由网关路由到 `taskProjectService`，但含中文标题时仍 `djangoPost` → `/api/internal/taskproject/translate-branch-title/`。该 Django internal **未注册**，导致 502 与前端「任务标题翻译失败」。

## 方案（采用）

在 `taskProjectService` 内直连 `conf/core/django/config.yaml` 的 `ai_agent_config.fanyi_agent`（OpenAI 兼容 `chat/completions`），用 `tracelog.DirectClient` 出站（禁止环境 Proxy）。无中文路径保持本地 sanitize。

**不采用**：补 Django internal（仅延长 Python 依赖）；迁到 `taskAIEndPoint`（过重，无租户 LLM 代理需求）。

## 契约（不变）

| 字段 | 说明 |
|------|------|
| 请求 | `{ "title": string }` |
| 200 | `{ source_title, translated_title, used_ai }` |
| 400 | title 空 / JSON 无效 |
| 405 | 非 POST |
| 502 | fanyi 配置缺失或上游失败。`error` 为用户可读短句；超时/空 body/JSON 解析细节只写日志（含 status、bytes、trace_id） |

## 权限

沿用网关 JWT / `X-Auth-*`；无新角色。成员校验由网关 + 既有 projects 前缀策略承担（与改造前一致）。

## 事件

纯工具查询/变换，**无领域事件**（书面例外：不改变任务/项目状态）。

## 架构交付物

- `docs/architecture/v42-application-integration-20260720-1555-claude.{puml,archimate,mermaid.md}`
- 更新 `docs/architecture/api-route-to-owner.md`：标记 Django internal 路径废弃

## 验收

1. 非中文 → 200 `used_ai=false`
2. 中文 + mock fanyi → 200 `used_ai=true`
3. 配置缺失 / 上游 500 → 502
4. `go test ./src -run TranslateBranch`
5. 本机 `:8016` 中文标题不再因 Django 404 秒级 502
