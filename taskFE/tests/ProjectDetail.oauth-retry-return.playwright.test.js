// @ts-check
/**
 * 项目详情「授权异常 → 重试」：模拟 OAuth 往返后必须跳回本页（含 accessCode），
 * 不得落到 /profile/git-site-oauth/。
 *
 * 运行：bash tests/ProjectDetail.oauth-retry-return.playwright.test.sh
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const portConfig = loadPortConfig();
const vueHost = clientReachableHost(portConfig.vue?.host, '127.0.0.1');
const vuePort = portConfig.vue?.port || 4000;
const DEFAULT_BASE = `http://${vueHost}:${vuePort}`;
const BASE = (process.env.PLAYWRIGHT_SITE_ORIGIN || DEFAULT_BASE).replace(/\/$/, '');

const TENANT_ID = process.env.TEST_TENANT_ID || '877397588196749312';
const PROJECT_ID = process.env.TEST_PROJECT_ID || 'proj_880498883115905024';
const ACCESS_CODE = process.env.TEST_ACCESS_CODE || '9aaHjbryhL';
const PROJECT_PATH = `/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const PROJECT_URL = `${PROJECT_PATH}?accessCode=${encodeURIComponent(ACCESS_CODE)}`;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const REPO_URL = 'https://gitlab.com/acme/oauth-retry-demo.git';

/**
 * @param {import('@playwright/test').Page} page
 */
async function mockTokenErrorAndOauthBounce(page) {
  /** @type {{ next: string, returnKey: string }} */
  const captured = { next: '', returnKey: '' };

  await page.route('**/api/projects/**', async (route) => {
    const url = route.request().url();
    if (url.includes('/nested-git-repos')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ parent_repo_url: REPO_URL, nested_repos: [], error: '' }),
      });
      return;
    }
    if (url.includes('/validate-git-repos')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [{ url: REPO_URL, repo_url: REPO_URL, token_status: 'token_error' }],
        }),
      });
      return;
    }
    if (url.includes('/validate-git-repo') || url.includes('/branches/') || url.includes('/repo-access-check')) {
      await route.fallback();
      return;
    }
    if (route.request().method() !== 'GET') {
      await route.fallback();
      return;
    }
    if (!url.includes(`/api/projects/${PROJECT_ID}/`)) {
      await route.fallback();
      return;
    }
    const statusRow = {
      repo_url: REPO_URL,
      token_status: 'token_error',
      oauth_provider: 'gitlab',
      oauth_service_provider: 'default',
    };
    const patch = {
      git_repos: [REPO_URL],
      git_repo_entries: [{ url: REPO_URL }],
      git_repos_status: [statusRow],
    };
    try {
      const response = await route.fetch();
      const body = await response.json();
      await route.fulfill({
        response,
        json: { ...body, ...patch, name: body?.name || '云端开发' },
      });
    } catch {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: PROJECT_ID,
          name: '云端开发',
          company: TENANT_ID,
          workspaces: [],
          ...patch,
        }),
      });
    }
  });

  await page.route(/\/api\/git-oauth\/(gitlab|github)-start-from-gateway\//, async (route) => {
    const reqUrl = new URL(route.request().url());
    captured.next = reqUrl.searchParams.get('next') || '';
    captured.returnKey = reqUrl.searchParams.get('return_key') || '';
    const origin = new URL(page.url()).origin;
    const bounce =
      `${origin}/oauth/github-app/callback/` +
      `?returnKey=${encodeURIComponent(captured.returnKey)}` +
      `&provider=gitlab&gitlab=ok`;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ authorize_url: bounce }),
    });
  });

  return captured;
}

test.describe('项目详情 授权异常重试后跳回本页', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的 OAuth 回流 E2E');
  test.setTimeout(180_000);

  test('点击重试后经回调落地页跳回项目详情且保留 accessCode', async ({ page }) => {
    const captured = await mockTokenErrorAndOauthBounce(page);

    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: BASE,
    });
    const reconsent = page.getByRole('button', { name: '同意并继续' });
    if (await reconsent.isVisible({ timeout: 4000 }).catch(() => false)) {
      await reconsent.click();
    }

    await page.goto(`${BASE}${PROJECT_URL}`, {
      waitUntil: 'domcontentloaded',
      timeout: 60_000,
    });
    await page.waitForLoadState('networkidle').catch(() => {});

    const heading = page.getByRole('heading', { name: '项目详情' });
    await expect(heading).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText('云端开发').first()).toBeVisible();

    const status = page.getByTestId('git-repo-oauth-status-0');
    await expect(status).toHaveText('授权异常', { timeout: 20_000 });
    const retryBtn = page.getByTestId('git-repo-oauth-action-0');
    await expect(retryBtn).toHaveText('重试');

    const beforeClick = new URL(page.url());
    const beforeAccess = beforeClick.searchParams.get('accessCode') || ACCESS_CODE;
    expect(beforeClick.pathname.replace(/\/+$/, '/')).toContain(
      PROJECT_PATH.replace(/\/+$/, '/'),
    );

    await retryBtn.click();

    await expect.poll(() => captured.next, { timeout: 20_000 }).not.toBe('');
    expect(captured.next).toContain(PROJECT_PATH);
    expect(captured.next).toContain('accessCode=');
    expect(captured.returnKey).toMatch(/^[0-9a-f]{32}$/);

    await expect(page).toHaveURL(new RegExp(`${PROJECT_PATH.replace(/\//g, '\\/')}`), {
      timeout: 45_000,
    });
    await expect(page).not.toHaveURL(/\/profile\/git-site-oauth/);
    await expect(page).not.toHaveURL(/\/oauth\/github-app\/callback/);

    const landed = new URL(page.url());
    expect(landed.searchParams.get('accessCode')).toBe(beforeAccess);

    await expect(heading).toBeVisible({ timeout: 20_000 });
    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('授权异常', {
      timeout: 20_000,
    });
    await expect(page.getByTestId('git-repo-oauth-action-0')).toHaveText('重试');
  });
});
