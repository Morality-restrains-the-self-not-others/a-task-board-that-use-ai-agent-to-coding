import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const vue = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'Register.vue'), 'utf8')

test('Register.vue captures query accessCode and submits access_code', () => {
  assert.match(vue, /captureReferralAccessCodeFromSearch\(window\.location\.search\)/)
  assert.match(vue, /submitData\.access_code = accessCode/)
  assert.match(vue, /readStoredReferralAccessCode\(/)
  assert.doesNotMatch(vue, /u\{userId\}/)
})

test('Register.vue 有 accessCode 时隐藏邀请码字段并提示无需填写', () => {
  assert.match(vue, /accessCodePresent\.value = Boolean\(captureReferralAccessCodeFromSearch\(window\.location\.search\)\)/)
  assert.match(vue, /data-testid="register-access-code-notice"/)
  assert.match(vue, /您已通过推荐邀请链接访问，无需填写邀请码/)
  // 邀请码字段仅在无 accessCode 时渲染
  assert.match(vue, /v-else/)
  // 提交校验：accessCode 存在时不再强制邀请码
  assert.match(vue, /invitePolicyEnabled\.value && !normalizedInvite && !accessCodePresent\.value/)
})

test('Register.vue 将 phone_register 通用业务错误展示在可见横幅（含 data-traceId）', () => {
  assert.match(vue, /mapRegisterApiError\(/)
  assert.match(vue, /data-testid="register-submit-error"/)
  assert.match(vue, /:data-traceId="submitErrorTraceId \|\| undefined"/)
  assert.doesNotMatch(vue, /errors\.value\.email = data\.error/)
})

test('Register.vue 注册成功 toast 仅在 response.ok 内（已注册 400 不会提示注册成功）', () => {
  const okIdx = vue.indexOf('if (response.ok)')
  const successToastIdx = vue.indexOf("toastService.success('注册成功，即将跳转到登录页面')")
  assert.ok(okIdx >= 0, 'missing response.ok branch')
  assert.ok(successToastIdx > okIdx, 'success toast must be inside response.ok')
})
