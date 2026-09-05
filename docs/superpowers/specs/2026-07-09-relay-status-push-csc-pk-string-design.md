# 设计：relay status-push 兼容 Go 侧 `csc_*` 字符串主键

- 日期：2026-07-09
- 状态：adopted（goal-mode 自动采用）
- 相关事故：`task_12675381068363715869` 启动后 status push HTTP 500 `INTERNAL_DISPATCH_ERROR`，随后 missed ACK 注销；约 1h 后出现 401「无效的 access_token」

## 1. 问题陈述

任务详情页 `?relayToTrae=true` 点击启动后：

1. token-exchange / open_runtime_session / 容器引导与克隆可成功；
2. `go_relayToTrae` → `relay-status-push` 连续 500；
3. 5 次 missed ACK 后注销任务，前端 SSE 状态丢失；
4. 大量 `/api/saas-heartbeat-probe` 200（与 status push **无关**，属双向心跳正常行为）；
5. 后续偶发 401「无效的 access_token」。

用户假设「启动未创建新 token 记录」对主事故 **不成立**：03:52 已成功 `open_runtime_session`（`csc_1783569134419`），token 校验通过后在构建快照时崩溃。

## 2. 根因

| 现象 | 根因 |
|------|------|
| HTTP 500 / `INTERNAL_DISPATCH_ERROR` | `relay_status_push_cfg_cache._cfg_snapshot_from_model` 对 `cfg.pk` 做 `int()`；Go 迁移后 `CloudServerConfig` 主键为字符串 `csc_<ms>`，触发 `ValueError`，被 `internal_dispatch` 吞成 500 |
| 连续 missed ACK → unregister | status push 失败不计 ACK，达阈值 5 注销 |
| 后续 401 | token TTL≈1h；注销后仍用过期/无效 token 推送，或 Go validate 失败 |
| heartbeat-probe 刷屏 | 预期行为（默认约 20s 一次），非故障 |

`TRAE_SKIP_CONTAINER_TOKEN_EXCHANGE` 仅跳过容器内二次换票，**不是**根因。

## 3. 方案（已选）

**A. 快照主键一律 `str(cfg.pk)`**（已落地于源码 + 单测）

- 符合 `.ai/03_technical_implementation/11_id_field_string_transit.md`：进程内 ID 保持 string。
- `StatusPushCfgSnapshot.pk: str`。

**B. go_relay：HTTP 401 立即注销，避免过期 token 刷屏**

- 鉴权失败不可靠重试；与「5 次网络/5xx missed ACK」区分。

**C. E2E：任务详情启动 → 无 status-push 500/401 → 出现「任务引导完成」/克隆完成信号**

## 4. 非目标

- 不改 heartbeat-probe 频率（属正常探测）。
- 不在本轮改 token TTL。
- 不把克隆引导绑定到 status push 成功（二者独立）。

## 5. 验收

- [ ] `_cfg_snapshot_from_model` 对 `csc_*` 不抛异常
- [ ] status-push 对有效 token 返回 200 + `ack`
- [ ] go_relay 遇 401 立即 unregister，日志明确
- [ ] Playwright：指定任务页启动后可见引导/克隆完成，且日志无 `INTERNAL_DISPATCH_ERROR` / `无效的 access_token`

## 6. 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-09 | 初版：定位 `int(csc_*)`；加固 401；扩展 E2E |
