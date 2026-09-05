# 功能意图：HTML head 标明提供页面的服务

## 用户故事

作为排障的研发或 Agent，我希望在浏览器查看页面源代码时，能从 `<head>` 立刻看出当前 HTML 由哪个服务提供，以便在多域名 / APISIX 转发场景下避免误判归属。

## 验收标准

1. 各入口 HTML 的 `<head>` 含 `<meta name="trae-service" content="<serviceId>" />`。
2. `content` 与 monorepo 服务目录名一致（如 `taskAiProvider`、`task2app`）。
3. `python3 db/scripts/ci/check_frontend_head_trae_service.py` 对清单内入口页全部通过。
4. DevTools 可用 `document.querySelector('meta[name="trae-service"]')?.content` 读取。

## 范围

- 入口 HTML：taskAiProvider、task2app（Vite + Django spa）、onlineServiceJS、runAll、valueStream、taskChromePlugin
- 规范：`.ai/01_project_constraints/26_frontend_head_trae_service.md`
- 门禁：`db/scripts/ci/check_frontend_head_trae_service.py` + YAML 清单

## 业务意图 → 事件对照

**无对应事件**：纯前端标识 / 可观测性治理，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| HTML head 标明提供页面的服务 | — | — | — | 纯前端标识，无服务端业务状态变更 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-17 | 初版 |
