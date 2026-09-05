// @ts-check
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || process.env.PW_TENANT_ID || PW_TENANT_ID;
const PROJECT_ID = process.env.PW_BRANCH_PREVIEW_PROJECT_ID || '861581450509701120';
const AUTH_TOKEN =
  process.env.PW_AUTH_TOKEN ||
  process.env.PLAYWRIGHT_AUTH_TOKEN ||
  'a8266ccd793329242ddeb0f269211c62924c383c';
const REPO_URL =
  process.env.PW_BRANCH_PREVIEW_REPO_URL ||
  'http://127.0.0.1:8012/example-user/task2app.git';

const PROJECT_URL = `/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const BRANCHES_API_PATH = `/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/branches/`;

test.describe('ProjectDetail 分支列表预览（Go branches API）', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端就绪的全栈 E2E');

  test.beforeEach(async ({ page }) => {
    await page.addInitScript((token) => {
      window.localStorage.setItem('authToken', token);
    }, AUTH_TOKEN);
  });

  test('点击分支列表预览不应再返回 501 Not Implemented', async ({ page }) => {
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

    expect(response.status()).not.toBe(501);
    expect(String(payload?.error || '')).not.toContain('endpoint not yet implemented in taskProjectService');
    expect(Array.isArray(payload?.branches)).toBe(true);
    expect(payload?.branches).not.toBeNull();

    const errorText = String(payload?.error || '');
    const branches = payload.branches;
    const hasBranches = branches.length > 0;
    const hasAuthHint =
      errorText.includes('未检测到可用授权') ||
      errorText.includes('OAuth 授权') ||
      errorText.includes('GitLab 会话无效');
    expect(hasBranches || hasAuthHint || errorText === '').toBe(true);
    expect(errorText).not.toContain('400 Client Error: Bad Request for url');
  });

  test('branches API 应携带 repo_url 查询参数', async ({ page }) => {
    let capturedUrl = '';
    await page.route(`**${BRANCHES_API_PATH}**`, async (route) => {
      capturedUrl = route.request().url();
      await route.continue();
    });

    await page.goto(PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const previewButton = page.getByRole('button', { name: '分支列表预览' });
    await expect(previewButton).toBeVisible({ timeout: 15000 });
    await previewButton.click();

    await expect.poll(() => capturedUrl).not.toBe('');
    expect(capturedUrl).toContain('repo_url=');
    expect(decodeURIComponent(capturedUrl)).toContain(REPO_URL.replace(/\/$/, ''));
  });
});
