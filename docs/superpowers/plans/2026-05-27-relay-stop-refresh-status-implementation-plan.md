# Implementation Plan: relayToTrae 停止后刷新状态

> Design: `docs/superpowers/specs/2026-05-27-relay-stop-refresh-status-design.md`

## Task 1: Playwright 回归（Red）
- [x] stop → refresh（health 失败 + status 200 stopped）
- [x] stop → refresh（health 与 status 均失败，stopped 收敛态 preserve）

## Task 2: generation token + 可达性判定（Green）
- [x] `relayStatusFetchGeneration` 递增与 stale 丢弃
- [x] status HTTP 200 回退即 relay 在线
- [x] `shouldPreserveRelayOnlineOnProbeFailure` — stopped 收敛态 + stop 显式 preserve

## Task 3: stopped 文案（Green）
- [x] `applyRelayToTraeStatusPayload`: stopped 文案

## Task 4: 验证（Refactor）
- [x] Playwright 子集通过
