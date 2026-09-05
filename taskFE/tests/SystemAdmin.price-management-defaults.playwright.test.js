// @ts-check
/**
 * OPT-20260725-015: 核验价格管理页正确展示资源定价（替代旧价格套餐）。
 *
 * 测试策略：
 * - 使用 userId cookie 绕过登录
 * - 拦截 resource-pricing API 返回当前定价
 * - 验证资源定价面板数据
 *
 * 运行：
 *   npx playwright test tests/SystemAdmin.price-management-defaults.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'

test.describe('SystemAdmin 价格管理 — 资源定价', () => {
  test('价格管理页展示任务、GitLab 磁盘与流量费的当前定价', async ({ page, baseURL }) => {
    const origin = baseURL || 'http://127.0.0.1:4000'

    await page.context().addCookies([
      { name: 'userId', value: 'playwright-admin', url: origin },
      { name: 'csrftoken', value: 'playwright-csrf', url: origin },
    ])

    // 拦截 resource-pricing API
    await page.route('**/api/system-admin/resource-pricing/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            message: '资源定价直接管理，修改后即时生效。新订单按最新价格计算。',
            pricing: {
              task_post: { price_yuan: '0.55', price_cents: 55 },
              gitlab_disk: { price_yuan: '4.00', price_cents: 400, min_quantity: 10 },
              gitlab_traffic: { price_yuan: '1.00', price_cents: 100 },
            },
            updated_at: '2026-07-25T00:00:00Z',
          }),
        })
        return
      }

      await route.continue()
    })

    // 拦截 product-pricing API（旧端点，仅做兼容）
    await page.route('**/api/system-admin/product-pricing/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          pricing: {
            task_post: { price_yuan: '0.55', price_cents: 55 },
            gitlab_disk: { price_yuan: '4.00', price_cents: 400, min_quantity: 10 },
            gitlab_traffic: { price_yuan: '1.00', price_cents: 100 },
          },
          updated_at: '2026-07-25T00:00:00Z',
          policy_summary: '资源定价直接管理，修改后即时生效。新订单按最新价格计算。不再使用价格套餐。',
        }),
      })
    })

    await page.goto('/system-admin/price-management/', {
      waitUntil: 'networkidle',
    })

    // 等待资源定价卡片加载
    const pricingCard = page.getByTestId('admin-current-pricing-card')
    await expect(pricingCard).toBeVisible({ timeout: 15000 })

    // 验证显示当前定价（元，非积分）
    await expect(page.getByText(/0\.55.*元\/帖/)).toBeVisible()
    await expect(page.getByText(/4\.00.*元\/GB\/月/)).toBeVisible()
    await expect(page.getByTestId('admin-gitlab-disk-min-gb')).toContainText('起购 10 GB')
    await expect(page.getByText(/1\.00.*元\/GB/)).toBeVisible()
  })
})
