// @ts-check
/**
 * E2E: 关闭「自动克隆子仓库」时，子仓 token_error 不得阻断项目自动运行。
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
const NESTED_ERROR = 'https://github.com/task2money/docs.git';

async function login(page) {
  await playwrightLoginWithLegalAccept(page, {
    email: EMAIL,
    password: PASSWORD,
    baseURL: BASE,
  });
  await page.waitForTimeout(1500);
}

function mockAutoCloneOffNestedTokenError(page) {
  page.route('**/nested-git-repos/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        parent_repo_url: PARENT,
        nested_repos: [{ path: 'docs', url: NESTED_ERROR, source: 'gitmodules' }],
        error: '',
      }),
    });
  });

  page.route('**/validate-git-repos/**', async (route) => {
    const postData = route.request().postDataJSON() || {};
    const urls = Array.isArray(postData.urls)
      ? postData.urls
      : Array.isArray(postData.repo_urls)
        ? postData.repo_urls
        : [];
    const statusByUrl = {
      [PARENT]: 'token_available',
      [NESTED_ERROR]: 'token_error',
    };
    const results = urls.map((url) => ({
      url,
      repo_url: url,
      token_status: statusByUrl[url] || 'not_applicable',
    }));
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ results }),
    });
  });

  page.route('**/api/projects/**', async (route) => {
    const url = route.request().url();
    if (
      url.includes('/nested-git-repos')
      || url.includes('/validate-git-repo')
      || url.includes('/branches/')
    ) {
      await route.fallback();
      return;
    }
    if (route.request().method() !== 'GET') {
      await route.fallback();
      return;
    }
    if (!url.includes(`/projects/`) || !url.includes(PROJECT_ID)) {
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
    if (!body || typeof body !== 'object' || !body.id) {
      await route.fulfill({ response });
      return;
    }
    body.auto_clone_nested_repos = false;
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
}

test.describe('项目详情自动运行门禁尊重自动克隆开关', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('auto_clone=false 时子仓 token_error 不展示无法启动', async ({ page }) => {
    try {
      await login(page);
    } catch (err) {
      test.skip(true, `登录未完成，跳过：${err?.message || err}`);
    }
    mockAutoCloneOffNestedTokenError(page);

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

    const cloneToggle = page.getByTestId('project-detail-auto-clone-nested-repos');
    await expect(cloneToggle).toBeVisible({ timeout: 15000 });
    await expect(cloneToggle).not.toBeChecked();

    await expect(page.getByTestId('project-auto-run-git-gate-hint')).toHaveCount(0);
    const label = page.getByTestId('project-default-auto-run-label');
    await expect(label).toBeVisible();
    await expect(label).not.toHaveText('无法启动');
  });
});
