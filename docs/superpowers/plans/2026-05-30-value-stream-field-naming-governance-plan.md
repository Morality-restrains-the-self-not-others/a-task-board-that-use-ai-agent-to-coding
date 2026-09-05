# value-stream 字段命名治理 — 实施计划

> Design: `docs/superpowers/specs/2026-05-30-value-stream-field-naming-governance-design.md`

## Slice 1: Go 测试

- [x] **T1** `production_config_test.go` — `TestLoadProductionValueStream`
- [x] **T2** `design_doc_fields_test.go` — `TestDesignDocFieldNames`
- [x] **T3** `repo_root.go` — 共享仓库根路径解析

## Slice 2: 集成与登记

- [x] **T4** `tests/test_value_stream_config_governance.py` pytest 桥接
- [x] **T5** `value-stream.yaml` 新增 `value-stream-config-governance` 流
- [x] **T6** `scripts/ci/check_ddd_bdd_compliance.py` 调用 Go 治理测试
- [x] **T7** `valueStream/scripts/hooks/pre-commit` 暂存触发

## Slice 3: 文档与 Skill

- [x] **T8** 修正 llm-budget 两篇 design specs 字段示例
- [x] **T9** 更新 `/1-brainstorming`、`/3-value-stream` skill
- [x] **T10** `value-stream.yaml.ai.md` 增补 JSON 嵌套映射行

## 验证

```bash
cd valueStream && go test ./src/... -count=1 -run 'TestLoadProduction|TestDesignDoc'
cd task2app/Saas_project && DJANGO_SETTINGS_MODULE=saas_project.settings_test \
  pytest tests/test_value_stream_config_governance.py -q
cd valueStream && ./bin/valueStream --config ../value-stream.yaml --ui-port :9999 &
# 无 Config error
```
