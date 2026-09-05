# 删除 Django rewrite_sub_token_providers_for_proxy

- **状态**: completed（2026-07-14 完成，Python 代码已删除，无回归）
- **归档说明**: 该功能已完整迁移至 Go `taskCloudService/feature_params_proxy_rewrite.go`，Python 源文件及测试引用已清理。本意图保留作为迁移历史记录，不再有活跃测试意图。

## 意图

删除 Python `projects.services.task_ai_endpoint_service.rewrite_sub_token_providers_for_proxy`；运行时 SSOT 为 taskCloudService `feature_params_proxy_rewrite.go`。

## 验收

1. `task_ai_endpoint_service.py` 已删除
2. Django 测试不再引用该函数；仅断言 `task-ai-endpoint` internal URL 404
3. Go `TestRewriteSubTokenProvidersForProxy` / FeatureParams proxy 单测仍绿

## 变更日期

2026-07-14


## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：删除 Python 实现，无新增业务事件投递。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 删除 Django rewrite_sub_token_providers_for_proxy | — | — | — | — | 删除 Python 实现，无新增业务事件投递 |
