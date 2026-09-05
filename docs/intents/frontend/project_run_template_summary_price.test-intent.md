# 测试意图：项目详情运行模版摘要显示价格

对应功能意图：`project_run_template_summary_price.intent.md`

## 单元测试

| 用例 | 覆盖点 | 位置 |
|------|--------|------|
| extractPriceLabelFromHardwareSpecSummary | 从规格摘要提取价格段；加载中；无价格为空 | `projectRunTemplateUtils.test.js` |
| appendPriceToRunTemplateSummary | 追加价格、去重、未设置不追加 | 同上 |
| summarizeRunTemplateWithPrice | 身份摘要 + 规格摘要中的价格；纯金额补「价格」前缀 | 同上 |

## 手工 / E2E 检查点

1. 打开已配置阿里云实例的项目详情，等待硬件面板价格返回。
2. `data-testid="project-run-template-summary-text"` 文案含 `价格` 与 `元/时`（或加载中文案）。
3. 同页「规格」行仍含 CPU/内存/GPU/价格。

## 变更记录

| 日期 | 差异 |
|------|------|
| 2026-07-20 | 新增与功能意图对齐的单测与手工点 |
