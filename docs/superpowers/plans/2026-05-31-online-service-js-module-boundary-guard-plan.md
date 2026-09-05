# Plan: onlineServiceJS 模块边界回归防护

> Design: `docs/superpowers/specs/2026-05-31-online-service-js-module-boundary-guard-design.md`

## Tasks

- [x] **T1** 新增 `src/jobsRuntime.bootstrapBinding.test.mjs`（buildLayersSnapshot / mirror / delete）
- [x] **T2** `package.json` `test:unit` 登记 T1
- [x] **T3** 验收：`cd trae-agent/onlineServiceJS && npm run test:unit`（59 pass）

## Verify

```bash
cd trae-agent/onlineServiceJS && npm run test:unit
```
