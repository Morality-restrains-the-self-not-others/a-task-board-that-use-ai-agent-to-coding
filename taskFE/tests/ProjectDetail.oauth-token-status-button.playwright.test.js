// @ts-check
/**
 * E2E: 项目详情页 OAuth 徽章。OPT-20260829-013：他人仓 validate-git-repos
 * 返回 token_error + is_accessible=false 时为「授权异常」+「重试」，不得「已授权」。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const portConfig = loadPortConfig();
const vueHost = clientReachableHost(portConfig.vue?.host);
const vuePort = portConfig.vue?.port || 4000;
const ORIGIN = `http://${vueHost}:${vuePort}`;

const LOGIN_URL = '/auth/login/';
const LIVE_PROJECT_URL = '/tenant/850256677331562496/projects/858546008673890304/';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;

const MOCK_TENANT = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const MOCK_PROJECT_ID = 'proj_oauth_token_error_other_repo';
const OTHER_GITHUB_URL = 'https://github.com/test-ruandao/helloworld';
const MOCK_PROJECT_PATH = `/tenant/${MOCK_TENANT}/projects/${MOCK_PROJECT_ID}/`;

/**
 * Mock 项目详情 GET 的 git_repos_status（旧路径，不走 validate-git-repos）。
 */
async function mockProjectDetailApi(page, tokenStatusOverrides) {
  await page.route('**/api/projects/**', async (route) => {
    const url = route.request().url();
    if (url.includes('/branches/') || url.includes('/repo-access-check') || url.includes('validate-git-repos')) {
      await route.continue();
      return;
    }
    if (!url.includes('/tenant_id/')) {
      await route.continue();
      return;
    }

    const response = await route.fetch();
    const body = await response.json();

    if (Array.isArray(body.git_repos) && body.git_repos.length > 0) {
      body.git_repos_status = body.git_repos.map((repoUrl, i) => {
        if (tokenStatusOverrides[i]) {
          return {
            repo_url: repoUrl,
            token_status: tokenStatusOverrides[i],
            oauth_provider: 'gitlab',
            oauth_service_provider: 'default',
          };
        }
        return {
          repo_url: repoUrl,
          token_status: 'not_applicable',
          oauth_provider: '',
          oauth_service_provider: '',
        };
      });
    }

    await route.fulfill({ response, json: body });
  });
}

async function login(page) {
  await page.goto(LOGIN_URL);
  await page.waitForLoadState('networkidle');
  await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD });
  await page.waitForTimeout(2000);
}

test.describe('项目详情页 OAuth 按钮 token_status 感知（live 项目）', () => {
  test.skip(!PASSWORD || process.env.PLAYWRIGHT_LIVE_PROJECT !== '1', '需 PLAYWRIGHT_LIVE_PROJECT=1 与 PASSWORD');

  test('token_available 时显示已授权徽章且不显示 OAuth 按钮', async ({ page }) => {
    await mockProjectDetailApi(page, ['token_available']);
    await login(page);

    await page.goto(LIVE_PROJECT_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    const projectHeading = page.getByRole('heading', { name: '项目详情' });
    if (!(await projectHeading.isVisible({ timeout: 10000 }).catch(() => false))) {
      test.skip(true, '项目详情页未加载，跳过');
    }

    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('已授权');
    await expect(page.getByRole('button', { name: 'OAuth 授权' })).toHaveCount(0);
    await expect(page.getByRole('button', { name: '重试' })).toHaveCount(0);
  });

  test('not_bound 时显示需要授权徽章与 OAuth 授权按钮', async ({ page }) => {
    await mockProjectDetailApi(page, ['not_bound']);
    await login(page);

    await page.goto(LIVE_PROJECT_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    const projectHeading = page.getByRole('heading', { name: '项目详情' });
    if (!(await projectHeading.isVisible({ timeout: 10000 }).catch(() => false))) {
      test.skip(true, '项目详情页未加载，跳过');
    }

    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('需要授权');
    await expect(page.getByRole('button', { name: 'OAuth 授权' })).toHaveCount(1);
  });

  test('token_error 时显示授权异常徽章与重试按钮', async ({ page }) => {
    await mockProjectDetailApi(page, ['token_error']);
    await login(page);

    await page.goto(LIVE_PROJECT_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    const projectHeading = page.getByRole('heading', { name: '项目详情' });
    if (!(await projectHeading.isVisible({ timeout: 10000 }).catch(() => false))) {
      test.skip(true, '项目详情页未加载，跳过');
    }

    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('授权异常');
    await expect(page.getByRole('button', { name: '重试' })).toHaveCount(1);
    await expect(page.getByRole('button', { name: 'OAuth 授权' })).toHaveCount(0);
  });
});

test.describe('OPT-20260829-013 他人仓 validate-git-repos token_error', () => {
  test('他人 GitHub 仓 probe token_error 显示授权异常与重试，不得已授权', async ({ page }) => {
    test.setTimeout(120000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: `${ORIGIN}/` },
      { name: 'sessionid', value: 'e2e-session-oauth-token-error', url: `${ORIGIN}/` },
      { name: 'csrftoken', value: 'e2e-csrf', url: `${ORIGIN}/` },
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'sessionid', value: 'e2e-session-oauth-token-error', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes('/api/accounts/users/me/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'e2e-user',
            username: 'e2e',
            is_superuser: true,
            phone_verified: true,
            companies: [{ id: MOCK_TENANT, name: 'E2E Co' }],
          }),
        });
        return;
      }

      if (url.includes('validate-git-repos') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            results: [
              {
                url: OTHER_GITHUB_URL,
                repo_url: OTHER_GITHUB_URL,
                token_status: 'token_error',
                is_accessible: false,
                oauth_provider: 'github',
                oauth_service_provider: 'default',
              },
            ],
          }),
        });
        return;
      }

      if (
        url.includes(`/api/projects/${MOCK_PROJECT_ID}/tenant_id/${MOCK_TENANT}`) &&
        method === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: MOCK_PROJECT_ID,
            name: 'helloworld-other',
            description: '他人仓 token_error',
            git_repos: [OTHER_GITHUB_URL],
            git_repos_status: [],
            workspaces: [],
            company: MOCK_TENANT,
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
            providers: [
              {
                provider: 'github',
                service_provider: 'default',
                website: 'https://github.com',
              },
            ],
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(MOCK_PROJECT_PATH);
    await page.waitForLoadState('domcontentloaded');

    await expect(page.getByRole('heading', { name: '项目详情' })).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('git-repo-url-link')).toHaveText(OTHER_GITHUB_URL, { timeout: 15000 });
    await expect(page.getByTestId('git-repo-oauth-status-0')).toHaveText('授权异常', { timeout: 20000 });
    await expect(page.getByTestId('git-repo-oauth-status-0')).not.toHaveText('已授权');
    await expect(page.getByRole('button', { name: '重试' })).toHaveCount(1);
    await expect(page.getByRole('button', { name: 'OAuth 授权' })).toHaveCount(0);
  });
});
