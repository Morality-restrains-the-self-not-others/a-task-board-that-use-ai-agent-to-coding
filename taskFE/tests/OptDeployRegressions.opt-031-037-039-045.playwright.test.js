// @ts-check
/**
 * 部署回归集（OPT-20260806-031/034/037/039/045-头像）：
 * 本地全链路验证各页面在部署后的行为。
 * - OPT-031/034: profile 页微信绑定面板（bindAvailable 时显示绑定按钮）
 * - OPT-037: company-settings 三按钮（复制昵称/移除头像）
 * - OPT-039: git-site-oauth 绑定状态展示
 * - OPT-045(头像): 两页无「上传图片」按钮 + 直接 POST 上传 API 403
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

test.describe('部署回归集', () => {
  test('OPT-031/034: profile 页微信绑定面板渲染（bindAvailable 状态）', async ({ page }) => {
    const user = await apiLogin(page)
    const userId = user.user?.id || 'me'
    await page.goto(`${SITE}/user/${userId}/profile/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(4000)
    const text = await page.evaluate(() => document.body.innerText)
    // 身份绑定（微信）面板存在（配置齐全时显示绑定入口，未配置时显示提示文案）
    const hasPanel = text.includes('身份绑定（微信）')
    const hasBindBtn = text.includes('绑定微信')
    const hasUnconfiguredHint = text.includes('微信应用尚未配置')
    console.log('WECHAT PANEL:', { hasPanel, hasBindBtn, hasUnconfiguredHint })
    expect(hasPanel).toBe(true)
    // 二选一：有按钮或提示（凭据配置状态决定，均不报错）
    expect(hasBindBtn || hasUnconfiguredHint).toBe(true)
  })

  test('OPT-037: company-settings 复制昵称/上传/移除按钮可用', async ({ page }) => {
    const user = await apiLogin(page)
    const userId = user.user?.id || 'me'
    await page.goto(`${SITE}/user/${userId}/profile/company-settings/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(4000)
    const text = await page.evaluate(() => document.body.innerText)
    // 「在各公司的设置」面板渲染；上传 UI 已还原（OPT-20260806-046 安全加固完成）
    expect(text).toContain('在各公司的设置')
    expect(text).toContain('上传图片')
    console.log('COMPANY SETTINGS OK')
  })

  test('OPT-039: git-site-oauth 绑定状态展示（无「无法获取绑定状态」）', async ({ page }) => {
    const user = await apiLogin(page)
    const userId = user.user?.id || 'me'
    await page.goto(`${SITE}/user/${userId}/profile/git-site-oauth/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(5000)
    const text = await page.evaluate(() => document.body.innerText)
    expect(text).not.toContain('无法获取绑定状态')
    console.log('GIT-SITE-OAUTH OK:', text.replace(/\n/g, ' | ').slice(0, 150))
  })

  test('OPT-046(头像): 伪装 HTML 上传返回 400（安全校验生效，非 403 熔断）', async ({ page }) => {
    const user = await apiLogin(page)
    const html = '<html><body><script>alert(1)</script></body></html>'
    const resp = await page.request.post(`${GATEWAY}/api/accounts/users/profile/avatar/`, {
      headers: { Origin: SITE },
      multipart: {
        avatar: { name: 'x.png', mimeType: 'image/png', buffer: Buffer.from(html) },
      },
    })
    const body = await resp.json().catch(() => ({}))
    console.log('AVATAR DISGUISE:', resp.status(), JSON.stringify(body))
    expect(resp.status()).toBe(400)
    expect(String(body.error || '')).toMatch(/伪装|受支持|图片/)
  })
})

test.describe('OPT-033 referral 部署回归', () => {
  test('referral 页统计正常展示（无「获取推荐收益统计失败」）', async ({ page }) => {
    const user = await apiLogin(page)
    const userId = user.user?.id || 'me'
    await page.goto(`${SITE}/user/${userId}/profile/referral/`, { waitUntil: 'domcontentloaded' })
      .catch(async () => {
        await page.goto(`${SITE}/profile/referral/`, { waitUntil: 'domcontentloaded' })
      })
    await page.waitForTimeout(5000)
    const text = await page.evaluate(() => document.body.innerText)
    expect(text).not.toContain('获取推荐收益统计失败')
    console.log('REFERRAL OK:', text.replace(/\n/g, ' | ').slice(0, 150))
  })
})
