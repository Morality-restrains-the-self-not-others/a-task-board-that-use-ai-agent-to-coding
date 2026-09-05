// @ts-check
import { test, expect } from '@playwright/test'

test.describe('Login 手机号入口策略开关', () => {
  test('策略关闭时隐藏手机号登录入口', async ({ page }) => {
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            enable_phone_login: false,
          },
        }),
      })
    })

    await page.goto('/auth/login/')
    await page.waitForLoadState('networkidle')

    await expect(page.getByRole('button', { name: '邮箱/密码' })).toBeVisible()
    await expect(page.getByRole('button', { name: '手机号/密码' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '手机号/验证码' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '使用短信验证码登录' })).toHaveCount(0)
  })

  test('策略开启时手机号/密码可见、手机号/验证码已移除（2026-08-24）', async ({ page }) => {
    await page.route('**/api/public/system-feature-policy/', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            enable_phone_login: true,
          },
        }),
      })
    })

    await page.goto('/auth/login/')
    await page.waitForLoadState('networkidle')

    await expect(page.getByRole('button', { name: '手机号/密码' })).toBeVisible()
    // 验证码登录入口已整体移除：即使策略开启也不得出现
    await expect(page.getByRole('button', { name: '手机号/验证码' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: '使用短信验证码登录' })).toHaveCount(0)
  })
})
