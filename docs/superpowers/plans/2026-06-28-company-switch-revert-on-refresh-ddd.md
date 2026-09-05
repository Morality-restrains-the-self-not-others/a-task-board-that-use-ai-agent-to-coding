# DDD 领域建模: 公司切换后页面刷新回退 — 修复

> 输入:
> - 设计文档: `docs/specs/company-switch-revert-on-refresh-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-nfr-clarification.md`

## 跳过声明

**DDD 领域建模已跳过。** 理由：

| 判断条件 | 评估 |
|----------|------|
| 新限界上下文？ | 无 — 变更在现有 `company-management` 和 `taskFE` 范围内 |
| 新实体/值对象？ | 无 — `CompanyMember`、`UserSerializer.current_company` 均为已有概念 |
| 新聚合/聚合根？ | 无 — 不改变数据模型 |
| 新领域事件？ | 无 — 无跨上下文通信需求 |
| 新仓储接口？ | 无 — 不新增持久化操作 |
| 变更性质 | 前端状态解析策略变更：`currentTenant` 来源从 API-only → URL-first + API-fallback |

此修复的核心动作是调整 `Navbar.logic.vue` 和 `Sidebar.vue` 中 `currentTenant` 的解析优先级：
- **before**: `currentTenant = api.current_company || api.companies[0]`
- **after**: `currentTenant = route.params.tenant (if valid) || api.current_company || api.companies[0]`

不涉及任何后端数据模型、业务规则或领域概念的变更。可选的后端 `get_current_company` 增强（增加 `tenant_id` aware filter）是已有 serializer 方法的行为微调，不构成新的领域概念。

## 无需新增的领域产物

- `domain/entities/` — 无新实体
- `domain/value_objects/` — 无新值对象
- `domain/repositories/` — 无新仓储接口
- `domain/services/` — 无新领域服务
- `domain/events/` — 无新领域事件

相关领域概念已在 `conf/value-stream.yaml` 的 `company-management` 流中建模完毕。
