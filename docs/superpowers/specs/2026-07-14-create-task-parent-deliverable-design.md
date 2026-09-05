# 创建任务：非顶层类别须选上层交付物

- **日期**: 2026-07-14
- **状态**: approved（goal-mode 自动采用）
- **范围**: 前端 CreateTaskModal + WorkPanel 提交；复用既有 `parent_task` 字段
- **架构影响**: 无

## 方案

- 顶层类别 = 体系内 `order` 最小；非顶层展示「上层交付物」下拉
- 候选 = 上一 `order` 层任务；必选后方可创建
- 提交写入 `parent_task`；无需新 API
