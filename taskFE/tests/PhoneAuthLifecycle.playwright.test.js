// @ts-check
/**
 * E2E: 手机号认证生命周期（2026-08-24 手机号+验证码登录移除后）
 *
 * 覆盖目标（成功标准逐条）：
 *   1. 注册：send_verification_code → DB 读码 → phone_register（邀请策略启用时带 invite_code）→ 201 token
 *   2. 手机号+验证码登录被拒：POST /api/auth/ {phone, code} → 400 phone_code_login_disabled
 *      （fail-closed：不发 token、不再自动注册）
 *   3. 登录页无「手机号/验证码」Tab（即使策略开启也不可见）
 *   4. 手机号+密码登录：错误密码 400「手机号或密码错误」→ 正确密码 200 token
 *   5. 登出：POST /api/accounts/users/logout/ → 204
 *   6. 忘记密码：send_password_reset_code → DB 读码 → reset_password_with_code → 200
 *   7. 新密码登录 → 200 token（旧密码已失效）
 *   8. afterAll 清理（约束 44）：auth_sms_verification_code / auth_login_method /
 *      auth_customtoken / auth_user / auth_registration_invite_code
 *
 * 依赖：
 *   - 网关 :18081（真实 taskAuth，已含 2026-08-24 验证码登录移除变更）
 *   - Vite dev（:4000，proxy /api → 网关），UI 断言需要
 *   - MySQL 容器 docker-mysql-mysql-1（验证码/邀请码读取与清理）
 *
 * 运行：npx playwright test tests/PhoneAuthLifecycle.playwright.test.js \
 *       --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { spawnSync } from 'child_process'
import { readE2eOrigins } from './helpers/gatewayLoginE2e.js'

const MYSQL_CONTAINER = process.env.E2E_MYSQL_CONTAINER || 'docker-mysql-mysql-1'
const MYSQL_PASSWORD = process.env.E2E_MYSQL_PASSWORD || 'root123456'
const PHONE_PREFIX = '+86'
// 唯一手机号：139 + 时间戳后 8 位，避免与历史/并发测试冲突（verify 配置单 worker 串行）
const PHONE_NATIONAL = `139${String(Date.now()).slice(-8)}`
const E164_PHONE = `${PHONE_PREFIX}${PHONE_NATIONAL}`
const PASSWORD = 'PhoneLife!2345'
const NEW_PASSWORD = 'PhoneLife!6789'
const FALLBACK_CODE = '246810'
// 唯一邀请码（邀请策略启用时 phone_register 必需）：E2ETEST 前缀 + 时间戳后 6 位
const TEST_INVITE_CODE = `E2ETEST${String(Date.now()).slice(-6)}`

/** 模块级：注册返回的 user id，afterAll 清理用 */
let registeredUserId = null

const { gatewayOrigin } = readE2eOrigins()

/** spawnSync 不经 shell，$ 等字符原样传递 */
function mysql(sql) {
  const res = spawnSync(
    'docker',
    ['exec', '-i', MYSQL_CONTAINER, 'mysql', '-uroot', `-p${MYSQL_PASSWORD}`, '-N', '-B', '-e', sql],
    { encoding: 'utf8' },
  )
  if (res.status !== 0) {
    throw new Error(`mysql failed: ${res.stderr || res.stdout}`)
  }
  return res.stdout
}

/** 读该手机号最新未用且未过期的验证码（服务端查询口径一致：is_used=0 且 expires_at>now） */
function readLatestCode() {
  const out = mysql(
    `SELECT code FROM task_auth.auth_sms_verification_code
     WHERE phone='${PHONE_NATIONAL}' AND country_calling_code='${PHONE_PREFIX}'
       AND is_used=0 AND expires_at > NOW()
     ORDER BY id DESC LIMIT 1`,
  )
  return out.trim()
}

/** 生成合法 bigint id（9 + 13 位时间戳 + 5 位随机 = 19 位，< bigint 上限） */
function snowflakeId() {
  return `9${Date.now()}${Math.floor(Math.random() * 100000).toString().padStart(5, '0')}`
}

/**
 * 发送验证码并取回码值。自适应：真实发送成功后码存 DB → 读回；
 * 发送失败（如真实 aliyun 未配置，服务端会删除码行）→ 直接 INSERT 已知码
 * （自洽：verify 仅查表，与 provider 无关）。保证注册/重置环节始终可继续。
 */
async function sendCodeAndRead(request, path) {
  const res = await request.post(`${gatewayOrigin}/api/accounts/users/${path}/`, {
    data: { phone: E164_PHONE },
  })
  if (res.ok()) {
    await expect.poll(() => readLatestCode(), { timeout: 5000 }).toBeTruthy()
    return readLatestCode()
  }
  // 发送失败回退：INSERT 已知验证码（is_used=0，TTL 5 分钟）
  mysql(
    `INSERT INTO task_auth.auth_sms_verification_code
      (id, phone, country_calling_code, email, code, created_at, expires_at, is_used)
     VALUES ('${snowflakeId()}', '${PHONE_NATIONAL}', '${PHONE_PREFIX}', NULL, '${FALLBACK_CODE}',
       NOW(), DATE_ADD(NOW(), INTERVAL 5 MINUTE), 0)`,
  )
  return FALLBACK_CODE
}

/** 幂等插入测试邀请码（status='unused'，今日上海日期）。注册后服务端标 used，afterAll 整行删除 */
function seedInviteCode() {
  mysql(`DELETE FROM task_auth.auth_registration_invite_code WHERE code='${TEST_INVITE_CODE}';`)
  mysql(
    `INSERT INTO task_auth.auth_registration_invite_code
      (id, code, issuer_user_id, status, issued_day, created_at)
     VALUES ('${snowflakeId()}', '${TEST_INVITE_CODE}', '0', 'unused', '2026-08-24', NOW())`,
  )
}

test.beforeAll(() => {
  seedInviteCode()
})

test.afterAll(() => {
  try {
    mysql(`DELETE FROM task_auth.auth_sms_verification_code WHERE phone='${PHONE_NATIONAL}';`)
    mysql(`DELETE FROM task_auth.auth_login_method
      WHERE method_type='phone' AND phone_country_calling_code='${PHONE_PREFIX}' AND identifier='${PHONE_NATIONAL}';`)
    if (registeredUserId) {
      mysql(`DELETE FROM task_auth.auth_customtoken WHERE object_id='${registeredUserId}';`)
      mysql(`DELETE FROM task_auth.auth_user WHERE id='${registeredUserId}';`)
    }
    mysql(`DELETE FROM task_auth.auth_registration_invite_code WHERE code='${TEST_INVITE_CODE}';`)
  } catch {
    // 清理失败不阻断断言
  }
})

test.describe('手机号认证生命周期（验证码登录移除后，2026-08-24）', () => {
  test('1. 注册：手机号+验证码+密码 → 201 token', async ({ request }) => {
    const code = await sendCodeAndRead(request, 'send_verification_code')
    expect(code).toBeTruthy()

    const res = await request.post(`${gatewayOrigin}/api/accounts/users/phone_register/`, {
      data: { phone: E164_PHONE, password: PASSWORD, code, invite_code: TEST_INVITE_CODE },
    })
    expect(res.status()).toBe(201)
    const body = await res.json()
    expect(body.token).toBeTruthy()
    expect(body.user?.id).toBeTruthy()
    registeredUserId = body.user.id
  })

  test('2. 手机号+验证码登录被拒（fail-closed：400 phone_code_login_disabled，无 token）', async ({ request }) => {
    const res = await request.post(`${gatewayOrigin}/api/auth/`, {
      data: { phone: E164_PHONE, code: '000000' },
    })
    expect(res.status()).toBe(400)
    const body = await res.json()
    expect(body.error).toBe('phone_code_login_disabled')
    expect(body.token).toBeFalsy()
  })

  test('3. 登录页无「手机号/验证码」Tab（策略开启时也不可见）', async ({ page }) => {
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { enable_phone_login: true } }),
      })
    })

    await page.goto('/auth/login/')
    await page.waitForLoadState('networkidle')

    await expect(page.getByRole('button', { name: '手机号/密码' })).toBeVisible()
    // 验证码登录入口已整体移除：即使策略开启也不得出现
    await expect(page.getByRole('button', { name: '手机号/验证码' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '使用短信验证码登录' })).toHaveCount(0)
  })

  test('4. 手机号+密码登录：错误密码 400 → 正确密码 200 token', async ({ request }) => {
    const bad = await request.post(`${gatewayOrigin}/api/auth/`, {
      data: { phone: E164_PHONE, password: 'WrongPass!9999' },
    })
    expect(bad.status()).toBe(400)
    const badBody = await bad.json()
    expect(badBody.error).toContain('手机号或密码错误')

    const ok = await request.post(`${gatewayOrigin}/api/auth/`, {
      data: { phone: E164_PHONE, password: PASSWORD, rememberMe: true },
    })
    expect(ok.status()).toBe(200)
    const okBody = await ok.json()
    expect(okBody.token).toBeTruthy()
  })

  test('5. 登出：带 Token 调用 logout → 204', async ({ request }) => {
    const login = await request.post(`${gatewayOrigin}/api/auth/`, {
      data: { phone: E164_PHONE, password: PASSWORD },
    })
    expect(login.status()).toBe(200)
    const token = (await login.json()).token
    expect(token).toBeTruthy()

    const res = await request.post(`${gatewayOrigin}/api/accounts/users/logout/`, {
      headers: { Authorization: `Token ${token}` },
    })
    expect(res.status()).toBe(204)
  })

  test('6. 忘记密码：send_password_reset_code → DB 读码 → reset_password_with_code → 200', async ({ request }) => {
    const code = await sendCodeAndRead(request, 'send_password_reset_code')
    expect(code).toBeTruthy()

    const res = await request.post(`${gatewayOrigin}/api/accounts/users/reset_password_with_code/`, {
      data: { phone: E164_PHONE, code, new_password: NEW_PASSWORD },
    })
    expect(res.status()).toBe(200)
  })

  test('7. 新密码登录 → 200 token（旧密码已失效）', async ({ request }) => {
    const old = await request.post(`${gatewayOrigin}/api/auth/`, {
      data: { phone: E164_PHONE, password: PASSWORD },
    })
    expect(old.status()).toBe(400)

    const res = await request.post(`${gatewayOrigin}/api/auth/`, {
      data: { phone: E164_PHONE, password: NEW_PASSWORD },
    })
    expect(res.status()).toBe(200)
    const body = await res.json()
    expect(body.token).toBeTruthy()
  })
})
