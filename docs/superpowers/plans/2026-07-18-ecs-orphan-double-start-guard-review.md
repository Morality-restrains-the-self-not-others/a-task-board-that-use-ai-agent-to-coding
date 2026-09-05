# Review：ECS 二次启动孤儿防护

- **日期**: 2026-07-18
- **结论**: Pass（无 critical）

## 检查

| 项 | 结果 |
|----|------|
| start 前 supersede 旧 instance | ✅ `supersedeExistingInstance` |
| ClearAfterStop 按 instance_id 收口 | ✅ 保留更新绑定 |
| InstanceName 对账回收 | ✅ `reconcileOrphanInstancesByName` |
| 单测 | ✅ Clear/Orphan/Supersede |
| 日志 | ✅ supersede_* / orphan_reconcile_* |
| 全量 taskCloudService 套件 | ⚠️ 基线已有 403 污染失败（与本变更无关） |

## Intent→Event

无新增事件；start 前同步 StopVM，STOPPED 路径传 `instance_id`。
