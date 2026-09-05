# 功能意图：多区域 Registry 副本与同区优先拉取

- **日期**: 2026-08-29
- **状态**: 已批准
- **接口**: 扩展 `POST/GET /api/vendor/container-images/{id}/cloud-server-image-association(s)/`；公开 catalog runtime_environments；start-vm 选址（无新 path）
- **页面**: 厂商门户区域运行环境

## 背景与目标

逻辑容器镜像只有一条 `image_url`，评论启机跨区拉仓库。目标：厂商按 CSI 区域登记公网仓库副本，平台推导内网地址；启机优先同区内网，否则同区公网，再否则规范 URL。

## 范围与边界

- 范围内：副本表、catalog/安装快照、start-vm 选址、registryhost 公网→内网、厂商门户字段。
- 范围外：平台 Harbor、代推 VPC、强制每区副本、改 relay resolve 为内网、手填内网 URL。

## 约束与风险

- 控制面禁止探测 VPC Registry（既有 RejectPrivateRegistry）。
- 副本键与 association 同为 `(platform_type, region)`；清 CSI 级联删副本。
- taskAiProvider 事件经 LogEventBus；须登记证据豁免。
- 单表所有权：源表 ai-provider；快照 task-cloud。

## 验收标准

1. 杭州公网 ACR URL 保存后 `intranet_url` 为 `registry-vpc.cn-hangzhou.aliyuncs.com` + 原 path。
2. 提交 VPC host 作为 public_url → 400。
3. start-vm `region_id` 匹配副本 → UserData 用 intranet（有则）否则 public。
4. 无副本安装行 → `container_image_url` 与今日 `image_url:version` 一致。
5. `GET /api/internal/image/resolve` 仍为规范公网引用。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 登记或更新区域仓库副本 | ContainerImageReplicasUpdated | LogEventBus | taskAiProvider upsert 成功 | 审计 | 证据豁免：taskAiProvider 无 Kafka |
| 清除区域副本 | ContainerImageReplicasUpdated | 同上 | csiID=0 级联删除后 | 审计 | 同上 |
| 启机选用拉取 URL | — | 既有 CLOUD_SERVER_STARTED | taskCloudService | UserData docker pull | 无新事实；写入现有事件 data.container_image_url |

## 实施计划

见设计文档第 13 节。

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-29 | 初稿 |
