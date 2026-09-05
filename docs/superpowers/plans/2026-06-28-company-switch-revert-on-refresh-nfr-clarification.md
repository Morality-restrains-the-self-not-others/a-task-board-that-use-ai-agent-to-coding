# NFR 澄清: 公司切换后页面刷新回退 — 修复

> 输入:
> - 设计文档: `docs/specs/company-switch-revert-on-refresh-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-company-switch-revert-on-refresh-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 数据一致性 (UI State) | L1 | Navbar/Sidebar 的 currentTenant 在页面加载后 100% 与 URL `/tenant/:tenant` 一致 |
| 性能 | L0 | 不适用 — 无新后端调用，仅调整已有 API 响应的消费逻辑 |
| 安全性 | L0 | 不适用 — 无新增认证/授权/加密逻辑 |
| 可用性 | L0 | 不适用 — 纯前端变更，无服务依赖变更 |
| 可伸缩性 | L0 | 不适用 — 无新增数据量或并发路径 |
| 可观测性 | L0 | 不适用 — 无新增日志/指标/追踪需求 |
| 合规与隐私 | L0 | 不适用 — 无新增数据收集或存储 |
| 可维护性 | L0 | 不适用 — 变更范围小（3 文件），无 API 版本影响 |

## 逐增量 NFR 分析

### Increment 1: URL Tenant 优先 — 公司上下文一致性

#### NFR 类别: 数据一致性 (UI State)
- **等级**: L1 - 基础
- **量化目标**: Navbar `currentTenant` 在 `applyMePayload` 执行后与 `route.params.tenant` 一致（当 URL 包含 `/tenant/:id` 且用户属于该公司时）
- **无专项测试**: 此修复是去 bug（状态源不一致），而非新增功能；正确性由现有 Playwright E2E + 单元测试覆盖

## 质量场景

### QS-01: 公司切换后 Navbar 显示 URL 对应的公司
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 (UI State) |
| 等级 | L1 |
| 刺激源 | 用户在公司 A 页面通过 Navbar 下拉框切换到公司 B |
| 刺激 | `switchCompany(B)` → `window.location.href = /tenant/B/work-panel` |
| 制品 | Navbar.logic.vue `applyMePayload` + Sidebar.vue `initData` |
| 环境 | 正常（页面完整加载） |
| 响应 | Navbar 下拉框显示公司 B；Sidebar 的 `tenantPath` = `/tenant/B` |
| 响应度量 | `currentTenant` === `route.params.tenant` 且 `currentTenant` 存在于 `userCompanies` 列表中 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| UI State 一致性 L1 | 不引入新领域概念。`current_company` 的解析从「API-only」变为「URL-first + API-fallback」——这是一个前端状态解析策略变更，而非领域模型变更。 | DDD 步骤可跳过（无新领域概念） |

## 权衡与边界

### 取舍
- 不引入新的后端 API 或 session 字段——URL 已是真源，不增加状态同步复杂度

### 明确不做什么
- 不改变 `switchCompany` 的页面跳转方式（保持与 workspace 切换一致）
- 不在后端新增 `set_current_company` API
- 不做 SPA 客户端路由内的公司切换（保持完整页面刷新）

### 升级触发条件
- 如果未来需要跨页面保持公司上下文而不依赖 URL → 可引入 sessionStorage 或后端 session 字段

## 跳过声明

以下 NFR 类别因变更性质（纯前端 bug fix，无新数据流/新外部依赖/新后端 API）而跳过：

- **性能**: 跳过。变更仅调整已有 API 响应的消费逻辑（增加一次 `Array.find`），无新增网络调用。
- **安全性**: 跳过。无新增认证/授权/加密逻辑，不改变现有安全边界。
- **可用性**: 跳过。无服务依赖变更，不引入新故障模式。
- **可伸缩性**: 跳过。无新增数据量或并发路径。
- **可观测性**: 跳过。无新增日志/指标/追踪需求。
- **合规与隐私**: 跳过。无新增数据收集或存储。
- **可维护性**: 跳过。变更范围小（3 文件），不涉及 API 版本或配置格式变更。
