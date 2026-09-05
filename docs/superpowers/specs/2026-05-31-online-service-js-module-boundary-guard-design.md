# onlineServiceJS 模块边界与回归防护

**日期：** 2026-05-31  
**状态：** 已审批（0-auto-flow）  
**关联：** `2026-05-31-relay-to-trae-startup-reliability-design.md`（同类 ReferenceError：`configFilePath`）

## 背景

任务详情 `container-layer-delete` 转发至容器 `DELETE /api/layers/{id}`。删除成功后需调用 `mirrorLayerGraphToTaskCloudSSE()` 向 SaaS 推送层图。`jobsRuntime.mjs` 使用该路径时引用了 `bootstrapCloneLayerId` 但未从 `bootstrap.mjs` 导入，运行时抛出 `ReferenceError: bootstrapCloneLayerId is not defined`，Django 包装为 **502**。

与 post-listen 引导中 `configFilePath is not defined` 属同一故障模式：ESM 无编译期符号检查，模块级可变导出跨文件引用易遗漏 import。

## 目标

1. **CI 回归：** `npm run test:unit` 覆盖 `mirrorLayerGraphToTaskCloudSSE` 与 `deleteLayerAndMirrorToSaas`，确保无裸符号 ReferenceError。
2. **契约文档化：** 明确 `jobsRuntime` ↔ `bootstrap` 的 import 边界。
3. **热修锁定：** `bootstrapCloneLayerId` import 已合入，本迭代以测试锁定为主。

## 非目标

- 不重构 `bootstrapCloneLayerId` 为注入式状态。
- 不将 Playwright e2e 并入 `test:unit`。
- 不改变 Django 502 / 容器 400 错误包装语义。

## 模块边界

| 模块 | 职责 | 对外符号 |
|------|------|----------|
| `bootstrap.mjs` | 引导运行时状态 | `bootstrapCloneLayerId`, `bootstrapRegisterCloneJob`, … |
| `jobsRuntime.mjs` | 任务/层队列、层图快照 | `mirrorLayerGraphToTaskCloudSSE`, `deleteLayerAndMirrorToSaas`, `buildLayersSnapshot` |
| `server.mjs` | HTTP 边界 | `DELETE /api/layers/:id` → `deleteLayerAndMirrorToSaas` |

**契约：** `jobsRuntime.mjs` 正文若使用 `bootstrapCloneLayerId`，必须从 `./bootstrap.mjs` 显式 import。

## 测试设计

| 用例 | 断言 |
|------|------|
| `buildLayersSnapshot(bootstrapCloneLayerId)` | 返回含 `layers` 数组的对象 |
| `mirrorLayerGraphToTaskCloudSSE()` | 无 TaskApiEndPoint 时 publish 早退，不抛 ReferenceError |
| `deleteLayerAndMirrorToSaas(childId)` | 子层目录删除，不抛 ReferenceError |

文件：`trae-agent/onlineServiceJS/src/jobsRuntime.bootstrapBinding.test.mjs`，登记于 `package.json` `test:unit`。

## 价值流影响

| 流 | 影响 |
|----|------|
| `task-detail-repo-clone-credentials-decoupling` / `decoupling-regression-guard` | 补充容器侧 ReferenceError 回归 |
| `2026-05-31-relay-to-trae-startup-reliability` | 并列横切防护，不合并 spec |

**字段：** 无新数据库字段。

## 领域概念（供 DDD）

| 概念 | 边界上下文 |
|------|------------|
| BootstrapRuntimeState | 容器运行时 |
| LayerGraphSnapshot | 任务协作 / 容器 |
| LayerDeletion | 容器 |

## 验收标准

1. `jobsRuntime.bootstrapBinding.test.mjs` 通过。
2. `cd trae-agent/onlineServiceJS && npm run test:unit` 全绿。
3. 故意移除 import 时单测失败。

## 实施切片

| 切片 | 内容 |
|------|------|
| S0 | 确认 `bootstrapCloneLayerId` import |
| S1 | 新增单测 + `test:unit` 登记 |
| S2 | 价值流 / NFR / 计划文档 |
