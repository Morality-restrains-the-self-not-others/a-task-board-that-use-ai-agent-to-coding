# DDD 领域模型确认: 容器启动时功能参数来源选择

> 上游:
> - 设计文档: `docs/designs/env-var-preset-switching.md`
> - 价值流: `docs/superpowers/plans/2026-07-01-feature-params-source-selector-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-07-01-feature-params-source-selector-nfr-clarification.md`
>
> 结论: **不引入新领域概念，全部复用已有领域构件。**

## 已有领域构件复用清单

| 构件 | 路径 | 复用方式 |
|------|------|---------|
| `FeatureParamsSource` (VO) | `projects/domain/feature_params/value_objects/` | 预览端点接收 `source` 参数，用 `FeatureParamsSource(value)` 校验 |
| `FeatureParamsResolver` (Domain Service) | `projects/domain/feature_params/services/` | 预览端点调用 `resolver.resolve(source=personal, ...)` |
| `FeatureParamsEnvSerializer` (Domain Service) | `projects/domain/tenant_llm_config/services/` | 预览端点调用 `serializer.serialize(config)` 返回 env dict |
| `PersonalFeatureParamsConfig` (Entity) | `projects/domain/feature_params/entities/` | 归属校验: `config.user_id == request.user.id` |
| `Todo` (Entity) | `projects/models/todo.py` (已有 `feature_params_source` + `personal_feature_params_config_id` 字段) | 启动时写入 source + config_id |
| `WorkspaceGovernanceAdapter` | `projects/infrastructure/adapters/persistence/` | 已存在，`allow_personal_feature_params` 校验由前端读取 workspace 属性 |

## 不需要新增的构件

| 通常需要 | 本次不需要 | 原因 |
|---------|-----------|------|
| 新 Entity | ❌ | 预览是只读查询，不引入新持久化对象 |
| 新 Value Object | ❌ | `FeatureParamsSource` 已覆盖 company/workspace/personal |
| 新 Repository Port | ❌ | 查询直接使用 Django ORM（查询不走领域层，CLAUDE.md 允许视图层直接 ORM） |
| 新 Domain Event | ❌ | NFR 决定用 Django logging 而非领域事件 |
| 新 Domain Service | ❌ | `FeatureParamsResolver` 已覆盖所有解析路径 |
| 新 Application Service | ❌ | 预览是简单查询，view 函数直接编排（遵循项目现有 `cloud_compute_views.py` 风格） |

## 架构分层确认

```
interfaces (cloud/views/cloud_compute_views.py)
  ├── 新: get_feature_params_env_preview  ← view 函数，直接调用 resolver + serializer
  └── 改: post_relay_to_trae_start       ← 接收新参数，写入 Todo
        │
        ├── 调用 domain/services/FeatureParamsResolver (已有)
        ├── 调用 domain/services/FeatureParamsEnvSerializer (已有)
        └── 查询 models/PersonalFeatureParamsConfig (已有，归属校验)
```

**依赖方向**: views → domain services ✅ (view 依赖 domain，domain 不依赖 view)

## 自检

- [x] 无新增领域文件（全部复用）
- [x] 领域层无新增基础设施依赖
- [x] 依赖方向正确: views → domain
- [x] 不新建端口接口（无新外部依赖）
