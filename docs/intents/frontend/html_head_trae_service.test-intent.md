# 测试意图：HTML head trae-service

## 对应功能意图

`html_head_trae_service.intent.md`

## 测试点

1. **单元**：`extract_trae_service` 能解析合法 meta；缺失时返回 `None`。
2. **门禁**：清单内每个入口 HTML 含正确 `content`；故意删 meta 时脚本 exit 1。
3. **手工/浏览器**：打开各服务入口页，DevTools 读取 `meta[name="trae-service"]` 与期望一致。

## 自动化落点

| 测试点 | 落点 |
|--------|------|
| 解析与清单校验 | `db/scripts/ci/test_check_frontend_head_trae_service.py` |
| 仓库现状门禁 | `python3 db/scripts/ci/check_frontend_head_trae_service.py` |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-17 | 初版 |
