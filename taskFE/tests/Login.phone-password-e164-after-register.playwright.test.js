// @ts-check
/**
 * E2E: 手机号注册后（auth_login_method 存 national + phone_country_calling_code），
 * 立即用同一 E.164（+86 前缀）+ 密码登录，断言离开 /auth/login/。
 *
 * 回归 OPT-20260819-035：此前 handleLogin 对 E.164 精确匹配 identifier 失败
 * （identifier 存 national，无国码）→ 返回「手机号或密码错误」。Go 单测
 * TestHandleLoginPhonePasswordAcceptsE164AfterRegister 已覆盖 handler 层；
 * 本测经真实网关 POST /api/auth/（→ taskAuth）闭环服务端规范化，并验证登录页
 * 表单以完整 E.164 提交、登录成功后离开登录页。
 *
 * 依赖：
 *   - Vite dev（:4000，proxy /api → 网关 18081 / dev taskAuth）
 *   - 网关（:18081 → taskAuth），真实 /api/auth/ 可用
 *   - MySQL 容器 docker-mysql-mysql-1（幂等 seed 手机号登录方式，beforeAll）
 *
 * 运行：npx playwright test tests/Login.phone-password-e164-after-register.playwright.test.js \
 *       --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { spawnSync } from 'child_process'
import { readE2eOrigins } from './helpers/gatewayLoginE2e.js'

const MYSQL_CONTAINER = process.env.E2E_MYSQL_CONTAINER || 'docker-mysql-mysql-1'
const MYSQL_PASSWORD = process.env.E2E_MYSQL_PASSWORD || 'root123456'
const PHONE_PREFIX = '+86'
const PHONE_NATIONAL = process.env.E2E_E164_PHONE_NATIONAL || '13901234567'
const E164_PHONE = `${PHONE_PREFIX}${PHONE_NATIONAL}`
const PASSWORD = 'E164Reg!2345'
/** bcrypt(E164Reg!2345), cost 12; Go x/crypto/bcrypt 兼容 $2b$ 前缀 */
const BCRYPT_HASH = '$2b$12$SvxDEFIyWUIcpAQ0392dx.b5yjYQIiOeOZlayZF1Wl4qfIcvKUuZO'
const TEST_USER_ID = '8800000000000000001'
const TEST_LM_ID = '8800000000000000002'
const USER_CONTENT_TYPE_ID = '4'

const { gatewayOrigin } = readE2eOrigins()

/** spawnSync 不经 shell，bcrypt $ 字符原样传递 */
function mysql(sql) {
  const res = spawnSync(
    'docker',
    ['exec', '-i', MYSQL_CONTAINER, 'mysql', '-uroot', `-p${MYSQL_PASSWORD}`, '-N', '-B', '-e', sql],
    { encoding: 'utf8' },
  )
  if (res.status !== 0) {
    throw new Error(`mysql seed failed: ${res.stderr || res.stdout}`)
  }
  return res.stdout
}

function seedPhoneLoginMethod() {
  // 幂等：先删后插，模拟 createUserWithPhoneLogin 产出的 phone login_method
  //（identifier=national、phone_country_calling_code=+86、bcrypt 明文密码）
  mysql(`DELETE FROM task_auth.auth_login_method
    WHERE method_type='phone' AND phone_country_calling_code='${PHONE_PREFIX}' AND identifier='${PHONE_NATIONAL}';`)
  mysql(`DELETE FROM task_auth.auth_user WHERE id='${TEST_USER_ID}';`)
  const now = new Date().toISOString().slice(0, 19).replace('T', ' ')
  mysql(`
    INSERT INTO task_auth.auth_user (id, password, last_login, is_superuser, is_staff, is_active, is_tenant, date_joined)
    VALUES ('${TEST_USER_ID}', '', NULL, 0, 0, 1, 0, '${now}');
    INSERT INTO task_auth.auth_login_method (
      id, content_type_id, object_id, method_type, identifier,
      phone_country_calling_code, password_hash, is_verified, created_at, updated_at
    ) VALUES (
      '${TEST_LM_ID}', ${USER_CONTENT_TYPE_ID}, '${TEST_USER_ID}', 'phone', '${PHONE_NATIONAL}',
      '${PHONE_PREFIX}', '${BCRYPT_HASH}', 1, '${now}', '${now}'
    );
  `)
}

function cleanupPhoneLoginMethod() {
  try {
    mysql(`DELETE FROM task_auth.auth_login_method
      WHERE method_type='phone' AND phone_country_calling_code='${PHONE_PREFIX}' AND identifier='${PHONE_NATIONAL}';`)
    mysql(`DELETE FROM task_auth.auth_user WHERE id='${TEST_USER_ID}';`)
  } catch {
    // 清理失败不阻断断言
  }
}

test.beforeAll(() => {
  seedPhoneLoginMethod()
})

test.afterAll(() => {
  cleanupPhoneLoginMethod()
})

test.describe('手机号注册后 E.164 登录（OPT-20260819-035）', () => {
  test('真实网关接受 E.164 手机号+密码并返回 token（服务端规范化回归）', async ({ request }) => {
    const res = await request.post(`${gatewayOrigin}/api/auth/`, {
      headers: { 'Content-Type': 'application/json' },
      data: { phone: E164_PHONE, password: PASSWORD, rememberMe: true },
    })
    expect(res.status()).toBe(200)
    const body = await res.json()
    expect(body.token).toBeTruthy()
    expect(body.user?.id).toBe(TEST_USER_ID)
  })

  test('登录页手机号/密码表单提交完整 E.164 并离开 /auth/login/', async ({ page }) => {
    // 捕获提交到 /api/auth/ 的请求体，断言 phone 为完整 E.164
    let submittedPhone = null
    await page.route('**/api/auth/', async (route) => {
      const req = route.request()
      if (req.method() === 'POST') {
        submittedPhone = req.postDataJSON()?.phone ?? null
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            token: 'e2e-e164-token',
            user: { id: TEST_USER_ID, username: 'e164user' },
          }),
        })
        return
      }
      await route.continue()
    })

    // 启用手机号登录策略（保持区号回退 +86）
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { enable_phone_login: true } }),
      })
    })

    await page.goto('/auth/login/')
    await page.waitForLoadState('networkidle')

    // 切换「手机号/密码」标签
    const phonePwdTab = page.getByRole('button', { name: '手机号/密码' })
    await expect(phonePwdTab).toBeVisible({ timeout: 15000 })
    await phonePwdTab.click()

    // 国家码默认 +86；填入国内号 + 密码
    await page.locator('#login-phone-national-pwd').fill(PHONE_NATIONAL)
    await page.locator('#password').fill(PASSWORD)

    // 勾选隐私/服务协议（策略接口失败时前端跳过勾选，此处幂等处理）
    const acceptAll = page.getByTestId('login-accept-all')
    if (await acceptAll.isVisible().catch(() => false)) {
      await acceptAll.check()
    }

    // 提交登录
    const loginBtn = page.getByRole('button', { name: '登录', exact: true })
    if (await loginBtn.isDisabled().catch(() => false)) {
      // 部分环境 checkbox 需要二次点击行
      await page.getByText('我已阅读并同意全部条款', { exact: false }).first().click({ force: true }).catch(() => {})
    }
    await loginBtn.click()

    // 断言表单提交的是完整 E.164（+86 前缀 + 国内号）
    await expect.poll(() => submittedPhone).toBe(E164_PHONE)

    // 断言离开 /auth/login/
    await page.waitForURL((url) => !url.pathname.includes('/auth/login/'), { timeout: 15000 })
    expect(page.url()).not.toContain('/auth/login/')
  })
})
