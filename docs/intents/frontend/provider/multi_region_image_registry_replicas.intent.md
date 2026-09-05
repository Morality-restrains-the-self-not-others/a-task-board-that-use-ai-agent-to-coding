# 功能意图：厂商按区域登记容器仓库公网地址

- **日期**: 2026-08-29
- **状态**: 已批准
- **接口**: 区域运行环境 association（后端 B-082）
- **页面**: 厂商门户「区域运行环境」弹层

## 背景与目标

厂商只填一条规范 `image_url` 时，各区服务器无法声明同区仓库。目标：在已有 CSI 区域行上可选填写该区公网 Registry，展示平台推导的内网地址（只读）。

## 范围与边界

- 范围内：区域运行环境表单字段、保存 payload `registry_public_url`、只读 `intranet_url`、错误节点 `data-traceId`。
- 范围外：手填内网、平台代推、拆 VendorPortal 行数（沿用现有 tab/modal 文件）。

## 约束与风险

- 内网地址只展示 API 返回值，前端不自己改写 host（SSOT 在 Go registryhost）。
- 保存须同步门闩（既有 association 保存按钮防重放约定）。
- 纯展示内网只读框：`Anti-Replay-OK: read-only derived field`。

## 验收标准

1. 填写杭州公网 ACR 并保存 → 弹层显示推导的 `registry-vpc.cn-hangzhou...`。
2. 填写 VPC host → 错误文案含无法触及，且带 `data-traceId`。
3. 留空公网仓库 → 仍可只绑 CSI；不展示内网。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|----------|--------|---------|
| 保存区域仓库副本 | ContainerImageReplicasUpdated | 前端只调 API；事件由 taskAiProvider 发布（见后端意图） |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-29 | 初稿 |
