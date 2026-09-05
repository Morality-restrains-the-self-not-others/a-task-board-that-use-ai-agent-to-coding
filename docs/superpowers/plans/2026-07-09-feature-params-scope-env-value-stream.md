# 价值流 — TASK_FEATURE_PARAMS_SCOPE

- **日期**: 2026-07-09
- **设计**: `docs/superpowers/specs/2026-07-09-feature-params-scope-env-design.md`

## 价值主张

配置消费者（容器脚本 / Agent）能从环境变量直接识别当前功能参数来自公司、工作空间还是个人，无需另查任务元数据。

## 最小增量（单增量交付）

| 步骤 | 名称 | 验收 |
|------|------|------|
| VS1 | 序列化注入 scope | `FeatureParamsEnvSerializer` 输出含正确 `TASK_FEATURE_PARAMS_SCOPE`；用户同名键无法覆盖 |
| VS2 | 设置页预览一致 | 三页 `systemEnv` / `env_preview` 展示对应值 |
| VS3 | 容器拉取一致 | `resolve_and_snapshot` 使用 `meta.source` 注入 |

## 测试映射

- VS1 → `Saas_project/tests/test_tenant_feature_params_env.py`
- VS2 → 前端 systemEnv（必要时轻量组件测）+ API env_preview 断言
- VS3 → resolver/application service 相关测试扩展
