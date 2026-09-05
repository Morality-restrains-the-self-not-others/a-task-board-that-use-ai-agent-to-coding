# taskEvents Handler Go 化 — G4.5 / G5 验收清单

> 设计：§5.1 `2026-06-01-taskevents-handlers-relocation-design.md` v3.1  
> 端口：`2026-06-01-taskevents-port-conflict-resolution-design.md`（**18020–18037**）

## 状态：**已完成**

| 阶段 | 状态 |
|------|------|
| G0–G4 Go handler + 18 intent 二进制 | ✅ |
| v4 intent 布局 + company D2 fan-out | ✅ |
| 端口治理 18020–18037 | ✅ |
| G4.5 `g45_verify.sh` | ✅ |
| G5 删 Python handlers / dispatch / 域级 cmd | ✅ |
| runAll `domain-events-intents` | ✅ |
| value-stream 更新 | ✅（g45 step 使用 runAll intent provider 名） |
| DOMAIN_EVENTS.md | ✅ v4 intent + 18020–18037 |

## P0 — 注册链

- [x] runAll：`domain-events-intents`（18 intent，18020–18037）
- [x] Go integration：`TestRegistrationChainUserToWorkspace`
- [x] Go integration：`TestCompanyCreatedFanOutIntents`
- [ ] Playwright 全栈 UI 注册（可选）

## P1 — billing / sse / 通知

- [x] Go integration：billing Redis 往返
- [x] Go integration：SSE Redis publish
- [x] inactive intent（email/invitation/user_activated welcome）默认 disabled，runAll 可单独启

## 横切

- [x] `go test -tags=integration ./integration/...`
- [x] `scripts/check_event_health.sh`（18020–18037）
- [x] `scripts/g45_verify.sh`
- [x] `run.sh port_preflight`

## G5

- [x] 删除 `core/kafka/handlers/`
- [x] 删除域级 `cmd/*`、Django `/dispatch/` 路由
- [x] `port_config.json` 嵌套 intents，删 8005–8012

## 运维

```bash
cd taskEvents
bash run.sh stop all
bash run.sh start events          # 或 runAll domain-events-intents
bash scripts/check_event_health.sh
bash scripts/g45_verify.sh        # 完整门禁
```

- 勿双开多套 consumer（同一 event_type 双消费）
- 端口冲突：`bin/{event}/{intent}/config.yaml` 或 `port_config.json` overlay

## 已知缺口（非阻塞）

- `cloud_server_start_auto`：`auto_create_*` 未实现（G4 文档化 skip）
- Playwright E2E 未覆盖通知类 inactive intent
