// @ts-check
/**
 * OPT-20260806-045 生产回归（www.daydaymoney.com）：
 * 登录 → feature-params 配置列表正常加载；人为 500 时错误元素带 data-traceId。
 */
import { test, expect } from '@playwright/test'
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js'

const SITE = 'https://www.daydaymoney.com'
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com'
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD || ''

test.describe('OPT-045 生产回归 feature-params', () => {
  test('登录后配置列表正常加载（无「加载个人配置失败」）', async ({ page }) => {
    test.skip(!PASSWORD, 'no test password in env')
    await loginViaGatewayApi(page, {
      email: EMAIL,
      password: PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: SITE,
    })

    await page.goto(`${SITE}/user/me/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      .catch(async () => {
        await page.goto(`${SITE}/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      })
    await page.waitForTimeout(5000)

    const text = await page.evaluate(() => document.body.innerText)
    expect(text).not.toContain('加载个人配置失败')
    const hasList = text.includes('新建配置') || text.includes('暂无个人配置')
    expect(hasList).toBe(true)
    console.log('PROD OK:', text.replace(/\n/g, ' | ').slice(0, 180))
  })

  test('接口 500 时错误元素带 data-traceId 且等于请求 traceId', async ({ page }) => {
    test.skip(!PASSWORD, 'no test password in env')
    await loginViaGatewayApi(page, {
      email: EMAIL,
      password: PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: SITE,
    })

    let capturedTraceId = ''
    await page.route('**/api/personal/feature-params/**', async (route) => {
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

    await page.goto(`${SITE}/user/me/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      .catch(async () => {
        await page.goto(`${SITE}/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      })
    await page.waitForTimeout(5000)

    const el = page.locator('[data-traceId]').first()
    await expect(el).toBeVisible({ timeout: 10000 })
    const attr = await el.getAttribute('data-traceId')
    console.log('PROD ERROR traceId:', attr, '| request:', capturedTraceId)
    expect(attr && attr.length > 0).toBe(true)
    if (capturedTraceId) expect(attr).toBe(capturedTraceId)
  })
})
