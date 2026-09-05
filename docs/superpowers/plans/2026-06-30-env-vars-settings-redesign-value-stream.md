# Value Stream: 环境变量设置页面重设计

> 设计文档: `docs/designs/env-vars-settings-redesign.md`

## Value Summary

租户/工作空间/个人三级环境变量设置页面重布局：环境变量管理提升为主体内容（80%空间），LLM 供应商配置折叠为附属模块；增强编辑器支持校验、.env 导入导出、保留字警告；修复 env 预览不包含用户自定义变量的 bug。

## Related Value Streams

- **feature-params-hierarchy** (`conf/value-stream.yaml` L2623, `docs/superpowers/plans/2026-06-30-feature-params-hierarchy-value-stream.md`): **修改** — Increment 3（前端 UI）的布局从「LLM 主体 + env 附属」改为「env 主体 + LLM 折叠」
- **feature-params-env-agent-agnostic** (`docs/superpowers/plans/2026-05-29-feature-params-env-agent-agnostic-value-stream.md`): **扩展** — 其 Increment 3 的前端 env 预览有 bug（不含 extra_env_vars），本次修复

## End-to-End Flow

```text
[管理员] 打开"环境变量"设置页
  → 看到环境变量表格为主体（可添加/导入/导出/校验）
  → 可点击折叠面板展开 LLM 配置
  → 底部预览区显示完整 env（系统 TASK_* + 用户自定义）
  → 保存 → 同一 feature-params API
  → 容器启动时拉取 env → 收到完整的合并 env map
```

## Value Increments

### Increment 1: 增强版环境变量编辑器 (Thin Slice)

**Value to user:** 用户可在独立、专业的编辑器中管理环境变量，支持校验和导入导出

**Scope:**
- `EnvKeyValueEditor.vue` → `EnvVarTableEditor.vue` 重构
  - 三列布局: key + value + description
  - 实时校验: key 格式 `^[A-Z_][A-Z0-9_]*$`，非法字符红色提示
  - 保留字警告: `TASK_*` / `ACCESS_TOKEN` / `BUSINESS_API_*` 黄色 tooltip
  - 导入 .env: textarea 弹窗解析 `KEY=VALUE` → 多行
  - 导出 .env: 下载 `KEY=VALUE\n` 文件
  - 敏感值遮蔽: value 含 token/key/secret/password 时 `●●●●`
  - 空值处理: key 为空过滤，value 可为空字符串
- 后端: `extra_env_vars` 序列化/反序列化支持 `description` 字段
- 后端: 新增 key 格式校验函数 (`validate_env_var_key`)
- 后端: 新增保留字冲突检测 (`check_reserved_key_conflict`)

**Depends on:** 无

**Fields:**
- `taskFE.runtime.env_var_table_editor` — 增强版编辑器组件
- `saas-backend.projects_tenant_feature_params.extra_env_vars` — schema 扩展 description

### Increment 2: 页面布局重设计

**Value to user:** 环境变量成为页面的视觉主体，LLM 配置退居附属

**Scope:**
- `CollapsibleLLMConfigPanel.vue` — 新建折叠面板组件，包裹 LLM 配置内容（默认折叠）
- `MergedEnvPreview.vue` — 新建完整预览组件，系统变量 + 用户变量合并展示，分隔线标识来源
- `WorkspaceSettingsFeatureParams.vue` → 重布局: 标题改"环境变量设置"，env 编辑器主体 + 预览 + LLM 折叠
- `WorkspaceFeatureParamsSettings.vue` → 同布局调整（工作空间级）
- `PersonalFeatureParamsConfigs.vue` → 编辑弹窗内同布局调整（个人级）
- `Sidebar.vue` → 菜单文字 "公司功能参数" → "环境变量"
- 修复 `buildLocalEnvPreview`: 合并 extra_env_vars 到预览中

**Depends on:** Increment 1

**Fields:**
- `taskFE.runtime.collapsible_llm_config_panel` — LLM 折叠面板
- `taskFE.runtime.merged_env_preview` — 合并 env 预览
- `taskFE.runtime.tenant_env_vars_settings_page` — 租户级页面
- `taskFE.runtime.workspace_env_vars_settings_page` — 工作空间级页面
- `taskFE.runtime.personal_env_vars_configs_page` — 个人级页面

### Increment 3: 后端 env_preview 修复 + 校验强化

**Value to user:** API 返回的 env_preview 完整反映容器将收到的所有变量

**Scope:**
- `feature_views.py` `manage_feature_params` GET: env_preview 合并 extra_env_vars
- `workspace_feature_params_views.py` `manage_workspace_feature_params` GET: 同上
- `personal_feature_params_views.py` 列表/详情: 同上
- `tenant_feature_params_env.py` `build_tenant_feature_params_env`: 确保 extra_env_vars 注入
- 更新 `test_manage_feature_params.py`: 验证 env_preview 含 extra_env_vars
- 更新 `test_tenant_feature_params_env.py`: 验证合并逻辑
- 新增 `test_env_var_validation.py`: key 格式校验 + 保留字冲突

**Depends on:** Increment 1

**Fields:**
- `saas-backend.projects_tenant_feature_params.extra_env_vars` — env_preview 补全
- `saas-backend.projects_workspace_feature_params.extra_env_vars` — 同上
- `saas-backend.projects_personal_feature_params_config.extra_env_vars` — 同上

### Increment 4: E2E 测试

**Value to user:** 生产就绪，完整用户流程验证

**Scope:**
- Playwright E2E: `front_project/tests/env-vars-editor.spec.ts`
  - 添加变量 → 校验错误提示 → 修正 → 保存 → 刷新后验证加载
  - 导入 .env 文本 → 验证解析为多行
  - 导出 .env → 验证下载文件内容
  - 展开 LLM 折叠面板 → 配置供应商 → 保存
  - 验证预览区包含 TASK_* + 用户变量
  - 验证保留字输入时出现警告

**Depends on:** Increment 2, Increment 3

## 验收标准

| 增量 | 验收 |
|------|------|
| 1 | EnvVarTableEditor 组件可独立渲染、校验、导入导出；后端校验拒绝非法 key 和保留字 |
| 2 | 三个页面（租户/工作空间/个人）均以 env 为主体、LLM 折叠；预览含完整变量 |
| 3 | pytest 通过：env_preview 含 extra_env_vars、key 校验覆盖边界 |
| 4 | Playwright E2E 通过完整用户流程 |
