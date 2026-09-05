# Value Stream 字段命名治理设计

**日期：** 2026-05-30  
**状态：** 已批准（0-auto-flow）  
**背景：** `value-stream.yaml` 字段名须为 `<service>.<table>.<column>` 三段式；2026-05-28、2026-05-30 两次因四段 JSON 路径导致 valueStream 启动失败。

## 1. 问题陈述

设计文档中的 YAML 示例（如 `providers.budget_enabled`）被原样复制到 `value-stream.yaml`，而 `ParseFieldName` 正则只允许三段。校验仅在 runAll 启动 valueStream 时触发，提交/CI 无门禁。

## 2. 价值流影响

| 项 | 影响 |
|----|------|
| 现有流 | 无业务流语义变更 |
| 新流 | `value-stream-config-governance`（系统管理与策略） |
| 字段 | `value-stream.config.production_yaml_valid` |
| 测试 | `tests/test_value_stream_config_governance.py`（pytest 桥接 Go 校验） |

## 3. 方案对比

| 方案 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **A. 仅 CI** | `go test TestLoadProductionValueStream` | 实现快 | 本地仍可能提交非法 YAML |
| **B. pre-commit + CI + 文档扫描（推荐）** | Go 测试 + 设计文档 `- name:` 扫描 + skill 约束 | 前移发现点，阻断污染源 | 略增 pre-commit 时间 |
| **C. 仅文档/Skill** | 不改自动化 | 零代码 | 复发风险高 |

**选定：B**

## 4. 架构

```text
[设计文档 YAML 块 / value-stream.yaml]
  → ParseFieldName（三段正则）
  → TestLoadProductionValueStream（加载根目录生产配置）
  → TestDesignDocFieldNames（扫描 docs/specs/*-design.md）
  → pre-commit（暂存 value-stream.yaml 或 specs 时跑 Go 测试）
  → scripts/ci/check_ddd_bdd_compliance.py（CI 兜底）
  → pytest 桥接（value-stream UI 可执行）
```

### JSON 嵌套字段映射规则

| 逻辑路径 | 合法 `fields[].name` | `description` 保留 |
|----------|----------------------|---------------------|
| `providers[].budget_enabled` | `saas-backend.projects_tenant_feature_params.providers_budget_enabled` | `providers[].budget_enabled` |
| `providers[].use_sub_token` | `saas-backend.projects_tenant_feature_params.providers_use_sub_token` | `providers[].use_sub_token` |
| `port_config.domainEvents.transport` | `saas-backend.config.domain_events_transport` | JSON 路径 |

**规则：** `name` 永远三段；JSON/嵌套语义写在 `description`。

## 5. 组件

| 组件 | 职责 |
|------|------|
| `valueStream/src/production_config_test.go` | 加载 `../../value-stream.yaml` 全量校验 |
| `valueStream/src/design_doc_fields_test.go` | 扫描 `docs/superpowers/specs/*-design.md` 中 `- name:` 行 |
| `valueStream/scripts/hooks/pre-commit` | 暂存相关文件时跑治理测试 |
| `scripts/ci/check_ddd_bdd_compliance.py` | CI 调用 `go test -run TestLoadProduction\|TestDesignDoc` |
| `tests/test_value_stream_config_governance.py` | pytest 桥接，供 value-stream.yaml 登记 |
| Skill `/1-brainstorming`、`/3-value-stream` | 增加 JSON 扁平化示例与 ai.md 引用 |

## 6. 文档修正（一次性）

- `docs/superpowers/specs/2026-05-30-llm-budget-endpoint-level-design.md`
- `docs/superpowers/specs/2026-05-30-task-llm-budget-limit-design.md`

## 7. 验收标准

1. `cd valueStream && go test ./src/... -run 'TestLoadProduction|TestDesignDoc'` 通过。
2. 故意在四段 `name` 的 design doc 或 value-stream.yaml 中插入违规项时，上述测试失败。
3. valueStream 启动无 Config error。
4. 设计文档示例均为三段式 `name`。

## 8. 领域概念（轻量）

| 概念 | 说明 |
|------|------|
| `FieldName` | 值对象：service + table + column，ParseFieldName 校验 |
| `ConfigGovernance` | valueStream 启动前配置治理，非业务域 |
