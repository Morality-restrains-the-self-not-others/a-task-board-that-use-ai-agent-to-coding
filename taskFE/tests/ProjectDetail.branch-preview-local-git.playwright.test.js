// @ts-check
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const PROJECT_URL = '/tenant/${TENANT_ID}/projects/846027310833254400/';
const BRANCHES_API_PATH = `/api/projects/tenant_id/${TENANT_ID}846027310833254400/branches/`;
const GITLAB_OAUTH_START_PATH = '/api/git-oauth/gitlab-app-start/';
const AUTH_TOKEN = '29a79eb5c88dbde51b621a3e2c95631d28efbbe7';
const GENERIC_ERROR = 'Branches cannot be retrieved from generic Git repositories without credentials';
const GITLAB_NOT_FOUND_ERROR = 'GitLab repository not found';
const GITLAB_SESSION_COOKIE = '3c9c3563405eab0cb0f6b5d4bd8b54de';

test.describe('ProjectDetail 分支列表预览（本地 Git 服务）', () => {
  test('点击后不应再返回 generic 仓库报错', async ({ page }) => {
    await page.context().addCookies([
      {
        name: '_gitlab_session',
        value: GITLAB_SESSION_COOKIE,
        url: 'http://localhost:4000/'
      }
    ]);

    const expectGenericError = String(process.env.EXPECT_GENERIC_ERROR || '').trim() === '1';

    await page.addInitScript((token) => {
      window.localStorage.setItem('authToken', token);
    }, AUTH_TOKEN);

    await page.goto(PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const previewButton = page.getByRole('button', { name: '分支列表预览' });
    await expect(previewButton).toBeVisible({ timeout: 15000 });

    const responsePromise = page.waitForResponse((resp) => {
      try {
        return resp.url().includes(BRANCHES_API_PATH);
      } catch {
        return false;
      }
    });

    await previewButton.click();
    const response = await responsePromise;
    const payload = await response.json().catch(() => ({}));
    const errorText = String(payload?.error || '');

    if (expectGenericError) {
      expect(errorText).toContain(GENERIC_ERROR);
      return;
    }

    expect(errorText).not.toContain(GENERIC_ERROR);
    expect(errorText).not.toContain(GITLAB_NOT_FOUND_ERROR);
    expect(Array.isArray(payload?.branches) ? payload.branches : []).toContain('main');
    await expect(page.locator('text=Branches cannot be retrieved from generic Git repositories without credentials')).toHaveCount(0);
    await expect(page.locator('text=GitLab repository not found')).toHaveCount(0);
  });

  test('GitLab 仓库行应显示 OAuth 授权按钮并发起 start 请求', async ({ page }) => {
    await page.addInitScript((token) => {
      window.localStorage.setItem('authToken', token);
    }, AUTH_TOKEN);

    let startRequestUrl = '';
    await page.route(`**${GITLAB_OAUTH_START_PATH}**`, async (route) => {
      startRequestUrl = route.request().url();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          authorize_url: 'http://localhost:4000/oauth-mock-return',
        }),
      });
    });

    await page.goto(PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const oauthButton = page.getByRole('button', { name: 'OAuth 授权' }).first();
    await expect(oauthButton).toBeVisible({ timeout: 15000 });
    await oauthButton.click();

    await expect.poll(() => startRequestUrl).not.toBe('');
    expect(startRequestUrl).toContain('/api/git-oauth/gitlab-app-start/?');
    expect(startRequestUrl).toContain('repo_url=');
    expect(startRequestUrl).toContain('return_key=');
    expect(startRequestUrl).toContain('next=');
  });
});
