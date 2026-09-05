# 意图：UserData 模板逐步进度经 TASK_API_ENDPOINT → SSE

- **日期**: 2026-07-15
- **状态**: implemented

## 背景

云主机 UserData 初始化（安装依赖、Docker、拉镜像、启动容器）耗时长，任务详情页此前仅能看到平台侧粗粒度进度，看不到主机脚本逐步状态。

## 目标

1. 厂商门户「编辑 UserData 模板」生成的模板内容，在每个主要步骤后向 `TASK_API_ENDPOINT` 上报进度。
2. 平台经既有 `server-startup-status-sse` 转发，前端进度条/状态文案可见。

## 范围

- 新增 Go：`POST .../server-container-token/boot-progress/`（taskCloudService + taskAgentSupport 转发）
- 修改：`userDataScriptLinux.js` / `userDataScriptWindows.js` / 脚本生成器默认云前缀占位符
- 文档：`machine_container.md` §4.5a

## 验收

- [ ] 生成脚本含 `report_progress` / `Report-Progress` 与 `boot-progress` 路径
- [ ] Go 单测：progress 上限 99；success 改写为 processing
- [ ] 前端 unit：Linux/Windows 模板断言

## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| UserData 启动步骤进度上报 | SseMessagePublished | SSE_MESSAGE | taskCloudService boot-progress → server-startup-status-sse | task-sse / 任务详情进度条 | — |
