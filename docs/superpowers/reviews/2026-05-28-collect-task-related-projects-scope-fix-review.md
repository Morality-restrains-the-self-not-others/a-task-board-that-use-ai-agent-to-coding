# Review：collect_task_related_projects 范围修复

**结论：** 可合并

## 变更摘要

- `collect_task_related_projects` 改为仅通过 `projects_taskproject` 查询任务关联项目
- relay 直接启动：`TASK_API_ENDPOINT_ORIGIN` 默认 `http://localhost:8001`（`relayToTraeUtils` + vite 本地 Django 配置）
- 新增/更新 pytest 14 项全绿；`relayToTraeUtils` 单测 32 项全绿
- 任务 `846269443533955072`：`github: None`，不再误扫 workspace 内其他 GitHub 项目

## 风险

- 低：语义与函数名、TaskProject 模型一致；调用方测试已对齐
