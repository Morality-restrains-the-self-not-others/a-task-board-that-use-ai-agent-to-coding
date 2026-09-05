# 设计文档：移除手机号验证码登录（保留注册/密码登录/验证码重置密码）

- 日期：2026-08-24
- 状态：已定稿（goal-mode 入口自动批准）
- 服务：taskAuth（后端）、taskFE（前端）

## 1. 目标与成功标准

**目标**：去掉「手机号+验证码登录」功能。用户以手机号+验证码+密码注册后，仅可通过**手机号+密码**登录；忘记密码时通过**手机号+验证码重置密码**。端到端测试确保全链路畅通。

**成功标准**：
1. 后端 `handleLogin` 拒绝 `{phone, code}` 登录（400 明确错误，fail-closed），不再自动注册
2. 前端登录页不再提供「手机号/验证码」Tab 与提交路径
3. 手机号+密码登录保持可用
4. 手机号+验证码+密码注册保持可用（`phone_register`）
5. 忘记密码→手机号+验证码重置密码→新密码登录 全链路可用
6. E2E 测试覆盖 注册→密码登录→登出→重置→新密码登录，含 UI 无验证码登录入口断言

## 2. 现状（代码基线，来自 codegraph/grep 探索）

### 后端 taskAuth
- 单一登录入口 `handleLogin`（src/auth_login.go:13）按 body 字段分发：
  - `{phone, code}` → `handlePhoneOTPLogin`（auth_login.go:282-339）：验证码校验 → `findLoginMethodByPhone` 查账号，**查不到自动注册**（`createUserWithPhoneLogin(cc, nat, "", "")`）→ 入口分离校验
  - `{phone, password}` → 密码登录（region 白名单 + `findLoginMethodByCanonicalPhone`）
  - `{email|username, password}` → 密码登录
- `handlePhoneOTPLogin` 仅被 auth_login.go:26 调用
- 保留项（不动）：
  - 注册 `POST /api/accounts/users/phone_register/`（body `{phone, password, code}`，src/auth_phone_register.go）
  - 发码 `POST /api/accounts/users/send_verification_code/`（src/auth_verification_code.go，登录/注册/绑手机共用）
  - 重置发码 `POST /api/accounts/users/send_password_reset_code/`（src/auth_password_reset.go:130，`smsKindPasswordReset`）
  - 重置提交 `POST /api/accounts/users/reset_password_with_code/`（body `{phone, code, new_password}`，auth_password_reset.go:181）
- 验证码存 MySQL `auth_sms_verification_code`（TTL 5min），E2E 可从 DB 读码
- SMS provider 默认 mock（env `SMS_PROVIDER`），生产 conf 为 aliyun

### 前端 taskFE
- `Login.vue` 四种登录方式：emailPassword / phonePassword / phoneCode / accessToken
- `useLoginFeaturePolicy.js`：`PHONE_CODE_METHOD='phoneCode'`、`isPhoneLoginMethod` 含 phoneCode、`switchToPhoneCode`、`handleMethodChanged` phoneCode 特判
- `LoginMethodSelector.vue`：「手机号/验证码」Tab（`v-if="allowPhoneLogin"`）
- `useLoginSubmit.js`：phoneCode 分支（L171-181, L218-227），与 phonePassword 同 POST `/api/auth/`
- `useLoginVerificationCode.js`：仅 Login.vue 使用（+ 自身测试）
- `LoginPhoneCodeFields.vue`：仅 Login.vue 使用
- `useLoginPhoneFields.js`：phoneNationalCode/loginPhoneForApi/isPhoneValidForCode/onLoginPhoneNationalCodeInput/Paste 仅登录页验证码方式使用
- `AdminLogin.vue`：为 useLoginSubmit 提供 verificationCode stub（isPhoneCodeReadyForSubmit/codeForLogin）
- 测试：useLoginFeaturePolicy.test.js（phoneCode 用例）、useLoginVerificationCode.test.js、LoginPhoneCodeFields.test.js、useLoginSubmit.test.js

### 重置密码现状
- `ResetPasswordRequest.vue`：手机表单发码 `send_password_reset_code` → 前端 6 位码校验 → 跳 `/reset-password/?phone=..&code=..`
- `ResetPassword.vue`：phone+code 路径 POST `reset_password_with_code` body `{phone, code, new_password}` → 成功跳登录页
- **手机号验证码重置已可用**（后端 Go 单测 TestPasswordResetPhoneCodeLocal 覆盖）——本任务保留

## 3. 方案（决策）

### 3.1 后端（taskAuth）
1. `auth_login.go` handleLogin：`{phone, code}` 分支改为**显式拒绝**：
   ```go
   if strField(body, "phone") != "" && strField(body, "code") != "" && strField(body, "password") == "" {
       writeErrorDetail(w, r, http.StatusBadRequest, "验证码登录已关闭，请使用手机号+密码登录")
       return
   }
   ```
   fail-closed：旧客户端/绕过前端直接调 API 也无法使用验证码登录。
2. 删除 `handlePhoneOTPLogin`（死代码，规则 53 删除旧逻辑无需征求批准；连带其自动注册路径消失）。
3. 单测：
   - 新增 `TestHandleLoginRejectsPhoneCodeLogin`：`{phone, code}` → 400 + 明确错误；`{phone, code, password}` 也拒绝（password 存在时走密码路径——见下）
   - 改造 `TestCustomerPhoneOTPRejectsAdminStaff`（auth_entry_separation_test.go:179）→ `TestCustomerPhoneCodeLoginRejected`
   - 检查依赖 OTP 登录的其他测试（grep 确认无）

> 边界：`{phone, code, password}` 三字段都传？现有分发逻辑 `phone+code` 优先（OTP）。移除后应让 password 路径接管（phone+password 正常登录），即拒绝条件仅当 code 存在且 password 为空。这样混合请求按密码登录处理，语义合理。

### 3.2 前端（taskFE）
1. `useLoginFeaturePolicy.js`：删除 `PHONE_CODE_METHOD` 常量、`switchToPhoneCode`、`isPhoneLoginMethod` 中 phoneCode、`handleMethodChanged` 中 phoneCode 特判
2. `LoginMethodSelector.vue`：删除「手机号/验证码」Tab
3. `Login.vue`：删除 phoneCode 分支渲染、`useLoginVerificationCode` 接线、handleSendCode/onVerificationCodeInput/phoneNationalCode 相关、移动端按钮 phoneCode 特判
4. `useLoginSubmit.js`：删除 phoneCode 分支
5. `AdminLogin.vue`：删除 verificationCode stub
6. `useLoginPhoneFields.js`：删除 phoneNationalCode 相关导出（loginPhoneForApi/isPhoneValidForCode/onLoginPhoneNationalCodeInput/Paste/phoneNationalCode）
7. 删除死代码文件：`useLoginVerificationCode.js` + `.test.js`、`LoginPhoneCodeFields.vue` + `.test.js`
8. 单测更新：useLoginFeaturePolicy.test.js（phoneCode 用例）、useLoginSubmit.test.js

### 3.3 E2E（新增 taskFE/tests/PhoneAuthLifecycle.playwright.test.js）
唯一手机号（时间戳派生 `139` + 8 位）：
1. **注册**（API）：POST send_verification_code → 读 DB 码 → POST phone_register → 201 token
2. **验证码登录已移除**（API）：POST /api/auth/ {phone, code} → 400 + detail
3. **UI 无验证码 Tab**（UI）：/auth/login/ 断言「手机号/验证码」不可见、「手机号/密码」可见
4. **手机号密码登录**（API）：错误密码 400 → 正确 200 token
5. **登出**（API）：POST /api/auth/logout/ → cookie 清除
6. **忘记密码**（API）：send_password_reset_code → 读 DB 码 → reset_password_with_code {phone, code, new_password} → 200
7. **新密码登录**（API）：POST /api/auth/ {phone, new_password} → 200 token
8. **清理**（afterAll）：删除 auth_user/auth_login_method/auth_sms_verification_code 行（约束 44）

DB 读码：`docker exec mysql`（模式同 Login.phone-password-e164-after-register.playwright.test.js）。
SMS provider 自适应：真实调用 send 接口成功则读码；若失败（生产 aliyun 未配）则直接 INSERT 验证码行（校验逻辑只查表，绕过发送）。

## 4. 非目标（不做）
- 不删手机号密码登录、注册、重置密码
- 不改 feature policy 结构（`enable_phone_login` 语义不变：控制手机号相关登录入口显示）
- 不动微信登录、accessToken 登录、admin-login
- 不迁移既有验证码数据

## 5. 影响面
- 后端：auth_login.go、auth_entry_separation_test.go、auth_login_test.go（新增）
- 前端：Login.vue、LoginMethodSelector.vue、useLoginFeaturePolicy(.js/.test.js)、useLoginSubmit(.js/.test.js)、useLoginPhoneFields.js、AdminLogin.vue、删除 useLoginVerificationCode(.js/.test.js)、LoginPhoneCodeFields(.vue/.test.js)
- 部署：taskAuth + taskFE 需精准编译重启（runAll）
- 风险：微信「手机号验证码自动开通」入口随之消失（OTP 自动注册）——注册需显式走 phone_register；这是预期行为（用户要求去掉验证码登录）
