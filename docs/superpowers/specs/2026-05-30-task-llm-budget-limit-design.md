# 任务 LLM 预算限额设计

**日期：** 2026-05-30  
**状态：** 已评审（Increment 1–2 后端已落地；3–5 见实施计划）  
**相关页面：**
- `http://127.0.0.1:4000/tenant/<tenant>/settings/task-panel/` — 租户开关 + 工作空间默认预算
- `http://127.0.0.1:4000/tenant/<tenant>/people/manage/` — 预算上调权限
- 任务详情 — 任务级预算覆盖与临时上调

## 背景与问题

部分 LLM 模型 token 费用较高。当前平台仅在任务创建/服务器启动时扣积分（taskBill），**不跟踪 LLM 实际用量**，也无法按「接入端点 × 模型」限制单任务 spend。

租户需要在可控成本下使用大模型：工作空间设默认预算，任务详情可覆盖；超额硬拦截；有权限者可二次确认临时上调。

**前提约束（新增）：** 预算控制**仅对已在「功能参数」页为某条 provider 配置勾选「启用派生子 Key」（`use_sub_token=true`）的接入生效**。未启用派生子 Key 的 provider 仍使用主 Key（master），**不参与**预算配置、计量与拦截。原因：派生子 Key 按任务隔离，平台可在任务维度绑定预算 policy 与用量账本；主 Key 无法可靠做 per-task 硬拦截。

## 已确认的产品决策

| 决策项 | 选择 |
|--------|------|
| 限额单位 | **人民币（CNY）金额预算**（非 token 数、非平台积分） |
| 单价来源 | 租户在设置页为每个 `(端点 + 模型)` **手动填写** input/output 单价（元/1M tokens） |
| 货币 | **固定 CNY**，不提供 USD 或工作空间级货币切换 |
| 预算粒度 | **按 (provider + base_url + model) 分别**设预算与累计 |
| **生效范围** | **仅 `use_sub_token=true` 的 provider**（功能参数 `/settings/feature-params/`）；主 Key provider 排除 |
| 超额行为 | **硬拦截**；任务 Owner / 被授权成员或小组可在详情页**二次确认临时上调** |
| 权限配置 | `people/manage` 为成员/小组配置 `can_raise_task_budget` |
| 租户开关 | **租户可选是否开启**；未开启时所有相关 UI 与 enforcement **均不启用** |

## 目标

1. 租户级主开关：显式 opt-in，默认关闭。
2. 开启后：**仅对 `use_sub_token=true` 的 provider** 配置工作空间默认单价 + 预算；任务级继承/覆盖；运行时混合 enforcement。
3. 关闭后：无预算 UI、无 API 写入、无容器 policy 下发、agent 不做预算检查。
4. 某 provider 关闭派生子 Key 后：该 provider 下预算行只读/隐藏，enforcement 立即停止，历史用量保留。

## 非目标（本期）

- 对 **主 Key（master）** provider 做预算限额（无派生子 Key 则不支持）
- 与 taskBill 积分自动换算/扣费
- 系统全局价目表（仅租户手动单价）
- 按任务面板列（DeliverableObj）差异化预算
- 软提醒-only（必须硬拦截）

---

## 租户功能开关（新增）

### 数据

在 `projects_tenant_feature_params` 增加字段：

| 字段 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `llm_budget_enabled` | `BooleanField` | `False` | 租户是否启用 LLM 预算限额 |

> 挂在 `TenantFeatureParams`（租户 OneToOne）而非工作空间：开关是**租户级**决策；具体预算数值仍按工作空间配置。

### 开关 UI 位置

**任务面板设置页**（`/tenant/{id}/settings/task-panel/`）顶部增加卡片：

- 标题：「LLM 预算限额」
- 控件：启用 / 关闭（toggle）
- 说明：关闭后，工作空间默认预算、任务详情预算面板、成员预算上调权限等入口均隐藏；已有累计数据保留但不拦截。
- 仅**租户管理员**（`CompanyMember.is_admin`）可修改开关。

### 开关关闭时的行为矩阵

| 区域 | 行为 |
|------|------|
| task-panel · 工作空间「LLM 预算默认设置」入口 | **隐藏** |
| 任务详情 · 「LLM 预算」面板 | **不渲染** |
| people/manage · 「允许上调任务 LLM 预算」 | **不渲染** |
| API `.../model-budget-defaults/`、`.../model-budgets/`、`.../budget-permissions/` | 返回 `403` + `{ "code": "llm_budget_disabled" }` |
| 容器 `TASK_LLM_BUDGET_POLICY` env | **不下发** |
| trae-agent 预算检查 | **跳过**（无 policy 即 no-op） |
| 历史 `TaskModelBudgetUsage` 数据 | **保留**（便于重新开启后查看；不删除） |

### 开关开启时

- 上述 UI 按权限正常展示。
- 新任务创建时从工作空间默认展开预算行。
- 容器 bootstrap 拉取 budget policy；agent 本地预检 + 平台账本。

### API 契约补充

`GET /api/tenant/{id}/feature-params/` 响应增加：

```json
{
  "llm_budget_enabled": false
}
```

`PATCH` 同路径可更新（仅 tenant admin）。前端路由守卫与页面 `onMounted` 均读此字段，**避免仅靠 UI 隐藏而 API 仍可写**。

公开只读接口（供任务详情、task-panel 轻量判断）：

`GET /api/tenant/{id}/llm-budget-feature/` → `{ "enabled": bool }`

（可与 feature-params 合并，避免重复请求；实现时二选一。）

---

## 派生子 Key 前置条件（新增）

### 与功能参数的关系

预算能力与 `TenantFeatureParams.providers[]` 中每条 provider 的 **`use_sub_token`** 字段绑定：

| `llm_budget_enabled` | `use_sub_token`（某 provider） | 该 provider 预算能力 |
|------------------------|-------------------------------|----------------------|
| false | 任意 | **不可用**（全局关闭） |
| true | false | **不可用**（主 Key，不参与预算） |
| true | true | **可用**（可配默认预算、任务覆盖、运行时拦截） |

数据来源：
- `use_sub_token` 在 **功能参数页** `/tenant/{id}/settings/feature-params/` 维护（已有 checkbox）
- 供应商须出现在系统 `SubTokenProvider` 列表中（已有 `isSubTokenProvider()` 前端逻辑）

### 判定函数（后端/前端共用语义）

```python
def is_budget_eligible_provider(provider_entry: dict, *, llm_budget_enabled: bool) -> bool:
    return (
        llm_budget_enabled
        and bool(provider_entry.get("use_sub_token"))
        and str(provider_entry.get("provider", "")).strip() != ""
    )
```

容器下发 env 时：`TaskApiKeyUsage.key_type` 已为 `sub` 的 provider 才纳入 `TASK_LLM_BUDGET_POLICY.models`。

### 功能参数页联动（`/settings/feature-params/`）

当 `llm_budget_enabled=true` 时，在 provider 卡片上增加说明：

- 勾选「启用派生子 Key」→ 该 provider 的模型可在 task-panel 配置预算、任务详情展示预算
- 取消勾选 → 提示「关闭后将停止该 provider 的预算拦截」；已有 `WorkspaceModelBudgetDefault` / `TaskModelBudget` **保留但不生效**

可选 UX：对已在系统登记、支持派生但未勾选 `use_sub_token` 的 provider，在 task-panel 预算模态中显示灰色提示：「请先在功能参数页启用派生子 Key」。

### provider 关闭派生子 Key 时

| 区域 | 行为 |
|------|------|
| task-panel 预算表格中该 provider 行 | **隐藏或只读禁用** |
| 任务详情预算面板中该 provider 模型 | **不展示** |
| `TASK_LLM_BUDGET_POLICY` | **不包含**该 provider |
| 已有 `TaskModelBudgetUsage` | **保留**，不再累加 |

---

## 分层架构

```text
┌─────────────────────────────────────────────────────────────────┐
│ 功能参数页：providers[].use_sub_token（派生子 Key 开关）          │
│  task-panel / task-detail / people（llm_budget_enabled 时）      │
└───────────────────────────┬─────────────────────────────────────┘
                            │ 仅 use_sub_token=true 的 provider
┌───────────────────────────▼─────────────────────────────────────┐
│ task2app Django                                                  │
│  TenantFeatureParams.llm_budget_enabled                           │
│  WorkspaceModelBudgetDefault / TaskModelBudget / Usage         │
│  TenantBudgetPermission                                          │
└───────────────────────────┬─────────────────────────────────────┘
                            │ env: TASK_LLM_BUDGET_POLICY（sub-token providers only）
┌───────────────────────────▼─────────────────────────────────────┐
│ onlineServiceJS + trae-agent                                     │
│  派生 Key 任务：本地 spent 累计 + 超额硬拦截；异步上报 usage       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 数据模型

> **货币约定：** 全平台 LLM 预算统一以 **人民币（CNY）** 计量与展示。API、DB、容器 env 中金额均为 CNY；UI 文案使用「元」或「¥」。不引入 `budget_currency` 字段或多币种换算。

### WorkspaceModelBudgetDefault

工作空间级默认（task-panel 设置，**需 `llm_budget_enabled=true` 且 provider `use_sub_token=true`**）。

| 字段 | 说明 |
|------|------|
| `workspace_id` | FK Workspace |
| `provider` | 来自 `TenantFeatureParams.providers[].provider`；**须 `use_sub_token=true`** |
| `base_url` | 端点 |
| `model_name` | 模型名 |
| `input_price_per_1m` | Decimal，单位：元/1M input tokens |
| `output_price_per_1m` | Decimal，单位：元/1M output tokens |
| `budget_limit` | Decimal，单位：元；该模型默认预算上限 |

唯一约束：`(workspace_id, provider, base_url, model_name)`。

### TaskModelBudget

任务级预算（继承 + 覆盖）。

| 字段 | 说明 |
|------|------|
| `todo_id` | FK Todo |
| `provider`, `base_url`, `model_name` | 同上 |
| `budget_limit` | Decimal（元）；`null` 表示继承工作空间默认 |
| `budget_limit_source` | `inherited` / `override` / `raised` |

### TaskModelBudgetUsage

平台账本（累计用量）。

| 字段 | 说明 |
|------|------|
| `todo_id`, `provider`, `base_url`, `model_name` | 联合标识 |
| `input_tokens`, `output_tokens` | 累计 |
| `spent_amount` | 按单价换算，单位：元 |
| `last_reported_at` | 最后上报时间 |

### TenantBudgetPermission

成员/小组预算上调权限（**仅 llm_budget_enabled=true 时在 people/manage 展示**）。

| 字段 | 说明 |
|------|------|
| `company_id` | FK Company |
| `subject_type` | `member` / `group` / `owner_role` |
| `subject_id` | CompanyMember.id / CompanyGroup.id / 固定 `owner` |
| `can_raise_task_budget` | bool |

### 费用公式

```
spent = input_tokens/1e6 * input_price_per_1m + output_tokens/1e6 * output_price_per_1m
```

### 继承规则

- 租户**开启**功能且任务创建时：从 `TenantFeatureParams.providers` 中 **`use_sub_token=true`** 的条目展开 `(provider, base_url, model)` × 工作空间默认，写入 `TaskModelBudget`（`source=inherited`）。
- 租户**关闭**功能：不创建新 `TaskModelBudget` 行；已有行只读保留。
- 某 provider **`use_sub_token` 改为 false**：不再为新任务创建该 provider 预算行；已有任务该 provider 行停止 enforcement（policy 不下发）。
- **主 Key provider**（`use_sub_token=false`）：永不创建预算行、永不进入 policy。

---

## UI 规格

### A. 任务面板设置 `/tenant/{id}/settings/task-panel/`

1. **租户开关卡片**（始终可见，仅 admin 可改）；说明中注明「须先在功能参数页为 provider 启用派生子 Key」
2. `llm_budget_enabled === true` 时，每个工作空间操作区增加「LLM 预算默认设置」按钮（与「套餐设置」并列）
3. 模态内容：模型预算表格（供应商 | 端点 | 模型 | Input 单价（元/1M） | Output 单价（元/1M） | 默认预算上限（元））；**仅列出 `use_sub_token=true` 的 provider 及其模型**
4. 若租户无任何 `use_sub_token=true` 的 provider：模态展示空状态 + 跳转功能参数页的链接
5. 模型列表只读引用 `TenantFeatureParams.providers`；未启用派生子 Key 的 provider **整行不出现**

### A2. 功能参数 `/tenant/{id}/settings/feature-params/`（联动）

- `llm_budget_enabled=true` 时，「启用派生子 Key」checkbox 旁增加提示：「启用后可配置该 provider 的任务 LLM 预算限额」
- 取消 `use_sub_token` 时二次确认（若该 provider 已有工作空间默认预算）

### B. 任务详情

- `llm_budget_enabled === false`：**整个「LLM 预算」面板不挂载**
- 开启时：**仅展示 `use_sub_token=true` provider 下的模型**；每模型进度条、已用/剩余（元）、覆盖输入、超额 banner、「临时上调」（权限 + 二次确认）

### C. 成员管理 `/tenant/{id}/people/manage/`

- `llm_budget_enabled === false`：不显示预算权限 checkbox
- 开启时：成员/小组「允许上调任务 LLM 预算」

---

## API 设计

| 端点 | 说明 | enabled 校验 |
|------|------|--------------|
| `GET/PATCH .../feature-params/` | 含 `llm_budget_enabled` | PATCH 开关不依赖 enabled |
| `GET/PATCH .../workspaces/{wid}/model-budget-defaults/` | 工作空间默认 CRUD | **必须 enabled** |
| `GET/PATCH .../task/{tid}/model-budgets/` | 任务预算读写 | **必须 enabled** |
| `POST .../model-budgets/raise/` | 临时上调 | **必须 enabled** + 权限 |
| `POST .../task/{tid}/cloud/model-budget-usage/` | 容器上报 | **必须 enabled** |
| `GET/PATCH .../budget-permissions/` | 权限配置 | **必须 enabled** |

后端统一 decorator：

1. `require_llm_budget_enabled(tenant_id)` → 403 `llm_budget_disabled`
2. `require_budget_eligible_provider(tenant_id, provider, base_url)` → 校验 `providers[].use_sub_token`；失败 403 `llm_budget_provider_not_sub_token`

容器 env（**enabled + 该任务 provider 为 sub-token** 且任务有 policy 时）：

```json
{
  "TASK_LLM_BUDGET_POLICY": {
    "currency": "CNY",
    "models": [{
      "provider": "openai",
      "base_url": "https://api.openai.com/v1",
      "model": "gpt-4.1",
      "input_price_per_1m": "18.00",
      "output_price_per_1m": "72.00",
      "budget_limit": "50.00",
      "spent_amount": "12.30"
    }]
  }
}
```

> `currency` 固定为 `"CNY"`，只读常量；客户端不应提供修改入口。若未来扩展多币种，需另开设计评审，本期不预留 USD 字段。

---

## 运行时 enforcement（混合方案）

1. Agent 每次 LLM 调用前：若存在 `TASK_LLM_BUDGET_POLICY`，检查 `spent < budget_limit`。
2. 超额：拒绝新 LLM 调用，推送 `budget_exhausted` 状态至任务详情。
3. 调用后：本地累加 token → spent；异步 POST usage delta（幂等键 `task+model+step`）。
4. **无 policy**（功能关闭、provider 未启用派生子 Key、或未配置）：与今日行为一致，零开销。

---

## 权限矩阵

| 操作 | 租户 admin | 工作空间 admin | 任务 Owner | can_raise 成员/小组 |
|------|-----------|---------------|-----------|-------------------|
| 切换 llm_budget_enabled | ✓ | ✗ | ✗ | ✗ |
| 配置工作空间默认 | ✓ | ✓ | ✗ | ✗ |
| 任务详情修改预算 | ✓ | ✓ | ✓ | ✗ |
| 超额临时上调 | ✓ | ✓ | ✓* | ✓* |

\*需在 people/manage 授权。

---

## 领域概念清单（供 `/5-ddd`）

| 类型 | 候选 |
|------|------|
| **限界上下文** | `TaskBudget`（预算与用量）、`TenantLLMConfig`（含开关 + `use_sub_token`  eligibility）、`MemberPermission` |
| **实体** | `TenantFeatureParams`（+`llm_budget_enabled`；`providers[].use_sub_token` 为 eligibility 输入）、`WorkspaceModelBudgetDefault`、`TaskModelBudget`、`TaskModelBudgetUsage`、`TenantBudgetPermission`、`TaskApiKeyUsage`（`key_type=sub` 关联） |
| **聚合根** | `TaskBudgetPolicy`（任务级：各模型子预算 + 累计 spent） |
| **值对象** | `ModelEndpointKey(provider, base_url, model_name)`、`CnyAmount(value)`（固定 CNY，无 currency 维度）、`UnitPrice(input_per_1m, output_per_1m)` |
| **领域事件** | `TenantLlmBudgetFeatureEnabled`、`TenantLlmBudgetFeatureDisabled`、`TaskModelBudgetExhausted`、`TaskModelBudgetRaised`、`TaskModelBudgetUsageReported` |

---

## 价值流影响

| 现有流 | 影响 |
|--------|------|
| `tenant-feature-params` | 新增 `llm_budget_enabled`；`providers[].use_sub_token` 为预算 eligibility 输入；测试 `tests/test_manage_feature_params.py` 扩展 |
| `task-management` | 任务创建时仅为 **sub-token provider** seed `TaskModelBudget` |
| `task-detail-runtime-relay` | 条件性下发 `TASK_LLM_BUDGET_POLICY`；exec log 可展示费用 |
| `billing`（taskBill） | 无直接影响 |

**建议新增价值流：** `task-llm-budget-governance`（domain: 任务协作）

| 步骤 | 状态 | 预期测试 |
|------|------|----------|
| `tenant-llm-budget-feature-toggle` | planned | `tests/test_llm_budget_feature_toggle.py` |
| `llm-budget-sub-token-eligibility` | planned | `tests/test_llm_budget_sub_token_eligibility.py` |
| `workspace-model-budget-defaults` | planned | `tests/test_workspace_model_budget_defaults.py` |
| `task-model-budget-override` | planned | `tests/test_task_model_budget_override.py` |
| `task-model-budget-enforcement` | planned | trae-agent + container usage API 集成测试 |
| `budget-raise-permission` | planned | `tests/test_tenant_budget_permission.py` |

**Fields 预览：**

- `saas-backend.projects_tenant_feature_params.llm_budget_enabled`
- `saas-backend.projects_tenant_feature_params.providers_use_sub_token`（JSON 字段 providers[].use_sub_token，eligibility）
- `saas-backend.projects_task_api_key_usage.key_type`（`sub` 与 budget 关联）
- `saas-backend.projects_workspace_model_budget_default.*`
- `saas-backend.projects_task_model_budget.*`
- `saas-backend.projects_task_model_budget_usage.*`
- `saas-backend.projects_tenant_budget_permission.*`

完整 value stream 切片在 `/3-value-stream-价值流` 完成。

---

## 测试要点

1. 默认 `llm_budget_enabled=false`；相关 API 403；前端无预算 UI。
2. Admin 开启后 UI 出现；关闭后 UI 消失且 API 再次 403。
3. 开关关闭期间容器 env 无 `TASK_LLM_BUDGET_POLICY`。
4. 开启后：工作空间 CRUD、任务继承、详情覆盖、超额拦截、raise 权限。
5. **sub-token 约束**：`use_sub_token=false` 的 provider 无法写入默认预算 API（403）；policy 不包含 master Key provider。
6. 关闭某 provider 的 `use_sub_token` 后：policy 立即移除该 provider；历史 usage 保留。
7. Playwright：开关 off → 无预算 UI；on 但无 sub-token provider → 空状态；on + sub-token → 完整流程。

---

## 方案选择记录

| 方案 | 结论 |
|------|------|
| A. 纯运行时本地拦截 | 审计弱，未选 |
| B. 纯平台账本 | 延迟高，未选 |
| **C. 混合（本地预检 + 平台账本）** | **采用** |
| 限额单位：Token / 积分 / 外币 | **CNY 金额（元）** |
| 货币：USD/CNY 可选 | **仅 CNY，无切换** |
| 单价：系统价目表 / 手动 | **租户手动** |
| 租户开关 | **opt-in，默认关，关则 UI+API+enforcement 全禁用** |
| 生效范围 | **仅 `use_sub_token=true` 的 provider；主 Key 不支持预算** |
