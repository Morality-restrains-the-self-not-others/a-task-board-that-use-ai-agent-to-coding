# 功能意图：FAQ 联系邮箱从 conf 读取

## 用户故事

作为平台运营，我希望 `/faq/` 上展示的联系邮箱来自 `conf/frontend/vue/config.yaml` 的 `contactEmail`，这样改配置即可换地址，不必改 FAQ 文案源文件。

## 验收标准

1. `https://www.daydaymoney.com/faq/`（及同源 `/faq/`）正文出现 `contactEmail` 的当前值，不出现 `{{contactEmail}}`。
2. `taskFE/app/src/faq/*.md` 不含裸邮箱，使用 `{{contactEmail}}` 占位符。
3. 构建期 Vite 从 `conf-read.py snapshot-json` 的 `vue.contactEmail` 注入 `import.meta.env.VITE_CONTACT_EMAIL`。
4. 占位符存在但配置为空时渲染失败（不静默兜底）。

## 范围

- `conf/frontend/vue/config.yaml`（SSOT）
- `taskFE/app/vite.config.js` 注入
- `Faq.vue` + `applyFaqPlaceholders` + `src/faq/*.md`

## 业务意图 → 事件对照

**无对应事件**：纯前端展示配置值，无服务端业务状态变更。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| FAQ 联系邮箱从 conf 读取 | — | — | — | 纯前端展示/交互或设计治理，无服务端业务状态变更意图 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-30 | 初版 |
