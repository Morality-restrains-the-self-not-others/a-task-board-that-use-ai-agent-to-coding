# [运行时] 新建任务基准分支 datalist 选中后再次弹出

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 工作面板「新建开发任务」中，基准分支输入框（`#task-base-branch-0-0`，原生 `<datalist>`）从下拉选中分支后，建议列表会立刻再次弹出。
- 目标分支 / 工作分支 datalist 在同类受控绑定下也可能复现。

## 根因

1. Vue `v-model` 受控 input + 原生 `list`/`<datalist>`：选中候选触发 `input`/`change`，值写入响应式状态后重渲染，输入框仍聚焦。
2. Chrome 对「聚焦且值匹配候选」的 input 会再次打开建议 UI。
3. 仅 `await nextTick()` 后 `blur()` **不足**：模板上的 `:list="..."` 在补丁阶段会把 `list` 写回，建议列表仍可被浏览器再次拉起。生产包（`WorkPanel-*.js`）已含旧 blur 逻辑仍复现。

## 解决方案

- 抽取 `createDatalistDismissController`（`front_project/app/src/utils/datalistDismiss.js`）：
  - 选中候选（精确匹配）时：将该 input `id` 加入抑制集，使 `:list` 变为 `undefined`
  - **同步** `removeAttribute('list')` 再 `blur`，避免等 Vue 补丁前的一帧被 Chrome 再次拉起
  - `@input` 仅在 `insertReplacementText` / `insertFromDrop`（datalist 点选）时 dismiss；普通 `insertText` 不打断输入
  - `@change` 始终按精确匹配 dismiss；下次 `@focus` 恢复 `list`
- 接入：`CreateTaskModal.vue`、`TaskDetailBranchStrategyPanel.vue`、`TaskDetailLinkedProjectsPanel.vue`
- 注意：该 util **必须入库**；仅改 Vue 引用而未提交 `datalistDismiss.js` 会导致公网仍为旧 blur-only 逻辑

## 预防

- 凡 Vue 受控 text + 原生 datalist，禁止只依赖 blur；须暂时摘掉 `list`（或改用自定义下拉）。
- 回归：`datalistDismiss.test.js`、`CreateTaskModal.test.js`（选中后 `list` 被摘掉、focus 后恢复）。
- 公网生效须 `bash scripts/runall-lifecycle.sh build`（build + collectstatic）。

## 验证

```bash
cd taskFE/app
npm test -- --run src/utils/datalistDismiss.test.js src/components/CreateTaskModal.test.js
bash scripts/runall-lifecycle.sh build
```
