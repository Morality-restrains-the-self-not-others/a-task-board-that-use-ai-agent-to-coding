# 公司成员多 Git 身份与加入自动创建

> 状态: **implemented** | 日期: 2026-08-12  
> 设计：`docs/superpowers/specs/2026-08-12-member-joined-auto-git-identity-design.md`

## 意图

一名公司成员可持有多个 Git 身份；创建成员成功后发布 `MEMBER_JOINED`；消费者自动创建默认身份（哈希邮箱 + 成员名）；租户/小组管理员与成员本人可管理。

## 验收

1. 所有成员创建路径（invite join / 建公司创建者 / 内部 upsert 新建）发布 `MEMBER_JOINED`
2. `member_joined/1_create_default_git_identity` 消费后写入 `task_git_identities`，邮箱格式 `{hash16(member_id)}.{hash16(tenant_id)}@daydaymoney.com`，用户名为成员名
3. 同一成员重复事件幂等，不重复插入
4. PeopleManage 成员行可管理该公司下 Git 身份；本人亦可在 UserGitIdentities 管理
5. 鉴权：self / `member:manage` / `group-members:manage`

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 消费者 | 例外 |
|---------|--------|-----|--------|--------|------|
| 公司成员行创建成功 | MEMBER_JOINED | Kafka member-joined | taskTenantService | taskEvents 1_create_default_git_identity | — |
| 查询/管理 Git 身份 | — | — | — | — | 纯查询/同步 CRUD |
