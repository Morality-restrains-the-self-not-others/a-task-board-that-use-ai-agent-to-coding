# 创建任务默认置顶 — 测试意图

## 变更记录

- 2026-07-13：与功能意图同步创建。

## 测试意图

1. **连续创建顺序**：同一工作区先后创建任务 A、B 后，列表接口返回顺序为 B、A（新在前）。
2. **order 数值**：后创建任务的 `order` 严格小于先创建任务。
3. **空工作区首任务**：无既有任务时创建成功，`order` 为 0，列表仅含该任务。
4. **回归**：拖拽重排仍可把任务移到任意位置（既有拖拽写 `order: index` 行为不变）。

## 自动化落点

- `taskTaskService/src/handlers_test.go`：`TestCreateTaskDefaultsToTopOfList`
