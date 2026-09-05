# 环境变量设置页面重设计

> **状态**: 头脑风暴 / 待审批
> **日期**: 2026-06-30
> **来源**: `/1-brainstorming-设计文档`

---

## 1. 问题分析

### 1.1 当前状态

当前 `/tenant/:tenant/settings/feature-params/` 页面 (`WorkspaceSettingsFeatureParams.vue`)：

- **页面标题**: "功能参数设置"
- **主体 (~80%)**: LLM 供应商列表 + 模型选择 + 预算面板
- **附属 (~10%)**: `EnvKeyValueEditor` — 额外环境变量键值对编辑器，缩在页面底部
- **预览**: 仅展示 `TASK_*` 标准变量，不包含 `extra_env_vars`（bug）
- **心智模型**: 这是一个「LLM 配置页」，环境变量是附件

用户的实际诉求是设置**自己的环境变量注入到容器**。当前页面没有给这个核心需求足够的空间和工具。

### 1.2 容器注入链路（保持不变）

```
用户配置 env vars → TenantFeatureParams.extra_env_vars
                         ↓
容器启动时调用 POST .../server-container-token/feature-params-env/
                         ↓
FeatureParamsEnvSerializer: 合并 TASK_* 标准变量 + extra_env_vars
                         ↓
docker run -e KEY=VALUE ...
```

---

## 2. 设计方案：环境变量为主体，LLM 配置为附属

### 2.1 核心思路

**不拆分页面、不新增 API** — 在现有 `feature-params` 页面内重新布局：

- **主体空间 (80%)**: 环境变量管理 — 结构化表格、导入导出、校验、预览
- **附属模块 (20%)**: LLM 供应商配置 — 可折叠/展开的手风琴面板，默认折叠

```
┌─────────────────────────────────────────────────────────────┐
│  环境变量设置                                                │
│  这些变量将在容器启动时注入。LLM 相关变量由系统自动生成。      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─ 用户自定义变量 ──────────────────────────────────────┐  │
│  │  [+ 添加]  [📥 导入 .env]  [📤 导出]                   │  │
│  │                                                       │  │
│  │  KEY              VALUE              说明       操作   │  │
│  │  ──────────────────────────────────────────────────   │  │
│  │  DEBUG_AGENT      true               调试模式    [✕]   │  │
│  │  LOG_LEVEL        debug              日志级别    [✕]   │  │
│  │  MY_ENDPOINT      https://...        [空]        [✕]   │  │
│  │  TIMEOUT_SEC      300                [空]        [✕]   │  │
│  │                                                       │  │
│  │  [+ 添加变量]                                          │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌─ 环境变量完整预览 (合并视图) ─────────────────────────┐  │
│  │  TASK_LLM_PROVIDERS_JSON=[...]    ← 系统自动生成       │  │
│  │  TASK_AGENT_MODEL=gpt-4.1          ← 系统自动生成       │  │
│  │  TASK_AGENT_MODEL_PROVIDER=openai  ← 系统自动生成       │  │
│  │  TASK_AGENT_MAX_STEPS=200          ← 系统自动生成       │  │
│  │  TASK_SUMMARY_MODEL=gpt-4.1-mini   ← 系统自动生成       │  │
│  │  TASK_SUMMARY_MODEL_PROVIDER=openai ← 系统自动生成      │  │
│  │  ───────────────────────────────────────────────────    │  │
│  │  DEBUG_AGENT=true                  ← 用户自定义         │  │
│  │  LOG_LEVEL=debug                   ← 用户自定义         │  │
│  │  MY_ENDPOINT=https://...           ← 用户自定义         │  │
│  │  TIMEOUT_SEC=300                   ← 用户自定义         │  │
│  │                                                       │  │
│  │  [📋 复制到剪贴板]                                      │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  [保存]  [重置]           保存状态: ✓ 已保存                │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ▶ LLM 供应商 & 模型配置 (点击展开)           ← 默认折叠   │
│                                                             │
│     ┌─ 展开后内容 ─────────────────────────────────────┐   │
│     │  供应商列表 (FeatureParamsProvidersEditor)         │   │
│     │  模型选择 (agent / summary)                        │   │
│     │  迭代次数 / 预算面板                                │   │
│     └──────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 与当前页面的差异对比

| 维度 | 当前 | 设计后 |
|------|------|--------|
| **页面标题** | "功能参数设置" | "环境变量设置" |
| **侧栏菜单名** | "公司功能参数" | "环境变量" |
| **主体内容** | LLM 供应商列表 (默认展开) | 环境变量表格 (主体) |
| **LLM 配置** | 主体，始终可见 | 折叠面板，点击展开 |
| **env 预览** | 仅 TASK_* 变量 | TASK_* + 用户自定义变量完整合并视图 |
| **编辑器** | 裸 key-value 两列 | 三列 (key + value + 说明) + 校验 + 导入/导出 |
| **推荐供应商** | 可折叠面板 (默认展开) | 移至 LLM 折叠面板内 |
| **路由** | `/settings/feature-params/` | 保持不变 (仅前端组件重写) |
| **API** | `GET/POST /api/tenant/{id}/feature-params/` | 保持不变 |

### 2.3 增强版环境变量编辑器

当前 `EnvKeyValueEditor.vue` → 重构为 `EnvVarTableEditor.vue`:

| 特性 | 说明 |
|------|------|
| **三列布局** | `变量名` \| `变量值` \| `说明`，提交时仍为 `[{key, value, description}]` |
| **变量名校验** | 实时校验 `^[A-Z_][A-Z0-9_]*$`，非法字符红色边框 + tooltip |
| **保留字警告** | 输入 `TASK_*` / `ACCESS_TOKEN` / `BUSINESS_API_*` 前缀时给出黄色警告："此变量由系统管理，自定义设置可能被覆盖" |
| **导入 .env** | 按钮触发 textarea 弹窗，粘贴 `KEY=VALUE` 文本，解析为多行 |
| **导出 .env** | 一键生成 `KEY=VALUE\n` 文本并下载为 `.env` 文件 |
| **模板建议** | 输入框 placeholder 下拉: `DEBUG_AGENT`、`LOG_LEVEL`、`CUSTOM_ENDPOINT`、`TIMEOUT_SEC` 等常用变量名 |
| **敏感值遮蔽** | value 包含 `token\|key\|secret\|password\|api_key` 时默认 `●●●●` 遮蔽，点击切换 |
| **空值允许** | key 为空的行过滤不提交；value 可以为空字符串（有意清空） |
| **拖拽排序** | 可选（nice-to-have，不影响功能） |

### 2.4 LLM 折叠面板内部布局

```
▶ LLM 供应商 & 模型配置 (点击展开)               ← 默认折叠

  展开后:
  ┌──────────────────────────────────────────────┐
  │  ┌─ 推荐供应商 ──────────────────────────┐  │
  │  │  [查看文档链接 x 3]   [展开更多]        │  │
  │  │  免责声明                                │  │
  │  └──────────────────────────────────────┘  │
  │                                            │
  │  供应商列表:                                │
  │  ┌──────────────────────────────────────┐  │
  │  │ 提供商标识 │ API Key │ Base URL │ ... │  │
  │  │  [ + 添加供应商 ]                    │  │
  │  └──────────────────────────────────────┘  │
  │                                            │
  │  模型配置 (水平双栏):                        │
  │  ┌──────────────────────┐ ┌──────────────┐  │
  │  │ 🤖 智能体模型        │ │ 📋 摘要模型   │  │
  │  │   模型名称 [____]    │ │   模型 [____] │  │
  │  │   供应商   [____]    │ │   供应商[____]│  │
  │  │   迭代次数 [____]    │ │              │  │
  │  └──────────────────────┘ └──────────────┘  │
  │                                            │
  │  [LLM 预算面板]                             │
  └──────────────────────────────────────────────┘
```

推荐供应商面板**移入** LLM 折叠区域内，不再占据页面顶部独立空间。

### 2.5 预览区域

预览面板改为展示**完整合并视图** — 系统变量 + 用户变量：

```javascript
// 当前 buildLocalEnvPreview (bug: 不含 extra_env_vars)
const buildLocalEnvPreview = () => {
  // 只返回 TASK_* 变量
}

// 修复后: 合并系统变量 + 用户变量
const buildLocalEnvPreview = () => {
  const system = {
    TASK_LLM_PROVIDERS_JSON: JSON.stringify(providers),
    TASK_AGENT_MODEL: form.agent_model,
    TASK_AGENT_MODEL_PROVIDER: form.agent_model_provider,
    TASK_AGENT_MAX_STEPS: form.agent_max_steps,
    TASK_SUMMARY_MODEL: form.summary_model,
    TASK_SUMMARY_MODEL_PROVIDER: form.summary_model_provider,
  }
  const user = {}
  ;(form.extra_env_vars || []).forEach(e => {
    if (e.key) user[e.key] = e.value || ''
  })
  return { ...system, ...user }  // 用户变量可覆盖系统变量 (user wins)
}
```

预览 textarea 中用分隔线区分系统变量 / 用户变量区域，视觉上清晰标识来源。

### 2.6 组件树对比

**当前**:
```
WorkspaceSettingsFeatureParams.vue
├── 推荐供应商面板 (默认展开，占据顶部独立卡片)
├── FeatureParamsProvidersEditor (供应商列表)
├── 模型选择 grid (4 个 input)
├── EnvKeyValueEditor (底部，弱存在感)          ← 2 列 key-value
├── FeatureParamsWorkspaceBudgetPanel
├── 保存/复制按钮
└── 环境变量预览 textarea (仅 TASK_*)
```

**设计后**:
```
TenantEnvVarsSettings.vue (重命名，标题"环境变量设置")
├── EnvVarTableEditor (主体，占据主要空间)       ← 3 列 + 校验 + 导入导出
├── 环境变量完整预览 (合并视图，TASK_* + 用户)   ← 修复 bug
├── 保存/重置按钮
└── CollapsibleLLMConfigPanel (折叠面板)         ← 默认折叠
    ├── 推荐供应商 (移入此处)
    ├── FeatureParamsProvidersEditor
    ├── 模型选择
    └── FeatureParamsWorkspaceBudgetPanel
```

### 2.7 数据流 (不变)

```
form = {
  providers: [...],       // ← 仍在 LLM 折叠面板内编辑
  agent_model: '',
  agent_model_provider: '',
  agent_max_steps: '200',
  summary_model: '',
  summary_model_provider: '',
  extra_env_vars: [       // ← 新 EnvVarTableEditor 编辑，主体位置
    { key: 'X', value: 'Y', description: 'Z' }
  ],
}

POST /api/tenant/{id}/feature-params/   ← API 不变
```

`extra_env_vars` JSON 格式从 `[{key, value}]` 扩展为 `[{key, value, description}]` — `description` 是新增可选字段。旧数据无 description 时 editor 显示为空，向后兼容。

### 2.8 导航调整

`Sidebar.vue`:
- 菜单文字: `公司功能参数` → `环境变量`
- 路由不变: `/tenant/:tenant/settings/feature-params/`
- (可选) 加一个 `aliases` 路由 `/tenant/:tenant/settings/env-vars/` 指向同一组件，方便 URL 可读性

### 2.9 工作空间级 & 个人级

同样的布局调整应用到：

| 页面 | 当前组件 | 调整后 |
|------|---------|--------|
| 租户级 | `WorkspaceSettingsFeatureParams.vue` | 重命名为 `TenantEnvVarsSettings.vue`，主体=env vars |
| 工作空间级 | `WorkspaceFeatureParamsSettings.vue` | 同布局，主体=env vars，LLM 折叠 |
| 个人级 | `PersonalFeatureParamsConfigs.vue` | 同布局，列表 + 编辑弹窗中 env vars 为主体 |

---

## 3. 价值流影响

现有价值流 `feature-params-hierarchy` 所有步骤标记 `planned`。本次重设计：

| 步骤 | 影响 |
|------|------|
| `fix-company-admin-guard` | 无影响 |
| `workspace-personal-config-governance` | 无影响 |
| `workspace-feature-params-model` | 无影响 (数据模型不变) |
| `personal-feature-params-model` | 无影响 |
| `feature-params-resolution` | 无影响 |
| `feature-params-snapshot-audit` | 无影响 |
| `task-feature-params-binding` | 无影响 |

**新增字段影响** (三段式):
- `taskFE.runtime.env_var_table_editor` — 增强版编辑器组件
- `saas-backend.projects_tenant_feature_params.extra_env_vars` — schema 扩展 `description` 字段

**测试影响**:
- 更新: `tests/test_manage_feature_params.py` (验证 extra_env_vars 含 description 的读写)
- 更新: `tests/test_tenant_feature_params_env.py` (验证 env_preview 包含 extra_env_vars)
- 新增: `tests/test_env_var_validation.py` (key 格式校验、保留字冲突)
- 新增: E2E `front_project/tests/env-vars-editor.spec.ts` (导入 .env / 导出 / 折叠面板)

---

## 4. 领域概念清单

| 有界上下文 | 概念 | 说明 |
|-----------|------|------|
| 特征参数 | `EnvironmentVariable` (VO) | `{key, value, description}` — 扩展后结构 |
| 特征参数 | `SystemReservedKeys` (VO) | 保留变量名集合 (`TASK_*`, `ACCESS_TOKEN`, `BUSINESS_API_*`) |
| 特征参数 | `FeatureParamsEnvSerializer` (已有领域服务) | 合并 TASK_* + extra_env_vars（逻辑不变，已支持） |
| 特征参数 | `TenantFeatureParams` (已有聚合) | `extra_env_vars` 字段承载环境变量 |

不引入新聚合或实体，仅在 VO 层面扩展 `EnvironmentVariable` 结构。

---

## 5. 实现概览

### 阶段 1: 组件重构 (前端)

1. `EnvKeyValueEditor.vue` → `EnvVarTableEditor.vue` (三列 + 校验 + 导入导出 + 保留字警告)
2. 新建 `CollapsibleLLMConfigPanel.vue` — 折叠面板，包裹现有 LLM 配置内容
3. 新建 `MergedEnvPreview.vue` — 完整合并预览 (系统 + 用户，分隔线标识来源)
4. 重写 `WorkspaceSettingsFeatureParams.vue` → 新布局：env vars 主体 + LLM 折叠 + 合并预览
5. 重写 `WorkspaceFeatureParamsSettings.vue` → 同布局
6. 重写 `PersonalFeatureParamsConfigs.vue` → 编辑弹窗内同布局
7. `Sidebar.vue` — 菜单文字改 "环境变量"

### 阶段 2: 后端增强

8. `extra_env_vars` 序列化/反序列化支持 `description` 字段 (validator 扩展，零 migration)
9. 新增 key 格式校验: `^[A-Z_][A-Z0-9_]*$`、禁止保留字前缀
10. `env_preview` 响应补全 extra_env_vars (修复现有 bug)

### 阶段 3: 测试

11. 更新 `test_manage_feature_params.py` — description 字段读写
12. 更新 `test_tenant_feature_params_env.py` — preview 含 extra_env_vars
13. 新增 `test_env_var_validation.py` — 校验规则
14. Playwright E2E `env-vars-editor.spec.ts`

---

## 6. 边界与约束

| 约束 | 处理 |
|------|------|
| API 不变 | 仍用 `GET/POST /api/tenant/{id}/feature-params/` |
| 数据模型不变 | `extra_env_vars` JSONField，仅扩展 schema |
| 零 migration | JSONField schema 变化不触发 Django migration |
| 向后兼容 | 旧 `extra_env_vars` 数据 (无 description) 正常加载，新 editor 处理缺失 |
| 文件行数 ≤500 | 拆分 EnvVarTableEditor / CollapsibleLLMConfigPanel / MergedEnvPreview 各独立文件 |

---

## 7. 关键代码变更清单

### 前端文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `views/WorkspaceSettingsFeatureParams.vue` | **重写** | 新布局：env 主体 + LLM 折叠 |
| `components/EnvKeyValueEditor.vue` | **重写** → `EnvVarTableEditor.vue` | 三列 + 校验 + 导入导出 |
| `components/CollapsibleLLMConfigPanel.vue` | **新建** | 折叠面板包装 LLM 配置 |
| `components/MergedEnvPreview.vue` | **新建** | 系统 + 用户合并预览 |
| `views/WorkspaceFeatureParamsSettings.vue` | **重写** | 同布局调整 |
| `views/PersonalFeatureParamsConfigs.vue` | **重写** | 编辑弹窗布局调整 |
| `components/Sidebar.vue` | **编辑** | 菜单文字 "公司功能参数" → "环境变量" |

### 后端文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `projects/views/feature_views.py` | **编辑** | env_preview 补全 extra_env_vars |
| `projects/views/workspace_feature_params_views.py` | **编辑** | env_preview 补全 |
| `projects/views/personal_feature_params_views.py` | **编辑** | env_preview 补全 |
| `projects/services/tenant_feature_params_providers.py` | **编辑** | 新增 key 校验函数 |
| `cloud/services/tenant_feature_params_env.py` | **编辑** | ensure extra_env_vars in preview |

---

## 总结清单

- **布局策略**: 环境变量为主体 (80%)，LLM 配置折叠为附属 (20%)，一页内完成
- **API 策略**: 复用现有 feature-params 端点，不新增路由/API — 改动最小化
- **编辑器增强**: 三列 (key+value+description)、校验、导入/导出 .env、保留字警告、敏感值遮蔽
- **预览修复**: 合并显示系统变量 + 用户变量，用户变量可覆盖系统变量
- **导航**: 菜单改名为 "环境变量"，路由不变
- **零迁移**: 纯前端重布局 + 后端 validator 扩展，不碰数据库 schema
