# Plan: taskAIEndPoint Phase 1

> Design: `docs/superpowers/specs/2026-05-30-task-ai-endpoint-budget-gateway-design.md`

## Goal

Go 服务 taskAIEndPoint（8013）：全部 sub-token LLM 请求经网关；budget_enabled 端点权威预算 gate；支持 stream=true。

## Increments

### Inc 1 — 服务骨架 + runAll
- [x] `taskAIEndPoint/` Go 项目（health、config、port_config）
- [x] `port_config.json` + `runAll.yaml` 条目

### Inc 2 — Django internal API
- [x] routes / upstream-credentials / budget reserve-or-deny / commit-usage
- [x] internal secret 鉴权（同 taskAgentSupport 模式）

### Inc 3 — 代理 + 流式
- [x] OpenAI-compatible `chat/completions` 转发
- [x] SSE stream 透传 + 结束后 commit usage
- [ ] sub-token derive（读 SubTokenProvider）— Phase 1 暂用 master key 转发

### Inc 4 — Bootstrap 契约
- [x] sub-token provider env 改为 proxy base_url + TASK_LLM_PROXY_TOKEN
- [x] 启用 taskAIEndPoint 时不下发 TASK_LLM_BUDGET_POLICY

### Inc 5 — 测试
- [x] Go handler 单元测试
- [x] Django internal API pytest

## Verify

```bash
cd taskAIEndPoint && ./build.sh && go test ./...
cd task2app/Saas_project && pytest tests/test_task_ai_endpoint_internal.py -q
```
