// @ts-check
/**
 * OPT-20260806-045 本地全链路回归：API 登录 → feature-params 配置列表正常加载；
 * 人为 500 时错误元素带 data-traceId。
 */
import { test, expect } from '@playwright/test'

const SITE = 'http://127.0.0.1:4000'
const GATEWAY = 'http://127.0.0.1:18081'
const EMAIL = 'e2e.local@example.com'
const PASSWORD = 'TestPass123!'

/** API 登录（与 Login.vue 同契约：明文密码，后端 bcrypt 校验），写入 page cookie */
async function apiLogin(page) {
  const resp = await page.request.post(`${GATEWAY}/api/auth/`, {
    headers: { 'Content-Type': 'application/json', Origin: SITE, Accept: 'application/json' },
    data: { username: EMAIL, password: PASSWORD, rememberMe: true },
  })
  const body = await resp.json().catch(() => ({}))
  if (!resp.ok()) throw new Error(`登录失败 ${resp.status()}: ${JSON.stringify(body).slice(0, 200)}`)
  const setCookie = resp.headers()['set-cookie'] || ''
  const userIdMatch = setCookie.match(/userId=([^;]+)/i)
  await page.context().addCookies([
    { name: 'userId', value: userIdMatch ? userIdMatch[1] : String(body.user?.id || '') , url: SITE },
    { name: 'csrftoken', value: 'e2e-csrf', url: SITE },
  ])
  return body
}

test.describe('OPT-045 本地全链路 feature-params', () => {
  test('登录后配置列表正常加载（无「加载个人配置失败」）', async ({ page }) => {
    await apiLogin(page)
    await page.goto(`${SITE}/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(5000)

    const text = await page.evaluate(() => document.body.innerText)
    expect(text).not.toContain('加载个人配置失败')
    const hasList = text.includes('新建配置') || text.includes('暂无个人配置')
    expect(hasList).toBe(true)
    console.log('LOCAL OK:', text.replace(/\n/g, ' | ').slice(0, 180))
  })

  test('接口 500 时错误元素带 data-traceId 且等于请求 traceId', async ({ page }) => {
    await apiLogin(page)

    let capturedTraceId = ''
    await page.route('**/api/personal/feature-params-configs/**', async (route) => {
      const req = route.request()
      if (req.method() === 'GET' && !req.url().includes('/items/')) {
        capturedTraceId = req.headers()['x-trace-id'] || req.headers()['x-request-id'] || ''
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ detail: '人为故障', trace_id: capturedTraceId }),
        })
        return
      }
      await route.continue()
    })

    await page.goto(`${SITE}/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(5000)

    const el = page.locator('[data-traceId]').first()
    await expect(el).toBeVisible({ timeout: 10000 })
    const attr = await el.getAttribute('data-traceId')
    console.log('LOCAL ERROR traceId:', attr, '| request:', capturedTraceId)
    expect(attr && attr.length > 0).toBe(true)
    if (capturedTraceId) expect(attr).toBe(capturedTraceId)
  })
})
