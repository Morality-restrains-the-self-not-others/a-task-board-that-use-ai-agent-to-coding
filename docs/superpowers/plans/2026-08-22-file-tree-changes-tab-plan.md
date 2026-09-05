# 实施计划 — 项目文件树与变动列表 Tab

- **日期**: 2026-08-22
- **设计 / 权限 / 价值流 / NFR / DDD**: 同主题 `2026-08-22-file-tree-changes-tab-*`

## 事件任务（例外）

- [x] 无新事件契约：纯前端 Tab。意图已写例外。

## Tasks

- [x] **T1** Red：`TaskDetailLayerFilesTabs.test.js` — 默认 changes、切换 v-show、计数徽章、layerId 复位
- [x] **T2** Green：`TaskDetailLayerFilesTabs.vue`
- [x] **T3** Red：`TaskDetailTaskLayerAssociationPanel.layer-files-tabs.test.js` — 选中层出现 tablist；点变动 Tab 后 layer-changes 可见
- [x] **T4** Green：面板改用 Tabs 包裹两个既有组件
- [x] **T5** Playwright helper `openLayerFilesChangesTab` + 更新既有 layer-changes 测例先点 Tab
- [x] **T6** 价值流图测试点；登记精准重启 `taskFE`

## 验收

Vitest 上述文件全绿；相关 Playwright mock 用例在切 Tab 后仍能操作变动列表与文件树。
