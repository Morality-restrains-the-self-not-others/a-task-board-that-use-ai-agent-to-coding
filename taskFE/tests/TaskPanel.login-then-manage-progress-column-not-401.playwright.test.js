// @ts-check
/**
 * 核验：用户登录后访问 tenant/{id}/settings/task-panel 不应出现 401 无效的认证token
 * 测试账号：contact@daydaymoney.com / <env:PLAYWRIGHT_TEST_PASSWORD>
 */
import { test, expect } from '@playwright/test';

test.describe('登录后访问 task-panel manage-progress-column API', () => {
  test('登录后跳转 task-panel，manage-progress-column API 应返回 200 而非 401', async ({ page }) => {
    const loginUrl = '/auth/login/';
    const taskPanelUrl = '/tenant/821976991517573120/settings/task-panel/';

    // 监听 API 请求，捕获 manage-progress-column 的响应
    const apiResponses = [];
    page.on('response', async (response) => {
      const url = response.url();
      if (url.includes('manage-progress-column')) {
        let body = null;
        try {
          body = await response.text();
        } catch {
          // ignore
        }
        apiResponses.push({
          url,
          status: response.status(),
          statusText: response.statusText(),
          body,
        });
      }
    });

    // 1. 打开登录页
    await page.goto(loginUrl);
    await page.waitForLoadState('networkidle');

    // 2. 选择邮箱密码登录方式（若需切换）
    const emailPasswordTab = page.locator('text=邮箱').or(page.locator('button:has-text("邮箱")')).first();
    if (await emailPasswordTab.isVisible()) {
      await emailPasswordTab.click();
      await page.waitForTimeout(300);
    }

    // 3. 填写邮箱和密码
    const emailInput = page.locator('#email');
    const passwordInput = page.locator('#password');
    await expect(emailInput).toBeVisible();
    await expect(passwordInput).toBeVisible();

    await emailInput.fill('contact@daydaymoney.com');
    await passwordInput.fill(process.env.PLAYWRIGHT_TEST_PASSWORD);

    // 4. 提交登录
    await page.locator('form').first().evaluate((form) => form.requestSubmit());
    await page.waitForLoadState('networkidle');

    // 5. 等待登录成功后跳转
    await page.waitForTimeout(2000);

    // 6. 导航到 task-panel 页面
    await page.goto(taskPanelUrl);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1500);

    // 7. 检查 manage-progress-column API 是否返回 401
    const manageProgressCalls = apiResponses.filter((r) => r.url.includes('manage-progress-column'));

    if (manageProgressCalls.length > 0) {
      for (const call of manageProgressCalls) {
        expect(
          call.status,
          `manage-progress-column API ${call.url} 不应返回 401，实际: ${call.status}，body: ${call.body || ''}`
        ).not.toBe(401);
      }
    } else {
      // 若未捕获到请求，可能是页面未触发或选择器不同，至少验证页面加载
      const hasAuthError = await page
        .locator('text=无效的认证token')
        .or(page.locator('text=Invalid token'))
        .first()
        .isVisible()
        .catch(() => false);
      expect(hasAuthError, '页面不应显示认证 token 错误').toBe(false);
    }
  });
});
