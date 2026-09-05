# 功能意图：系统管理用户「测试」角色

- **日期**: 2026-08-23
- **状态**: 实施中
- **页面**: `/system-admin/users/`
- **服务**: taskAuth + taskFE

## 背景与目标

为账号增加角色类别「测试」。测试等同租户：可使用租户产品能力，并额外可使用开发模式 GitLab 区域。

## 范围与边界

- 范围内：列表筛选/展示、新增与编辑勾选 `is_tester`、JSON 字段、网关 `X-User-Is-Tester`、Kafka 事件。
- 范围外：把 tester 做成平台 RBAC；给测试账号系统管理菜单。

## 约束与风险

- 禁止 tester 进入 `X-User-Roles`。
- 勾选测试必须同时 `is_tenant=1`。
- 日志不输出邮箱/手机号。

## 验收标准

1. 角色筛选项含「测试」；选中后列表仅 `is_tester=1` 用户。
2. 列表角色列对测试账号显示「测试」（优先于「租户」）。
3. 新增/编辑可勾选「测试」；保存后 `is_tester=true` 且 `is_tenant=true`。
4. `/me/` 含 `is_tester`。
5. 测试账号请求经网关后下游可见 `X-User-Is-Tester: 1`。
6. 设置/取消测试投递 `UserTesterFlagChanged`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 消费者 | 例外理由 |
|----------|--------|----|--------|--------|----------|
| 设置或取消测试角色 | UserTesterFlagChanged | user-tester-flag-changed | patchUserAsAdmin / 创建用户 | 无自动消费者（审计）；handler 清 forward-auth 缓存 | — |
| 列表筛选/展示 | — | — | — | — | 纯查询 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-23 | 初稿 |
