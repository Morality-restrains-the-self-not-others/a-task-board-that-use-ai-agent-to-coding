# DDD Model: runAll 链式关闭混合所有权

## 限界上下文

- **service-orchestration**（扩展）：级联计划执行时为每步选择有效 actor。

## 新增

| 类型 | 名称 | 说明 |
|------|------|------|
| Domain Service | `CascadeStepActorResolver` | initiating vs registered owner 解析 |
| Value Object | `CascadeStepActor` | 解析结果（session id 字符串） |

## 复用

- `ServiceOwnership`、`ServiceOwnershipGuardService`（单点路径不变）
- `executeLifecyclePlan`（应用层 Runner）

## 不变量

- 委托不写入 ownership 记录（非 takeover）
- 无 ownership 记录 → initiating actor
