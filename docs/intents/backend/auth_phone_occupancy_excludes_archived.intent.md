# 功能意图：手机/邮箱占用仅计活跃用户

## 背景与目标

超管用户列表默认「活跃」页（`is_archived=false`）看不到已归档用户，但手机注册仍提示「该手机号已被注册，请直接登录」。根因是占用查询只看 `auth_login_method` 未作废绑定，不要求对应 `auth_user` 存在且未归档。归档操作也不作废登录方式。

目标：注册/绑定占用 = 仅活跃用户；新注册成功后作废归档/孤儿遗留绑定；超管按手机/邮箱/`q` 检索时即使在活跃页也能看到占用者。

## 范围与边界

- 范围内：`findLoginMethodByPhone` / `ByEmail` / `ByIdentifier`；手机/邮箱注册作废陈旧绑定；超管列表在标识检索时跳过 `is_archived=false`。
- 范围外：解档时的标识冲突 409（作废后旧用户解档不再持有该号）；前端改默认 Tab。

## 约束与风险

- 日志禁止输出完整手机号/邮箱；可记 `voided_count`。
- 不作废活跃用户的绑定。

## 验收标准

1. 占用者已归档或用户行缺失时，手机注册返回 201，旧绑定 `binding_voided_at` 非空。
2. 活跃占用者仍 400，文案「该手机号已被注册，请直接登录」。
3. `is_archived=false&phone=` 能返回已归档占用者。
4. `findLoginMethodByEmail` 忽略已归档占用者。

## 实施计划

1. 查找 SQL INNER JOIN 活跃 `auth_user`。
2. 注册成功路径作废陈旧 phone/email 绑定。
3. 列表 `localWhere` 在标识检索时不套用 `is_archived=false`。

## 业务意图 → 事件对照

| 意图 | 事件名 | 发布点 | 消费者 | MQ类型/契约 |
|------|--------|--------|--------|-------------|
| 占用判定改为仅活跃用户 | 无对应新事件 | 注册占用查询 / 超管列表过滤 | — | 例外：查询语义与标识回收，不改变对外用户领域事实；新用户仍走既有 `USER_CREATED`。证据豁免：`auth-phone-occupancy-excludes-archived-no-new-event` |
