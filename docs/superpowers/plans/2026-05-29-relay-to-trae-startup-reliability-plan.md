# Plan: relayToTrae 启动可靠性

> Design: `docs/superpowers/specs/2026-05-29-relay-to-trae-startup-reliability-design.md`

## Tasks

- [x] **T1** taskAgentSupport: 修复 status-push 路径 `len>=10` + `handlers_test.go`
- [x] **T2** internal_dispatch: OperationalError→503 RELAY_DOWNSTREAM_BUSY + pytest
- [x] **T3** reachability.mjs: register-reachability 503 退避重试
- [x] **T4** 前端 mergeRelayToTraeLogLines 清理/刷新语义 + Vitest
- [ ] **T5** 验收: `go test ./...` taskAgentSupport; `pytest test_task_agent_support_internal.py`; relay 直启手工冒烟

## Verify

```bash
cd taskAgentSupport/src && go test ./...
cd task2app/Saas_project && pytest tests/test_task_agent_support_internal.py -q
cd task2app/front_project/app && npm run test -- src/utils/relayToTraeUtils.test.js
```
