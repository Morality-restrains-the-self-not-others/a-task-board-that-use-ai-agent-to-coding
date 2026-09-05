# Value Stream: 修复预览环境变量按钮无响应 (Bug Fix)

> 设计文档: `docs/designs/fix-env-preview-button-no-response.md`
> 类型: Bug Fix (Repair)

## Value Summary

修复任务详情「直接启动」面板中点击「预览环境变量」按钮无响应的问题。用户可正常预览个人配置对应的环境变量。

## Related Value Streams

- **[feature-params-source-selector](2026-07-01-feature-params-source-selector-value-stream.md)**: **修复** — 该流 Increment 1 的后端预览端点正确挂在 `cloud/compute/feature-params-env-preview/`，但 Increment 2 的前端 `fetchEnvPreview()` 错误地通过 `relayToTraeApiUrl()` 构造 URL，注入了多余的 `relay-to-trae/` 段导致 404。

## Root Cause

`ServerConfig.logic.vue` 中 `fetchEnvPreview()` 使用 `relayToTraeApiUrl()` 构造 URL:
- 前端构造: `.../cloud/compute/relay-to-trae/feature-params-env-preview/`
- 后端路由: `.../cloud/compute/feature-params-env-preview/`
- `relay-to-trae/` 段不在 `CloudComputeViewSet` 路由下 → 404 → 静默失败

## Value Increments

### Increment 1: 修复 URL 构造 + 增加错误反馈 (单增量)

**Scope:**
- `ServerConfig.logic.vue`: `fetchEnvPreview()` 绕过 `relayToTraeApiUrl()`，直接用上下文 ID 拼接 URL
- 增加非 ok 响应的 `console.error` 日志

**Fields:** 无新增

---

*本次为纯 Bug Fix，不新增 value stream YAML 条目。*
