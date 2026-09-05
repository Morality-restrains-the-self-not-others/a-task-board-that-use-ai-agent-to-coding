# 实施计划：工作面板进度纵轴恢复

- **日期**: 2026-07-15
- **设计**: `docs/superpowers/specs/2026-07-15-work-panel-progress-vertical-lanes-design.md`

## 任务

- [x] 更新意图 004（纵=进度；下拉换纵轴）
- [x] `DeliverableKanbanBoard.vue` 恢复 `.progress-lane`
- [x] `TaskPanel.vue` 拖拽写回 `progress_column_id`；下拉按同交付物计数 order
- [x] 更新面包屑 Playwright；新增 `progress-status-moves-vertical-lane` E2E
- [ ] 单元测试（既有 aggregation / kanban utils）
- [ ] `runall-lifecycle.sh build`（build + collectstatic）
- [ ] Playwright 冒烟 / 相关用例

## 验证命令

```bash
cd task2app/front_project/app && npx vitest run src/utils/workPanelKanbanUtils.test.js src/utils/workPanelDeliverableAggregation.test.js
bash scripts/runall-lifecycle.sh build
# Playwright（需本机服务 + 凭证）
```
