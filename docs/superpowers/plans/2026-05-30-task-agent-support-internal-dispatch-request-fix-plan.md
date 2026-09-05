# Plan: taskAgentSupport internal_dispatch Request 修复

> Design: `docs/superpowers/specs/2026-05-30-task-agent-support-internal-dispatch-request-fix-design.md`

## Tasks

- [x] **T1** `internal_dispatch.py`：`_build_inbound_http_request` 返回 HttpRequest；`view_func(http_request, ...)`
- [x] **T2** `internal_dispatch.py`：`OperationalError` locked → 503 `RELAY_DOWNSTREAM_BUSY`；其它异常 + `error_code`
- [x] **T3** `tests/test_task_agent_support_internal_dispatch.py`：exchange-refresh 不返回 500 internal error
- [x] **T4** 验收：`pytest tests/test_task_agent_support_internal_dispatch.py tests/test_task_agent_support_internal.py -q`

## Verify

```bash
cd task2app/Saas_project && pytest tests/test_task_agent_support_internal_dispatch.py tests/test_task_agent_support_internal.py -q
```
