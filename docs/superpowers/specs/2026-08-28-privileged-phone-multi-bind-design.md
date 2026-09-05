# 设计文档：特权角色一号多账号绑定（最多 5 个）

- **日期:** 2026-08-28
- **状态:** accepted（/goal 入口自动采用）
- **作者:** cursor
- **范围:** taskAuth 手机号占用策略 + 登录/重置消歧 + 资料页绑定错误码；taskFE 错误展示

## 目标

超级用户、员工、测试这三类角色允许**同一个手机号绑定多个账号**，每个号码最多 **5** 个活跃绑定。普通客户仍保持一号一账号。

## 成功标准

1. 绑定主体为超级用户 / 员工 / 测试，且该号现有活跃占用者**全部**为同类特权账号、占用数 `< 5` 时，资料页绑定成功（不 409、不 reclaim）。
2. 同一号码第 6 个绑定 → 409 `code=phone_bind_limit`，`reclaim_available=false`。
3. 普通客户绑定已被占用的号码 → 仍 409 `code=phone_taken`（含特权占用者）。
4. 特权账号绑定已被**普通客户**占用的号码 → 仍 409 `phone_taken` + reclaim（SIM 所有权转移，不作静默共享）。
5. 公开 `phone_register` 仍一号一户：号码已有任意活跃绑定时拒绝注册。
6. 手机号+密码登录：按密码匹配唯一账号；0 命中沿用现有失败文案；≥2 命中 → 400 `phone_ambiguous`。
7. 手机号验证码重置密码：活跃绑定 ≥2 时 → 400 `phone_ambiguous`，不发送/不消耗验证码去改随机账号。
8. 日志不含完整手机号。

## 当前基线（CodeGraph / 源码）

- 占用判定：`phoneLoginMethodTaken` → `findLoginMethodByPhone` `LIMIT 1`（`taskAuth/src/auth_profile_internal.go`）。
- 绑定入口：`handleBindPhone`（资料页 bind/replace + `/bind_phone/`）。
- 登录：`handleLogin` 手机路径同样 `LIMIT 1`。
- 重置：`findLoginMethodForPasswordReset` → 同一 LIMIT 1。
- 无 DB UNIQUE 索引；占用是应用层约束。
- 角色字段：`auth_user.is_superuser` / `is_staff` / `is_tester`；平台角色 `super_admin` ⇔ 超管、`employee` ⇔ 员工（`isPlatformAdminStaff`）。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 4 files / 0 新节点（本任务开始前工作区测试文件）。
- 爆炸半径（codegraph explore）：`handleBindPhone`、`upsertPhoneLoginMethod`、`phoneLoginMethodTaken`、`findLoginMethodByPhone`、`handleLogin`、`handleSendPasswordResetCode`、`handlePhoneTaken`、`handleUpsertPhoneLoginMethod`、前端 `isPhoneTakenBindError`。
- 无新服务/表/事件拓扑。

## 架构变更判定

**不更新 `docs/architecture/` 视图。** 无新 Application_Component、无新 DataObject、无新领域事件、无跨服务协议变更。属既有 `auth_login_method` 占用规则收窄/放宽。

No-ADR: trivial tech choice, no architectural impact（应用层策略常量，无新中间件/存储）。

## 方案（采用）

集中领域策略 `domain.EvaluatePhoneShareBind`：

| 绑定者 | 现有占用者 | 结果 |
|---|---|---|
| 普通客户 | 任意他人 | Taken（reclaim） |
| 特权 | 含普通客户 | Taken（reclaim） |
| 特权 | 全为特权且 count&lt;5（不含自己） | Allow |
| 特权 | 全为特权且 count≥5 | Limit |
| 任何人 | 仅自己 | Allow（换绑/幂等） |

特权定义（满足任一）：

- `is_superuser=1` 或平台角色 `super_admin`
- `is_staff=1` 或平台角色 `employee`
- `is_tester=1`

常量：`MaxSharedPhoneBindings = 5`。

`reclaim=true`：作废该号**全部**其他活跃绑定后绑定当前用户（修正现 LIMIT 1 在共享后漏作废）。

登录消歧：列出该号全部活跃 phone methods，对非空 `password_hash` 做 bcrypt；恰好 1 个匹配则登录该 `object_id`。

内部 `POST .../phone-taken/`：`taken=true` 表示「按 exclude_user 策略不能直接 upsert」，与 bind 一致。

## 拒绝的方案

1. **DB 部分唯一索引 / 触发器按角色分叉** — 角色可变，约束无法表达「全员特权」；拒绝。
2. **登录弹出账号选择器** — 暴露同号下其他 user_id；拒绝。用密码唯一匹配 + 邮箱/用户名兜底。
3. **公开注册也允许多绑** — 新注册账号默认非特权，会被客户用来撞号；拒绝。共享仅限已具备特权标志的账号走绑定。

## API 契约（向后兼容）

既有 409 `phone_taken` 不变。新增：

```json
{ "error": "该手机号已达绑定上限", "code": "phone_bind_limit", "reclaim_available": false, "limit": 5 }
```

```json
{ "error": "phone_ambiguous", "detail": "该手机号绑定了多个账号，请使用邮箱或用户名登录", "code": "phone_ambiguous" }
```

无新 endpoint。

## 业务意图 → 事件

绑定成功仍不发新领域事件（存量 `auth-profile-phone-bind-no-new-event`）。共享绑定只改变 `auth_login_method` 行数，身份事实仍由该表表达。登录成功仍走既有 `USER_LOGGED_IN`。

## Python API

无。全部落 taskAuth Go。

## 可观测性

- `phone_bound` 增加 `shared`、`holder_count`（int）、`reclaim`；禁止 phone 明文。
- `phone_bind_rejected`：`reason=taken|limit|ambiguous`。
- `login rejected`：`reason=phone_ambiguous`。
