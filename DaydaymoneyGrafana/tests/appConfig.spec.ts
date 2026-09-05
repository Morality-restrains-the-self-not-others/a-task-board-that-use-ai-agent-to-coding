import { test, expect } from './fixtures';
import { testIds } from '../src/components/testIds';

test('should be possible to save app configuration after login', async ({ appConfigPage, page }) => {
  const sessionToken = 'e2e-save-config-session-token';
  const account = 'save-config@example.com';
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

  await page.getByTestId(testIds.appConfig.apiBaseUrl).fill(apiBaseUrl);
  await page.getByTestId(testIds.appConfig.account).fill(account);
  await page.getByTestId(testIds.appConfig.accessToken).fill('at_e2e_demo_token');
  await page.getByRole('button', { name: /登录/i }).click();
  await expect(page.getByText(/已登录/i)).toBeVisible();

  const saveButton = page.getByRole('button', { name: /保存配置/i });
  await expect(saveButton).toBeEnabled();

  const saveResponse = appConfigPage.waitForSettingsResponse();
  await saveButton.click();
  await expect(saveResponse).toBeOK();

  // 保存后页面会 reload；确认配置页仍可打开且旧版回退区块不存在
  await expect(page.getByRole('group', { name: /旧版回退/i })).toHaveCount(0);
  await expect(page.getByTestId(testIds.appConfig.lokiPollEnabled)).toBeVisible();
});
