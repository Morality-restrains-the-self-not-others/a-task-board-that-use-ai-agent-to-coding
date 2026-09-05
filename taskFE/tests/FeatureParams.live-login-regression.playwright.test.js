// @ts-check
/**
 * OPT-20260806-045 部署回归（登录态）：feature-params 配置列表正常加载；
 * 人为 5xx 时错误元素带 data-traceId 且值等于请求 traceId。
 */
import { test, expect } from '@playwright/test'
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js'

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com'
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD || ''

test.describe('OPT-045 feature-params 部署回归', () => {
  test('登录后配置列表正常加载', async ({ page }) => {
    test.skip(!PASSWORD, 'no test password in env')
    await loginViaGatewayApi(page, { email: EMAIL, password: PASSWORD })

    const origin = process.env.SITE_BASE || 'http://127.0.0.1:4000'
    await page.goto(`${origin}/user/me/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      .catch(async () => {
        await page.goto(`${origin}/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      })
    await page.waitForTimeout(4000)

    const text = await page.evaluate(() => document.body.innerText)
    // 不应出现「加载个人配置失败」
    expect(text).not.toContain('加载个人配置失败')
    // 出现配置列表容器（新建配置 / 暂无配置 任一）
    const hasList = text.includes('新建配置') || text.includes('暂无个人配置')
    expect(hasList).toBe(true)
    console.log('PAGE OK, snippet:', text.replace(/\n/g, ' | ').slice(0, 200))
  })

  test('接口 500 时错误元素带 data-traceId', async ({ page }) => {
    test.skip(!PASSWORD, 'no test password in env')
    await loginViaGatewayApi(page, { email: EMAIL, password: PASSWORD })

    // 人为制造后端 500：拦截 feature-params 列表接口
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

    const origin = process.env.SITE_BASE || 'http://127.0.0.1:4000'
    await page.goto(`${origin}/user/me/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      .catch(async () => {
        await page.goto(`${origin}/profile/feature-params/`, { waitUntil: 'domcontentloaded' })
      })
    await page.waitForTimeout(4000)

    // 错误文案元素应带 data-traceId
    const el = page.locator('[data-traceId]').first()
    await expect(el).toBeVisible({ timeout: 10000 })
    const attr = await el.getAttribute('data-traceId')
    console.log('ERROR ELEMENT traceId:', attr, '| request traceId:', capturedTraceId)
    expect(attr && attr.length > 0).toBe(true)
    if (capturedTraceId) expect(attr).toBe(capturedTraceId)
  })
})
