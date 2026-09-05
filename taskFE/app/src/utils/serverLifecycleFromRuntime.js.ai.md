# Companion：`serverLifecycleFromRuntime.js`

## 用途

将云 `server-runtime-status` 的 `runtime_status` 映射为任务详情生命周期 hydrate 意图，供 `updateServerStatus({ status: 'runtime_hydrate' })` 使用。

## 约束

- **只改本模块 + `serverLifecycleFromRuntime.test.js`** 来扩展云状态码映射；不要在 `ServerConfig.logic.vue` 内联 if/else 表。
- 未知/空状态返回 `null`：调用方不得仅凭「空串」覆盖 SSE；但 API **明确**「尚未创建云实例」时由 `notifyRuntimeAbsentHydrate`（`Stopped`）回落，见 `serverRuntimeAbsent.js` / `serverRuntimeHydrate.js`。
- 与 `serverLifecycleStatus.js`（码→文案）分工：本文件管「云态→标志位意图」，后者管「标志位→UI 文案」。

## 自检

```bash
cd taskFE/app && npm test -- src/utils/serverLifecycleFromRuntime.test.js
```
