# 功能意图：容器→SaaS 接口版本（taskAiProvider）

- **日期**: 2026-08-20
- **状态**: 已实施
- **服务**: taskAiProvider

## 背景与目标

镜像行必须声明已发布的 inbound skill 版本；公开目录与审核 API 返回该字段；未知版本拒绝写入。

## 验收标准

1. `GET /api/ai-provider/saas-inbound-skill-versions/` 返回 current + versions（公开）。
2. `GET /saas-machine-container.md?version=1` 返回 v1 原文；未知 version 404。
3. 创建/更新镜像缺少或非 published 版本 → 400。
4. 列表/详情/公开目录 JSON 含 `saas_inbound_skill_version`。
5. 存量行迁移后为 `"1"`。
6. 写成功发布 `ContainerImageSaasInboundSkillVersionAssigned`（LogEventBus）。

## 业务意图 → 事件对照

| 业务意图 | 事件 | 发布点 | 消费者 |
|----------|------|--------|--------|
| 镜像写入接口版本 | ContainerImageSaasInboundSkillVersionAssigned | vendor container-images POST/PUT | LogEventBus |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-20 | 初版 |
| 2026-09-01 | 生产 clone-run 无 docs 子仓导致 GET catalog 500；运行时改为 embed 副本 |
