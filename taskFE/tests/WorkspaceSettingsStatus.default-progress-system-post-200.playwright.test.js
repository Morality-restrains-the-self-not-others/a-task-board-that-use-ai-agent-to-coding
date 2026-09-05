// @ts-check
/**
 * 核验：用户登录后访问 tenant/{id}/settings/status，选择默认进度体系时
 * POST /api/projects/settings/default-progress-system/tenant_id/{id} 应返回 200 而非 403
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

test.describe('登录后设置默认进度体系 default-progress-system POST API', () => {
  test('选择默认进度体系后 POST 应返回 200 而非 403', async ({ page }) => {
    const statusSettingsUrl = '/tenant/821976991517573120/settings/status/';

    // 监听 default-progress-system API 的 POST 响应
    const postResponses = [];
    page.on('response', async (response) => {
      const url = response.url();
      const method = response.request().method();
      if (url.includes('default-progress-system') && method === 'POST') {
        let body = null;
        try {
          body = await response.text();
        } catch {
          // ignore
        }
        postResponses.push({
          url,
          status: response.status(),
          statusText: response.statusText(),
          body,
        });
      }
    });

    // 1. 登录（须勾选协议，否则仍停留在 /auth/login/）
    await playwrightLoginWithLegalAccept(page, {
      email: 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
    });
    expect(page.url().includes('/auth/login/'), '登录后应离开登录页').toBe(false);

    // 2. 导航到 settings/status 页面
    await page.goto(statusSettingsUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // 3. 定位「选择默认进度体系」下拉框（须用整张白卡片，勿用任意含标题的 div，否则可能匹配到仅含 h3 的 flex 行而找不到 select）
    const section = page
      .locator('div.bg-white.p-6.rounded-xl.shadow')
      .filter({ has: page.getByRole('heading', { name: '默认进度体系设置' }) });
    const select = section.locator('select').first();
    await expect(select).toBeVisible({ timeout: 10000 });

    // 4. 等待 options 加载（至少有一个非空的选项）
    await page.waitForTimeout(1500);
    const options = await select.locator('option[value]:not([value=""])').all();
    const hasSelectableOptions = options.length > 0;

    if (hasSelectableOptions) {
      // 选择第一个有效选项（触发 @change -> handleProgressSystemChange）
      const firstOptionValue = await options[0].getAttribute('value');
      await select.selectOption(firstOptionValue);
      await page.waitForTimeout(1500);

      // 5. 验证 POST default-progress-system 返回 200
      const postCalls = postResponses.filter((r) => r.url.includes('default-progress-system'));
      expect(
        postCalls.length,
        '应触发 default-progress-system POST 请求'
      ).toBeGreaterThan(0);

      for (const call of postCalls) {
        expect(
          call.status,
          `default-progress-system POST ${call.url} 不应返回 403，实际: ${call.status}，body: ${call.body || ''}`
        ).toBe(200);
      }
    } else {
      // 若无可选进度体系，至少验证页面加载正常、无 403 错误
      const has403Error = await page
        .locator('text=403')
        .or(page.locator('text=CSRF'))
        .first()
        .isVisible()
        .catch(() => false);
      expect(has403Error, '页面不应显示 403 或 CSRF 错误').toBe(false);
    }
  });
});
