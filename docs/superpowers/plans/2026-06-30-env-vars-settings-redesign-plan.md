# 实施计划: 环境变量设置页面重设计

> 输入:
> - 设计文档: `docs/designs/env-vars-settings-redesign.md`
> - 价值流: `docs/superpowers/plans/2026-06-30-env-vars-settings-redesign-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-30-env-vars-settings-redesign-nfr-clarification.md`
> - DDD: 跳过（纯前端变更，无新领域概念）

---

## 任务清单

### Increment 1: 增强版环境变量编辑器 + 后端校验

#### 1.1 后端: env var key 校验函数

- [ ] **Task 1.1.1**: 在 `task2app/Saas_project/projects/services/tenant_feature_params_providers.py` 新增 `validate_env_var_key(key: str)` 函数
  - 校验格式 `^[A-Z_][A-Z0-9_]*$`，2-128 字符
  - 校验保留字冲突: `TASK_*`, `ACCESS_TOKEN`, `BUSINESS_API_*` 前缀
  - 返回 `(is_valid: bool, error_message: str)`
  - 若 key 以保留字前缀开头 → `"TASK_* 为系统保留变量名前缀，请使用其他名称"`
  - 若 key 格式非法 → `"变量名仅允许大写字母、数字和下划线，且必须以字母或下划线开头"`
- [ ] **Task 1.1.2**: 新增 `validate_env_vars_list(env_vars: list)` 函数
  - 遍历每个 entry，调用 `validate_env_var_key`
  - 检测重复 key（后者覆盖前者，WARN 级别日志）
  - 返回 `(is_valid, errors: list[str])`
- [ ] **Task 1.1.3**: 新增测试文件 `tests/test_env_var_validation.py`
  - 测试合法 key: `DEBUG`, `MY_VAR`, `LOG_LEVEL_2`
  - 测试非法 key: `task_var` (小写), `123VAR` (数字开头), `MY-VAR` (连字符), `TASK_LLM_PROVIDERS_JSON` (保留字)
  - 测试边界: 1 字符、128 字符、空字符串、纯下划线
  - 测试重复 key 检测

#### 1.2 后端: extra_env_vars 序列化支持 description

- [ ] **Task 1.2.1**: 在 `feature_views.py` 的 `manage_feature_params` POST 中调用 `validate_env_vars_list`
  - 校验失败返回 400
  - description 字段透传（可选，0-256 字符）
- [ ] **Task 1.2.2**: 在 `workspace_feature_params_views.py` 的 `manage_workspace_feature_params` POST 中同样处理
- [ ] **Task 1.2.3**: 在 `personal_feature_params_views.py` 的 POST/PUT 中同样处理
- [ ] **Task 1.2.4**: 更新 `test_manage_feature_params.py` — 增加 description 字段读写 + 非法 key 拒绝测试

#### 1.3 前端: EnvVarTableEditor 组件

- [ ] **Task 1.3.1**: 新建 `components/EnvVarTableEditor.vue`
  - 三列表格: 变量名 (KEY) | 变量值 (VALUE) | 说明 (description)
  - Props: `modelValue: Array<{key, value, description}>`
  - Emits: `update:modelValue`
  - 变量名输入框实时校验: 非法字符红色边框 + tooltip
  - 保留字冲突: 黄色警告 tooltip "此变量由系统管理，自定义设置可能被覆盖"
  - 添加行 / 删除行按钮
  - 空 key 行过滤不 emit
  - **敏感值遮蔽**: 检测 value 包含 `token|key|secret|password|api_key` 时默认显示 `●●●●`，点击眼睛图标切换
  - **导入 .env**: 按钮 → 弹窗 textarea → 解析 `KEY=VALUE\n` 格式 → 合并到现有列表
  - **导出 .env**: 按钮 → 生成 `KEY=VALUE\n` 文本 → 触发浏览器下载
  - **模板建议**: 变量名输入框 focus 时下拉显示常用变量名 (`DEBUG_AGENT`, `LOG_LEVEL`, `CUSTOM_ENDPOINT`, `TIMEOUT_SEC`, `MAX_RETRIES`, `API_BASE_URL`)
- [ ] **Task 1.3.2**: 保留旧 `EnvKeyValueEditor.vue` 不动（feature-params 页面仍可用），新组件用新文件名

### Increment 2: 页面布局重设计

#### 2.1 新组件

- [ ] **Task 2.1.1**: 新建 `components/CollapsibleLLMConfigPanel.vue`
  - Props: `providers`, `agentModel`, `summaryModel`, `agentModelProvider`, `summaryModelProvider`, `agentMaxSteps`, `subTokenProviders`, `recommendedProviders`
  - 默认折叠，点击标题栏展开/收起
  - 标题栏显示 "▶ LLM 供应商 & 模型配置" / "▼ LLM 供应商 & 模型配置"
  - 内部包裹: 推荐供应商列表 + FeatureParamsProvidersEditor + 模型选择 grid + FeatureParamsWorkspaceBudgetPanel
  - 从当前 `WorkspaceSettingsFeatureParams.vue` 提取 LLM 相关模板到此处
- [ ] **Task 2.1.2**: 新建 `components/MergedEnvPreview.vue`
  - Props: `systemEnv: Record<string, string>`, `userEnvVars: Array<{key, value}>`
  - 合并展示: 先展示系统变量（灰色背景），分隔线，再展示用户变量
  - 系统变量区带标签 "系统自动生成"
  - 用户变量区带标签 "用户自定义"
  - 复制按钮: 复制全部合并后的 `KEY=VALUE\n` 文本
  - 高度自适应（至少 400px）

#### 2.2 租户级页面重写

- [ ] **Task 2.2.1**: 重写 `views/WorkspaceSettingsFeatureParams.vue`
  - 标题改为 "环境变量设置"
  - 副标题: "这些变量将在容器启动时注入。LLM 相关变量由系统根据下方配置自动生成。"
  - 布局: EnvVarTableEditor (主体) → MergedEnvPreview (预览) → CollapsibleLLMConfigPanel (折叠) → 保存按钮
  - `buildLocalEnvPreview()` 修复: 合并 extra_env_vars
  - 推荐供应商面板移入 CollapsibleLLMConfigPanel 内部
  - 移除旧的 `showProvidersPanel` 逻辑
- [ ] **Task 2.2.2**: 页面加载时从 `extra_env_vars` 填充 EnvVarTableEditor（含 description 字段）
- [ ] **Task 2.2.3**: 保存时确保 `extra_env_vars` 含 description 字段

#### 2.3 工作空间级页面重写

- [ ] **Task 2.3.1**: 重写 `views/WorkspaceFeatureParamsSettings.vue`
  - 同 2.2 布局（env 主体 + LLM 折叠 + 预览）
  - 保留 "允许个人配置" 治理开关和 "使用公司默认" 切换
  - 当 `use_company_default=true` 时，env 编辑器只读（显示公司配置）
  - 当 `use_company_default=false` 时，env 编辑器可编辑
  - 继承可视化: 显示哪些变量来自公司默认、哪些被工作空间覆盖

#### 2.4 个人级页面重写

- [ ] **Task 2.4.1**: 重写 `views/PersonalFeatureParamsConfigs.vue`
  - 列表页保持（配置列表 + 新建/编辑/删除/复制）
  - 编辑弹窗内布局同 2.2（env 主体 + LLM 折叠 + 预览）
  - 弹窗标题显示配置名称

#### 2.5 导航

- [ ] **Task 2.5.1**: 编辑 `components/Sidebar.vue`
  - 菜单文字: `公司功能参数` → `环境变量`
  - 路由不变: `/tenant/:tenant/settings/feature-params/`

### Increment 3: 后端 env_preview 修复

- [ ] **Task 3.1**: 修复 `cloud/services/tenant_feature_params_env.py` 的 `build_tenant_feature_params_env()`
  - 确保 `env` dict 包含 extra_env_vars 的所有有效条目
  - 用户变量可覆盖同名系统变量 (user wins)
- [ ] **Task 3.2**: 修复 `feature_views.py` `manage_feature_params` GET 的 `env_preview`
  - `env_preview` 目前仅含 TASK_* 变量，补全 extra_env_vars
- [ ] **Task 3.3**: 修复 `workspace_feature_params_views.py` `manage_workspace_feature_params` GET 的 `env_preview`
- [ ] **Task 3.4**: 修复 `personal_feature_params_views.py` 的列表/详情 `env_preview`
- [ ] **Task 3.5**: 更新 `tests/test_tenant_feature_params_env.py` — 验证 env_preview 含 extra_env_vars
- [ ] **Task 3.6**: 更新 `tests/test_manage_feature_params.py` — 验证 GET env_preview 完整性

### Increment 4: E2E 测试

- [ ] **Task 4.1**: 新建 Playwright 测试 `front_project/tests/env-vars-editor.spec.ts`
  - **添加变量**: 输入 key + value + description → 保存 → 刷新 → 验证三列数据回显
  - **校验错误**: 输入非法 key (如 `task_var`) → 验证红色边框 + 保存时后端拒绝
  - **保留字警告**: 输入 `TASK_MY_VAR` → 验证黄色 warning tooltip 出现
  - **导入 .env**: 粘贴 `DEBUG=true\nLOG_LEVEL=info\n` → 验证解析为 2 行
  - **导出 .env**: 点击导出 → 验证下载文件内容正确
  - **LLM 折叠**: 点击展开 → 配置供应商 → 保存 → 验证供应商回显
  - **预览完整性**: 配置供应商 + 添加 env var → 验证预览 textarea 同时包含 TASK_* + 用户变量
  - **敏感值遮蔽**: 输入 value 含 `api_key` → 验证默认显示 `●●●●`
- [ ] **Task 4.2**: 更新 `test_manage_feature_params.py` Playwright 替代测试（如有）验证新 UI 流程

---

## 文件变更汇总

| 文件 | 操作 | 行数估算 |
|------|------|---------|
| `components/EnvVarTableEditor.vue` | **新建** | ~200 |
| `components/CollapsibleLLMConfigPanel.vue` | **新建** | ~120 |
| `components/MergedEnvPreview.vue` | **新建** | ~80 |
| `views/WorkspaceSettingsFeatureParams.vue` | **重写** | ~250 (从 ~420 缩减) |
| `views/WorkspaceFeatureParamsSettings.vue` | **重写** | ~250 |
| `views/PersonalFeatureParamsConfigs.vue` | **重写** | ~300 |
| `components/Sidebar.vue` | **编辑** | +1 行 |
| `projects/services/tenant_feature_params_providers.py` | **编辑** | +40 |
| `projects/views/feature_views.py` | **编辑** | +15 |
| `projects/views/workspace_feature_params_views.py` | **编辑** | +15 |
| `projects/views/personal_feature_params_views.py` | **编辑** | +20 |
| `cloud/services/tenant_feature_params_env.py` | **编辑** | +10 |
| `tests/test_env_var_validation.py` | **新建** | ~80 |
| `tests/test_manage_feature_params.py` | **编辑** | +30 |
| `tests/test_tenant_feature_params_env.py` | **编辑** | +20 |
| `front_project/tests/env-vars-editor.spec.ts` | **新建** | ~150 |

总新增/变更: ~1600 行，所有文件 ≤250 行（符合 ≤500 约束）。

## 执行顺序

```
1.1 (后端校验) → 1.2 (后端 description) → 1.3 (前端编辑器)
                                               ↓
2.1 (新组件) ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ┘
  ↓
2.2 (租户页面) → 2.3 (工作空间) → 2.4 (个人) → 2.5 (导航)
  ↓
3.x (后端 preview 修复)
  ↓
4.x (E2E 测试)
```

Increment 1.1/1.2 可与 1.3 并行（前后端独立）。Increment 2.x 依赖 1.3。Increment 3.x 依赖 2.x 验证页面正确连线。

## 验证命令

```bash
# 后端测试
cd task2app/Saas_project
source activate_env.sh
pytest tests/test_env_var_validation.py -v
pytest tests/test_manage_feature_params.py -v
pytest tests/test_tenant_feature_params_env.py -v

# E2E 测试
cd task2app/front_project
npx playwright test tests/env-vars-editor.spec.ts --headed
```
