# 工作面板：进度状态下拉后同横轴换纵轴

- **日期**: 2026-07-15
- **状态**: approved（goal-mode 自动采用）
- **范围**: `task2app/front_project` 工作面板 UI；不新增后端 HTTP
- **架构影响**: 无

## 1. 问题

看板曾改为「每交付物纵列仅一个任务区」，导致卡片上改「进度状态」只写库、**视觉不换位**。用户期望：改进度状态 → 卡片在**同一交付物横轴**内移到对应**进度纵轴**。

## 2. 方案

恢复设计方案 B（2026-07-13）：横=交付物、纵=进度（`.progress-lane`）。

| 交互 | 行为 |
|------|------|
| 下拉改进度 | PATCH `progress_column_id` + order → `tasks-updated` 刷新 → 卡片进目标 lane |
| 跨进度拖拽 | 写 `progress_column_id` |
| 跨交付物拖拽 | 写 `deliverable_obj_id`（可兼带进度） |

## 3. 验收

1. `.deliverable-column` 内 `.progress-lane` 数 ≥ 进度体系列数（通常 ≥2）
2. 下拉改进度后，卡片仅出现在同交付物栏目标 lane
3. 既有跨交付物 / 跨进度拖拽 Playwright 不回归
