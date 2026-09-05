# NFR 澄清: 公司切换后项目列表为空 — 修复

> 输入:
> - 设计文档: `docs/specs/company-switch-project-list-empty-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-company-switch-project-list-empty-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 数据一致性 (Context Propagation) | L1 | `/me` API 在收到 `?tenant_id=` query param 时 100% 返回该 tenant 对应的 workspace |
| 性能 | L0 | 不适用 — 仅增加 < 10 字节 query param，无新网络调用 |
| 安全性 | L0 | 不适用 — query param 仅作查询过滤，不改变授权边界 |
| 可用性 | L0 | 不适用 |
| 可伸缩性 | L0 | 不适用 |
| 可观测性 | L0 | 不适用 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L0 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: `/me` API tenant 上下文传播

#### NFR 类别: 数据一致性 (Context Propagation)
- **等级**: L1 - 基础
- **量化目标**: `get_current_workspace` 在 `?tenant_id=<id>` 存在且用户属于该公司时，返回该公司的 workspace（而非旧公司的 workspace）

## 质量场景

### QS-01: 公司切换后 workspace 解析正确
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L1 |
| 刺激源 | WorkPanel 在用户切换公司后调用 `/me?tenant_id=<new_id>` |
| 刺激 | `/me` API 收到 `tenant_id=<new_id>` query param |
| 制品 | `UserSerializer.get_current_workspace` |
| 环境 | 正常 |
| 响应 | 返回 `current_workspace` 属于 `tenant_id` 对应的公司 |
| 响应度量 | `current_workspace.company_id` === `tenant_id`（query param），且 workspace 关联了项目 |

## 领域模型影响

不引入新领域概念。这是上下文传播的修复——`tenant_id` 从 URL path 到 query param 的传递链补齐。

## 权衡与边界

### 取舍
- 使用 query param 而非修改 URL 结构——最小侵入，向后兼容

### 明确不做什么
- 不新增 session key 或持久化状态
- 不改变 `/me` API 的 URL 路径结构

### 升级触发条件
- 如果未来需要更强的上下文传递 → 可考虑通过 middleware 自动注入 `X-Tenant-Id` header

## 跳过声明

以下 NFR 类别因变更性质（纯 bug fix，无新数据流/外部依赖/安全边界变更）而跳过：性能、安全性、可用性、可伸缩性、可观测性、合规、可维护性。
