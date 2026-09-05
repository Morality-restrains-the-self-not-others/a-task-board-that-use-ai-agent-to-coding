// @ts-check
/**
 * OPT-20260721-007: 公开致谢页 Playwright 冒烟测试。
 *
 * 断言 ack-entry-* 与 ack-repo-link-* 可见且链接指向正确地址。
 * 无需登录。
 *
 * 运行：
 *   npx playwright test tests/Acknowledgments.smoke.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'
import { playwrightSiteOrigin } from './playwrightApiEnv.js'

test.describe('Acknowledgments 公开致谢页', () => {
  test('致谢条目可见且链接正确', async ({ page }) => {
    const origin = playwrightSiteOrigin()
    await page.goto(`${origin}/acknowledgments/`, { waitUntil: 'domcontentloaded' })

    // 等待页面加载
    await expect(page.getByTestId('view-acknowledgments-page')).toBeVisible({ timeout: 15000 })

    // 断言 trae-agent 条目可见
    const entry = page.getByTestId('ack-entry-trae-agent')
    await expect(entry).toBeVisible({ timeout: 10000 })

    // 断言条目名称可见
    await expect(entry.locator('h2')).toContainText('Trae Agent')

    // 断言 repo link 可见且链接指向正确仓库
    const repoLink = page.getByTestId('ack-repo-link-trae-agent')
    await expect(repoLink).toBeVisible({ timeout: 5000 })
    await expect(repoLink).toHaveAttribute('href', 'https://github.com/bytedance/trae-agent')

    // 断言返回首页链接可见
    const backLink = page.getByTestId('ack-back-home-link')
    await expect(backLink).toBeVisible({ timeout: 5000 })
  })
})
