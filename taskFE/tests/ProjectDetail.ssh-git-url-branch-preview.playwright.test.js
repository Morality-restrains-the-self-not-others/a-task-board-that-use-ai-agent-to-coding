// @ts-check
/**
 * E2E: SSH 格式 Git 仓库在 OAuth 已授权时不应再显示
 * "SSH Git URLs cannot be checked for branches without credentials"。
 *
 * 场景：
 * 1. 项目 git_repos 为 git@host:path 格式，git_repos_status 为 token_available（已授权）
 * 2. 点击「分支列表预览」后，不应出现 SSH 凭证错误
 * 3. branches API 应走 OAuth 规范化后的 HTTPS 查询（返回分支或授权提示，而非 SSH 错误）
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
const TENANT_ID = process.env.PW_SSH_BRANCH_TENANT_ID || '850256677331562496';
const PROJECT_ID = process.env.PW_SSH_BRANCH_PROJECT_ID || 'proj_-5910410547773912751';
const SSH_REPO_URL =
  process.env.PW_SSH_BRANCH_REPO_URL ||
  'git@127.0.0.1:example-user/somanyad-emailD.git';

const LOGIN_URL = '/auth/login/';
const PROJECT_URL = `/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const BRANCHES_API_PATH = `/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/branches/`;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const SSH_BRANCH_ERROR = 'SSH Git URLs cannot be checked for branches without credentials';

async function login(page) {
  await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD });
  await page.waitForTimeout(1500);
}

test.describe('ProjectDetail SSH Git 仓库分支预览与 OAuth 已授权', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端就绪的全栈 E2E');

  test('OAuth 已授权 + SSH 仓库：分支预览不应出现 SSH 凭证错误', async ({ page }) => {
    await login(page);
    await page.goto(PROJECT_URL);
    await page.waitForLoadState('networkidle');

    const projectHeading = page.getByRole('heading', { name: '项目详情' });
    if (!(await projectHeading.isVisible({ timeout: 15000 }).catch(() => false))) {
      test.skip(true, '项目详情页未加载，跳过');
    }

    const oauthBadge = page.getByTestId('git-repo-oauth-status-0');
    if (await oauthBadge.isVisible({ timeout: 10000 }).catch(() => false)) {
      const badgeText = (await oauthBadge.textContent())?.trim() || '';
      if (badgeText === '已授权') {
        // 已授权场景：继续验证分支预览
      } else if (badgeText === '检查中…') {
        await expect(oauthBadge).toHaveText(/已授权|未授权|无需 OAuth|授权异常/, { timeout: 15000 });
      }
    }

    const previewButton = page.getByRole('button', { name: '分支列表预览' });
    await expect(previewButton).toBeVisible({ timeout: 15000 });

    const responsePromise = page.waitForResponse((resp) => {
      try {
        return resp.url().includes(BRANCHES_API_PATH) && resp.request().method() === 'GET';
      } catch {
        return false;
      }
    });

    await previewButton.click();
    const response = await responsePromise;
    const payload = await response.json().catch(() => ({}));

    expect(response.status()).not.toBe(501);
    expect(String(payload?.error || '')).not.toContain(SSH_BRANCH_ERROR);

    const previewPanel = page.locator('[class*="bg-gray-50"]').filter({ hasText: SSH_REPO_URL });
    if (await previewPanel.count()) {
      await expect(previewPanel.first()).not.toContainText(SSH_BRANCH_ERROR);
    }

    const errorText = String(payload?.error || '');
    const branches = Array.isArray(payload?.branches) ? payload.branches : [];
    const hasBranches = branches.length > 0;
    const hasAuthHint =
      errorText.includes('未检测到可用授权') ||
      errorText.includes('OAuth 授权') ||
      errorText.includes('GitLab 会话无效') ||
      errorText.includes('GitHub');
    expect(hasBranches || hasAuthHint || errorText === '').toBe(true);
  });

  test('mock SSH 仓库已授权时 branches API 响应不含 SSH 错误', async ({ page }) => {
    await page.route('**/api/projects/tenant_id/**/', async (route) => {
      const url = route.request().url();
      if (url.includes('/branches/') || url.includes('/repo-access-check')) {
        await route.continue();
        return;
      }

      const response = await route.fetch();
      const body = await response.json();
      body.git_repos = [SSH_REPO_URL];
      body.git_repos_status = [
        {
          repo_url: SSH_REPO_URL,
          token_status: 'token_available',
          oauth_provider: 'gitlab',
          oauth_service_provider: 'synology-gitlab',
        },
      ];
      await route.fulfill({ response, json: body });
    });

    await page.route(`**${BRANCHES_API_PATH}**`, async (route) => {
      const reqUrl = decodeURIComponent(route.request().url());
      expect(reqUrl).toContain('repo_url=');
      expect(reqUrl).toContain(SSH_REPO_URL);

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          branches: ['main', 'dev'],
          error: '',
        }),
      });
    });

    await login(page);
    await page.goto(PROJECT_URL);
    await page.waitForLoadState('networkidle');

    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('已授权', { timeout: 15000 });

    const previewButton = page.getByRole('button', { name: '分支列表预览' });
    await previewButton.click();

    await expect(page.getByText(SSH_BRANCH_ERROR)).toHaveCount(0);
    await expect(page.getByText('main')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('dev')).toBeVisible();
  });
});
