# 价值流 — Work Panel 任务状态 SSE

## 最小可交付增量（MVP）

用户打开看板 → 看到当前任务列 → **他人/他标签改进度后本机卡片秒级更新**。

## 端到端步骤

1. 用户打开 work-panel（价值起点）
2. HTTP 拉取 todos + columns（已有）
3. 订阅 work-panel-events-sse（新增）
4. 协作者 PATCH 进度（已有写路径）
5. TASK_STATUS_CHANGED → fan-out → SSE → 卡片 patch（新增）
6. 用户看到一致看板（价值终点）

## 测试点映射

见 `docs/intents/*/work_panel_task_status_sse.test-intent.md`（T1–T6 / B1–B4）。
