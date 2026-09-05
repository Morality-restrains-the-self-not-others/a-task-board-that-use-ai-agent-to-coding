# LLM 预算：端点级启用设计（修订）

**日期：** 2026-05-30  
**状态：** 已批准（方案 A，0-auto-flow 2026-05-30）  
**前置文档：** `docs/superpowers/specs/2026-05-30-task-llm-budget-limit-design.md`（金额、拦截、任务/工作空间粒度仍有效）

## 1. 问题陈述

当前实现将 **「租户 LLM 预算功能」** 作为 `TenantFeatureParams.llm_budget_enabled` 租户级布尔开关，放在功能参数页顶部独立卡片；而 **「启用派生子 Key」** 已是 `providers[]` 每一行（供应商 + `base_url` + 模型列表）的 **端点级** 配置。

用户反馈：二者粒度不一致。**LLM 预算的启用应与派生子 Key 一样，按 API 端点（功能参数中的一条 provider 配置）设置**，而不是租户顶部一个总开关。

### 现状对照

| 配置项 | 当前粒度 | 存储位置 |
|--------|----------|----------|
| 派生子 Key | **端点级**（每条 provider 行） | `providers[].use_sub_token` |
| LLM 预算功能开关 | **租户级** | `llm_budget_enabled` |
| 单价 / 默认预算上限 | **端点 × 模型**（已正确） | `WorkspaceModelBudgetDefault` 等 |

后端 eligibility 已按 `(provider, base_url, model)` 计量，但 **「是否参与预算」** 仅依赖租户开关 + `use_sub_token`，无法在「同一租户、多 endpoint」场景下只给部分 endpoint 开预算。

---

## 2. 价值流影响（`value-stream.yaml`）

受影响流：**`task-llm-budget-governance`**（任务协作域）

| 步骤 | 影响 |
|------|------|
| `tenant-llm-budget-feature-toggle` | **语义调整**：从「租户 opt-in」改为「端点 opt-in 聚合」或废弃租户字段；测试需更新 |
| `llm-budget-sub-token-eligibility` | **扩展**：增加 `providers[].budget_enabled`（或等价字段）；eligibility = `use_sub_token ∧ budget_enabled` |
| `workspace-model-budget-defaults` | 模态/表格仅列出 **budget_enabled** 的端点模型 |
| `task-model-budget-*` / `budget-raise-permission` | 全局 gate 改为「租户存在至少一个 eligible 端点」或按任务实际用到的 endpoint 判断 |

建议在 `value-stream.yaml` 新增字段条目：

```yaml
- name: saas-backend.projects_tenant_feature_params.providers_budget_enabled
  description: providers[].budget_enabled；端点级 LLM 预算启用；须 use_sub_token=true
```

**跨流依赖：** 仍依赖 `tenant-feature-params`（功能参数 CRUD）、`task-management`（任务详情面板）、容器 bootstrap（`TASK_LLM_BUDGET_POLICY`）。

---

## 3. 领域概念清单（供 `/5-ddd`）

| 概念 | 边界上下文 | 说明 |
|------|------------|------|
| `LlmProviderEntry` | 租户 LLM 配置 | 扩展 `budget_enabled` 值对象字段 |
| `BudgetEligibleEndpoint` | 任务 LLM 预算 | 派生概念：`use_sub_token ∧ budget_enabled` 的端点 |
| `WorkspaceModelBudgetDefault` | 任务 LLM 预算 | 不变；键仍为 `(workspace, provider, base_url, model)` |
| `TenantBudgetFeature` | 任务 LLM 预算 | 从「租户布尔」弱化为「是否存在 eligible 端点」的查询 |

领域事件（可选）：`EndpointBudgetEnabled` / `EndpointBudgetDisabled` — 用于审计与清理 policy 缓存（本期可只做同步校验，不发事件）。

---

## 4. 方案对比

### 方案 A：纯端点级（推荐）

- 在 `providers[]` 每行增加 **`budget_enabled`**（默认 `false`）。
- **移除** UI 上的租户总开关；`llm_budget_enabled` 列 **废弃**（迁移期只读聚合，见 §6）。
- Eligibility：`use_sub_token && budget_enabled && provider 非空`。
- 功能参数页：在「启用派生子 Key」**同一行或紧邻下一行** 增加「启用 LLM 预算」；未勾选派生子 Key 时禁用并提示。

**优点：** 与派生子 Key 心智一致；多 endpoint 可独立控制。  
**缺点：** 需数据迁移与 API 兼容；「一键关闭全租户预算」需逐个端点关（可接受）。

### 方案 B：租户总开关 + 端点级细开关（双层）

- 保留 `llm_budget_enabled` 作租户 **主闸**（合规/紧急关停）。
- 每端点 `budget_enabled` 仅当主闸开启时可编辑。
- Eligibility：`llm_budget_enabled && use_sub_token && budget_enabled`。

**优点：** 保留「全租户关停」运维能力。  
**缺点：** 两层开关易混淆；与用户「端点级」诉求不完全一致。

### 方案 C：与 `use_sub_token` 合并（不新增字段）

- 勾选「派生子 Key」即视为启用该端点预算；取消即关闭。

**优点：** 零新字段。  
**缺点：** 无法「派生 Key 但不限额」；与「派生用于隔离、预算可选」产品路径冲突。**不推荐。**

### 推荐：**方案 A**，迁移期 API 保留 `llm_budget_enabled` 作为 **计算字段**（`any(providers[].budget_enabled)`），便于旧前端/测试过渡，**不再作为独立写入项**。

---

## 5. 详细设计（方案 A）

### 5.1 数据模型

**`TenantFeatureParams.providers[]` JSON 每项增加：**

| 字段 | 类型 | 默认 | 约束 |
|------|------|------|------|
| `budget_enabled` | `bool` | `false` | 仅当 `use_sub_token=true` 时可置 `true`；保存时若 `use_sub_token=false` 则强制 `budget_enabled=false` |

**`llm_budget_enabled`（DB 列）：**

- **Phase 1（本设计）：** 写入时由后端根据 `any(budget_enabled)` 同步更新，保持旧测试/只读 API 兼容。
- **Phase 2（可选）：** 删除列，GET feature-params 仅返回 `llm_budget_enabled: computed`。

### 5.2 Eligibility 语义（替换原表）

| `use_sub_token` | `budget_enabled` | 该端点预算能力 |
|-----------------|------------------|----------------|
| false | 任意 | **不可用** |
| true | false | **不可用**（派生 Key 可无预算） |
| true | true | **可用**（可配默认预算、任务覆盖、拦截） |

```python
def is_budget_eligible_provider_entry(entry, *, tenant_budget_legacy: bool = True) -> bool:
    # tenant_budget_legacy: Phase1 可忽略；Phase2 删除
    return bool(
        entry.use_sub_token
        and entry.budget_enabled
        and entry.provider.strip()
    )
```

### 5.3 功能参数页 UX（`/settings/feature-params/`）

**删除** 顶部「租户 LLM 预算功能」独立卡片。

**每条 provider 卡片内**（`base_url` 字段下方或派生子 Key 同行）：

```
[ ] 启用派生子 Key          [ ] 启用 LLM 预算（本端点）
     ↑ 已有                      ↑ 新增；未勾选左侧时 disabled
```

文案：

- 「启用 LLM 预算」：仅对本 API 端点（`base_url`）下模型生效；须先启用派生子 Key。
- 勾选派生子 Key 时 **不自动** 勾选预算（显式 opt-in）。
- 取消派生子 Key：二次确认；同时清空 `budget_enabled`。

**工作空间默认预算** 区块：

- 展示条件：存在至少一个 `budget_enabled=true` 的端点（不再读租户总开关）。
- 模态表格：仅列出 eligible 端点的模型（与现逻辑一致，过滤条件加 `budget_enabled`）。

### 5.4 其他页面

| 页面 | 调整 |
|------|------|
| 任务详情 · LLM 预算面板 | 可见条件：任务所属租户存在 eligible 端点，且任务 policy 中有对应模型 |
| people/manage · 上调权限 | 同上，无 eligible 端点则不展示 |
| 任务面板（若仍有入口） | 与功能参数一致，按 eligible 端点展示「LLM 预算默认」 |

### 5.5 API

**`GET/POST/PATCH .../feature-params/`**

- `providers[]` 读写 `budget_enabled`。
- 响应保留 `llm_budget_enabled`（**只读、计算**）至少一个 major 版本，文档标注 deprecated。
- 校验：`budget_enabled=true` 且 `use_sub_token=false` → `400`。

**预算 CRUD / usage / raise**

- `require_llm_budget_enabled` 改为 `require_any_budget_eligible_endpoint(tenant_id)` 或按请求的 `(provider, base_url)` 校验 `budget_enabled`。
- 403 码保留 `llm_budget_disabled`（语义：该租户无任何启用预算的端点，或目标端点未启用）。

### 5.6 迁移

1. 部署时：对所有 `providers[].use_sub_token=true` 的行，若旧 `llm_budget_enabled=true`，则设 `budget_enabled=true`；否则 `false`。
2. 旧 `llm_budget_enabled=false`：全部 `budget_enabled=false`。
3. 前端：移除租户卡片；发布说明旧书签 `/llm-budget/` 已重定向 feature-params。

### 5.7 测试计划

| 用例 | 文件 |
|------|------|
| 端点 `budget_enabled` 校验、与 `use_sub_token` 联动 | `test_llm_budget_endpoint_enable.py`（新）或扩展现有 |
| eligibility 仅 budget 端点出现在 defaults | 更新 `test_llm_budget_sub_token_eligibility.py` |
| feature-params POST 拒绝非法组合 | 更新 `test_manage_feature_params.py` |
| 计算 `llm_budget_enabled` 兼容 | 更新 `test_llm_budget_feature_toggle.py` |

---

## 6. 与原文档关系

- **保留：** CNY 计量、端点×模型预算键、硬拦截、工作空间默认、任务覆盖、权限矩阵、容器 policy 结构。
- **修订：** 「租户功能开关」章节 → 「端点级 budget_enabled」；UI 从顶部卡片 → provider 卡片内。
- **不变：** `WorkspaceModelBudgetDefault` / `TaskModelBudget` 表结构（已是端点×模型）。

---

## 7. 待确认项（审批前）

1. **是否完全移除租户级主闸（方案 A）**，还是保留方案 B 的 `llm_budget_enabled` 供管理员一键关停？（推荐 A + 计算字段兼容）
2. **勾选派生子 Key 时是否默认勾选预算？**（推荐：**不默认**，避免隐性开启拦截）
3. **Phase 2 是否删除 DB 列 `llm_budget_enabled`？**（推荐：至少观察一个版本后再删）

---

## 8. 验收标准

1. 功能参数页无独立「租户 LLM 预算功能」卡片；每条 provider 可在端点级勾选「启用 LLM 预算」。
2. 仅 `use_sub_token && budget_enabled` 的端点出现在工作空间默认预算模态与任务详情预算面板。
3. 关闭某端点 `budget_enabled` 后，该 `(provider, base_url)` 下模型立即停止拦截，历史 usage 保留。
4. 迁移后，原 `llm_budget_enabled=true` 的租户，其已启用派生的端点行为与迁移前一致。
