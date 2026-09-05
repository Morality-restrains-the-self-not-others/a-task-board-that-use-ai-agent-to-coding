// @ts-check
/**
 * OPT-20260717-028: 核验所有入口页面包含正确的 <meta name="trae-service"> 标签。
 *
 * 测试策略：
 * - 导航到 task2app 主页面
 * - 断言 <meta name="trae-service" content="task2app"> 存在
 * - 若有配置且可访问，也校验 taskAiProvider 页面（port 8010）
 * - 此测试防止 HTML 替换中间件误删除 meta 标签
 *
 * SSOT: db/scripts/ci/frontend_head_trae_service.yaml
 * Rule: .ai/01_project_constraints/26_frontend_head_trae_service.md
 *
 * 运行：
 *   npx playwright test tests/EntryPage.trae-service-meta.playwright.test.js --config=playwright.verify.config.js
 */
import { test, expect } from '@playwright/test'

test.describe('入口页面 trae-service meta 标签', () => {
  test('task2app 主页面应包含 <meta name="trae-service" content="task2app">', async ({
    page,
    baseURL,
  }) => {
    const origin = baseURL || 'http://127.0.0.1:4000'

    // 导航到 task2app 首页
    await page.goto(origin, { waitUntil: 'domcontentloaded' })

    // 检查 meta[name="trae-service"] 存在且 content 为 task2app
    const meta = page.locator('meta[name="trae-service"]')
    await expect(meta).toHaveAttribute('content', 'task2app', { timeout: 10000 })
  })

  test('taskAiProvider 页面应包含 <meta name="trae-service" content="taskAiProvider">', async ({
    page,
  }) => {
    // taskAiProvider 默认在 8010 端口
    const aiProviderOrigin = process.env.PLAYWRIGHT_AI_PROVIDER_ORIGIN || 'http://127.0.0.1:8010'

    // 尝试导航到 AI 容器镜像市场页面
    await page.goto(aiProviderOrigin, { waitUntil: 'domcontentloaded', timeout: 15000 })

    const meta = page.locator('meta[name="trae-service"]')
    await expect(meta).toHaveAttribute('content', 'taskAiProvider', { timeout: 10000 })
  })
})
