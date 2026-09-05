# 测试意图：FAQ 联系邮箱从 conf 读取

## 对应功能意图

`docs/intents/frontend/faq_contact_email_from_conf.intent.md`

## 测试点

| ID | 场景 | 期望 | 层级 |
|----|------|------|------|
| T1 | 多处 `{{contactEmail}}` | 全部替换为给定邮箱 | Vitest |
| T2 | 无占位符 | 原文返回 | Vitest |
| T3 | 有占位符且邮箱为空 | 抛错含 contactEmail | Vitest |
| T4 | 挂载 Faq.vue | 正文含 `VITE_CONTACT_EMAIL`、无 `{{contactEmail}}` | Vitest |
| T5 | `src/faq/*.md` 原文 | 有占位符、无裸邮箱 | Vitest |
| T6 | `vite.config.js` 源码 | 含 `VITE_CONTACT_EMAIL` 与 `contactEmail` | Vitest |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-30 | 初版 |
