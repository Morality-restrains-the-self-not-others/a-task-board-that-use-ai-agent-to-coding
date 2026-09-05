// @ts-check
/**
 * OPT-20260806-038: 「各公司的设置」面板（UserProfileCompanySettingsPanel）自动化测试。
 *
 * 测试策略（全部 mock，不依赖真实后端）：
 * - mock GET /api/accounts/users/profile/ 返回含 company_nicknames 的 profile
 * - 导航到 /profile/company-settings/
 * - 断言面板渲染：公司名 / 从个人昵称与头像复制按钮 / 昵称输入 / 头像占位
 * - 点「从个人昵称与头像复制」→ 断言发出正确 POST body（copy-personal-to-company）
 * - mock 404 时错误文案展示
 *
 * 运行（OPT-20260806-040 恢复的配置加载链路）：
 *   npx playwright test tests/UserProfileCompanySettings.copy-personal.playwright.test.js --config=playwright.config.headless.js
 */
import { test, expect } from '@playwright/test'
import { playwrightSiteOrigin } from './playwrightApiEnv.js'

const PROFILE_BODY = {
  user_id: 'e2e-user',
  personal_nickname: 'E2E 个人昵称',
  email: 'e2e@example.com',
  avatar_url: null,
  company_nicknames: [
    {
      company_id: 'c-100',
      company_name: '软刀科技',
      member_name: '软刀',
      member_avatar_url: '',
      is_admin: true,
      is_creator: true,
      workspace_id: 'w-1',
    },
  ],
}

test.describe('UserProfileCompanySettingsPanel (OPT-20260806-038)', () => {
  test('面板渲染：公司名 / 复制按钮 / 昵称输入 / 头像占位', async ({ page }) => {
    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      if (url.includes('/api/accounts/users/profile/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(PROFILE_BODY),
        })
        return
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    const origin = playwrightSiteOrigin()
    await page.goto(`${origin}/profile/company-settings/`, { waitUntil: 'domcontentloaded' })

    // 面板标题与公司名
    await expect(page.getByText('在各公司的设置').first()).toBeVisible({ timeout: 30000 })
    await expect(page.getByText('公司：软刀科技')).toBeVisible({ timeout: 10000 })

    // 「从个人昵称与头像复制」按钮存在
    const copyBtn = page.getByRole('button', { name: '从个人昵称与头像复制' })
    await expect(copyBtn).toBeVisible({ timeout: 5000 })

    // 昵称输入框回显 member_name
    const nicknameInput = page.getByPlaceholder('在本公司显示的名称')
    await expect(nicknameInput).toBeVisible({ timeout: 5000 })
    await expect(nicknameInput).toHaveValue('软刀')

    // 头像区：无头像时渲染默认首字母占位（img 存在；限定在面板内，排除侧边栏头像）
    const panel = page.locator('div', { hasText: '在各公司的设置' }).last()
    const avatarImg = panel.locator('img[alt=""]')
    await expect(avatarImg).toHaveCount(1, { timeout: 5000 })
    // 无 member_avatar_url → 不显示「移除头像」
    await expect(panel.getByRole('button', { name: '移除头像' })).toHaveCount(0)
  })

  test('点「从个人昵称与头像复制」发出正确 POST body', async ({ page }) => {
    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    let copyPostBody = null
    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      if (url.includes('/api/accounts/users/profile/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(PROFILE_BODY),
        })
        return
      }
      if (url.includes('/api/accounts/users/profile/copy-personal-to-company/') && method === 'POST') {
        copyPostBody = req.postData()
        await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
        return
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    const origin = playwrightSiteOrigin()
    await page.goto(`${origin}/profile/company-settings/`, { waitUntil: 'domcontentloaded' })

    const copyBtn = page.getByRole('button', { name: '从个人昵称与头像复制' })
    await expect(copyBtn).toBeVisible({ timeout: 30000 })
    await copyBtn.click()

    // POST body 应含 company_id 与 personal_nickname
    await expect.poll(() => copyPostBody, { timeout: 10000 }).not.toBeNull()
    const body = JSON.parse(copyPostBody)
    expect(body.company_id).toBe('c-100')
    expect(body.personal_nickname).toBe('E2E 个人昵称')
  })

  test('profile 接口 404 时展示错误文案', async ({ page }) => {
    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      if (url.includes('/api/accounts/users/profile/') && req.method() === 'GET') {
        await route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({ detail: 'profile not found' }),
        })
        return
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    const origin = playwrightSiteOrigin()
    await page.goto(`${origin}/profile/company-settings/`, { waitUntil: 'domcontentloaded' })

    // 错误文案展示（页面级 onChildError）
    await expect(page.locator('p.text-danger').first()).toBeVisible({ timeout: 30000 })
  })
})
