# 测试意图：Relay status-push Cloud converge

## 对应功能意图

`relay_status_push_cloud_converge.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `TestConvergeRelayStatusPushByScopeUpdatesSession` | phase→running，last_status_seq 更新 |
| T2 | `TestConvergeRelayStatusPushByScopeMissingSession` | `status_converge_session_missing`，不 panic |
| T3 | `TestHandleRelayStatusPushReturnsAckAndConvergesLocally` | 立即 200 ack；异步 SSE publish + Redis converge |
| T4 | `TestHandleRelayStatusPushAckWithoutSessionStill200` | 无 session 仍 200 ack |
| T5 | Django `relay-status-push-effects` URL | 404（compat 已删）；SSE/converge 改测 service 直调或 Go |

## 命令

```bash
cd taskCloudService && go test ./src/ -count=1 -run 'Relay|Inbound|Converge'
cd task2app/Saas_project && pytest tests/test_relay_to_trae_status.py tests/test_task_agent_support_internal_dispatch.py -q
```
