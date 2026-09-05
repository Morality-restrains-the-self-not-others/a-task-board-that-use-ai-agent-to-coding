# 设计：OTP/SMS 迁 taskAuth（首切）+ Navbar add_account AC7

**日期**: 2026-07-15  
**状态**: 已批准（goal-mode 自动采用）  
**迭代**: `otp-go-migration-inc1-and-add-account-ac7`  
**作者**: claude

## 目标与成功标准

| # | 标准 |
|---|------|
| S1 | 租户页 Navbar「添加账号」不把 `/tenant/...` 写入 `next` |
| S2 | Vitest 覆盖 `resolveAddAccountNext` |
| S3 | `accounts_sms_verification_code` owner → `task-auth`；auth.db 建表 |
| S4 | `POST /api/accounts/users/send_verification_code/` 在 taskAuth 本地发码（mock/aliyun） |
| S5 | Django 手机验码改为调 taskAuth internal verify；邮箱码仍可暂留 Django |
| S6 | 登录 OTP 路径继续 forward-login，但验码读 Go 表（经 Django→Go） |
| S7 | Go 测试 + Django 适配测试通过；开 PR |

## 方案选型

### A — AC7

采用 `resolveAddAccountNext(currentPath)`：含 `/tenant/` → `/`（省略业务 next）；否则去 `workspace_id` 后保留。Login 侧：`add_account=1` 且 next 仍含 `/tenant/` 时用新 userId 经 `resolveSwitchHref` 二次纠偏。

### B — OTP 首切

| 方案 | 结论 |
|------|------|
| 只发码进 Go、表仍 Django | 否（违反单表所有权） |
| 表+发+验进 Go；登录仍 forward-login | **采用** |
| 一次做完登录纯 Go + 充值/重置 | 否（范围过大） |

SMS：Go 侧 `mock`/`none`/`disabled` 成功；`aliyun` 走 dysmsapi（配置不全则失败，不静默成功除非 mock）。腾讯云首切可后置（未配置时返回明确错误）。

## 🐍 Python 新接口

无新公网 Python 接口。Django 改为 **调用** taskAuth internal verify/send（收薄本地写码）。

`python_api_approval: n/a`

## 业务意图 → 事件

| 意图 | 事件 | 例外 |
|------|------|------|
| 发短信验证码 | VerificationCodeSent（可选） | 首切可证据豁免：日志审计 |
| 验码成功 | VerificationCodeVerified | 同上 |
| add_account AC7 | — | 纯前端 |

## 架构

需更新 table ownership；application-integration 可轻量标注 VerificationCode 迁 taskAuth（本迭代写设计 + ownership；若改 Rel_Access 则补 target 架构）。首切以 ownership YAML + 设计文档为主；若时间允许补 vN application-integration。

## 非目标（后置）

- 充值 SMS / 密码重置码迁 Go
- phone+code 登录完全脱离 Django forward-login
- 腾讯云 SMS Go SDK
