// @ts-check
/**
 * relayToTrae：OAuth 未绑定时「启动」按钮应 disabled。
 *
 * - **mock** 套件：route 拦截，无需登录，验证 UI 阻断逻辑（默认运行）
 * - **integration** 套件：真实登录 + 远端/本地站点（需 PLAYWRIGHT_INTEGRATION=1）
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '860371538948571136';

const REPO_URL = 'https://github.com/mock-org/mock-relay-oauth.git';
const MOCK_USER_ID = '827923618451263488';

const SITE_ORIGIN = String(process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const RELAY_TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true`;
const INTEGRATION_TASK_PATH = `/tenant/${process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496'}/workspace/${process.env.PLAYWRIGHT_WORKSPACE_ID || '857903329669984256'}/task-detail/${process.env.PLAYWRIGHT_RELAY_TASK_ID || '860371538948571136'}/?relayToTrae=true`;

async function installRelayOAuthMockRoutes(page, { oauthConnected = false } = {}) {
  await page.context().addCookies([
    { name: 'userId', value: MOCK_USER_ID, url: 'http://127.0.0.1:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://127.0.0.1:4000/' },
  ]);

  await page.route('**/*', async (route) => {
    const req = route.request();
    const url = req.url();
    if (!url.includes('/api/')) {
      await route.continue();
      return;
    }
    const method = req.method();

    if (url.includes('/api/git-oauth/providers/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          providers: [
            {
              provider: 'github',
              service_provider: 'default',
              website: 'https://github.com/',
            },
          ],
        }),
      });
      return;
    }

    if (url.includes(`/api/user/${MOCK_USER_ID}/accounts/github/app/connection`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          connected: oauthConnected,
          connections: oauthConnected ? [{ connected: true, github_user_id: 1001 }] : [],
        }),
      });
      return;
    }

    if (url.includes('/cloud/compute/github-credential-status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          github_connections: [],
          repo_bindings: [],
        }),
      });
      return;
    }

    if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET' && url.includes('workspace_id=')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'proj-mock-relay',
            name: 'Mock Relay 项目',
            git_repos: [REPO_URL],
          },
        ]),
      });
      return;
    }

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'Relay OAuth Mock Task',
          description: 'Mock',
          created_at: '2026-01-01T00:00:00Z',
          created_by: { username: 'mock-user' },
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
          container_image_id: 'img-relay-oauth-1',
          projects: [{ project_id: 'proj-mock-relay', repo_index: 0, base_branch: 'main' }],
          parameters: { repo_clone_git_identities: {} },
        }),
      });
      return;
    }

    if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'img-relay-oauth-1', name: 'Relay Image', version: '1.0' }]),
      });
      return;
    }

    if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          env: {
            TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
            BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
          },
        }),
      });
      return;
    }

    if (url.includes('/relay-to-trae/health/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'ok' }) });
      return;
    }

    if (url.includes('/relay-to-trae/register/') && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'ok' }) });
      return;
    }

    if (url.includes('/relay-to-trae/status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          running: false,
          port_listening: false,
          online_service_up: false,
          logs: [],
        }),
      });
      return;
    }

    if (url.includes('/relay-to-trae/token-init/') && method === 'POST') {
      await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ detail: 'should-not-call' }) });
      return;
    }

    if (url.includes('/server-runtime-status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
      });
      return;
    }

    if (url.includes('/server-content/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
      });
      return;
    }

    if (url.includes('/cloud/compute/server-start-history/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', records: [] }),
      });
      return;
    }

    if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
      return;
    }

    if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
      });
      return;
    }

    if (url.includes('/api/user/') && url.includes('/profile/git-identities/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ identities: [] }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

async function openRelayDirectPanel(page, taskPath = RELAY_TASK_PATH) {
  await page.goto(taskPath);
  await page.waitForLoadState('domcontentloaded');

  await expect(page.getByText('关联项目', { exact: true })).toBeVisible({ timeout: 30_000 });

  const relayTab = page.getByTestId('server-config-relay-direct-tab');
  await expect(relayTab).toBeVisible({ timeout: 30_000 });
  await relayTab.click();

  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  await expect(startBtn).toBeVisible({ timeout: 30_000 });
  return startBtn;
}

async function waitForOAuthBindStateSettled(page) {
  const oauthBindBtns = page.getByTestId('comment-repo-oauth-bind-btn').or(page.getByTestId('task-repo-oauth-bind-btn'));
  await expect
    .poll(
      async () => {
        const bindCount = await oauthBindBtns.count();
        const detectingCount = await page.getByRole('button', { name: '检测中...' }).count();
        return bindCount > 0 || detectingCount === 0;
      },
      { timeout: 45_000, message: '等待 OAuth 绑定状态检测完成' },
    )
    .toBe(true);
  return oauthBindBtns;
}

test.describe('TaskDetail relayToTrae OAuth 启动阻断（mock）', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑 Playwright mock');

  test('OAuth 未绑定时：显示 OAuth 绑定按钮且启动 disabled', async ({ page }) => {
    test.setTimeout(120_000);
    await installRelayOAuthMockRoutes(page, { oauthConnected: false });

    const startBtn = await openRelayDirectPanel(page);
    await waitForOAuthBindStateSettled(page);

    await expect(startBtn).toBeDisabled();
    await expect(page.getByTestId('relay-to-trae-oauth-unbound-guide')).toBeVisible();
    await expect(page.getByTestId('relay-to-trae-oauth-unbound-guide')).toContainText('创建或编辑任务');
  });

  test('OAuth 已绑定时：不显示 OAuth 绑定按钮且启动可点（镜像已选）', async ({ page }) => {
    test.setTimeout(120_000);
    await installRelayOAuthMockRoutes(page, { oauthConnected: true });

    const startBtn = await openRelayDirectPanel(page);
    await waitForOAuthBindStateSettled(page);

    await expect(page.getByTestId('comment-repo-oauth-bind-btn')).toHaveCount(0);
    await expect(page.getByTestId('task-repo-oauth-bind-btn')).toHaveCount(0);
    await expect(startBtn).toBeEnabled();
    await expect(page.getByTestId('relay-to-trae-oauth-unbound-guide')).toHaveCount(0);
  });
});

test.describe('TaskDetail relayToTrae OAuth 启动阻断（integration）', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑全栈集成');
  test.skip(() => process.env.PLAYWRIGHT_INTEGRATION !== '1', '需 PLAYWRIGHT_INTEGRATION=1');

  test('存在 OAuth 绑定按钮时启动按钮应 disabled 且展示引导', async ({ page }) => {
    test.setTimeout(180_000);

    const email = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
    const password = process.env.PLAYWRIGHT_TEST_PASSWORD ;

    await playwrightLoginWithLegalAccept(page, { email, password, baseURL: SITE_ORIGIN });
    await page.goto(`${SITE_ORIGIN}${INTEGRATION_TASK_PATH}`);
    await page.waitForLoadState('domcontentloaded');

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await expect(page.getByText('关联项目', { exact: true })).toBeVisible({ timeout: 60_000 });
    await page.getByTestId('server-config-relay-direct-tab').click();
    await expect(startBtn).toBeVisible({ timeout: 30_000 });

    const oauthBindBtns = await waitForOAuthBindStateSettled(page);
    const bindCount = await oauthBindBtns.count();
    test.skip(bindCount === 0, '当前任务所有仓库 OAuth 已绑定，跳过未绑定场景');

    await expect(oauthBindBtns.first()).toBeVisible();
    await expect(startBtn).toBeDisabled();
    await expect(page.getByTestId('relay-to-trae-oauth-unbound-guide')).toBeVisible();
  });
});
