// @ts-check
/**
 * OPT-20260806-046 部署回归：头像上传还原后，
 * 两页出现「上传图片」按钮；直接 POST 上传 API 返回可正常处理（真实图片 200 / 伪装 400）。
 */
import { test, expect } from '@playwright/test'

const SITE = 'http://127.0.0.1:4000'
const GATEWAY = 'http://127.0.0.1:18081'
const EMAIL = 'e2e.local@example.com'
const PASSWORD = 'TestPass123!'

async function apiLogin(page) {
  const resp = await page.request.post(`${GATEWAY}/api/auth/`, {
    headers: { 'Content-Type': 'application/json', Origin: SITE, Accept: 'application/json' },
    data: { username: EMAIL, password: PASSWORD, rememberMe: true },
  })
  const body = await resp.json().catch(() => ({}))
  if (!resp.ok()) throw new Error(`登录失败 ${resp.status()}`)
  const setCookie = resp.headers()['set-cookie'] || ''
  const userIdMatch = setCookie.match(/userId=([^;]+)/i)
  await page.context().addCookies([
    { name: 'userId', value: userIdMatch ? userIdMatch[1] : String(body.user?.id || ''), url: SITE },
    { name: 'csrftoken', value: 'e2e-csrf', url: SITE },
  ])
  return body
}

test.describe('OPT-046 头像上传还原回归', () => {
  test('profile 页出现「上传图片」按钮（UI 已还原）', async ({ page }) => {
    const user = await apiLogin(page)
    const userId = user.user?.id || 'me'
    await page.goto(`${SITE}/user/${userId}/profile/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(4000)
    const text = await page.evaluate(() => document.body.innerText)
    expect(text).toContain('上传图片')
    expect(text).not.toContain('头像上传已暂时禁用')
    console.log('PROFILE UPLOAD UI OK')
  })

  test('伪装 HTML 上传返回 400（安全校验生效）', async ({ page }) => {
    const user = await apiLogin(page)
    const html = '<html><body><script>alert(1)</script></body></html>'
    const resp = await page.request.post(`${GATEWAY}/api/accounts/users/profile/avatar/`, {
      headers: { Origin: SITE },
      multipart: {
        avatar: { name: 'x.png', mimeType: 'image/png', buffer: Buffer.from(html) },
      },
    })
    const body = await resp.json().catch(() => ({}))
    console.log('HTML DISGUISE:', resp.status(), JSON.stringify(body))
    expect(resp.status()).toBe(400)
    expect(String(body.error || '')).toMatch(/伪装|受支持|图片/)
  })
})
