// @ts-check
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const AUTH_TOKEN = '29a79eb5c88dbde51b621a3e2c95631d28efbbe7';
const TENCENT_PROJECT_URL = '/tenant/${TENANT_ID}/projects/848537693488873472/';
const GENERIC_ERROR = 'Branches cannot be retrieved from generic Git repositories without credentials';

test.describe('ProjectDetail 分支预览（配置驱动 IP GitLab）', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端就绪的全栈 E2E');

  test.beforeEach(async ({ page }) => {
    await page.addInitScript((token) => {
      window.localStorage.setItem('authToken', token);
    }, AUTH_TOKEN);

    await page.route('**/api/git-oauth/providers/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          providers: [
            {
              provider: 'gitlab',
              service_provider: 'tencent-gitlab',
              provider_key: 'gitlab:tencent-gitlab',
              website: 'http://1.117.67.121:8012',
              label: 'GitLab (tencent-gitlab)',
            },
          ],
        }),
      });
    });
  });

  test('IP GitLab 仓库行应显示 OAuth 授权按钮', async ({ page }) => {
    await page.goto(TENCENT_PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const oauthButton = page.getByRole('button', { name: 'OAuth 授权' }).first();
    await expect(oauthButton).toBeVisible({ timeout: 15000 });
  });

  test('分支预览不应返回 generic 仓库报错', async ({ page }) => {
    await page.route('**/projects/848537693488873472/branches/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          branches: [],
          error: '无法获取 GitLab 分支：未检测到可用授权。请先在个人资料完成 GitLab 绑定，或先登录本地 GitLab（localhost）后重试。',
        }),
      });
    });

    await page.goto(TENCENT_PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const previewButton = page.getByRole('button', { name: '分支列表预览' });
    await expect(previewButton).toBeVisible({ timeout: 15000 });
    await previewButton.click();

    await expect(page.locator(`text=${GENERIC_ERROR}`)).toHaveCount(0, { timeout: 10000 });
    await expect(page.locator('text=未检测到可用授权')).toHaveCount(1);
  });

  test('分支预览 slow backend 在 15s 内应展示业务错误而非超时', async ({ page }) => {
    await page.route('**/projects/848537693488873472/branches/**', async (route) => {
      await new Promise((r) => setTimeout(r, 6000));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          branches: [],
          error:
            '无法获取 GitLab 分支：未检测到可用授权。请先在个人资料完成 GitLab 绑定，或先登录本地 GitLab（localhost）后重试。',
        }),
      });
    });

    await page.goto(TENCENT_PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const previewButton = page.getByRole('button', { name: '分支列表预览' });
    await previewButton.click();

    await expect(page.locator('text=未检测到可用授权')).toBeVisible({ timeout: 20000 });
    await expect(page.locator('text=分支列表查询超时')).toHaveCount(0);
  });
});
