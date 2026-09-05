// @ts-check
/**
 * 调试：项目详情 GitLab OAuth 授权后 profile_failed 根因
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const PROJECT_URL = '/tenant/${TENANT_ID}/projects/846027310833254400/';
const AUTH_TOKEN = '29a79eb5c88dbde51b621a3e2c95631d28efbbe7';
const GITLAB_USER = process.env.GITLAB_USER || 'ljy';
const GITLAB_PASSWORD = process.env.GITLAB_PASSWORD;
test.skip(!GITLAB_PASSWORD, 'GITLAB_PASSWORD env required (no hardcoded fallback)');

test.describe('ProjectDetail GitLab OAuth profile_failed 调试', () => {
  test('完整 OAuth 流程并捕获最终 URL 与 Toast', async ({ page, context }) => {
    test.setTimeout(180000);

    await page.addInitScript((token) => {
      window.localStorage.setItem('authToken', token);
    }, AUTH_TOKEN);

    const networkLog = [];
    page.on('response', async (resp) => {
      const url = resp.url();
      if (
        url.includes('/oauth/') ||
        url.includes('/api/accounts/gitlab') ||
        url.includes('/api/v4/user') ||
        url.includes('8012')
      ) {
        networkLog.push({ url: url.slice(0, 300), status: resp.status() });
      }
    });

    await page.goto(PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const oauthButton = page.getByRole('button', { name: 'OAuth 授权' }).first();
    await expect(oauthButton).toBeVisible({ timeout: 15000 });

    const popupPromise = context.waitForEvent('page', { timeout: 30000 }).catch(() => null);
    await oauthButton.click();

    let gitlabPage = await popupPromise;
    if (!gitlabPage) {
      await page.waitForURL(/8012|8002|oauth/, { timeout: 60000 }).catch(() => {});
      gitlabPage = page.url().includes('8012') || page.url().includes('oauth') ? page : page;
    }

    const activePage = gitlabPage || page;
    await activePage.waitForLoadState('domcontentloaded');

    for (let step = 0; step < 20; step++) {
      const u = activePage.url();
      console.log('[step', step, ']', u.slice(0, 200));

      if (u.includes('localhost:4000') && (u.includes('gitlab=') || u.includes('github-app/callback'))) {
        break;
      }

      if (u.includes('8012/users/sign_in') || u.includes('8012/login')) {
        await activePage.locator('#login_field, input[name="user[login]"]').first().fill(GITLAB_USER);
        await activePage.locator('#password, input[name="user[password]"]').first().fill(GITLAB_PASSWORD);
        await activePage.locator('button[type="submit"], input[type="submit"]').first().click();
        await activePage.waitForTimeout(2000);
        continue;
      }

      if (u.includes('8012/oauth/authorize')) {
        const errHeading = activePage.getByRole('heading', { name: /An error has occurred|error/i });
        if (await errHeading.isVisible().catch(() => false)) {
          const errText = await activePage.locator('main').innerText().catch(() => '');
          throw new Error(`GitLab 授权页错误: ${errText.slice(0, 300)}`);
        }
        const authorizeBtn = activePage.getByTestId('authorization-button');
        if (await authorizeBtn.isVisible().catch(() => false)) {
          await authorizeBtn.click({ force: true });
        } else {
          await activePage.locator('form').first().evaluate((form) => form.requestSubmit());
        }
        await activePage.waitForTimeout(3000);
        continue;
      }

      if (u.includes('8002/api/accounts')) {
        await activePage.waitForTimeout(2000);
        continue;
      }

      await activePage.waitForTimeout(1000);
    }

    await page.waitForTimeout(3000);
    const finalUrl = page.url();
    console.log('[final url]', finalUrl);

    const toastText = await page.locator('[class*="toast"], [role="alert"], .Toastify__toast-body').allTextContents().catch(() => []);
    console.log('[toast]', toastText);

    console.log('[network sample]', JSON.stringify(networkLog.slice(-15), null, 2));

    if (finalUrl.includes('profile_failed') || toastText.some((t) => t.includes('无法读取 GitLab'))) {
      throw new Error(`profile_failed 复现: url=${finalUrl} toast=${JSON.stringify(toastText)}`);
    }

    expect(finalUrl).toMatch(/846027310833254400/);
    expect(finalUrl).not.toContain('profile_failed');
    expect(toastText.join(' ')).not.toContain('无法读取 GitLab');
  });
});
