import { test, expect } from './fixtures';
import { testIds } from '../src/components/testIds';

test.describe('Daydaymoney Grafana 登录状态持久化', () => {
  test('登录成功后自动 POST plugin settings 并显示 Session 已配置', async ({ appConfigPage, page }) => {
    const pluginId = 'daydaymoney-grafana-app';
    const sessionToken = 'e2e-persisted-session-token';
    const account = 'persisted-user@example.com';
    const apiBaseUrl = 'http://127.0.0.1:18081';

    await page.route('**/api/accounts/users/login-with-access-token/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: sessionToken,
          user: { id: '850256677331562496', companies: [{ id: '850256677331562496', name: 'demo-co' }] },
        }),
      });
    });

    await page.route('**/api/user/*/accounts/users/me/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '850256677331562496',
          companies: [{ id: '850256677331562496', name: 'demo-co' }],
        }),
      });
    });

    await page.route('**/api/tenant/*/workspaces/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    const settingsRequest = page.waitForRequest(
      (request) => request.method() === 'POST' && request.url().includes(`/api/plugins/${pluginId}/settings`)
    );

    await page.getByTestId(testIds.appConfig.apiBaseUrl).fill(apiBaseUrl);
    await page.getByTestId(testIds.appConfig.account).fill(account);
    await page.getByTestId(testIds.appConfig.accessToken).fill('at_e2e_demo_token');
    await page.getByRole('button', { name: /登录/i }).click();

    await expect(page.getByText(/已登录/i)).toBeVisible();
    await expect(page.getByTestId(testIds.appConfig.loggedInAccount)).toContainText(account);
    await expect(page.getByTestId(testIds.appConfig.accessToken)).toHaveCount(0);
    await expect(page.getByTestId(testIds.appConfig.logout)).toBeVisible();

    const request = await settingsRequest;
    const payload = request.postDataJSON() as { jsonData?: Record<string, string> };
    expect(payload.jsonData?.task2appSessionToken).toBe(sessionToken);
    expect(payload.jsonData?.task2appAccount).toBe(account);
    expect(payload.jsonData?.task2appApiBaseUrl).toBe(apiBaseUrl);
  });
});
