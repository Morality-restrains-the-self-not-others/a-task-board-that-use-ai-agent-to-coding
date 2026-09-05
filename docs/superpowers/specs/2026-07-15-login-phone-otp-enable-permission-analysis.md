# 角色权限分析：登录页电话验证码登录可用

**日期**: 2026-07-15  
**设计**: `docs/superpowers/specs/2026-07-15-login-phone-otp-enable-design.md`  
**迭代**: `login-phone-otp-enable`

## 角色矩阵

| 角色 | 读 public policy | 改 system-feature-policy | 发验证码 | 手机 OTP 登录 | 说明 |
|------|------------------|--------------------------|----------|---------------|------|
| 匿名访客 | ✅ | ❌ | ✅（策略开时） | ✅（策略开时） | 登录页目标用户 |
| 已登录用户（add_account） | ✅ | ❌ | ✅ | ✅ | 添加账号槽 |
| 系统管理员 | ✅ | ✅ | ✅ | ✅ | 可再关闭手机登录 |
| 普通成员 | ✅ | ❌ | ✅ | ✅ | 同访客登录能力 |

## 变更点审计

| 变更 | 权限边界 | 风险 | 缓解 |
|------|----------|------|------|
| Data migration 打开 `enable_phone_login` | 部署时一次性；运行时仍可由管理员关闭 | 现网突然出现手机入口 | 管理员可关；SMS 失败有明确错误 |
| `next` 同站 path 回跳 | 仅相对 path | 开放重定向 | `PostLoginReturnUrl.normalize` |
| `persistLoginAccountSlot` | 本机 localStorage | XSS 放大面 | 既有上限 5；不新增跨站 cookie |
| 无新 endpoint | — | — | 复用既有鉴权 |

## 结论

无新权限角色；匿名可 OTP 登录仅在策略开启时。权限分析通过，可进入价值流。
