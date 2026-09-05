// @ts-check
/**
 * OPT-20260829-014：项目详情 Git OAuth 徽章三态（需要授权 / 已授权）。
 * 授权异常已由 ProjectDetail.oauth-token-status-button 覆盖。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';
import { addE2eSession, matchTenantShellApi } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const PROJECT_ID = 'proj_oauth_l2_badge';
const GITHUB_URL = 'https://github.com/ruandao/helloworld.git';
const PATH = `/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} tokenStatus
 */
async function installProject(page, tokenStatus) {
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    const shell = matchTenantShellApi(url, method, TENANT_ID);
    if (shell) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(shell) });
      return;
    }
    if (url.includes('validate-git-repos') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [{
            url: GITHUB_URL,
            repo_url: GITHUB_URL,
            token_status: tokenStatus,
            is_accessible: tokenStatus === 'token_available',
            oauth_provider: 'github',
            oauth_service_provider: 'default',
          }],
        }),
      });
      return;
    }
    if (url.includes(`/api/projects/${PROJECT_ID}/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: PROJECT_ID,
          name: 'helloworld-l2',
          description: 'l2 badges',
          git_repos: [GITHUB_URL],
          git_repos_status: [],
          workspaces: [],
          company: TENANT_ID,
          tags: [],
          server_run_template: {},
        }),
      });
      return;
    }
    if (url.includes('/api/git-oauth/providers/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          providers: [{ provider: 'github', service_provider: 'default', website: 'https://github.com' }],
        }),
      });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

test.describe('项目详情 OAuth L2 徽章（OPT-20260829-014）', () => {
  test('not_bound 显示需要授权', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-l2-not-bound' });
    await installProject(page, 'not_bound');
    await page.goto(PATH);
    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('需要授权', { timeout: 20000 });
  });

  test('token_available 显示已授权', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-l2-available' });
    await installProject(page, 'token_available');
    await page.goto(PATH);
    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('已授权', { timeout: 20000 });
  });
});
