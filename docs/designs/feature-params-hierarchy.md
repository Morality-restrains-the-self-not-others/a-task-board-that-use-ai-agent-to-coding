# 设计文档：功能参数多层级配置（公司/工作空间/个人）

**状态**: 待审批  
**日期**: 2026-06-30  
**作者**: Claude (brainstorming)

---

## 1. 问题陈述

### 1.1 当前状态

功能参数（LLM 供应商配置、模型选择、Agent 参数等）**仅支持公司级别**配置：

- 数据模型：`TenantFeatureParams` 与 `Company` 1:1 绑定
- 管理入口：`/tenant/{tenant}/settings/feature-params/`（仅租户管理员可编辑）
- 运行时：容器启动后通过 `POST .../feature-params-env/` 拉取公司级配置，注入环境变量

```mermaid
graph LR
    A[Company] -->|1:1| B[TenantFeatureParams]
    B -->|env| C[Task Container]
```

### 1.2 需求

用户希望能够：

1. **公司级** — 保留现有能力，作为全局默认配置
2. **工作空间级** — 每个 Workspace 可覆盖公司配置（如不同项目用不同 LLM endpoint）
3. **个人级** — 用户可创建**多份**个人配置（如 "DeepSeek 预算版"、"OpenAI 完整版"）
4. **任务选择** — 创建/编辑任务时可选择使用哪一级的哪个配置
5. **工作空间管控** — 工作空间可控制是否允许成员使用个人配置（**默认不允许**），同一公司下不同工作空间可独立设置
6. **运行审计** — 每次容器启动拉取配置时，记录快照（用了哪个配置 + 完整内容），可追溯可复现

### 1.3 典型场景

| 场景 | 说明 |
|------|------|
| 公司统一配置 | 管理员设置公司级 DeepSeek provider，所有任务默认使用 |
| 工作空间差异化 | "AI 研发部"工作空间用 GPT-4.1，"测试部"用 DeepSeek-V3 |
| 个人多配置 | 高级用户创建 "低成本日常任务" 和 "高能力复杂任务" 两套配置 |
| 任务灵活切换 | 创建任务时从下拉菜单选择：公司默认 / 工作空间默认 / 个人:低成本 / 个人:高能力 |
| 工作空间禁用个人配置 | 工作空间管理员关闭"允许个人配置"后，该空间内的任务只能使用公司或工作空间配置；其他工作空间不受影响 |
| 运行记录追溯 | 任务跑了 3 次，每次用的什么配置、API key 是什么，都能在运行记录里查看 |

---

## 2. 设计方案

### 2.1 整体架构

```
┌──────────────────────────────────────────────────────────────┐
│                   Company Level                               │
│  TenantFeatureParams (公司级，必须存在)                        │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  providers, agent_model, agent_max_steps, ...             │ │
│  └──────────────────────────────────────────────────────────┘ │
│                         │                                      │
│          ┌──────────────┼──────────────┐                       │
│          ▼              ▼              ▼                       │
│  ┌──────────────┐ ┌──────────────┐ ┌────────────────┐         │
│  │ Workspace A  │ │ Workspace B  │ │ Workspace C    │         │
│  │ (use co.     │ │ (override)   │ │ (use co.       │         │
│  │  default)    │ │              │ │  default)      │         │
│  │              │ │ allow_perso- │ │                │         │
│  │ allow_perso- │ │ nal=true ✓   │ │ allow_perso-   │         │
│  │ nal=false ✗  │ │              │ │ nal=true ✓     │         │
│  └──────────────┘ └──────────────┘ └────────────────┘         │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │ Personal Configs (per user, per company)                  │ │
│  │  仅在任务所属工作空间的 allow_personal_feature_params=True  │ │
│  │  时才可被选择使用                                          │ │
│  │  - "低成本日常任务" (DeepSeek, max_steps=50)               │ │
│  │  - "高能力复杂任务" (GPT-4.1, max_steps=500)               │ │
│  └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │  Task (Todo)        │
              │  feature_params:    │
              │    source: personal │
              │    config_id: "..." │
              └─────────────────────┘
                         │
                         ▼
              ┌─────────────────────┐     ┌──────────────────────────┐
              │  Container Runtime  │────▶│ TaskFeatureParamsSnapshot │
              │  env = resolve(task)│ 写入 │ (运行记录：配置来源+完整内容) │
              └─────────────────────┘     └──────────────────────────┘
```

### 2.2 解析策略

任务运行时，按以下优先级解析有效配置：

```
resolve(task):
  workspace = task.workspace
  company_params = TenantFeatureParams.objects.get(company_id=workspace.company_id)

  if task.feature_params_source == 'personal':
    // ── 治理检查：工作空间是否允许个人配置 ──
    if not workspace.allow_personal_feature_params:
      log_warning("personal config blocked by workspace policy, fallback to company")
      return company_params  // 回退到公司默认
    config = PersonalFeatureParamsConfig.objects.get(id=task.personal_feature_params_config_id)
    return config  // 若 config 已删除，回退到 company_params

  if task.feature_params_source == 'workspace':
    ws_params = WorkspaceFeatureParams.objects.filter(workspace_id=task.workspace_id).first()
    if ws_params and not ws_params.use_company_default:
      return ws_params  // 工作空间自定义配置
    return company_params  // 工作空间继承公司配置

  // task.feature_params_source == 'company' (default)
  return company_params
```

**关键设计决策**：

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 工作空间覆盖粒度 | 全字段覆盖（非逐字段） | 简化实现，与现有 `TenantFeatureParams` 结构一致 |
| 个人配置存储 | 完整配置副本（非增量覆盖） | 用户创建的就是完整配置，不依赖上级；避免多层 merge 的复杂度 |
| 任务配置绑定 | 运行时引用（非快照） | 容器启动时拉取最新配置；用户可修改配置后重启任务生效 |
| 运行审计 | 拉取时同步写快照表 | 引用 + 快照双轨：引用保证灵活性，快照保证可追溯可复现 |
| 个人配置权限 | 工作空间级开关 `Workspace.allow_personal_feature_params`，**默认 false** | 管控粒度更细：同一公司下不同工作空间可独立决定是否允许个人配置 |
| 默认行为 | 任务不选时默认 workspace → company 回退 | 向后兼容，现有任务无需迁移 |

### 2.3 数据模型

#### 2.3.1 公司级（保留现有，基本不改）

```python
# projects/models/tenant_feature_params.py — 现有模型，不改结构
class TenantFeatureParams(models.Model):
    id = SnowflakeIDField()
    company = models.OneToOneField(Company, ...)
    providers = models.JSONField(default=list)
    agent_model = models.CharField(...)
    agent_model_provider = models.CharField(...)
    agent_max_steps = models.PositiveIntegerField(default=200)
    summary_model = models.CharField(...)
    summary_model_provider = models.CharField(...)
    llm_budget_enabled = models.BooleanField(default=False)
    # ...
```

#### 2.3.2 工作空间级（新增）

**Workspace 模型增加治理字段**（修改现有 `projects/models/workspace.py`）：

```python
class Workspace(models.Model):
    # ... 现有字段 ...

    # ── 新增：工作空间级治理开关 ──
    allow_personal_feature_params = models.BooleanField(
        default=False,
        help_text='是否允许成员在此工作空间的任务中使用个人功能参数配置。'
                  '关闭时仅可使用公司或工作空间配置。'
    )
```

**WorkspaceFeatureParams（新增）**：

```python
# projects/models/workspace_feature_params.py — 新文件
class WorkspaceFeatureParams(models.Model):
    """工作空间功能参数覆盖配置"""
    id = SnowflakeIDField()
    workspace = models.OneToOneField(
        'Workspace', on_delete=models.CASCADE,
        related_name='feature_params'
    )
    company_id = models.CharField(max_length=64)  # 冗余，加速查询
    use_company_default = models.BooleanField(
        default=True,
        help_text='为 True 时完全继承公司配置，忽略下方所有覆盖字段'
    )

    # 以下字段仅在 use_company_default=False 时生效
    providers = models.JSONField(default=list)
    agent_model = models.CharField(max_length=255, default='', blank=True)
    agent_model_provider = models.CharField(max_length=255, default='', blank=True)
    agent_max_steps = models.PositiveIntegerField(default=200)
    summary_model = models.CharField(max_length=255, default='', blank=True)
    summary_model_provider = models.CharField(max_length=255, default='', blank=True)
    llm_budget_enabled = models.BooleanField(default=False)

    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        db_table = 'projects_workspace_feature_params'
        verbose_name = '工作空间功能参数'
```

#### 2.3.3 个人级（新增 — 支持多份）

```python
# projects/models/personal_feature_params_config.py — 新文件
class PersonalFeatureParamsConfig(models.Model):
    """个人功能参数配置（完整配置，非增量覆盖）"""
    id = SnowflakeIDField()
    user_id = models.CharField(max_length=64)  # taskAuth user_id
    company_id = models.CharField(max_length=64)  # 属于哪个公司
    name = models.CharField(max_length=255)  # 用户命名的配置名

    # 完整配置字段（与 TenantFeatureParams 字段一致）
    providers = models.JSONField(default=list)
    agent_model = models.CharField(max_length=255, default='', blank=True)
    agent_model_provider = models.CharField(max_length=255, default='', blank=True)
    agent_max_steps = models.PositiveIntegerField(default=200)
    summary_model = models.CharField(max_length=255, default='', blank=True)
    summary_model_provider = models.CharField(max_length=255, default='', blank=True)
    llm_budget_enabled = models.BooleanField(default=False)

    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        db_table = 'projects_personal_feature_params_config'
        verbose_name = '个人功能参数配置'
        # 同一用户在同一公司下，配置名唯一
        unique_together = ('user_id', 'company_id', 'name')
```

#### 2.3.4 任务绑定（修改 Todo）

```python
# projects/models/todo.py — 新增字段

FEATURE_PARAMS_SOURCE_COMPANY = 'company'
FEATURE_PARAMS_SOURCE_WORKSPACE = 'workspace'
FEATURE_PARAMS_SOURCE_PERSONAL = 'personal'

FEATURE_PARAMS_SOURCE_CHOICES = [
    (FEATURE_PARAMS_SOURCE_COMPANY, '公司默认'),
    (FEATURE_PARAMS_SOURCE_WORKSPACE, '工作空间默认'),
    (FEATURE_PARAMS_SOURCE_PERSONAL, '个人配置'),
]

class Todo(models.Model):
    # ... 现有字段 ...

    # 新增字段
    feature_params_source = models.CharField(
        max_length=20,
        choices=FEATURE_PARAMS_SOURCE_CHOICES,
        default=FEATURE_PARAMS_SOURCE_COMPANY,
        help_text='功能参数配置来源'
    )
    personal_feature_params_config_id = models.CharField(
        max_length=64, default='', blank=True,
        help_text='当 source=personal 时，指向 PersonalFeatureParamsConfig.id'
    )
```

#### 2.3.5 运行记录快照（新增）

```python
# projects/models/task_feature_params_snapshot.py — 新文件
class TaskFeatureParamsSnapshot(models.Model):
    """每次容器拉取功能参数时写入的运行记录快照。

    记录「哪个任务、在什么时间、用了哪个来源的配置、配置的完整内容」，
    即使后续配置被修改或删除，运行记录仍可追溯可复现。
    """
    id = SnowflakeIDField()
    task_id = models.CharField(max_length=64)
    workspace_id = models.CharField(max_length=64)
    tenant_id = models.CharField(max_length=64)

    # ── 配置来源标识 ──
    source = models.CharField(
        max_length=20,
        choices=FEATURE_PARAMS_SOURCE_CHOICES,
    )
    source_config_id = models.CharField(
        max_length=64, default='', blank=True,
        help_text='具体配置的 ID：公司 TenantFeatureParams.id / 工作空间 WorkspaceFeatureParams.id / 个人 PersonalFeatureParamsConfig.id'
    )
    source_display_name = models.CharField(
        max_length=512, default='', blank=True,
        help_text='人类可读的配置名，如 "公司默认"、"工作空间:研发部"、"个人:低成本日常任务"'
    )

    # ── 完整 env 快照（含 API key，需注意访问控制） ──
    resolved_env = models.JSONField(
        help_text='容器实际收到的完整 env map，如 TASK_LLM_PROVIDERS_JSON / TASK_AGENT_MODEL 等'
    )

    # ── 脱敏摘要（供 UI 展示，不含完整 API key） ──
    providers_summary = models.JSONField(
        default=list,
        help_text='providers 摘要：[{provider, base_url, supported_models, api_key_hash}]'
    )
    agent_model = models.CharField(max_length=255, default='', blank=True)
    agent_model_provider = models.CharField(max_length=255, default='', blank=True)
    agent_max_steps = models.IntegerField(default=200)
    summary_model = models.CharField(max_length=255, default='', blank=True)
    summary_model_provider = models.CharField(max_length=255, default='', blank=True)

    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        db_table = 'projects_task_feature_params_snapshot'
        verbose_name = '任务功能参数运行记录'
        indexes = [
            models.Index(fields=['task_id', '-created_at']),
        ]
```

### 2.4 配置解析服务（新增领域服务）

```python
# projects/services/feature_params_resolver.py — 新文件

def resolve_feature_params_for_task(task: Todo) -> tuple[TenantLlmConfig, dict]:
    """
    根据任务绑定，解析出有效的功能参数配置。

    Returns:
        (TenantLlmConfig, snapshot_meta) — 配置对象 + 快照元信息（供审计写入）
    """
    company_params = TenantFeatureParams.objects.get(
        company_id=task.workspace.company_id
    )

    if task.feature_params_source == FEATURE_PARAMS_SOURCE_PERSONAL:
        # ── 治理检查：工作空间是否允许个人配置 ──
        workspace = task.workspace
        if not workspace.allow_personal_feature_params:
            snapshot_meta = {
                'source': FEATURE_PARAMS_SOURCE_COMPANY,
                'source_config_id': str(company_params.id),
                'source_display_name': f'公司默认（个人配置被工作空间策略禁用，已回退）',
            }
            return _to_llm_config(company_params), snapshot_meta

        config_id = task.personal_feature_params_config_id
        if config_id:
            try:
                personal = PersonalFeatureParamsConfig.objects.get(id=config_id)
                snapshot_meta = {
                    'source': FEATURE_PARAMS_SOURCE_PERSONAL,
                    'source_config_id': str(personal.id),
                    'source_display_name': f'个人配置:{personal.name}',
                }
                return _to_llm_config(personal), snapshot_meta
            except PersonalFeatureParamsConfig.DoesNotExist:
                pass  # fallback below

        # 个人配置无效（已删除/ID 为空）→ 回退公司默认
        snapshot_meta = {
            'source': FEATURE_PARAMS_SOURCE_COMPANY,
            'source_config_id': str(company_params.id),
            'source_display_name': '公司默认（个人配置无效，已回退）',
        }
        return _to_llm_config(company_params), snapshot_meta

    if task.feature_params_source == FEATURE_PARAMS_SOURCE_WORKSPACE:
        ws_params = WorkspaceFeatureParams.objects.filter(
            workspace_id=task.workspace_id
        ).first()
        if ws_params and not ws_params.use_company_default:
            snapshot_meta = {
                'source': FEATURE_PARAMS_SOURCE_WORKSPACE,
                'source_config_id': str(ws_params.id),
                'source_display_name': f'工作空间配置:{task.workspace.name}',
            }
            return _to_llm_config(ws_params), snapshot_meta
        snapshot_meta = {
            'source': FEATURE_PARAMS_SOURCE_COMPANY,
            'source_config_id': str(company_params.id),
            'source_display_name': '公司默认（工作空间继承）',
        }
        return _to_llm_config(company_params), snapshot_meta

    # source == 'company' 或默认
    snapshot_meta = {
        'source': FEATURE_PARAMS_SOURCE_COMPANY,
        'source_config_id': str(company_params.id),
        'source_display_name': '公司默认',
    }
    return _to_llm_config(company_params), snapshot_meta
```

### 2.5 API 设计

#### 2.5.1 公司级（现有，不变）

| Method | Path | 说明 |
|--------|------|------|
| `GET` | `/api/tenant/{tenant}/feature-params/` | 获取公司级配置 |
| `POST` | `/api/tenant/{tenant}/feature-params/` | 保存公司级配置 |

#### 2.5.2 工作空间级（新增）

| Method | Path | 说明 |
|--------|------|------|
| `GET` | `/api/tenant/{tenant}/workspace/{workspace}/feature-params/` | 获取工作空间配置（含公司配置作为参考） |
| `POST` | `/api/tenant/{tenant}/workspace/{workspace}/feature-params/` | 保存工作空间覆盖配置 |

**GET 响应格式**：
```json
{
  "data": {
    "use_company_default": false,
    "providers": [...],
    "agent_model": "gpt-4.1",
    // ... 工作空间自身配置
    "company_config": {
      "providers": [...],
      "agent_model": "deepseek-v3",
      // ... 公司配置作为参考
    }
  }
}
```

#### 2.5.3 个人级（新增）

| Method | Path | 说明 |
|--------|------|------|
| `GET` | `/api/personal/feature-params-configs/` | 列出当前用户的所有个人配置 |
| `POST` | `/api/personal/feature-params-configs/` | 创建个人配置 |
| `GET` | `/api/personal/feature-params-configs/{id}/` | 获取单个配置详情 |
| `PUT` | `/api/personal/feature-params-configs/{id}/` | 更新个人配置 |
| `DELETE` | `/api/personal/feature-params-configs/{id}/` | 删除个人配置 |

#### 2.5.4 任务配置绑定（新增/修改）

| Method | Path | 说明 |
|--------|------|------|
| `PATCH` | `/api/tasks/{task_id}/feature-params/` | 修改任务的配置来源 |

**请求体**：
```json
{
  "feature_params_source": "personal",
  "personal_feature_params_config_id": "1234567890"
}
```

**任务创建时也可携带**（扩展现有 `POST /api/tasks/`）：
```json
{
  "title": "...",
  "workspace_id": "...",
  "feature_params_source": "workspace",       // 可选，默认 "company"
  "personal_feature_params_config_id": ""      // source=personal 时必填
}
```

#### 2.5.5 容器运行时（修改现有，不改路径签名）

```
POST /api/container/{tenant_id}/{workspace_id}/{task_id}/feature-params-env/
```

**内部逻辑修改**：

1. 调用 `resolve_feature_params_for_task(task)` 解析有效配置
2. 将解析结果序列化为 env map 返回给容器
3. **同步写入 `TaskFeatureParamsSnapshot`** — 记录本次运行的配置来源 + 完整 env 快照

> ⚠️ 路径签名不变 → 容器侧零改动，完全向后兼容。
> ⚠️ 快照写入失败不应阻塞 env 返回（降级为 log warning），保证容器启动不受影响。

#### 2.5.6 运行记录查询（新增）

| Method | Path | 说明 |
|--------|------|------|
| `GET` | `/api/tasks/{task_id}/feature-params-snapshots/` | 列出该任务的所有运行记录（按时间倒序） |

**响应格式**：
```json
{
  "snapshots": [
    {
      "id": "...",
      "source_display_name": "个人配置:低成本日常任务",
      "providers_summary": [
        {"provider": "deepseek", "base_url": "https://api.deepseek.com", "api_key_hash": "sha256:ab12..."}
      ],
      "agent_model": "deepseek-v3",
      "agent_max_steps": 50,
      "created_at": "2026-06-30T10:30:00Z"
    }
  ]
}
```
> 注意：`resolved_env`（含完整 API key）不直接在列表中返回；详情接口需要额外鉴权。

### 2.6 前端 UI 设计

#### 2.6.1 公司级（现有，不变）

`/tenant/:tenant/settings/feature-params/` — `WorkspaceSettingsFeatureParams.vue`

#### 2.6.2 工作空间级（新增页面）

`/tenant/:tenant/settings/workspace/:workspace/feature-params/`

```
┌─────────────────────────────────────────────────┐
│  工作空间功能参数设置                              │
│                                                 │
│  ○ 使用公司默认配置                               │
│  ● 自定义工作空间配置                             │
│                                                 │
│  ┌─ 公司配置参考（只读）─────────────────────────┐ │
│  │ providers: [DeepSeek, OpenAI]                │ │
│  │ agent_model: deepseek-v3                     │ │
│  │ agent_max_steps: 200                         │ │
│  └─────────────────────────────────────────────┘ │
│                                                 │
│  ┌─ 工作空间覆盖配置 ───────────────────────────┐ │
│  │ [LLM Provider Editor]                        │ │
│  │ [Agent Model Selector]                       │ │
│  │ [Summary Model Selector]                     │ │
│  │ [Max Steps Input]                            │ │
│  └─────────────────────────────────────────────┘ │
│                                                 │
│  [保存] [重置为公司默认]                          │
└─────────────────────────────────────────────────┘
```

#### 2.6.3 个人配置管理（新增页面）

`/personal/feature-params-configs/`

```
┌─────────────────────────────────────────────────┐
│  我的功能参数配置                                  │
│                                                 │
│  [+ 新建配置]                                    │
│                                                 │
│  ┌─────────────────────────────────────────────┐ │
│  │ 📋 低成本日常任务          DeepSeek V3 · 50步  │ │
│  │   默认选中                                    │ │
│  │   [编辑] [复制] [删除]                        │ │
│  ├─────────────────────────────────────────────┤ │
│  │ 📋 高能力复杂任务          GPT-4.1 · 500步     │ │
│  │   [编辑] [复制] [删除]                        │ │
│  ├─────────────────────────────────────────────┤ │
│  │ 📋 Claude 独占          Claude Opus · 300步   │ │
│  │   [编辑] [复制] [删除]                        │ │
│  └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

#### 2.6.4 任务创建/编辑（修改现有）

在任务创建表单中新增"功能参数"选择器：

```
┌─────────────────────────────────────────────────┐
│  创建任务                                        │
│                                                 │
│  标题: [________________________]                │
│  描述: [________________________]                │
│  工作空间: [v 研发部          ]                   │
│  功能参数: [v 个人:低成本日常任务]                 │
│            ─────────────────────                 │
│            公司默认                               │
│            工作空间默认                            │
│            ─────────────────────                 │
│            个人: 低成本日常任务                    │
│            个人: 高能力复杂任务                    │
│            个人: Claude 独占                      │
│                                                 │
│  [创建] [取消]                                   │
└─────────────────────────────────────────────────┘
```

默认选中"工作空间默认"（如果工作空间有自定义配置），否则回退到"公司默认"。

### 2.7 权限模型

| 操作 | 权限要求 |
|------|---------|
| 查看/修改公司配置 | 租户管理员 (`CompanyMember.is_admin=True`) |
| 查看/修改工作空间配置 | 工作空间管理员 (`WorkspaceAccess.role='admin'`) |
| 设置工作空间 `allow_personal_feature_params` | 工作空间管理员（或租户管理员） |
| 创建/编辑/删除个人配置 | 配置所有者 (`user_id == request.user.id`) |
| 查看个人配置（任务选择用） | 登录用户（只能看到自己的配置） |
| 任务选择配置来源 | 任务创建者/编辑者；选 personal 时须工作空间允许 |
| 查看运行记录快照 | 任务可见范围内的成员（脱敏摘要）；完整 env 需管理员 |

> 详细角色权限分析将由 `/2-role-permission-角色权限分析` 产出。

---

## 3. 价值流影响分析

### 3.1 受影响的现有 Value Stream

| Stream | Step | 影响 |
|--------|------|------|
| `project-workspace` | `tenant-feature-params` | 扩展：新增 workspace-level + personal-level 配置能力 |
| `task-management` | `todo-crud` | 修改：任务创建/编辑增加 `feature_params_source` + `personal_config_id` 字段 |
| `task-llm-budget-governance` | `workspace-model-budget-defaults` | 可能影响：workspace feature params 与 workspace budget defaults 需要协调 |

### 3.2 建议新增 Value Stream

```yaml
- name: feature-params-hierarchy
  domain: 项目与工作空间
  description: 功能参数多层级配置（公司→工作空间→个人），任务可选择配置来源
  steps:
    - name: workspace-feature-params-crud
      status: planned
      test_file: tests/test_workspace_feature_params.py
      fields:
        - name: saas-backend.projects_workspace_feature_params.use_company_default
        - name: saas-backend.projects_workspace_feature_params.providers
        - name: saas-backend.projects_workspace_feature_params.agent_model

    - name: personal-feature-params-crud
      status: planned
      test_file: tests/test_personal_feature_params_config.py
      fields:
        - name: saas-backend.projects_personal_feature_params_config.name
        - name: saas-backend.projects_personal_feature_params_config.user_id
        - name: saas-backend.projects_personal_feature_params_config.providers

    - name: task-feature-params-binding
      status: planned
      test_file: tests/test_task_feature_params_binding.py
      fields:
        - name: saas-backend.projects_todo.feature_params_source
        - name: saas-backend.projects_todo.personal_feature_params_config_id

    - name: feature-params-resolution
      status: planned
      test_file: tests/test_feature_params_resolver.py
      fields:
        - name: saas-backend.projects_todo.feature_params_source
        - name: saas-backend.projects_workspace_feature_params.id
        - name: saas-backend.projects_tenant_feature_params.providers
        - name: saas-backend.projects_workspace.allow_personal_feature_params

    - name: feature-params-snapshot-audit
      status: planned
      test_file: tests/test_task_feature_params_snapshot.py
      fields:
        - name: saas-backend.projects_task_feature_params_snapshot.task_id
        - name: saas-backend.projects_task_feature_params_snapshot.source
        - name: saas-backend.projects_task_feature_params_snapshot.resolved_env
        - name: saas-backend.projects_task_feature_params_snapshot.providers_summary

    - name: workspace-personal-config-governance
      status: planned
      test_file: tests/test_workspace_personal_config_governance.py
      fields:
        - name: saas-backend.projects_workspace.allow_personal_feature_params
        - name: saas-backend.projects_todo.feature_params_source
```

### 3.3 字段影响汇总

| Service | Table | Field | 操作 |
|---------|-------|-------|------|
| saas-backend | `projects_workspace` | `allow_personal_feature_params` | **新增** |
| saas-backend | `projects_workspace_feature_params` | (全表) | **新增** |
| saas-backend | `projects_personal_feature_params_config` | (全表) | **新增** |
| saas-backend | `projects_task_feature_params_snapshot` | (全表) | **新增** |
| saas-backend | `projects_todo` | `feature_params_source` | **新增** |
| saas-backend | `projects_todo` | `personal_feature_params_config_id` | **新增** |
| saas-backend | `projects_tenant_feature_params` | (全字段) | 不变 |

---

## 4. 领域概念清单（DDD 前置）

| 类别 | 概念 | 说明 |
|------|------|------|
| **Bounded Context** | FeatureParams | 功能参数配置的有界上下文，包含多层级管理 |
| **Bounded Context** | TaskConfigBinding | 任务与配置源的绑定关系 |
| **Bounded Context** | ConfigAudit | 配置运行审计（快照记录） |
| **实体** | TenantFeatureParams (CompanyFeatureParams) | 公司级配置，聚合根 |
| **实体** | WorkspaceFeatureParams | 工作空间级覆盖配置，聚合根 |
| **实体** | PersonalFeatureParamsConfig | 个人配置（多份），聚合根 |
| **实体** | TaskFeatureParamsSnapshot | 运行记录快照，只追加不修改 |
| **值对象** | FeatureParamsSource | 枚举：company / workspace / personal |
| **值对象** | LlmProviderEntry | 现有值对象，不变 |
| **值对象** | SnapshotMeta | 快照元信息：source + source_config_id + source_display_name |
| **聚合** | TenantLlmConfig | 现有聚合根，增加来源标记 |
| **领域服务** | FeatureParamsResolver | 根据 task 解析有效配置 + 治理检查 |
| **领域事件** | FeatureParamsConfigChanged | 配置变更时发布（可选，供预算系统消费） |
| **领域事件** | FeatureParamsResolved | 配置解析完成时发布 → 触发快照写入 |

---

## 5. 实施计划概要

### Increment 1: 数据模型（后端基础）

1. Workspace 模型增加 `allow_personal_feature_params` 字段 + migration
2. 创建 `WorkspaceFeatureParams` 模型 + migration
3. 创建 `PersonalFeatureParamsConfig` 模型 + migration
4. 创建 `TaskFeatureParamsSnapshot` 模型 + migration
5. Todo 模型增加 `feature_params_source` + `personal_feature_params_config_id` 字段 + migration

### Increment 2: 领域服务 + 容器运行时

1. 实现 `FeatureParamsResolver` 领域服务（含工作空间治理检查 + 回退逻辑）
2. 修改容器 env 拉取接口：调用 resolver → 写快照 → 返回 env
3. 编写 resolver 单元测试（覆盖 6 条路径：3 来源 × 治理开关 ± 回退）
4. 编写快照写入测试

### Increment 3: 管理 API（后端）

1. Workspace feature params CRUD API
2. Workspace `allow_personal_feature_params` 读写 API
3. Personal feature params configs CRUD API（受治理开关限制）
4. Task feature params binding PATCH API
5. Task feature params snapshots 查询 API
6. 任务创建/列表接口增加 feature_params 字段
7. 编写 API 测试

### Increment 4: 前端 UI

1. 工作空间配置页面 + "允许个人配置"开关
2. 个人配置管理页面（仅允许个人配置的工作空间可见入口）
3. 任务创建/编辑表单增加配置选择器（动态过滤个人配置可用性）
4. 任务详情展示当前使用的配置 + 历史运行记录面板

---

## 6. 设计决策记录

| # | 决策 | 备选方案 | 理由 |
|---|------|---------|------|
| 1 | 个人配置存储完整配置（非增量覆盖） | 增量覆盖（只存差异字段） | 完整配置更直观，用户创建的就是他看到的样子；避免多层 merge 的边界情况 |
| 2 | 工作空间配置使用 `use_company_default` 布尔开关 | 所有字段 nullable + null=继承 | 显式开关更清晰，避免 "字段为空到底是不设置还是想设为空" 的歧义 |
| 3 | 运行时引用（非任务创建时快照） | 快照到任务 | 用户可修改配置后重启任务生效；符合现有容器拉取模式 |
| 4 | 绑定信息存在 Todo 表上 | 独立绑定表 `TaskFeatureParamsBinding` | 减少 JOIN，当前需求足够简单不需要独立表 |
| 5 | 容器 env API 路径签名不变 | 修改路径添加新参数 | 零容器侧改动，完全向后兼容 |

---

## 7. 兼容性

| 方面 | 策略 |
|------|------|
| 现有任务 | `feature_params_source` 默认 `'company'`，行为完全不变 |
| 现有 API | 公司级 API 路径和响应格式不变 |
| 容器侧 | env 拉取路径签名不变，容器零改动 |
| 数据库 | 新增三张表 + Todo 增加两个字段，无破坏性变更 |
| LLM Budget | resolver 返回的配置继续被 budget 系统消费，无影响 |

---

## 8. 风险与注意事项

1. **删除个人配置** — 如果某任务绑定了已删除的个人配置，resolver 应回退到公司默认，并在快照 `source_display_name` 中注明回退原因
2. **工作空间关闭个人配置** — 管理员关闭 `allow_personal_feature_params` 后，已绑定个人配置的存量任务自动回退到公司默认（下次容器启动时生效）
3. **工作空间删除** — 工作空间删除时，`WorkspaceFeatureParams` 级联删除；已绑定该工作空间配置的任务自动回退到公司默认
4. **公司切换** — 用户可能属于多个公司，个人配置按 `(user_id, company_id)` 隔离
5. **权限边界** — 普通成员不应看到管理员的个人配置，确保 API 层 `user_id` 过滤
6. **快照中的 API Key** — `resolved_env` 包含完整 API key，快照查询 API 只返回脱敏摘要；完整内容须额外鉴权（仅任务 owner + 管理员）
7. **快照写入失败** — 不应阻塞容器 env 返回，降级为 log warning

---

## 总结清单

- **配置层级**: 公司级（保留）/ 工作空间级（新增覆盖）/ 个人级（新增多份完整配置）
- **解析策略**: personal（须工作空间允许）→ workspace override → workspace inherit → company（含治理检查的回退链）
- **治理开关**: 工作空间级 `Workspace.allow_personal_feature_params`，默认 false，管理员按工作空间粒度开放
- **运行审计**: 每次容器拉取配置时同步写 `TaskFeatureParamsSnapshot`，记录来源 + 完整 env（查询时脱敏）
- **数据模型**: 新增 3 张表 + Workspace 加 1 字段 + Todo 加 2 字段，`TenantFeatureParams` 不变
- **API 设计**: 公司级不变 / 工作空间新增 3 端点 / 个人新增 5 端点 / 任务绑定新增 1 端点 / 快照查询新增 1 端点
- **容器兼容**: 路径签名不变，容器零改动
- **实施顺序**: 数据模型 → 领域服务+运行时 → 管理 API → 前端 UI

---

## 9. 权限影响分析（来自 Step 2）

> 完整分析见 `docs/designs/feature-params-hierarchy-permission-analysis.md`

### 9.1 权限影响矩阵（摘要）

| 端点 | 主体 | 资源层级 | 缺失检查 | 风险 |
|------|------|----------|---------|------|
| POST /api/tenant/{t}/feature-params/ | tenant_admin | Tenant | ⚠️ 现有 POST 未强制 is_admin | 🟡 |
| POST .../workspace/{w}/feature-params/ | workspace_admin | Workspace | ⚠️ 需 WorkspaceAccess.role='admin' | 🟡 |
| GET/PUT/DELETE .../configs/{id}/ | config_owner | User | 🔴 需 user_id 归属校验（IDOR） | 🔴 |
| POST /api/personal/.../ | 登录用户 | User | 🔴 需强制 user_id=request.user.id | 🔴 |
| PATCH /tasks/{id}/feature-params/ | task_owner | Task | ⚠️ 需校验 personal_config 归属 | 🟡 |
| GET /tasks/{id}/feature-params-snapshots/ | workspace_member | Task | ⚠️ 脱敏 + workspace 归属 | 🟡 |

### 9.2 关键修复

1. **Increment 0**: 修复现有 `manage_feature_params` POST 缺少 `is_admin` 校验
2. **Increment 3**: 所有个人配置 `{id}` 端点实现 `IsPersonalConfigOwner` 权限类
3. **Increment 3**: 创建个人配置时 `user_id = request.user.id`（硬编码，拒绝注入）
4. **任务绑定**: personal source 须三重校验（workspace 允许 + config 属于用户 + config.company == workspace.company）

### 9.3 测试覆盖

28 条权限测试用例，覆盖 IDOR / 跨租户 / 权限提升 / user_id 注入 / 403 vs 404。详见 `feature-params-hierarchy-permission-analysis.md` §4。
