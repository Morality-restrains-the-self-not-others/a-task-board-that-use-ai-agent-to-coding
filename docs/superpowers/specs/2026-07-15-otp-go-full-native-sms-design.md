# 设计：OTP/SMS 全切 Go（原生 SDK + 重置码 + 脱离 forward-login）

**日期**: 2026-07-15  
**状态**: 已批准（goal-mode 自动采用）  
**迭代**: `otp-go-full-native-sms`  
**作者**: claude  
**基于**: `2026-07-15-otp-go-migration-inc1-and-add-account-ac7-design.md`

## 目标与成功标准

| # | 标准 |
|---|------|
| S1 | taskAuth 原生调用阿里云/腾讯云 SendSms；发码路径**不再**经 Django `dispatch-sms` |
| S2 | 密码重置手机码：存码+发短信+验码全在 Go（模板 `password_reset`） |
| S3 | 充值 SMS：继续经 `VerificationCodeService`→Go 发/验；真实短信走原生 SDK（与 S1 同一通道） |
| S4 | `POST /api/auth/` phone+code：**不**调用 `forward-login`；Go 验码→查/建用户→`enrich-login`→发 Token |
| S5 | Go 单测覆盖 mock SMS、重置码、OTP 登录；OpenAPI 更新；意图文档落地 |
| S6 | 开 PR（taskAuth / task2app / docs） |

## 方案选型

### A — 原生 SMS

| 方案 | 结论 |
|------|------|
| 继续 Django 桥 | 否（本切目标） |
| Go HTTP 签名直调云 API（无重型 SDK 依赖） | **采用**（Aliyun POP + 腾讯 TC3） |
| 引入官方全量 SDK | 否（依赖膨胀；测试难 mock） |

配置沿用现有环境变量（`SMS_PROVIDER`、`SMS_ALIYUN_*`、`SMS_TENCENT_*`、`SMS_TEMPLATE_CODE_*`）。缺配置时**失败**（不静默成功）；`mock`/`none`/`disabled` 本地成功。

`sendVerificationSMS(ctx, phone, code, kind)`：`kind=verification|password_reset` 选模板。

### B — 密码重置手机码

`handleSendPasswordResetCode`：手机路径查 LoginMethod 存在 → `sendPhoneVerificationCode(..., password_reset)`；邮箱仍委托 Django。  
`handleResetPasswordWithCode`：手机路径本地 `verifyPhoneVerificationCode`；邮箱仍 Django。

### C — phone+code 登录脱离 forward-login

对齐密码登录：

1. `verifyPhoneVerificationCode`
2. `findLoginMethodByPhone`；无则 `createUserWithPhoneLogin`（空密码哈希，已验证）
3. `getOrCreateToken` + `djangoEnrichLogin`（同意条款 + session + USER_CREATED 自愈）
4. **禁止** `proxyToDjangoAuth` / `forward-login`

隐私/功能开关：仍由 `enrich-login` 侧校验条款；手机登录总开关若未在 Go 侧可读，则依赖前端/策略（与密码手机登录一致：Go 已允许 phone+password）。

### D — 充值

不迁公网路由（仍 billing_bridge）；OTP 真源已在 Go。本切仅保证 SMS 走原生通道。无新 Python 公网接口。

`python_api_approval: n/a`

## 业务意图 → 事件

| 意图 | 事件 | 证据 |
|------|------|------|
| 发短信验证码 | VerificationCodeSent | 豁免：结构化日志（沿用） |
| 验码成功 | VerificationCodeVerified | 豁免：结构化日志 |
| OTP 自动注册 | USER_CREATED | enrich-login `_ensure_user_has_company` / post-register 路径 |

## 架构

新增 target **v31-application-integration**：taskAuth → Aliyun/Tencent SMS；废弃 Rel：taskAuth→Django `dispatch-sms` / `forward-login`（OTP）。伴生 `.puml` + `.archimate` + `.mermaid.md`。

## 非目标

- 邮箱 OTP 迁出 Django
- 充值门禁缓存迁入 Go
- 删除 Django `dispatch-sms` / `forward-login` 端点本体（可留作兼容，Go 不再调用）
