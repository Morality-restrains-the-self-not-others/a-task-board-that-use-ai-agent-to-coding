# 016 — 交付物过滤栏下拉展示任务编号

- 日期：2026-08-16
- 相关 URL：`/tenant/{tenantId}/work-panel`
- 元素：`nav.deliverable-filter-trail` / `[data-alias="deliverable-content-select"]`

## 背景与目标

过滤栏各类别下拉目前只渲染任务标题。同名任务（如多次「写一个 hello world程序」）无法区分。看板卡片已用人读序号 `#N`（`workspace_seq`）标识任务，下拉应与之对齐。

## 范围与边界

- **范围内**：工作面板交付物过滤栏内容下拉；创建/编辑任务「上层交付物」下拉（同类任务选项）。
- **范围外**：不改任务 ID 生成、不改过滤语义（仍按 content.id / 任务子树过滤）。

## 约束与风险

- 编号 SSOT 为 `workspace_seq` → `formatTaskDisplayNo` / `formatTaskIdTitleLabel`；无序号时不回退技术 ID 后六位。
- 原生 `<select>` 宽度较窄，编号须出现在 option 文案开头，关闭态也能看见。

## 验收标准

1. 有 `workspace_seq` 时，过滤栏 option 文案为 `#N 标题`。
2. 无序号时 option 仍为标题。
3. 上层交付物下拉同样展示 `#N 标题`。
4. 选中 option 的 value 仍为任务 id，过滤行为不变。

## 实施计划

1. 聚合层内容对象携带 `workspace_seq`。
2. 下拉用 `formatTaskIdTitleLabel` 渲染。
3. 单测覆盖聚合与组件 option 文案。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 过滤栏下拉展示任务编号 | — | — | — | — | 纯前端展示，无领域状态变更 |
