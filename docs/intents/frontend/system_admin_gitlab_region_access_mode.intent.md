# 功能意图：GitLab 镜像区域开发/发布模式

- **日期**: 2026-08-23
- **状态**: 实施中
- **页面**: `/system-admin/gitlab-resources`
- **服务**: taskBill + taskFE（门禁亦约束 taskGitOauth 绑区）

## 背景与目标

为镜像区域设置开发模式与发布模式。开发模式仅测试角色账号可使用。

## 范围与边界

- 范围内：区域字段 `access_mode`、管理页展示与保存、租户目录过滤、购买/开通/已购摘要门禁、Kafka 事件。
- 范围外：GitLab 实例 OIDC 直连登录拦截；改 GitLab 容器本身。

## 约束与风险

- 存量默认 `release`。
- 系统管理列表不过滤 development。
- 下游只读 `X-User-Is-Tester`，不查 auth 库。
- 错误带 `trace_id` / 前端 `data-traceId`。

## 验收标准

1. 区域卡片可见模式（开发/发布）；编辑可改并保存。
2. 新建区域可选模式，默认发布。
3. 非测试账号 GET 区域目录不含 development。
4. 测试账号目录含 development。
5. 非测试购买/连接 development → 403，文案含「仅测试角色」。
6. 改模式投递 `GitlabRegionAccessModeChanged`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 消费者 | 例外理由 |
|----------|--------|----|--------|--------|----------|
| 创建或更新区域 access_mode | GitlabRegionAccessModeChanged | gitlab-region-access-mode-changed | 系统管理创建/更新区域 | 无自动消费者（审计） | — |
| 租户列出可用区域 | — | — | — | — | 纯查询 |
| 拒绝使用开发区 | — | — | — | — | 拒绝无状态变更；打 warn 日志 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-23 | 初稿 |
