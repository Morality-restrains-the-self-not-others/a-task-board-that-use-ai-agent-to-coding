# 实施计划：移除手机号验证码登录

- 日期：2026-08-24
- 关联设计：2026-08-24-remove-phone-code-login-design.md

## 垂直切片

### 切片 1 — 后端拒绝 + 删除 OTP（taskAuth）
- auth_login.go：handleLogin 加 phone+code 拒绝分支；删除 handlePhoneOTPLogin
- 测试（Red→Green）：
  - 新增 `TestHandleLoginRejectsPhoneCodeLogin`（400 + 明确错误；混合 {phone,code,password} 走密码）
  - 改造 `TestCustomerPhoneOTPRejectsAdminStaff` → `TestCustomerPhoneCodeLoginRejected`
- 验证：`go test ./src/ -run 'TestHandleLogin|TestCustomer|TestEntrySeparation'`
- 提交：`refactor(auth): 移除手机号验证码登录（handleLogin 拒绝 phone+code，删除 OTP 自动注册）`

### 切片 2 — 前端移除入口与死代码（taskFE）
- useLoginFeaturePolicy.js：删 PHONE_CODE_METHOD/switchToPhoneCode/相关特判
- LoginMethodSelector.vue：删「手机号/验证码」Tab
- Login.vue：删 phoneCode 分支与 useLoginVerificationCode 接线
- useLoginSubmit.js：删 phoneCode 分支
- AdminLogin.vue：删 verificationCode stub
- useLoginPhoneFields.js：删 phoneNationalCode 相关导出
- 删文件：useLoginVerificationCode(.js/.test.js)、LoginPhoneCodeFields(.vue/.test.js)
- 单测更新：useLoginFeaturePolicy.test.js、useLoginSubmit.test.js
- 验证：vitest 单测全绿
- 提交：`refactor(login): 移除手机号验证码登录入口与死代码`

### 切片 3 — E2E 全链路（taskFE/tests/）
- 新增 `PhoneAuthLifecycle.playwright.test.js`：
  1. 注册（send_verification_code → DB 读码 → phone_register）
  2. 验证码登录 400 拒绝
  3. UI 无「手机号/验证码」Tab
  4. 手机号密码登录（错误→400，正确→200）
  5. 登出
  6. 重置（send_password_reset_code → DB 读码 → reset_password_with_code）
  7. 新密码登录 → 200
  8. afterAll 清理（约束 44）
- 验证：`npx playwright test tests/PhoneAuthLifecycle.playwright.test.js --config=playwright.verify.config.js`
- 提交：`test(e2e): 手机号认证生命周期 E2E（注册→密码登录→验证码重置→新密码登录）`

### 切片 4 — 回归与收尾
- taskAuth 全量 `go test ./src/`
- taskFE vitest 全量 + 相关 playwright 回归（Login.phone-password-e164-after-register 等）
- runAll 精准编译重启登记（taskAuth + taskFE）
- 提交文档（docs/superpowers/specs）

## 部署
- taskAuth：`scripts/register-precise-restart.sh task-auth`（服务名以 runAll.yaml 为准）
- taskFE：`scripts/register-precise-restart.sh taskFE`
- 回滚：git revert 切片 1/2 提交 + 重启；E2E 反证（验证码登录恢复可用）
