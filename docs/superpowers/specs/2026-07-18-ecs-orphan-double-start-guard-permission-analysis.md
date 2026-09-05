# 权限分析：ECS 二次启动孤儿防护

- **设计**: `2026-07-18-ecs-orphan-double-start-guard-design.md`
- **结论**: 无新公网 endpoint；沿用现有 internal 路由与云授权。

| 改动点 | 调用方 | 鉴权 | 数据范围 |
|--------|--------|------|----------|
| start 前 StopVM | taskEvents cloudserverstarted | 事件载荷 `authorization_id` → CloudAuthorization | 仅该 task 旧 `instance_id` |
| clear-after-stop + instance_id | taskEvents → taskCloudService internal | internal secret（既有） | company/workspace/task + 可选 instance |
| InstanceName 对账 | workspace-runtime-indicators / machine-summary 触发 | 租户 workspace 上下文（既有） | workspace 内 CSC 任务 |

无新增角色；不对未授权租户跨账号 DeleteInstance。
