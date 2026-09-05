# Review：创建任务可选字段显隐

**日期**: 2026-07-18

## 结论：可交付

| 检查 | 结果 |
|------|------|
| Go tests CreateTaskFieldSettings* | ✅ pass |
| Vitest createTaskFieldSettings + CreateTaskModal | ✅ 29 pass |
| OpenAPI path/schema | ✅ |
| 默认全 true 兼容 | ✅ |
| feature_params 隐藏跳过门禁 | ✅ |
| 日志 + traceId | ✅ handlers |
| MQ 事件 | 书面例外（配置 CRUD） |
| 权限 | 对齐 options，无新增 RBAC |

## 非阻塞

- Chrome 插件未对齐字段显隐 → OPT
- Playwright E2E 未跑（单元已覆盖核心路径）
