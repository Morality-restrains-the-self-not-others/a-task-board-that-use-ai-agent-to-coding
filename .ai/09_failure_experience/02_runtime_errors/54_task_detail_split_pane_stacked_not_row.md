# [运行时] 任务详情文件变动分栏变成上下堆叠

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-19
- 最后修改：2026-07-19
- 维护者：Trae AI 团队

## 现象

- 页面：任务详情「文件变动」`[data-testid="resizable-split-pane"]`
- 期望：左侧变动列表 + 右侧预览左右排列
- 实际：上下堆叠（列表在上、预览在下），分隔条不显示

## 根因

`ResizableSplitPane` 使用 `flex-col` + `md:flex-row`，且 gutter 为 `hidden md:flex`。在视口宽度 &lt; Tailwind `md`（768px）、或主内容区虽窄但用户期望桌面分栏时，会退化成纵向布局。

## 解决方案

固定 `flex-row`；左栏 `w-[var(--split-left-width)]` + 内联 `width` 兜底；gutter 始终可见。

## 验证

```bash
cd taskFE/app && npx vitest run src/components/ResizableSplitPane.unit.test.js
bash scripts/runall-lifecycle.sh build
```

硬刷新任务详情 → 文件变动区应为左右分栏。

## 关联

- `taskFE/app/src/components/ResizableSplitPane.vue`
