// @ts-check
/**
 * OPT-20260721-012: 个人资料访问令牌「有效/已吊销 Tab + 分页」。
 *
 * 测试策略：
 * - Mock access-tokens API 返回 >20 个令牌（含有效和已吊销）
 * - 导航到 /profile/
 * - 断言默认 Tab「有效」只显示有效令牌
 * - 切换到「已吊销」Tab，断言分页出现
 *
 * 运行：
 *   npx playwright test tests/UserProfile.access-token-tabs.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { PW_TENANT_ID } from './playwrightTenantEnv.js'
import { playwrightSiteOrigin } from './playwrightApiEnv.js'

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID

/** 生成 mock access-tokens（count 个） */
function generateMockTokens(count, isRevoked) {
  const tokens = []
  for (let i = 1; i <= count; i++) {
    const pad = String(i).padStart(3, '0')
    tokens.push({
      id: `tok-${isRevoked ? 'rev' : 'act'}-${pad}`,
      name: `${isRevoked ? '已吊销' : '有效'}令牌 #${i}`,
      is_revoked: isRevoked,
      last_4: String((1000 + i) % 10000).padStart(4, '0').slice(-4),
      created_at: `2026-01-${String(i).padStart(2, '0')}T00:00:00Z`,
      expires_at: null,
      last_used_at: `2026-06-${String(i).padStart(2, '0')}T00:00:00Z`,
    })
  }
  return tokens
}

const ACTIVE_TOKENS = generateMockTokens(5, false)
const REVOKED_TOKENS = generateMockTokens(12, true)

test.describe('UserProfile 访问令牌 Tab + 分页', () => {
  test('默认展示有效令牌、切换到已吊销 Tab 后分页出现', async ({ page }) => {
    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ])

    // 拦截所有 API（UserProfile 会 GET /api/accounts/users/profile/ 等）
    await page.route('**/api/**', async (route) => {
      const req = route.request()
      const url = req.url()
      const method = req.method()

      // access-tokens API ── 返回混合令牌
      if (url.includes('/api/accounts/users/access-tokens/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            tokens: [...ACTIVE_TOKENS, ...REVOKED_TOKENS],
          }),
        })
        return
      }

      // UserProfile 自身信息
      if (url.includes('/api/accounts/users/profile/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'e2e-user',
            username: 'e2e-user',
            email: 'e2e@example.com',
            is_superuser: false,
            nickname: 'E2E User',
            avatar_url: null,
            has_phone: false,
            phone_masked: '',
            company_nicknames: {},
          }),
        })
        return
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
    })

    const origin = playwrightSiteOrigin()
    await page.goto(`${origin}/profile/access-tokens/`, { waitUntil: 'domcontentloaded' })

    // 等待 access-token 面板加载
    const tokenPanel = page.getByTestId('access-token-panel')
    await expect(tokenPanel).toBeVisible({ timeout: 30000 })

    // 等待令牌列表出现
    await expect(page.getByTestId('access-token-list')).toBeVisible({ timeout: 10000 })

    // ── 断言默认 Tab「有效」展示的 token 均为有效（无 .access-token-row-revoked） ──
    const activeRows = tokenPanel.locator('[data-testid="access-token-row-active"]')
    const revokedRows = tokenPanel.locator('[data-testid="access-token-row-revoked"]')
    await expect(activeRows).toHaveCount(5, { timeout: 5000 })
    await expect(revokedRows).toHaveCount(0)

    // Tab 计数
    const activeTab = tokenPanel.getByTestId('access-token-tab-active')
    await expect(activeTab).toContainText('(5)')
    const revokedTab = tokenPanel.getByTestId('access-token-tab-revoked')
    await expect(revokedTab).toContainText('(12)')

    // ── 切换到「已吊销」Tab ──
    await revokedTab.click()

    // 等待列表刷新 → 显示已吊销令牌
    await expect(revokedRows.first()).toBeVisible({ timeout: 10000 })
    // 每页 10 条（ACCESS_TOKEN_PAGE_SIZE = 10），12 个已吊销 → 第一页应显示 10 条
    await expect(revokedRows).toHaveCount(10, { timeout: 5000 })
    // 无有效行
    await expect(activeRows).toHaveCount(0)

    // ── 断言分页出现 ──
    const pagination = tokenPanel.getByTestId('access-token-pagination')
    await expect(pagination).toBeVisible({ timeout: 5000 })

    // 分页信息：共 12 条，第 1 / 2 页
    await expect(pagination).toContainText('共 12 条')
    await expect(pagination).toContainText('第 1 / 2 页')

    // ── 翻到第二页 ──
    const nextBtn = pagination.locator('button', { hasText: '下一页' })
    await nextBtn.click()
    await expect(revokedRows).toHaveCount(2, { timeout: 5000 })
  })
})
