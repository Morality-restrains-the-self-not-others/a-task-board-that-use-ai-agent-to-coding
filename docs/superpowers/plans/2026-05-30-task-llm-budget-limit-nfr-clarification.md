# NFR 澄清: 任务 LLM 预算限额

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-30-task-llm-budget-limit-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-30-task-llm-budget-limit-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 预算 API P95 ≤ 500ms；usage 上报异步不阻塞 LLM |
| 可伸缩性 | L1 | 单租户任务级账本，日活 < 1 万 |
| 可用性 | L2 | 上报失败时 agent 本地仍拦截；恢复后补报 |
| 安全性 | L3 | 租户 admin 改开关；预算 API 需 enabled + sub-token；容器 usage 需 AccessToken |
| 数据一致性 | L2 | usage 幂等键去重；spent 读己之写（主库） |
| 容错机制 | L2 | usage 上报 at-least-once + 幂等；agent 无 policy 时 no-op |
| 可观测性 | L1 | 超额写 audit/log；无专项 Grafana |
| 合规与隐私 | L1 | 不存明文 api_key；仅累计 token 与 CNY |
| 可维护性 | L2 | 特性开关 `llm_budget_enabled`；关闭零 enforcement 开销 |

## 逐增量 NFR 分析

### Increment 1: 租户开关 + sub-token eligibility

#### 安全性 — L3
- 仅 tenant admin 可 PATCH `llm_budget_enabled`
- 预算写 API 双重 gate：enabled + `use_sub_token`
- 质量场景: QS-01, QS-02

#### 可维护性 — L2
- 默认 `llm_budget_enabled=false`，旧租户行为不变

### Increment 2–3: 工作空间默认 + 任务覆盖

#### 性能 — L2
- CRUD P95 ≤ 500ms（SQLite/Django ORM）
- 质量场景: QS-03

#### 数据一致性 — L2
- 任务创建与 `TaskModelBudget` seed 同事务边界

### Increment 4: 用量上报 + runtime policy

#### 性能 — L2
- usage POST 异步；agent 本地 O(1) 预检
- 质量场景: QS-04

#### 容错 — L2
- 幂等键 `task+model+step`；重复上报不 double-count

#### 可用性 — L2
- 上报失败：本地 agent 仍按最后已知 spent 拦截

### Increment 5: 上调权限

#### 安全性 — L3
- `can_raise_task_budget` RBAC；raise 需二次确认（前端）+ 403（后端）
- 质量场景: QS-05

## 质量场景

### QS-01: 非 admin 无法开启租户预算
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 普通成员 |
| 刺激 | PATCH feature-params 设置 llm_budget_enabled=true |
| 制品 | manage_feature_params |
| 环境 | 正常 |
| 响应 | 403 Forbidden |
| 响应度量 | pytest 断言 status=403 |

### QS-02: master Key provider 无法写预算默认
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 工作空间 admin |
| 刺激 | POST model-budget-defaults（use_sub_token=false 的 provider） |
| 制品 | workspace_model_budget_defaults |
| 环境 | llm_budget_enabled=true |
| 响应 | 403 + code=llm_budget_provider_not_sub_token |
| 响应度量 | pytest 断言 error code |

### QS-03: 工作空间预算列表响应时间
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 管理员 |
| 刺激 | GET model-budget-defaults（≤20 模型行） |
| 制品 | workspace_model_budget_defaults |
| 环境 | 正常负载 |
| 响应 | 200 + JSON 列表 |
| 响应度量 | 单测环境 < 500ms（无专项 APM） |

### QS-04: usage 幂等上报
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | 容器 agent |
| 刺激 | 同一 idempotency_key POST 两次 |
| 制品 | model-budget-usage API |
| 环境 | 正常 |
| 响应 | 第二次 200 idempotent；spent 不重复累加 |
| 响应度量 | pytest 断言 spent_amount 不变 |

### QS-05: 无 raise 权限被拒绝
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 普通协作者 |
| 刺激 | POST model-budgets/raise |
| 制品 | raise endpoint |
| 环境 | 超额任务 |
| 响应 | 403 |
| 响应度量 | pytest 断言 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| usage 幂等 L2 | spent 用绝对累加 + idempotency 表/键 | `TaskModelBudgetUsage` 聚合 + `record_usage(idempotency_key)` |
| 最终一致上报 L2 | agent 本地副本与平台账本可短暂偏差 | 领域服务 `resolve_effective_spent(local, remote)` |
| 安全 L3 | eligibility 校验独立于 UI | 值对象 `BudgetEligibleProvider`；领域服务 `LlmBudgetEligibilityService` |
| 租户开关 L2 | 关闭时不创建新 budget 行 | 领域事件 `TenantLlmBudgetFeatureDisabled` |

## 权衡与边界

### 取舍
- agent 本地预检优先低延迟，接受短暂 platform/agent spent 偏差（L2 最终一致）
- 不做 taskBill 积分联动（L0 财务集成）

### 明确不做什么
- 主 Key provider 预算（无派生子 Key）
- 多币种
- P99 < 100ms 预算 API

### 升级触发条件
- 日活 > 10 万 → usage 写入独立服务或 Kafka 流
- 客户要求 SOC2 → 审计升至 L3

## 跳过声明

- **可伸缩性 L4**：当前单 Django 实例 + SQLite 足够
- **合规 GDPR**：无跨境；CNY 国内部署
