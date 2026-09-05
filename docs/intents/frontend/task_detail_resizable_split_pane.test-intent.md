# 任务详情可拖动分隔 — 测试意图

## 单元

1. `clampLeftWidth`：低于 min / 高于 max / 超过容器 70% 时钳制正确
2. `readStoredLeftWidth` / `writeStoredLeftWidth` 往返
3. `ResizableSplitPane` 渲染 separator（`role=separator`、`cursor-col-resize`）与左右 slot
4. 键盘 ArrowRight 增加 `--split-left-width`
5. 既有 `TaskDetailProjectFileTree.refresh-stability`：`project-file-tree-body` 仍存在，刷新时 `opacity-70`

## 手工 / 公网

1. 打开任务详情 → 展开文件变动 → 拖动目录栏与预览之间的竖条，左侧宽度变化
2. 刷新页面后宽度保持（localStorage：`task-detail-layer-changes-split`）
3. 项目文件树同样可拖（`task-detail-project-file-tree-split`）
