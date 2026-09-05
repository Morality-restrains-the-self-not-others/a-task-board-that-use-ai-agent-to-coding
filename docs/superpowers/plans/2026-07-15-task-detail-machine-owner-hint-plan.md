# 实施计划：task-detail-machine-owner-hint

## Tasks

- [x] T1: Go — `attachMachineOwnerHint(resp, cfg, viewerTaskID)` + 单测
- [x] T2: OpenAPI — `ServerRuntimeStatusResponse` 增加字段
- [x] T3: 前端纯函数 `machineOwnerHint.js` + 单测
- [x] T4: 组件 `ServerConfigMachineOwnerHint.vue` + 单测
- [x] T5: 接入 `ServerConfig.logic.vue` 镜像卡
- [x] T6: 意图文档 + value-stream 测试点（已写入 VS 文档）
- [x] T7: `runall-lifecycle.sh build`（collectstatic）— 公网 `/static/TaskDetailContent.logic-BQ3D9l4q.js` 含「机器节点所属任务」且 200
- [x] T8: Review / Ship — 已合入 `task2app`/`taskCloudService` main；本轮 goal 复核通过

## 完成定义

S1–S6 全部满足；Go + 前端单测通过。

## 本轮 goal 复核（2026-07-15 19:50）

| 标准 | 结果 |
|------|------|
| S1 未运行不展示 | ✅ 目标任务 `instance_id=null`，DOM 无 `machine-owner-hint` |
| S2/S3/S4 运行时展示 | ✅ 单测覆盖；公网包已含组件文案 |
| S5 无新 Python HTTP + OpenAPI | ✅ |
| S6 意图文档 | ✅ `docs/intents/frontend/task_detail/025_*` |
