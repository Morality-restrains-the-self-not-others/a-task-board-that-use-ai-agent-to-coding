# 角色权限分析：特权角色一号多账号绑定

- **日期:** 2026-08-28
- **设计:** `docs/superpowers/specs/2026-08-28-privileged-phone-multi-bind-design.md`

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST bind-phone / replace-phone / bind_phone | 已登录用户 | Resource（本人 login_method） | write | Token `resolveTokenUserIDFromRequest` + 短信验证码 | ✅ | 共享资格看**目标账号自身**角色，不新增 endpoint 权限 |
| PATCH internal phone-login-method | 内部服务 | System | write | `X-TaskAuth-Internal-Secret` | ✅ | upsert 走同一策略，防内部绕过 5 上限 |
| POST internal phone-taken | 内部服务 | System | read | Internal secret | ✅ | taken 语义与策略对齐 |
| POST login（手机+密码） | 匿名 | Resource | auth | 密码哈希；入口分离 | ✅ | 消歧不泄露其他 user_id |
| POST send_password_reset_code（手机） | 匿名 | Resource | write | 既有 | ⚠️ 共享号 LIMIT 1 会改错账号 | 必须 ambiguous 拒绝 |
| POST phone_register | 匿名 | System | write | 验证码+占用 | ✅ | 保持拒绝已占用，不把注册变成共享入口 |
| 解档 identifier conflict | 系统管理员 | System | write | superuser 管理端 | ✅ | 全员特权且 ≤5 的手机共享不视为冲突 |

## 角色定义（沿用，不新增角色）

```
role: super_admin / is_superuser     display: 超级用户
role: employee / is_staff            display: 员工
role: is_tester                      display: 测试
```

共享资格是**账号属性**，不是新 RBAC 权限位。禁止把「可共享手机」写入 `X-User-Roles`。

## 威胁与缓解

| 威胁 | 缓解 |
|------|------|
| 普通客户把号码挂到员工账号上蹭登录 | 客户绑定特权占用号仍 Taken；登录需该账号自己的 password_hash |
| 员工把客户号码静默共享 | 占用含普通客户 → Taken，须 SMS reclaim（作废客户绑定） |
| 手机重置改掉共享集合里随机一人 | ≥2 绑定时拒绝重置 |
| 枚举同号账号列表 | 不返回 user_id 列表；ambiguous 统一文案 |
| 内部 upsert 绕过上限 | 同一 Evaluate 函数 |

## 新角色/权限建模

不引入新角色。不新增 endpoint。
