# Review：task-detail-machine-owner-hint

- **日期**: 2026-07-15
- **结论**: ✅ 通过（无 critical / important 阻断项）

## 对照计划

| 项 | 状态 |
|----|------|
| Go `attachMachineOwnerHint` 挂到 `writeServerRuntimeStatusJSON` | ✅ |
| OpenAPI `machine_owner_task_ids` / `container_running` | ✅ |
| 前端纯函数 + 组件 + 镜像卡接入 | ✅ |
| 意图无事件例外书面登记 | ✅ |

## Intent→Event 审计

纯查询展示；`025_machine_owner_hint.intent.md` 已声明无 MQ 事件例外。✅

## Log 审计

- bindings 失败：`logWarn(event=machine_owner_hint_bindings_failed …)` ✅
- 无敏感字段落日志 ✅

## 公网验证

- `/static/main-cVffA5CD.js` → 200
- `/static/TaskDetailContent.logic-BQ3D9l4q.js` → 200，含 `机器节点所属任务` / `machine-owner-hint`
- 目标页 runtime：`container_running=false`, `instance_id=null` → 提示按设计不渲染 ✅

## 非阻断建议

- 仅 mock/relay、无云 `instance_id` 时仍不展示「机器节点」归属（符合设计「机器节点=云实例绑定」）；若产品后续要覆盖本地启动，需另开意图扩展数据源。
