# 意图：厂商门户可查看容器 → SaaS 接口 Skill

## 背景与目标

容器镜像对接 SaaS 的 inbound 契约权威文档是 `docs/skills/saas-container/saas-machine-container.md`。厂商在 `https://provider.daydaymoney.com/` 发布镜像时需要直接打开该文件，而不是在仓库里翻路径。

目标：镜像市场顶栏提供真实 `<a href>`，打开同源 `GET /saas-machine-container.md`，正文与仓库 SSOT 一致。

## 范围与边界

- 范围内：`taskAiProvider` 公开提供该 markdown；`App.vue` 顶栏链接；Vite 开发态同样可打开。
- 范围外：不改接口鉴权、不渲染 Markdown 为富文本、不新增 Kafka 事件。

## 约束与风险

- 导航须真实 `href`，禁止 `@click.prevent` + `router.push`。
- 文件 SSOT 仍在 monorepo `docs/skills/saas-container/saas-machine-container.md`，禁止另维护一份会漂移的副本作为权威。
- 公开只读；`Cache-Control: no-cache`，避免网关缓存旧契约。
- 请求失败若在 SPA 内展示，须带 `data-traceId`；本链接打开的是静态正文，不走错误 toast。

## 验收标准

1. `https://provider.daydaymoney.com/` 顶栏可见「容器→SaaS 接口」链接，`href` 为 `/saas-machine-container.md`。
2. `GET /saas-machine-container.md` 返回 200，`Content-Type` 含 `text/plain`，正文含 `SaaS Machine Container Skill`。
3. 未登录也可打开（公开文档）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 查看容器→SaaS skill 文档 | — | — | — | — | 纯只读静态文档，无服务端事实变更 |

## 实施计划

1. Go：按 monorepo 根解析 SSOT 并注册 `GET /saas-machine-container.md`。
2. 前端：顶栏真实 `<a href="/saas-machine-container.md">`。
3. Vite：dev/preview 中间件与 build 写入 dist，保证本地也能打开。

## 变更记录

- 2026-08-16：厂商门户需要可点击查看 `saas-machine-container.md`。
