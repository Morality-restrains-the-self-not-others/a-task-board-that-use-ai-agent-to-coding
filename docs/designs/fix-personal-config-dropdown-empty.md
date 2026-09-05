# Fix: 任务详情页「个人配置」下拉选项为空

**日期**: 2026-07-01
**状态**: 待审批
**类型**: Bug Fix

---

## 问题描述

用户在任务详情页 (`/tenant/{tid}/workspace/{wid}/task-detail/{tid}/`) 的「功能参数来源」下拉中选择「个人配置」后，「个人配置」二级下拉选项始终为空。但用户自身确实拥有个人配置（可在 `/user/{uid}/profile/feature-params/` 页面正常查看和管理）。

**复现步骤**:
1. 登录 `contact@daydaymoney.com`
2. 进入任务详情页: `http://183.250.1.132:4000/tenant/850256677331562496/workspace/857903329669984256/task-detail/859354231702982656/?relayToTrae=true`
3. 在「直接启动」面板中，「功能参数来源」选择「个人配置」
4. 观察：「个人配置」下拉显示「暂无个人配置，请先在「环境变量」设置页创建」

**期望行为**: 下拉应列出用户已有的个人配置（如 profile 页面所示）。

---

## 根因分析

### 受影响文件

| 层 | 文件 | 行号 | 问题 |
|---|------|------|------|
| 前端 | `front_project/app/src/components/ServerConfig.logic.vue` | L473-475 | API 响应解析错误 |
| 前端 | `front_project/app/src/components/ServerConfig.logic.vue` | L458-465 | 初始加载时未触发 fetch |

### 根因 1 (主要): API 响应字段名不匹配

`ServerConfig.logic.vue:473-475` 中的 `fetchPersonalConfigs()` 方法:

```js
// ❌ 当前代码
const data = await resp.json()
personalConfigs.value = Array.isArray(data) ? data : (data.results || [])
```

后端 API `GET /api/personal/feature-params-configs/` 返回:

```json
{"configs": [{"id": "xxx", "name": "我的配置", ...}]}
```

前端解析逻辑:
1. `Array.isArray(data)` → `false`（`data` 是对象 `{configs: [...]}`）
2. 回退到 `data.results` → `undefined`（响应中没有 `results` 字段）
3. `data.results || []` → `[]`

**`personalConfigs` 永远是空数组**，无论用户有多少个人配置。

### 对比验证

| 位置 | 代码 | 正确? |
|------|------|--------|
| `ServerConfig.logic.vue:475` | `Array.isArray(data) ? data : (data.results \|\| [])` | ❌ |
| `TaskFeatureParamsSelector.vue:51` (已废弃) | `(await resp.json()).configs \|\| []` | ✅ |
| `PersonalFeatureParamsConfigs.vue:189` (profile 页) | `(await resp.json()).configs \|\| []` | ✅ |

### 根因 2 (次要): 初始加载未获取个人配置

`initFeatureParamsSource()` (L458-465) 仅在页面加载时设置 `featureParamsSource.value`，但**不调用** `fetchPersonalConfigs()`。当任务已保存 `feature_params_source='personal'` 时，页面刷新后不会自动拉取个人配置列表。

```js
// ❌ 当前代码
const initFeatureParamsSource = () => {
  if (props.task?.feature_params_source) {
    featureParamsSource.value = props.task.feature_params_source
  }
  if (props.task?.personal_feature_params_config_id) {
    selectedPersonalConfigId.value = props.task.personal_feature_params_config_id
  }
  // 缺少: 如果 source === 'personal'，应 fetchPersonalConfigs()
}
```

---

## 解决方案

### 修复 1: 更正 API 响应字段名

**文件**: `front_project/app/src/components/ServerConfig.logic.vue`
**行号**: L475

```diff
- personalConfigs.value = Array.isArray(data) ? data : (data.results || [])
+ personalConfigs.value = data.configs || []
```

### 修复 2: 初始加载时主动获取

**文件**: `front_project/app/src/components/ServerConfig.logic.vue`
**行号**: L458-465

```diff
 const initFeatureParamsSource = () => {
   if (props.task?.feature_params_source) {
     featureParamsSource.value = props.task.feature_params_source
   }
   if (props.task?.personal_feature_params_config_id) {
     selectedPersonalConfigId.value = props.task.personal_feature_params_config_id
   }
+  if (featureParamsSource.value === 'personal') {
+    fetchPersonalConfigs()
+  }
 }
```

### 不需要改动

- **后端 API** — 无需改动。API 响应格式正确，与 profile 页面一致
- **Workspace 治理** — `allow_personal_feature_params` 是容器拉起时的运行时校验，不影响前端下拉展示
- **其他前端组件** — `PersonalFeatureParamsConfigs.vue` 和 `TaskFeatureParamsSelector.vue` 的解析是正确的

---

## Domain Concept Inventory

| 维度 | 概念 |
|------|------|
| Bounded Context | 任务协作 (Task Collaboration) |
| Key Entities | `PersonalFeatureParamsConfig` (用户个人功能参数配置) |
| Candidate Aggregates | `Task (Todo)` 聚合根，持有 `feature_params_source` + `personal_feature_params_config_id` |
| Domain Events | `FeatureParamsResolved` — 容器拉起时触发，本次修复不涉及 |

---

## Value Stream Impact

本次修复为**纯前端 Bug Fix**，不涉及价值流变更:
- 不新增/修改 value stream step
- 不引入新 field
- 不需要新测试 step（已有 E2E 测试覆盖功能参数选择流程）
- 不产生跨 stream 依赖

---

## 测试建议

### 手动验证
1. 以 `contact@daydaymoney.com` 登录
2. 确保用户在 profile 页至少有一个个人配置
3. 进入任意任务详情页 → 直接启动 → 功能参数来源 → 选择「个人配置」
4. 验证：「个人配置」二级下拉显示所有个人配置
5. 选择一个个人配置 → 点击「预览环境变量」→ 验证返回正确环境变量

### 边界场景
- 用户无个人配置时，下拉显示「暂无个人配置」提示（已有逻辑，无需改动）
- 页面刷新后，若任务已绑定 `feature_params_source='personal'`，下拉应自动填充个人配置列表

---

## 总结清单

- 根因 1 (API 解析): 字段名 `data.configs` 误写为 `data.results` — 一行修复
- 根因 2 (初始加载): `initFeatureParamsSource` 未在 source=personal 时 fetch — 一行修复
