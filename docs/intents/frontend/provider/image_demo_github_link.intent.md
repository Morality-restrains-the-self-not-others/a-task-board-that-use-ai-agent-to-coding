# 意图：厂商门户顶栏提供镜像 Demo GitHub 链接

## 背景与目标

厂商在 `https://provider.daydaymoney.com/` 发布/对接容器镜像时，需要一份可运行的参考实现。权威 Demo 仓库是 `https://github.com/task2money/trae-agent`。

目标：镜像市场顶栏提供真实 `<a href>`，文案「镜像Demo」，打开该 GitHub 仓库。

## 范围与边界

- 范围内：`taskAiProvider/frontend` 顶栏导航；未登录也可看见并点击。
- 范围外：不镜像仓库内容、不新增后端路由、不新增 Kafka 事件、不改 SSO/审核流。

## 约束与风险

- 导航须真实 `href`，禁止 `@click.prevent` + `router.push`。
- 外链须 `target="_blank"` 且 `rel="noopener noreferrer"`，避免门户页被替换、降低 `window.opener` 风险。
- 链接为公开 GitHub URL，不携带 token/密钥。

## 验收标准

1. `https://provider.daydaymoney.com/` 顶栏可见「镜像Demo」链接。
2. 该链接 `href` 为 `https://github.com/task2money/trae-agent`。
3. 源码无 `@click.prevent`；未登录也可打开。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 打开镜像 Demo GitHub | — | — | — | — | 纯前端外链导航，无服务端事实变更 |

## 实施计划

1. 常量模块导出 href/文案。
2. `App.vue` 顶栏真实 `<a>`。
3. 前端单测读常量与 `App.vue` 源码断言。

## 变更记录

- 2026-08-17：厂商门户需要可点击打开 trae-agent Demo 仓库。
