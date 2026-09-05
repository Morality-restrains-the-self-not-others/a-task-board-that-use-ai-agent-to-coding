# Value Stream: 容器启动时功能参数来源选择

> 设计文档: `docs/designs/env-var-preset-switching.md`

## Value Summary

用户在任务详情「直接启动」面板中选择容器使用的功能参数来源（公司默认 / 工作空间默认 / 个人配置），选择后持久化到任务，容器启动后自动拉取对应层级的 LLM 配置与环境变量。

## Related Value Streams

- **feature-params-hierarchy** (`docs/superpowers/plans/2026-06-30-feature-params-hierarchy-value-stream.md`): **扩展** — 在已有三级 FeatureParams 模型 + Resolver + 管理 API 基础上，补充容器启动 UI 的来源选择入口。该流已交付 Increment 1-4（模型 + API + 前端设置页 + TaskFeatureParamsSnapshot）
- **env-vars-settings-redesign** (`docs/superpowers/plans/2026-06-30-env-vars-settings-redesign-value-stream.md`): **独立** — 该流重构了设置页（公司/工作空间/个人配置管理页）的布局，本流在任务详情启动面板新增来源选择器，不重叠

## End-to-End Flow

```text
[用户] 进入任务详情 → 点击"直接启动"tab
  → 看到功能参数来源选择器，当前值 = Todo.feature_params_source
  → 切换来源：公司默认 → 直接可用（不预览）
  → 切换来源：工作空间默认 → 直接可用（不预览）
  → 切换来源：个人配置 → 出现二级下拉（个人配置列表）+ [预览] 按钮
     → 选择具体配置 → 点击 [预览] → 看到解析后的 env 变量
  → 点击 [启动]
     → POST relay-to-trae/start/ + feature_params_source + personal_config_id
     → 后端持久化到 Todo → 启动 relayToTrae
  → 容器内部 fetch_tenant_feature_params_env_for_container()
     → 读取 Todo.feature_params_source → FeatureParamsResolver 解析 → 返回 env
```

## Value Increments

### Increment 1: 后端预览端点 + 启动参数扩展 (Thin Slice)

**Value to user:** 后端支持按来源预览和持久化，但尚无 UI

**Scope:**
- 新增 `GET .../feature-params-env-preview/` 端点
  - 仅允许 `source=personal`（company/workspace 返回 403）
  - 校验 `personal_config_id` 归属（IDOR 防护，复用 `_check_idor` 模式）
  - 调用 `FeatureParamsResolver` + `FeatureParamsEnvSerializer` 返回 env
- 修改 `POST .../relay-to-trae/start/` 
  - 接收可选 `feature_params_source` + `personal_feature_params_config_id`
  - `source=personal` 时校验 config 归属
  - 持久化到 `Todo.feature_params_source` / `personal_feature_params_config_id`
- 后端测试: `test_feature_params_env_preview.py` (5 个场景)

**Depends on:** feature-params-hierarchy Increment 1-4 (模型 + Resolver + Todo 字段已存在)

**Fields:**
- `saas-backend.projects_todo.feature_params_source` (写)
- `saas-backend.projects_todo.personal_feature_params_config_id` (写)
- `saas-backend.projects_personal_feature_params_config.user_id` (读，归属校验)

### Increment 2: 前端来源选择器 + 预览集成

**Value to user:** 在「直接启动」面板可视化选择来源，个人配置可预览

**Scope:**
- `ServerConfig.logic.vue`:
  - 新增状态: `featureParamsSource`, `personalConfigs`, `selectedPersonalConfigId`, `envPreview`
  - 新增方法: `initFeatureParamsSource()`, `fetchPersonalConfigs()`, `fetchEnvPreview()`, `onSourceChange()`
  - 修改 `startRelayToTrae()`: 传递 source 参数
- `ServerConfigRelayDirectPanel.vue`:
  - 新增来源选择器（三选一下拉）
  - 新增个人配置二级下拉（source=personal 时显示）
  - 新增 [预览] 按钮 + 折叠式 env 预览面板
- E2E 测试: `TaskDetail.feature-params-source-switching.playwright.test.js`

**Depends on:** Increment 1

**Fields:**
- `taskFE.runtime.feature_params_source_selector` — 来源选择器组件

---

*关联 YAML 待写入 `conf/value-stream.yaml`。*
