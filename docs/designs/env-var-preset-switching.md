# 设计文档：容器启动时功能参数来源选择

## 元信息

- **日期**: 2026-07-01
- **状态**: 待审批
- **关联页面**: `task-detail/{id}/?relayToTrae=true` 的「镜像」区域（功能参数来源；原在「直接启动」面板）
- **修正记录**: v1（误解为预设保存/切换）→ v2（误解为全层级预览）→ v3（明确：仅个人配置可预览，公司/工作空间不可预览）→ v4（2026-07-12：UI 从「直接启动」迁入「镜像」卡片）

---

## 1. 问题

### 1.1 已有能力

系统已有完整的三级功能参数（Feature Params）体系：

| 层级 | 模型 | 存储 |
|------|------|------|
| 公司级 | `TenantFeatureParams` | providers, agent_model, ..., extra_env_vars |
| 工作空间级 | `WorkspaceFeatureParams` | 同上 + use_company_default 开关 |
| 个人级 | `PersonalFeatureParamsConfig` | 同上 + user_id + name |

`Todo` 模型已有字段：

```python
feature_params_source = CharField(choices=['company','workspace','personal'], default='company')
personal_feature_params_config_id = CharField(...)
```

容器内部通过 `fetch_tenant_feature_params_env_for_container()` 读取 `Todo.feature_params_source`，由 `FeatureParamsResolver` 解析出最终环境变量。

### 1.2 缺失

「直接启动」面板 **完全没有暴露来源选择**。用户只能看到 4 个硬编码输入框（TASK_API_ENDPOINT_ORIGIN 等），不知道、也无法切换容器启动后将使用哪一级的功能参数。

---

## 2. 需求

在「镜像」区域增加**功能参数来源选择器**（任务详情页始终展示，不依赖 `relayToTrae=true`；与镜像选择同卡）：

1. **三选一下拉**：公司默认 / 工作空间默认 / 个人配置
2. **仅个人配置可预览**：选择个人配置后，可点击预览按钮查看解析出的环境变量（LLM 模型、providers、extra_env_vars 等）；公司和工作空间默认不可预览
3. **启动时持久化**：将选择写入 `Todo.feature_params_source`（及 `personal_feature_params_config_id`）
4. **容器自动拉取**：容器内部已有的 `fetch_tenant_feature_params_env_for_container()` 自然读取该设置

---

## 3. 设计

### 3.1 UI 布局

```
┌─────────────────────────────────────────────────┐
│  镜像                                            │
│  ┌──────────────────────────────┐                │
│  │  trae0630 …              [▼] │  ← 镜像选择    │
│  └──────────────────────────────┘                │
│  环境变量参数：                                   │
│  ┌──────────────────────────────┐                │
│  │  -- 请选择环境变量参数 -- [▼] │  ← 须先选择    │
│  └──────────────────────────────┘                │
│  （选择"个人配置"时出现二级下拉 + 预览）           │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│  直接启动                                        │
│  ┌────────────────────────────────────────────┐  │
│  │  relayToTrae 服务：在线  │  online：未启动  │  │
│  └────────────────────────────────────────────┘  │
│  基础环境变量 / 启动·停止 / 日志 …                │
└─────────────────────────────────────────────────┘
```

> 变更（2026-07-12）：功能参数来源从「直接启动」面板迁入上方「镜像」卡片；启动时仍读取同一套 Todo 字段。
>
> 变更（2026-07-12）：文案改为「环境变量参数」；未选择来源（或个人配置未选定）时禁用「直接启动」与「启动服务器」，并提示「请先选择环境变量参数」。默认不再预填公司默认，须用户显式选择（任务已持久化来源时仍回填）。
>
> 变更（2026-07-13）：工作面板「创建任务 / 编辑任务」模态框在「已安装镜像」下方复用同一选择器；新建无 task id 时不展示预览；提交 todos 时写入 `feature_params_source` / `personal_feature_params_config_id`。
>
> 变更（2026-07-13）：未选环境变量参数时创建/保存按钮禁用并展示原因；taskTaskService 创建/更新任务将合法 `feature_params_source` 视为必填（`FEATURE_PARAMS_SOURCE_REQUIRED`）。
>
> 变更（2026-07-13）：选择器与启动门禁均不再依赖 `relayToTrae=true`；普通任务详情页镜像区即可选择环境变量参数。
>
> 变更（2026-07-13）：选择就绪后即时 `PATCH /api/tasks/{id}/feature-params/` 持久化（不依赖 relay 启动写入）。
>
> 变更（2026-07-13）：PATCH 失败时在镜像区环境变量块展示可读错误（如非 Owner 403）。

### 3.2 交互流程

```
进入直接启动面板
       │
       ▼
读取 Todo.feature_params_source (默认 company)
       │
       ▼
来源选择器显示当前值
       │
       ├── 选择"公司默认" → 无预览，直接可用
       ├── 选择"工作空间默认" → 无预览，直接可用
       └── 选择"个人配置" → 出现二级下拉 + [预览] 按钮
              │
              ├── 选择具体配置 → 点击 [预览]
              │     └── GET .../feature-params-env-preview/?source=personal&config_id=xxx
              │         → 展示解析后的环境变量面板
              │
              └── 点击 [启动]
                    └── POST .../relay-to-trae/start/
                          body: { ..., feature_params_source: "personal",
                                  personal_feature_params_config_id: "xxx" }
                        → 后端持久化到 Todo
                        → 启动 relayToTrae
```

### 3.3 为什么仅个人配置可预览

| 层级 | 预览 | 原因 |
|------|------|------|
| 公司默认 | ❌ | 公司管理员统一配置，所有成员共享，无需预览 |
| 工作空间默认 | ❌ | 工作空间内共享，配置稳定，避免信息过载 |
| 个人配置 | ✅ | 用户自己创建/维护的多个配置，需要预览确认选中了正确的那一份 |

### 3.4 新增 API

`GET /api/tenant/{t}/workspace/{w}/task/{task_id}/cloud/compute/feature-params-env-preview/`

**参数**：`?source=personal&personal_config_id=<id>`

**响应**：
```json
{
  "source": "personal",
  "config_name": "我的Staging配置",
  "env": {
    "TASK_LLM_PROVIDERS_JSON": "[{\"provider\":\"anthropic\",...}]",
    "TASK_AGENT_MODEL": "claude-sonnet-4-6",
    "TASK_AGENT_MODEL_PROVIDER": "anthropic",
    "TASK_AGENT_MAX_STEPS": "200",
    "TASK_SUMMARY_MODEL": "claude-haiku-4-5",
    "TASK_SUMMARY_MODEL_PROVIDER": "anthropic",
    "MY_EXTRA_VAR": "my_value"
  }
}
```

**约束**：仅允许 `source=personal`，传 `company` 或 `workspace` 返回 403。

### 3.5 修改现有 API

`POST .../relay-to-trae/start/` 的 body 新增可选字段：

```json
{
  "feature_params_source": "personal",           // 可选，默认 'company'
  "personal_feature_params_config_id": "xxx"     // source=personal 时必填
}
```

后端处理：将这两个字段持久化到 `Todo`，使容器后续的 `fetch_tenant_feature_params_env_for_container()` 调用读出相同的配置。

### 3.6 已有 API 复用

| 端点 | 用途 |
|------|------|
| `GET /api/personal/feature-params-configs/` | 列出当前用户的个人配置列表（填充二级下拉） |

---

## 4. 实施概要

### 4.1 后端 (Django)

| 文件 | 改动 |
|------|------|
| `cloud/views/cloud_compute_views.py` | 新增 `get_feature_params_env_preview`（~25 行）；修改 `post_relay_to_trae_start` 接收 source 参数（+5 行） |
| `cloud/urls.py` | 注册 `feature-params-env-preview/` 路由（+2 行） |

### 4.2 前端 (Vue 3)

| 文件 | 改动 |
|------|------|
| `ServerConfig.logic.vue` | 新增 `featureParamsSource`、`personalConfigs`、`selectedPersonalConfigId`、`envPreview` 状态；新增 `fetchPersonalConfigs()`、`fetchEnvPreview()`、`onSourceChange()` 方法（+50 行） |
| `ServerConfigRelayDirectPanel.vue` | 新增来源选择器 + 个人配置选择器 + 预览区域（+50 行） |

### 4.3 测试

| 文件 | 说明 |
|------|------|
| `cloud/view_test/test_feature_params_env_preview.py` | 后端：验证仅 personal 可预览，company/workspace 返回 403 |
| `playwright/.../TaskDetail.feature-params-source-switching.playwright.test.js` | E2E：验证来源切换 → 启动 → 持久化 → 容器拉取正确配置 |

---

## 5. 价值流影响

- 影响 `task-detail-runtime-relay` 流，新增 `feature-params-source-selector` step
- 读写已有字段 `projects_todo.feature_params_source` / `projects_todo.personal_feature_params_config_id`，不新增字段
- 与 relay 启动 API 兼容（新参数 optional）

## 6. Domain Concept Inventory

所有领域概念**已存在**（`FeatureParamsSource`、`FeatureParamsResolver`、`PersonalFeatureParamsConfig` 等），本次仅做 UI 暴露，不引入新概念。

---

---

## 7. 权限影响分析

### 7.1 权限影响矩阵

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 缺失 | 建议 |
|---|--------|------|----------|------|----------|------|------|
| 1 | `GET .../feature-params-env-preview/` (新增) | workspace member | Personal | READ | `IsAuthenticated` (class-level) | ⚠️ **IDOR 风险**: 用户可传入他人 `personal_config_id` 预览他人 env vars | 添加 `_check_idor(config_id, request.user.id)` — 复用 `personal_feature_params_views.py:51` 已有模式 |
| 2 | `POST .../relay-to-trae/start/` 新增 `feature_params_source` 参数 (修改) | workspace member | Task | WRITE | `IsAuthenticated` + Todo 查询已 scope 到 workspace_id | ⚠️ 当 `source=personal` 时，未校验 `personal_config_id` 归属 | 在 `relay_to_trae_start` 服务函数中: 若 `source=personal`，调用 `_check_idor()` 验证 config 属于当前用户 |
| 3 | `GET /api/personal/feature-params-configs/` (复用，不改) | authenticated user | Personal | READ | `IsAuthenticated` + `user_id` 过滤 | ✅ 充分 | — |

### 7.2 安全检查清单

| 检查项 | 结论 | 说明 |
|--------|------|------|
| **IDOR 风险**: 含 `/<resource>_id/` 的 URL | ⚠️ 需修复 | 预览端点接收 `personal_config_id` query param，须校验归属 |
| **权限提升**: PATCH/PUT 端点 | ✅ 无风险 | 仅 GET 预览 + POST 启动，无 PATCH/PUT |
| **跨租户泄露**: 查询是否限定 tenant_id | ✅ 安全 | `FeatureParamsResolver` 已按 `company_id` 限定；`PersonalFeatureParamsConfig` 查询按 `user_id` 过滤 |
| **403 vs 404**: 不存在资源返回什么 | ✅ 一致 | 复用已有 `_check_idor()` — 不存在返回 404，无权返回 403 |
| **user_id 注入**: 能否操作他人资源 | ⚠️ 需修复 | 启动 API 允许传入 `personal_feature_params_config_id`，须校验归属 |
| **敏感操作**: 删除/转账等 | ✅ 不适用 | 无删除/转账/权限变更 |

### 7.3 测试用例

| # | 场景 | 角色 | 操作 | 预期 |
|---|------|------|------|------|
| T1 | 预览自己的个人配置 | workspace member | `GET .../feature-params-env-preview/?source=personal&personal_config_id=MY_ID` | 200, 返回 env |
| T2 | 预览他人的个人配置 | workspace member | `GET .../preview/?source=personal&personal_config_id=OTHER_ID` | 403 |
| T3 | 预览不存在的个人配置 | workspace member | `GET .../preview/?source=personal&personal_config_id=NONEXIST` | 404 |
| T4 | 预览公司配置 (被禁止) | workspace member | `GET .../preview/?source=company` | 403 |
| T5 | 预览工作空间配置 (被禁止) | workspace member | `GET .../preview/?source=workspace` | 403 |
| T6 | 启动时指定自己的个人配置 | workspace member | `POST .../start/` + `source=personal` + own config_id | 200, Todo 写入 |
| T7 | 启动时指定他人的个人配置 | workspace member | `POST .../start/` + `source=personal` + other's config_id | 403 |
| T8 | 未登录访问预览 | anonymous | `GET .../preview/?source=personal&config_id=...` | 401 |
| T9 | 跨 workspace 启动 | other workspace member | `POST .../start/` (不同 workspace 的 task) | 404 (Todo query scope 已限定) |

### 7.4 风险评级

| 风险 | 级别 | 缓解 |
|------|------|------|
| 预览端点 IDOR — 可查看他人 personal config 的 API keys | 🔴 高 | 实施前必须添加 `_check_idor()` 校验 |
| 启动时指定他人 personal config | 🟡 中 | 实施前必须添加归属校验 |
| extra_env_vars 中的敏感值在预览中暴露 | 🟢 低 | 前端已有 `SENSITIVE_PATTERN` 遮罩 (`EnvVarTableEditor.vue`) |

### 7.5 实施要求

预览端点和启动端点**必须**在服务函数中添加以下权限检查（复用现有模式）：

```python
# 伪代码 — 复用 personal_feature_params_views.py 的 _check_idor 逻辑

def get_feature_params_env_preview(request, ...):
    source = request.GET.get('source', '')
    if source == 'personal':
        config_id = request.GET.get('personal_config_id', '')
        config = PersonalFeatureParamsConfig.objects.filter(id=config_id).first()
        if not config:
            return Response({'message': '配置不存在'}, status=404)
        if str(config.user_id) != str(request.user.id):
            return Response({'message': '无权访问该配置'}, status=403)
        # ... resolve and return env
    else:
        return Response({'message': '仅支持预览个人配置'}, status=403)
```

---

*文档 v3 — 已追加权限分析。*
