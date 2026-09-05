# Value Stream: value-stream 字段命名治理

> Derived from: `docs/superpowers/specs/2026-05-30-value-stream-field-naming-governance-design.md`

## Related Value Streams

- **Greenfield tooling** — 无业务流变更；新增 `value-stream-config-governance` 元治理流。

## Value Summary

开发者在修改 `value-stream.yaml` 或设计文档中的字段示例时，**提交前**即可发现三段式违规，不再等到 runAll 启动 valueStream 才报错。

## Increments

### Increment 1: 生产配置加载测试（Thin Slice）

**Value:** 根目录 `value-stream.yaml` 在 CI 中自动校验，与 valueStream 启动等价。

**Scope:** `TestLoadProductionValueStream`、`check_ddd_bdd_compliance.py` 集成。

### Increment 2: 设计文档污染扫描

**Value:** 设计文档中的 `- name:` YAML 块不能再写入四段字段名。

**Scope:** `TestDesignDocFieldNames`、修正 llm-budget 两篇 specs。

### Increment 3: pre-commit + Skill

**Value:** 本地暂存相关文件时即时反馈；AI 流程不再产出非法示例。

**Scope:** valueStream pre-commit、pytest 桥接、skill 更新、`value-stream.yaml` 登记。

## YAML 登记

```yaml
- name: value-stream-config-governance
  domain: 系统管理与策略
  description: value-stream.yaml 字段三段式治理（pre-commit + CI）
  steps:
    - name: production-config-validation
      status: active
      test_file: tests/test_value_stream_config_governance.py
      fields:
        - name: value-stream.config.production_yaml_valid
          description: 根目录 value-stream.yaml 通过 LoadConfig 校验
        - name: value-stream.config.design_doc_field_names_valid
          description: docs/specs/*-design.md 中 - name: 行须三段式
```
