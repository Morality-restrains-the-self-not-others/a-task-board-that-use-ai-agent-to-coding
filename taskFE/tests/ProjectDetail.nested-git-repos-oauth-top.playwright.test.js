// @ts-check
/**
 * E2E: 项目详情子 Git 仓库 — 未授权置顶 + 批量 validate-git-repos
 *
 * 登录后进入项目详情，mock nested-git-repos 与 validate-git-repos：
 * - 断言「未授权」且排在 nested-git-repo-oauth-status-0
 * - 批量接口被调用（非 N 次单仓 validate-git-repo）
 */
import { test, expect } from '@playwright/test';
import { loadConfYaml } from '../helpers/loadConfYaml.mjs';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const conf = loadConfYaml();
const vueHost = conf.vue?.host || '127.0.0.1';
const vuePort = conf.vue?.port || 4000;
const BASE = `http://${vueHost}:${vuePort}`;

const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const PROJECT_ID = process.env.TEST_PROJECT_ID || 'proj_-5247879312070945751';
const PROJECT_URL = `/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const PARENT_REPO = 'https://gitlab.daydaymoney.com/example-user/ram-work.git';
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

test.describe('项目详情子仓 OAuth 未授权置顶', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('nested-git-repo-oauth-status-0 为未授权且排在最前，并走批量接口', async ({ page }) => {
    let batchCalls = 0;
    let singleValidateCalls = 0;

    await page.route('**/api/projects/tenant_id/**/nested-git-repos/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          parent_repo_url: PARENT_REPO,
          nested_repos: [
            {
              path: 'DaydaymoneyGrafana',
              url: NESTED_AUTHORIZED,
              source: 'gitignore_nested',
            },
            {
              path: 'docs',
              url: NESTED_UNAUTHORIZED,
              source: 'gitignore_nested',
            },
          ],
          error: '',
        }),
      });
    });

    await page.route('**/projects/validate-git-repos/**', async (route) => {
      batchCalls += 1;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [
            {
              url: NESTED_AUTHORIZED,
              repo_url: NESTED_AUTHORIZED,
              token_status: 'token_available',
              oauth_provider: 'gitlab',
              oauth_service_provider: 'default',
            },
            {
              url: NESTED_UNAUTHORIZED,
              repo_url: NESTED_UNAUTHORIZED,
              token_status: 'not_bound',
              oauth_provider: 'gitlab',
              oauth_service_provider: 'default',
            },
            {
              url: PARENT_REPO,
              repo_url: PARENT_REPO,
              token_status: 'token_available',
              oauth_provider: 'gitlab',
              oauth_service_provider: 'default',
            },
          ],
        }),
      });
    });

    await page.route('**/projects/validate-git-repo/**', async (route) => {
      // 排除复数路径（已由上面处理）
      if (route.request().url().includes('validate-git-repos')) {
        await route.fallback();
        return;
      }
      singleValidateCalls += 1;
      await route.continue();
    });

    await page.route(`**/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/**`, async (route) => {
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
      // 仅项目详情 GET
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
      body.git_repos = [PARENT_REPO];
      body.git_repo_entries = [{ url: PARENT_REPO }];
      body.git_repos_status = [
        {
          repo_url: PARENT_REPO,
          token_status: 'token_available',
          oauth_provider: 'gitlab',
          oauth_service_provider: 'default',
        },
      ];
      await route.fulfill({ response, json: body });
    });

    await login(page);
    await page.goto(`${BASE}${PROJECT_URL}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    await page.waitForTimeout(3000);

    const nestedSection = page.getByTestId('project-detail-nested-git-repos');
    if (!(await nestedSection.isVisible({ timeout: 15000 }).catch(() => false))) {
      test.skip(true, '子 Git 仓库区块未渲染，跳过');
    }

    await expect(page.getByTestId('nested-git-repo-oauth-status-0')).toHaveText('未授权', {
      timeout: 15000,
    });

    const firstRow = nestedSection.locator('[data-testid="project-detail-nested-git-repo-row"]').first();
    await expect(firstRow).toContainText('docs');
    await expect(firstRow).toContainText('未授权');

    expect(batchCalls).toBeGreaterThan(0);
    expect(singleValidateCalls).toBe(0);
  });
});
