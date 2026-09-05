# Plan: relayToTrae 启动可靠性（统一版）

> Design: `docs/superpowers/specs/2026-05-31-relay-to-trae-startup-reliability-design.md`

## Tasks

- [x] **T0**（历史）status-push / internal_dispatch 503 / reachability 重试 / 前端 Vitest
- [x] **T1** `bootstrap.mjs` 导入 `configFilePath`
- [x] **T2** `materializeAgentConfigFile` in `featureParamsEnvToYaml.mjs`
- [x] **T3** `runBootstrapAfterListen` 调用 `materializeAgentConfigFile`
- [x] **T4** `bootstrap.agentConfig.test.mjs` + `package.json` test:unit 登记
- [x] **T5** 验收：`npm run test:unit`（onlineServiceJS，58 pass）

## Verify

```bash
cd trae-agent/onlineServiceJS && npm run test:unit
```
