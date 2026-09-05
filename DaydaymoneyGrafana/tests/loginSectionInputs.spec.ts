import { test, expect } from './fixtures';
import { testIds } from '../src/components/testIds';

test.describe('Daydaymoney Grafana 配置页登录表单', () => {
  test('账号与访问令牌输入框可编辑', async ({ appConfigPage, page }) => {
    const accountInput = page.getByTestId(testIds.appConfig.account);
    const accessTokenInput = page.getByTestId(testIds.appConfig.accessToken);
    const loginButton = page.getByRole('button', { name: /登录/i });

    await expect(accountInput).toBeEditable();
    await expect(accessTokenInput).toBeEditable();
    await expect(loginButton).toBeDisabled();

    await accountInput.fill('e2e-demo-user');
    await accessTokenInput.fill('at_e2e_demo_token');

    await expect(accountInput).toHaveValue('e2e-demo-user');
    await expect(accessTokenInput).toHaveValue('at_e2e_demo_token');
    await expect(loginButton).toBeEnabled();
  });

  test('API 基地址输入框可编辑', async ({ appConfigPage, page }) => {
    const apiBaseUrlInput = page.getByTestId(testIds.appConfig.apiBaseUrl);

    await expect(apiBaseUrlInput).toBeEditable();
    await apiBaseUrlInput.fill('http://127.0.0.1:18081');
    await expect(apiBaseUrlInput).toHaveValue('http://127.0.0.1:18081');
  });
});
