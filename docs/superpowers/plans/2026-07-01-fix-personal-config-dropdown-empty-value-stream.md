# Value Stream: 修复个人配置下拉为空 (Bug Fix)

> 设计文档: `docs/designs/fix-personal-config-dropdown-empty.md`
> 类型: Bug Fix (Repair)

## Value Summary

修复任务详情「直接启动」面板中，选择「个人配置」来源后二级下拉始终为空的 Bug。用户可正常选择已创建的个人配置。

## Related Value Streams

- **[feature-params-source-selector](2026-07-01-feature-params-source-selector-value-stream.md)**: **修复** — 该流的 Increment 2 (前端来源选择器 + 预览集成) 中 `ServerConfig.logic.vue` 的 `fetchPersonalConfigs()` 存在 API 响应字段名不匹配 Bug。本修复将 `data.results` 更正为 `data.configs`，并在 `initFeatureParamsSource()` 中补充初始加载时的自动 fetch。
- **[feature-params-hierarchy](2026-06-30-feature-params-hierarchy-value-stream.md)**: **独立** — 该流提供后端 API (`GET /api/personal/feature-params-configs/`)，响应格式正确无需修改。

## End-to-End Flow

```text
[用户] 进入任务详情 → 点击"直接启动" → 功能参数来源选择「个人配置」
  → fetchPersonalConfigs() 调用 GET /api/personal/feature-params-configs/
  → 解析响应 data.configs (修复前: 误读 data.results → 永远为 [])
  → 个人配置下拉填充可用配置
  → 用户选择具体配置 → 启动容器
```

## Root Cause

`ServerConfig.logic.vue:475` 中 API 响应解析字段名错误:

```js
// ❌ 修复前: data.results 不存在，personalConfigs 永远是 []
personalConfigs.value = Array.isArray(data) ? data : (data.results || [])

// ✅ 修复后: 正确读取 data.configs
personalConfigs.value = data.configs || []
```

后端返回 `{"configs": [...]}`，前端却读 `data.results`。同一 API 在 `PersonalFeatureParamsConfigs.vue:189` 正确使用了 `data.configs`。

## Value Increments

### Increment 1: 修复响应解析 + 初始加载补充 (单增量)

**Value to user:** 个人配置下拉正常显示已创建的配置

**Scope:**
- `ServerConfig.logic.vue:475`: `data.results` → `data.configs`
- `ServerConfig.logic.vue:458-465`: `initFeatureParamsSource()` 中 source=personal 时调用 `fetchPersonalConfigs()`

**Depends on:** feature-params-source-selector Increment 2 (已交付，有 Bug)

**Fields:** 无新增字段

---

*本次为纯 Bug Fix，不新增 value stream YAML 条目。修复逻辑属于已有 `feature-params-source-selector` 流的 Increment 2 修正。*
