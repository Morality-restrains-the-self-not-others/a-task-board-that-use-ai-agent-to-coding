# 功能意图：GitLab 区域 access_mode 门禁（后端）

- **日期**: 2026-08-23
- **状态**: 实施中
- **服务**: taskBill

## 背景与目标

区域持久化 `access_mode`；租户目录与购买按测试头过滤/拒绝；管理 API 可读写模式。

## 范围与边界

- 范围内：DDL、list/get/create/update、购买门禁、事件。
- 范围外：OIDC 直连 GitLab。

## 验收标准

同 `system_admin_gitlab_region_access_mode.intent.md` 后端条目。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 消费者 |
|----------|--------|----|--------|--------|
| 更改区域模式 | GitlabRegionAccessModeChanged | gitlab-region-access-mode-changed | 系统管理创建/更新 | 无自动消费者 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-23 | 初稿 |
