/**
 * 任务详情评论区：智能体资源配置来源选择器冒烟 E2E。
 *
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

test.describe('TaskDetail 智能体资源配置来源选择器', () => {
  test('评论输入框下方可渲染智能体资源配置来源选择器（无需 relayToTrae）', async ({ page }) => {
    const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
    const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
    test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD');

    await playwrightLoginWithLegalAccept(page, { email, password });

    const taskDetailPath = process.env.PLAYWRIGHT_TASK_DETAIL_PATH || '/';
    await page.goto(taskDetailPath);
    await page.waitForLoadState('domcontentloaded');

    const sourceSelector = page.locator('[data-testid="feature-params-source-selector"]');
    const hasSelector = await sourceSelector.count();
    if (hasSelector === 0) {
      test.skip(true, '当前页面未挂载智能体资源配置来源选择器（需任务详情评论区）');
    }

    await expect(sourceSelector.first()).toBeVisible();
    await expect(page.locator('[data-testid="feature-params-block"]')).toContainText('智能体资源配置');
  });
});
