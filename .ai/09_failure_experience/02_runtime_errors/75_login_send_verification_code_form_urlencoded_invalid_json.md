# [运行时] 登录页发送验证码失败：重复文案且无 data-traceId

## 基本信息

- 版本：1.0.0
- 案例编号：FE-20260722-0075
- 录入日期：2026-07-22
- 最后更新：2026-07-22
- 录入人：Cursor Agent

## 现象

- 页面：`/auth/login/?add_account=1` 手机验证码登录，「获取验证码」
- 可见文案：`发送验证码失败：发送验证码失败`
- DOM：`<p class="taskplugin-el-highlight">…</p>`，**无** `data-traceId`

## 环境与上下文

- 前端：`useLoginVerificationCode.js` → `POST /api/accounts/users/send_verification_code/`
- 网关：APISIX 直连 **taskAuth**（Django `UserViewSet.send_verification_code` 仅返回 501 占位）
- taskAuth：`handleSendVerificationCode` → `readJSONBody`，仅接受 JSON

## 根因

1. **请求体格式错误**：登录页用 `Content-Type: application/x-www-form-urlencoded` + `URLSearchParams`；taskAuth 解析 JSON 失败 → HTTP 400 `{"error":"invalid json"}`。
2. **重复文案**：失败分支 `responseData.message || '发送验证码失败'` 取不到 `message`（真实字段是 `error`/`detail`），再拼前缀 → `发送验证码失败：发送验证码失败`。
3. **无 data-traceId**：走 `modalService.alert(...)` 未传 `traceId` / 未用 `showRequestError(message, response)`；响应头实际有 `X-Trace-Id`。

对照：`Register.vue` / `UserProfilePhoneBindingPanel.vue` 已用 JSON，故注册/换绑路径正常。

## 修复

- `useLoginVerificationCode`：改为 `application/json` + `JSON.stringify({ phone })`
- 失败用 `showRequestError(verificationCodeSendErrorMessage(data), response)`（挂 `data-traceId`）
- 错误字段优先级：`detail` → `error` → `message` → `phone[0]`，不再盲目加「发送验证码失败：」前缀

## 验收

```bash
# 错误复现（旧行为）
curl -sS -X POST '.../send_verification_code/' \
  -H 'Content-Type: application/x-www-form-urlencoded' --data 'phone=%2B8613800138000'
# → {"error":"invalid json"}

# 正确
curl -sS -X POST '.../send_verification_code/' \
  -H 'Content-Type: application/json' --data '{"phone":"+8613800138000"}'
# → 200 {"message":"验证码发送成功",...}

npm --prefix taskFE/app test -- src/composables/auth/useLoginVerificationCode.test.js
```

## 关联

- 元规则：`.ai/01_project_constraints/24_frontend_error_data_trace_id.md`
- 同类：`52_people_groups_create_failed_invalid_data_trace_id.md`（须 `showRequestError` + 合法 trace）
- 代码：`taskFE/app/src/composables/auth/useLoginVerificationCode.js`、`taskAuth/src/auth_verification_code.go`
