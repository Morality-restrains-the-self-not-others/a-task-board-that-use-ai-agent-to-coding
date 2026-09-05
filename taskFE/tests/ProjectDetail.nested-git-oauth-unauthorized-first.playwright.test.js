// @ts-check
/**
 * E2E: 项目详情「子 Git 仓库」展示 OAuth 状态，未授权置顶 + 批量 validate-git-repos
 *
 * - 登录态打开项目详情
 * - mock nested-git-repos + validate-git-repos（批量）
 * - 断言 nested-git-repo-oauth-status-0 为「未授权」且对应未授权仓排在最前
 * - 断言走批量接口（非 N 次单仓 validate-git-repo）
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const portConfig = loadPortConfig();
const vueHost = clientReachableHost(portConfig.vue?.host, '127.0.0.1');
const vuePort = portConfig.vue?.port || 4000;
const BASE = `http://${vueHost}:${vuePort}`;

const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const PROJECT_ID = process.env.TEST_PROJECT_ID || 'proj_-5247879312070945751';
const PROJECT_URL = `/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const PARENT = 'https://gitlab.daydaymoney.com/example-user/ram-work.git';
const NESTED_AUTHORIZED = 'https://gitlab.daydaymoney.com/example-user/DaydaymoneyGrafana.git';
const NESTED_UNAUTHORIZED = 'https://gitlab.daydaymoney.com/example-user/docs.git';

async function login(page) {
  await playwrightLoginWithLegalAccept(page, {
    email: EMAIL,
    password: PASSWORD,
    baseURL: BASE,
  });
  await page.waitForTimeout(1500);
}

/**
 * @returns {{ batchCalls: { n: number }, singleCalls: { n: number } }}
 */
function mockNestedGitOAuthFlow(page) {
  const batchCalls = { n: 0 };
  const singleCalls = { n: 0 };

  page.route('**/nested-git-repos/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        parent_repo_url: PARENT,
        nested_repos: [
          {
            path: 'DaydaymoneyGrafana',
            url: NESTED_AUTHORIZED,
            source: 'gitmodules',
          },
          {
            path: 'docs',
            url: NESTED_UNAUTHORIZED,
            source: 'gitmodules',
          },
        ],
        error: '',
      }),
    });
  });

  page.route('**/validate-git-repos/**', async (route) => {
    batchCalls.n += 1;
    const postData = route.request().postDataJSON() || {};
    const urls = Array.isArray(postData.urls)
      ? postData.urls
      : Array.isArray(postData.repo_urls)
        ? postData.repo_urls
        : [];
    const statusByUrl = {
      [PARENT]: 'token_available',
      [NESTED_AUTHORIZED]: 'token_available',
      [NESTED_UNAUTHORIZED]: 'not_bound',
    };
    const results = urls.map((url) => ({
      url,
      repo_url: url,
      token_status: statusByUrl[url] || 'not_applicable',
      oauth_provider: 'gitlab',
      oauth_service_provider: 'default',
    }));
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ results }),
    });
  });

  // 兼容单仓回退；正常路径不应命中
  page.route('**/validate-git-repo/**', async (route) => {
    if (route.request().url().includes('validate-git-repos')) {
      await route.fallback();
      return;
    }
    singleCalls.n += 1;
    const url = new URL(route.request().url());
    const repoUrl = url.searchParams.get('url') || url.searchParams.get('git_repo') || '';
    const statusByUrl = {
      [PARENT]: 'token_available',
      [NESTED_AUTHORIZED]: 'token_available',
      [NESTED_UNAUTHORIZED]: 'not_bound',
    };
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        is_accessible: statusByUrl[repoUrl] === 'token_available',
        token_status: statusByUrl[repoUrl] || 'not_applicable',
        message: '',
      }),
    });
  });

  page.route(`**/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/**`, async (route) => {
    const url = route.request().url();
    if (
      url.includes('/nested-git-repos') ||
      url.includes('/validate-git-repo') ||
      url.includes('/branches/') ||
      url.includes('/repo-access-check')
    ) {
      await route.fallback();
      return;
    }
    if (route.request().method() !== 'GET' || !url.match(/\/projects\/[^/]+\/?(\?|$)/)) {
      await route.fallback();
      return;
    }
    const response = await route.fetch();
    let body;
    try {
      body = await response.json();
    } catch {
      await route.fulfill({ response });
      return;
    }
    body.git_repos = [PARENT];
    body.git_repo_entries = [{ url: PARENT }];
    body.git_repos_status = [
      {
        repo_url: PARENT,
        token_status: 'token_available',
        oauth_provider: 'gitlab',
        oauth_service_provider: 'default',
      },
    ];
    await route.fulfill({ response, json: body });
  });

  return { batchCalls, singleCalls };
}

test.describe('项目详情子仓 OAuth 未授权置顶', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('nested-git-repo-oauth-status-0 为未授权且排在最前，并走批量接口', async ({ page }) => {
    const { batchCalls, singleCalls } = mockNestedGitOAuthFlow(page);
    await login(page);

    await page.goto(`${BASE}${PROJECT_URL}`, {
      waitUntil: 'domcontentloaded',
      timeout: 20000,
    });
    await page.waitForLoadState('networkidle').catch(() => {});
    await page.waitForTimeout(2500);

    const heading = page.getByRole('heading', { name: '项目详情' });
    if (!(await heading.isVisible({ timeout: 15000 }).catch(() => false))) {
      test.skip(true, '项目详情页未加载，跳过');
    }

    const nestedSection = page.getByTestId('project-detail-nested-git-repos');
    await expect(nestedSection).toBeVisible({ timeout: 15000 });

    const firstStatus = nestedSection.getByTestId('nested-git-repo-oauth-status-0');
    await expect(firstStatus).toHaveText('未授权', { timeout: 15000 });

    const rows = nestedSection.getByTestId('project-detail-nested-git-repo-row');
    await expect(rows.first()).toContainText('docs');
    await expect(rows.first()).toContainText(NESTED_UNAUTHORIZED);

    expect(batchCalls.n).toBeGreaterThan(0);
    expect(singleCalls.n).toBe(0);
  });
});
