# relayToTraeUtils.js

## 约束

- `mergeRelayToTraeLogLines` 在 `suppressedAfterClear=true` 时：
  - 空 `lines` **不得**解除抑制或清空 snapshot。
  - 同源全量/更短前缀/已知连续片段（SSE 增量回放）**不得**回填展示。
  - 仅当服务端缓冲相对 snapshot **增长**时追加新尾部；仍保持抑制，避免后续全量回填。
  - `awaitingFirstStatusAfterStart` 仍允许启动后首次全量展示。
  - 若全量缓冲**包含**当前 snapshot 为连续子序列（SSE 先推中段、随后 `/status` cursor=0）：只吸收 snapshot 之后的新尾部，**禁止**把 snapshot 前的历史前缀回填，也**禁止**把 bootstrap 段再拼一遍。
- 非抑制路径：当 snapshot 是 incoming 的连续子序列但非前缀时，以 incoming 全量为准**重建**展示（避免「已向 SaaS 注册 / clone_begin」重复）。
- 心跳行剥离规则见 `isContainerHeartbeatLogLine` / `partitionRelayToTraeLogLines`；勿把心跳行写回启动日志面板。
- `classifyBootstrapMilestoneLogLine` 识别 `BOOTSTRAP_COMPLETE` / `BOOTSTRAP_FAILED` / `BOOTSTRAP_PHASE=`（及旧文案「任务引导完成」）；启动日志面板须高亮这些行，勿改稳定前缀。
- `isTokenPersistFailedLogLine` / `classifyRelayStartupLogLine` 识别 `FAIL_PERSIST` / `TOKEN_PERSIST_FAILED` / `token-persist: FAIL`；面板须红字高亮，状态条徽章为「落盘失败」、文案「换票落盘失败」，勿与 `TOKEN_ACCESS_INVALID` 混用。
- `mapRelayToTraeStatusErrorMessage` 将 status.`error` / 启动失败 message 中的 `TOKEN_PERSIST_FAILED` 映射为启动按钮旁中文提示；`applyRelayToTraeStatusPayload` 与 `resolveRelayToTraeErrorMessage` 须走该映射。
- `resolveBootstrapStatusFromLogs` 推导状态条：`failed`（含落盘失败）> `complete` > 最近 `phase` > `idle`；失败须有独立徽章（不仅红字行）。
- `resolveRelayConsoleOpenUrl`：`mode=selected_image` 时必须用 `containerPageUrl`/`serverUrl`，禁止用侧车 `/ui/{token}` 或 task-detail SPA；host 模式仍用 `relayUiUrl`。
- `buildDefaultRelayToTraeEnvItems`：未配置 VITE 时，优先用当前页非 loopback hostname 拼 `http://{host}:8765`，避免默认 127.0.0.1 导致 register-reachability 对浏览器不可达。
