# 测试意图：Relay startup session Gateway 直写

## 对应功能意图

`relay_startup_session_gateway_direct_write.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `TestUpsertRelayStartupSessionTokenInitWritesKeys` | wf + scope key 写入，phase=`token_init_succeeded` |
| T2 | `TestUpsertRelayStartupSessionStartAcceptedMerges` | 合并 request_id，phase=`start_accepted` |
| T3 | `TestHandleInternalRelayStartupSessionUpsert` | HTTP 200 + secret 校验 |
| T4 | `TestUpsertRelayStartupSessionViaCloudOnTokenInitAndStart` | Gateway token-init/start 调 Cloud upsert |
| T5 | Django `test_relay_workflow_transition_accept` | 默认 persist=false 仍 200，不依赖 Redis 写 |

## 命令

```bash
cd taskCloudService && go test ./src/ -count=1 -run 'Relay|Session'
cd taskContainerGateway && go test ./src/ -count=1 -run 'Relay|Session|Token'
```

## 变更日期

2026-07-10
