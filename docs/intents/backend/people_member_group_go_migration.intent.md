# 人员页成员/邀请/分组 API 迁 Go

> 状态: **in_progress** | 日期: 2026-07-19  
> 设计：`docs/superpowers/specs/2026-07-19-people-member-group-go-migration-design.md`

## 意图

将租户人员相关公网 API 从 Django 迁至 **taskTenantService**，并清理 Django 公网入口；四表迁入 `data/task_tenant.db`。

## 验收

1. 网关将 `/api/tenant/*/accounts/members/**` 与 `.../groups/**` 路由到 task-tenant-service:8020
2. 邀请 / 管理人员 / 管理分组三页功能与迁前一致
3. Django 公网 members/groups ViewSet 不可达（404/410）
4. `table_ownership.yaml` 四表 owner = task-tenant-service
5. `INVITATION_CREATED` / `MEMBER_JOINED` 仍在成功写路径发布

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 发送或重发公司邀请 | INVITATION_CREATED | Kafka INVITATION_CREATED | taskTenantService invite/resend | taskEvents 发邮/短信 | — |
| 用户接受邀请加入公司 | MEMBER_JOINED | Kafka MEMBER_JOINED | taskTenantService join | taskEvents 1_create_default_git_identity + 公司切换器等 | — |
| 查询成员/分组/校验邀请 | — | — | — | — | 纯查询 |
