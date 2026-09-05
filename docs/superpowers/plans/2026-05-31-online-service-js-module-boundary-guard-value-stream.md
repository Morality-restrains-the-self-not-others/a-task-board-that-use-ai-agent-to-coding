# Value Stream: onlineServiceJS 模块边界回归防护

> 设计：`docs/superpowers/specs/2026-05-31-online-service-js-module-boundary-guard-design.md`

## Related Value Streams

- **2026-05-31-relay-to-trae-startup-reliability**：扩展 — Increment 4 已覆盖 `configFilePath`；本流覆盖 `bootstrapCloneLayerId` + 删层路径
- **task-detail-repo-clone-credentials-decoupling** / `decoupling-regression-guard`（planned）：follow-up 审计 error_code

## Value Summary

容器删层与层图 SSE 镜像路径在 CI 单测中可验证，避免 `ReferenceError` 泄漏到 Django 502。

## End-to-End Flow

[任务详情删层] → Django `container-layer-delete` → `DELETE /api/layers/:id` → `deleteLayerAndMirrorToSaas` → `mirrorLayerGraphToTaskCloudSSE` → [layer-graph-push]

## Value Increments

### Increment 1: bootstrap 绑定烟雾单测（本迭代）

**Value：** 合并前捕获 `jobsRuntime` 未导入 `bootstrapCloneLayerId`  
**Scope：** `jobsRuntime.bootstrapBinding.test.mjs` + `test:unit` 登记  
**Depends on：** `bootstrapCloneLayerId` import 热修  
**Test：** `npm run test:unit`（onlineServiceJS）

### Increment 2: e2e（可选，不阻塞 unit）

**Value：** 删层后 mock task cloud 收到 layer-graph-push  
**Scope：** `e2e/layer-graph-push-on-layer-delete.api.spec.mjs`  
**Depends on：** Increment 1
