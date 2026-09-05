# DDD: 公司切换后项目列表为空 — 修复

**跳过。** 理由：无新领域概念。变更为 `tenant_id` 上下文传播的纯技术修复——在 `/me` API 调用链中补齐 query param 传递路径，使 `get_current_workspace` 和 `get_current_company` 能正确解析当前租户上下文。不引入新实体、值对象、聚合、仓储或领域事件。
