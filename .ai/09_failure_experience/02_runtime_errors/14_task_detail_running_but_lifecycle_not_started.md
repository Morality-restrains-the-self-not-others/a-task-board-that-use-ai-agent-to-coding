# [运行时] 任务详情运行中但「服务器启动状态」显示未启动

## 现象

任务详情页「服务器运行状态」为运行中（云 Describe `Running`），同页「服务器启动状态」仍显示「未启动」。

典型路径：机器已启动后刷新页面 / 冷打开任务详情 URL。

## 环境与上下文

- 「服务器运行状态」：`GET .../cloud/compute/server-runtime-status/` → `runtime_status`
- 「服务器启动状态」：`resolveServerLifecycleLabel({ isServerRunning, isServerStarting, serverStatus })`（意图 027）
- `isServerRunning` 原先仅由本会话 SSE `success`（非停机文案）置位

## 根因

冷打开时 SSE 未必重放「启动成功」事件，`isServerRunning` 保持 `false`、`serverStatus` 为空或不可映射码 → 生命周期默认「未启动」；与已拉取的云运行态不一致。

## 修复

- `serverLifecycleFromRuntime.js`：云 `runtime_status` → hydrate 意图
- `updateServerStatus` 处理 `runtime_hydrate`（不写启动日志）；写入 `cloudRuntimeStatus`
- `ServerConfig.fetchServerRuntimeStatus` 成功后触发 hydrate
- `ServerConfig` 失配自愈：云 Running 但父 `isServerRunning=false` 时重新 hydrate
- 展示层 `resolveServerLifecycleLabel({ runtimeStatus })`：标志位未回填时仍显示「已启动」
- 层图门控确认 `running` 时 `onServerRuntimeServing`
- `markServerRuntimeServing` 清除 `serverRuntimeNotServing`

## 预防

- 新增/展示依赖「已启动」的 UI 时，须考虑冷打开：以云 runtime 或等价持久态回填，不得仅依赖本会话 SSE
- 意图 027 分行后，回归须覆盖「运行中 + 刷新」组合（见意图 029）
- **半落地陷阱**：仅有 `updateServerStatus('runtime_hydrate')` handler 不够；`fetchServerRuntimeStatus` 成功后必须主动调用，且面板须传入 `runtimeStatus` 作为展示回退（2026-07-18 复现：handler 在、接线丢）
