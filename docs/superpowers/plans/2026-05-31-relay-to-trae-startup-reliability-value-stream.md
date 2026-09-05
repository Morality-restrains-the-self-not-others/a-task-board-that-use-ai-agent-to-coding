# Value Stream: relayToTrae 启动可靠性（统一版）

> 设计：`docs/superpowers/specs/2026-05-31-relay-to-trae-startup-reliability-design.md`

## Related Value Streams

- **2026-05-29-relay-to-trae-startup-reliability**：扩展 — Increment 1–3 已交付（status-push、dispatch 503、reachability 重试、前端 Vitest）
- **2026-05-27-relay-bootstrap-gitlab-clone-auth**：依赖 — 克隆 HTTP Basic 已修复
- **online-service-two-phase-bootstrap**（planned）：长期 — 本流不激活

## Value Summary

relay 直启后容器 listen、可达性注册、post-listen 引导（含 `service_config.yaml`）全链路可完成，失败可诊断且有关键路径单测锁定。

## End-to-End Flow

[启动] → go_relay → onlineServiceJS listen → register-reachability → post-listen（详情/克隆/feature-params/写配置）→ status-push SSE → [Trae job 可跑]

## Value Increments

### Increment 4: post-listen Agent 配置物化回归（本迭代）

**Value：** 消除 `configFilePath` 类 ReferenceError；引导写配置可单测验证  
**Scope：** `materializeAgentConfigFile` + `bootstrap.agentConfig.test.mjs` + `runBootstrapAfterListen` 调用  
**Depends on：** Increment 1–3（已交付）  
**Test：** `trae-agent/onlineServiceJS/src/bootstrap.agentConfig.test.mjs`

### Increment 5: 验收冒烟（手工）

**Value：** 任务 `848546827193511936` relay 直启日志含「任务引导完成」  
**Scope：** 手工 + 既有 pytest/go test 套件  
**Depends on：** Increment 4
