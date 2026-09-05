# [运行时] 服务器已停止仍提示「服务器已在运行」

## 现象

任务详情硬件配置区 `data-testid="start-server-disabled-reason"` 仍显示「服务器已在运行，如需重新启动请先停止」，但机器/云态实际已停止（含 Mock）。

## 根因

1. `updateServerStatus` 对 SSE `success` 仅用 `message.includes('停止虚拟机成功')` 识别停机；Mock 成功文案「Mock 实例已停止」落入 else，把 `isServerRunning` 置为 `true`。
2. `ServerConfig` 的 `statusMessage` watch 同样未识别 Mock 停机文案，停止后不刷新 `server-runtime-status`。
3. 失配自愈 `shouldRematchRuntimeHydrate` 仅覆盖「本地未 running + 云 Running」，未覆盖「本地仍 running + 云已 Stopped」。
4. 禁用原因解析在云态已为 Stopped/Released 时仍优先相信陈旧 `isServerRunning`。

## 修复

- 统一使用 `isCloudServerStopSuccessMessage`（`stopVmSseMessage.js`）
- `shouldRematchRuntimeHydrate` 双向失配
- `resolve*StartDisabledReason`：云非服务态时不阻断为「已在运行」
- `stopServer` HTTP 200 后立即 `fetchServerRuntimeStatus`

## 关联

- 意图：`task2app/docs/intents/frontend/task_detail/029_runtime_hydrate_server_lifecycle.intent.md`
- 既有：`16_stop_server_idempotency_company_id_skip.md`（Mock 停机 SSE 文案约定）
