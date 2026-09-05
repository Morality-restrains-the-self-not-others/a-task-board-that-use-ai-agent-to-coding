# 实施计划: relay stop stale 端点作废

> Design: `docs/superpowers/specs/2026-05-31-relay-stop-stale-container-endpoint-invalidation-design.md`

## Increment 1 任务清单（方案 D 止血）

- [x] **T1** pytest：`test_relay_stop_clears_container_reachability`
- [x] **T2** `clear_container_reachability.py`
- [x] **T3** `relay_to_trae_stop` 成功回调
- [x] **T4** `stop_vm` 挂载
- [x] **T5** `stop_mock_run_container` 挂载
- [x] **T6** pytest smoke
- [x] **T7** 相关 pytest 全绿

## Increment 2 任务清单（方案 F History 会话）

- [x] **T8** migration `0045_cloudserverconfighistory_runtime_session`
- [x] **T9** `runtime_session_lifecycle.py` open/update/close
- [x] **T10** register-reachability sync open session
- [x] **T11** relay/mock-run/VM start open session
- [x] **T12** `get_server_start_history` 返回 started_at/stopped_at/URL
- [x] **T13** pytest `test_runtime_session_history.py`

## Increment 3 任务清单（前端乐观收敛）

- [x] **T14** `onContainerReachabilityCleared` callback
- [x] **T15** `stopRelayToTrae` / `stopMockRunContainer` / stop-vm SSE 调用

## 验证

```bash
cd task2app/Saas_project && pytest tests/test_relay_stop_clears_container_endpoint.py tests/test_stop_vm_clears_container_endpoint.py tests/test_runtime_session_history.py -q
```
