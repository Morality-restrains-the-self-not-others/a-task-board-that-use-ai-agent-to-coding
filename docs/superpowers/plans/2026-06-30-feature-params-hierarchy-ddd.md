# DDD 领域模型: 功能参数多层级配置

> 输入:
> - 设计文档: `docs/designs/feature-params-hierarchy.md`
> - 价值流: `docs/superpowers/plans/2026-06-30-feature-params-hierarchy-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-30-feature-params-hierarchy-nfr-clarification.md`
>
> 领域模型文件: `task2app/Saas_project/projects/domain/feature_params/`

## 限界上下文

| 上下文 | 职责 | 备注 |
|--------|------|------|
| **FeatureParams** | 多层级功能参数配置管理与解析 | 本次新增 |
| **TenantLlmConfig** (现有) | LLM 配置值对象和聚合根 | 复用，不修改 |
| **ConfigAudit** | 运行记录审计快照 | 本次新增，从 FeatureParams 分离以遵循单一职责 |

## 聚合与聚合根

```
FeatureParams 限界上下文
├── 聚合根: TenantFeatureParams (公司级)
│   └── TenantLlmConfig (现有值对象)
├── 聚合根: WorkspaceFeatureParams (工作空间级)
│   └── TenantLlmConfig (现有值对象)
├── 聚合根: PersonalFeatureParamsConfig (个人级)
│   └── TenantLlmConfig (现有值对象)
│
ConfigAudit 限界上下文
└── 聚合根: TaskFeatureParamsSnapshot (只追加)
    ├── SnapshotMeta (值对象)
    └── ProvidersSummary (值对象)
```

## 值对象

| 值对象 | 文件 | 说明 |
|--------|------|------|
| `FeatureParamsSource` | `value_objects/feature_params_source.py` | company / workspace / personal 枚举 |
| `SnapshotMeta` | `value_objects/snapshot_meta.py` | 解析来源 + config ID + 人读名称 |
| `ProvidersSummary` | `value_objects/providers_summary.py` | LLM 供应商脱敏摘要（api_key_hash 替代明文） |
| `ProviderSummaryEntry` | `value_objects/providers_summary.py` | 单个供应商摘要条目 |
| `LlmProviderEntry` (现有) | `tenant_llm_config/value_objects/llm_provider_entry.py` | LLM 供应商配置完整条目 |

## 领域服务

| 服务 | 文件 | 职责 |
|------|------|------|
| `FeatureParamsResolver` | `services/feature_params_resolver.py` | 多层级解析 + 治理检查 + 回退链 (Chain of Responsibility) |
| `FeatureParamsEnvSerializer` (现有) | `tenant_llm_config/services/feature_params_env_serializer.py` | TenantLlmConfig → env map 序列化 |

## 端口接口

| 端口 | 文件 | 方法 |
|------|------|------|
| `TenantFeatureParamsRepository` | `ports/repositories/tenant_feature_params_repository.py` | `find_by_company_id`, `save` |
| `WorkspaceFeatureParamsRepository` | `ports/repositories/workspace_feature_params_repository.py` | `find_by_workspace_id`, `save` |
| `PersonalFeatureParamsConfigRepository` | `ports/repositories/personal_feature_params_config_repository.py` | `find_by_id_and_user` (IDOR-safe), `find_all_by_user_id`, `save`, `delete` |
| `TaskFeatureParamsSnapshotRepository` | `ports/repositories/task_feature_params_snapshot_repository.py` | `save` (append-only), `find_by_task_id` |
| `WorkspaceGovernancePort` | `ports/repositories/workspace_governance_port.py` | `is_personal_config_allowed` |

## 领域事件

| 事件 | 文件 | 触发时机 |
|------|------|---------|
| `FeatureParamsResolved` | `events/feature_params_resolved.py` | 每次配置解析完成 → 驱动快照写入 |

## 应用服务

| 服务 | 文件 | 职责 |
|------|------|------|
| `FeatureParamsApplicationService` | `services/feature_params_application_service.py` | 编排 resolver + 序列化 + 快照写入 |

## 依赖反转验证

```
interfaces (API views) → application (FeatureParamsApplicationService)
                              ↓ depends on
                         domain/ports (ABC interfaces)
                              ↑ implements
                    infrastructure/adapters/persistence (Django ORM repos)
```

- ✅ 切换数据库：新建 DjangoAdapter，实现同一 Repository 接口 → 领域层零改动
- ✅ 切换治理检查方式：新建 ConfigFileGovernanceAdapter → 领域层零改动
- ✅ 切换快照存储：新建 S3SnapshotRepository → 领域层零改动

## NFR 决策对模型的影响

| NFR 决策 | 领域模型体现 |
|----------|-------------|
| 安全 L3: IDOR 防护 | `find_by_id_and_user(config_id, user_id)` — 强制双字段匹配 |
| 安全 L3: 快照脱敏 | `ProvidersSummary` 值对象分离脱敏视图；`to_summary_dict()` vs `to_detail_dict()` |
| 容错 L2: 多级回退 | Resolver 采用 Chain of Responsibility：personal → workspace → company |
| 审计 L2: 只追加 | `TaskFeatureParamsSnapshot` 聚合无 Update/Delete 操作，`save()` 语义为 INSERT |
| 可维护性 L2: 向后兼容 | `FeatureParamsSource.COMPANY` 为默认源；resolver 在缺失配置时自动回退 |
