// @ts-check
/**
 * 回归：派生任务（Fork）时，先确认是否自动运行，再预开 about:blank → 成功后导航到任务详情 URL。
 * 断言新标签页以 task-detail URL 打开，且无"浏览器拦截了新标签页"提示。
 *
 * 对应：taskDetailEditing.js forkTask → openTaskDetailInNewTab → window.open
 *       TaskDetailPageHeader.vue → ForkAutoRunConfirmModal
 *
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SOURCE_TASK_ID = 'task_fork_source_mock';
const FORKED_TASK_ID = 'task_forked_result_mock';
const PROGRESS_FIRST_COL_ID = '1000000000000000101';
const PROGRESS_WIP_COL_ID = '1000000000000000102';
const LINKED_PROJECT_ID = 'proj_fork_git_identity';
const LINKED_REPO_URL = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad';
const CURRENT_USER_GIT_IDENTITY_ID = 'gid-current-user';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${SOURCE_TASK_ID}/?accessCode=u824976301710503936`,
);

/**
 * mockForkApi 注册 fork 流程所需的全部 mock 路由。
 * @param {import('@playwright/test').Page} page
 * @param {{
 *   forkPosts: any[],
 *   forkPostHeaders?: string[],
 *   clientIp?: string,
 *   linkedRepo?: boolean,
 *   gitIdentities?: object[],
 *   oauthConnected?: boolean,
 * }} ctx clientIp 非空时 mock /api/accounts/users/client-ip/ 返回该 IP
 */
async function mockForkApi(page, {
  forkPosts,
  forkPostHeaders,
  clientIp,
  linkedRepo = false,
  gitIdentities = [],
  oauthConnected = true,
}) {
  await page.addInitScript(() => {
    try {
      localStorage.setItem('currentUserId', 'e2e-user');
    } catch {
      /* ignore quota / private mode */
    }
    // @ts-ignore
    window.open = (url) => {
      // @ts-ignore
      window.__lastForkOpenUrl = url;
      return { closed: false, location: { href: url } };
    };
  });

  await page.context().addCookies([
    { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
  ]);

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 源任务详情
    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${SOURCE_TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: SOURCE_TASK_ID,
          title: 'Fork Source Task',
          description: '',
          created_at: '2026-07-21T00:00:00Z',
          created_by: { username: 'mock' },
          comments: [],
          ai_comments: [],
          assignees: ['827923618451263488'],
          workspace_id: WORKSPACE_ID,
          feature_params_source: 'company',
          owner: '827923618451263488',
          operator: '',
          priority: 1,
          progress_column_id: PROGRESS_WIP_COL_ID,
          projects: linkedRepo
            ? [{ project_id: LINKED_PROJECT_ID, repo_index: 0, base_branch: 'main' }]
            : [],
        }),
      });
      return;
    }

    if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(linkedRepo
          ? [{ id: LINKED_PROJECT_ID, name: 'somanyad', git_repos: [LINKED_REPO_URL] }]
          : []),
      });
      return;
    }

    if (url.includes('/api/git-identities/user/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ identities: gitIdentities }),
      });
      return;
    }

    if (url.includes('/api/git-oauth/user-app-connection/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ connected: oauthConnected }),
      });
      return;
    }

    // Fork POST — 返回新任务 ID
    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}`) && method === 'POST') {
      let body = {};
      try {
        body = req.postDataJSON() || {};
      } catch {
        body = {};
      }
      forkPosts.push(body);
      if (Array.isArray(forkPostHeaders)) {
        const headers = req.headers();
        forkPostHeaders.push(headers['idempotency-key'] || headers['Idempotency-Key'] || '');
      }
      const seq = forkPosts.length;
      const createdId = seq <= 1 ? FORKED_TASK_ID : `${FORKED_TASK_ID}_${seq}`;
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ id: createdId }),
      });
      return;
    }

    // 公网客户端 IP（自动运行路径，OPT-20260810-046）
    if (url.includes('/api/accounts/users/client-ip/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ip: clientIp || '' }),
      });
      return;
    }

    // workspace-collaborators
    if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    // progress-system（按 order 第一列为待处理）
    if (url.includes('/progress-system/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          columns: [
            { id: PROGRESS_FIRST_COL_ID, name: '待处理', order_num: 0 },
            { id: PROGRESS_WIP_COL_ID, name: '进行中', order_num: 1 },
          ],
        }),
      });
      return;
    }

    // 容器 context
    if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
      });
      return;
    }

    // installed-images
    if (url.includes('/installed-images/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    if (url.includes('/personal/feature-params-configs/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ configs: [] }),
      });
      return;
    }

    if (url.includes('/feature-params/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            providers: [{ provider: 'openai', supported_models: ['gpt-4.1', 'gpt-4.1-mini'] }],
            agent_model: 'gpt-4.1',
            agent_model_provider: 'openai',
          },
        }),
      });
      return;
    }

    // 兜底
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

/** 断言已打开的新标签页 URL 且无浏览器拦截提示 */
async function assertOpenedTaskDetailUrl(page, confirmModal) {
  const openedUrl = await page.evaluate(() => {
    // @ts-ignore
    return window.__lastForkOpenUrl || '';
  });
  expect(openedUrl).toContain(FORKED_TASK_ID);
  expect(openedUrl).toContain('task-detail');

  const blockMsg = page.getByText(/浏览器拦截了新标签页/);
  await expect(blockMsg).toHaveCount(0);
  await expect(confirmModal).toHaveCount(0);
}

test.describe('TaskDetail Fork 弹出新标签页', () => {
  test('不自动运行：Fork 确认后新标签页打开 task-detail URL，无浏览器拦截提示', async ({ page }) => {
    test.setTimeout(120000);

    /** @type {any[]} */
    const forkPosts = [];
    await mockForkApi(page, { forkPosts });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // 点击 Fork → 确认模态（此时不应已 POST）
    const forkBtn = page.locator('#task-fork-btn');
    await expect(forkBtn).toBeVisible({ timeout: 15000 });
    await forkBtn.click();
    const confirmModal = page.getByTestId('fork-auto-run-confirm-modal');
    await expect(confirmModal).toBeVisible({ timeout: 5000 });
    expect(forkPosts).toHaveLength(0);

    await page.getByTestId('fork-mode-fork-only').check();
    await page.getByTestId('fork-confirm-submit').click();

    // 等待 API 响应 + window.open 被调用
    await page.waitForTimeout(2000);

    expect(forkPosts).toHaveLength(1);
    expect(forkPosts[0].fork_from).toBe(SOURCE_TASK_ID);
    expect(forkPosts[0].auto_run).toBe(false);
    expect(forkPosts[0].client_public_ip).toBeUndefined();
    expect(forkPosts[0].agent_models).toBeUndefined();
    expect(forkPosts[0].fork_count).toBeUndefined();
    expect(forkPosts[0].progress_column_id).toBe(PROGRESS_FIRST_COL_ID);

    await assertOpenedTaskDetailUrl(page, confirmModal);
  });

  test('自动运行并派生：mock 公网 IP，断言 POST auto_run=true 且携带 client_public_ip', async ({ page }) => {
    test.setTimeout(120000);

    const PUBLIC_IP = '203.0.113.42';
    /** @type {any[]} */
    const forkPosts = [];
    await mockForkApi(page, { forkPosts, clientIp: PUBLIC_IP });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const forkBtn = page.locator('#task-fork-btn');
    await expect(forkBtn).toBeVisible({ timeout: 15000 });
    await forkBtn.click();
    const confirmModal = page.getByTestId('fork-auto-run-confirm-modal');
    await expect(confirmModal).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('fork-auto-run-git-identity-section')).toHaveCount(0);
    expect(forkPosts).toHaveLength(0);

    await page.getByTestId('fork-mode-auto-run').check();
    await expect(page.getByTestId('fork-agent-copy-section')).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('fork-agent-model-gpt-4.1')).toBeVisible({ timeout: 10000 });
    await page.getByTestId('fork-confirm-submit').click();

    // 等待 client-ip 查询 + API 响应 + window.open
    await page.waitForTimeout(2500);

    expect(forkPosts).toHaveLength(1);
    expect(forkPosts[0].fork_from).toBe(SOURCE_TASK_ID);
    expect(forkPosts[0].auto_run).toBe(true);
    expect(forkPosts[0].client_public_ip).toBe(PUBLIC_IP);
    expect(forkPosts[0].fork_count).toBeUndefined();
    expect(forkPosts[0].agent_models).toEqual([{ provider: 'openai', model: 'gpt-4.1' }]);
    expect(forkPosts[0].progress_column_id).toBe(PROGRESS_FIRST_COL_ID);

    await assertOpenedTaskDetailUrl(page, confirmModal);
  });

  test('T6 关联仓未选身份：展示当前用户身份与 OAuth，自动运行禁用', async ({ page }) => {
    test.setTimeout(120000);
    /** @type {any[]} */
    const forkPosts = [];
    await mockForkApi(page, {
      forkPosts,
      linkedRepo: true,
      oauthConnected: true,
      gitIdentities: [{
        id: CURRENT_USER_GIT_IDENTITY_ID,
        git_user_name: 'current-user',
        git_user_email: 'current@example.com',
        is_default: false,
      }],
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    await expect(page.locator('#task-fork-btn')).toBeVisible({ timeout: 15000 });
    await page.locator('#task-fork-btn').click();
    const confirmModal = page.getByTestId('fork-auto-run-confirm-modal');
    await expect(confirmModal).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('fork-auto-run-git-identity-section')).toHaveCount(0);
    await page.getByTestId('fork-mode-auto-run').check();
    await expect(page.getByTestId('fork-auto-run-git-identity-section')).toBeVisible();
    await expect(page.getByTestId('create-task-repo-git-identity-select')).toBeVisible();
    await expect(page.getByTestId('fork-auto-run-oauth-bound')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('fork-auto-run-git-identity-blocked-reason')).toContainText('Git 提交身份');
    await expect(page.getByTestId('fork-confirm-submit')).toBeDisabled();
    await page.getByTestId('fork-mode-fork-only').check();
    await expect(page.getByTestId('fork-confirm-submit')).toBeEnabled();
    expect(forkPosts).toHaveLength(0);
  });

  test('T7 选当前用户身份后自动运行：POST repo_identities 不含他人身份', async ({ page }) => {
    test.setTimeout(120000);
    const PUBLIC_IP = '203.0.113.42';
    /** @type {any[]} */
    const forkPosts = [];
    await mockForkApi(page, {
      forkPosts,
      clientIp: PUBLIC_IP,
      linkedRepo: true,
      oauthConnected: true,
      gitIdentities: [{
        id: CURRENT_USER_GIT_IDENTITY_ID,
        git_user_name: 'current-user',
        git_user_email: 'current@example.com',
        is_default: false,
      }],
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    await expect(page.locator('#task-fork-btn')).toBeVisible({ timeout: 15000 });
    await page.locator('#task-fork-btn').click();
    const confirmModal = page.getByTestId('fork-auto-run-confirm-modal');
    await expect(confirmModal).toBeVisible({ timeout: 5000 });
    await page.getByTestId('fork-mode-auto-run').check();
    await expect(page.getByTestId('fork-auto-run-oauth-bound')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('fork-agent-model-gpt-4.1')).toBeVisible({ timeout: 10000 });
    await page.getByTestId('create-task-repo-git-identity-select').selectOption(CURRENT_USER_GIT_IDENTITY_ID);
    await expect(page.getByTestId('fork-confirm-submit')).toBeEnabled();
    await page.getByTestId('fork-confirm-submit').click();
    await page.waitForTimeout(2500);

    expect(forkPosts).toHaveLength(1);
    expect(forkPosts[0].fork_from).toBe(SOURCE_TASK_ID);
    expect(forkPosts[0].auto_run).toBe(true);
    expect(forkPosts[0].repo_identities).toEqual([{
      repo_url: LINKED_REPO_URL,
      git_identity_id: CURRENT_USER_GIT_IDENTITY_ID,
      oauth_gitsite: 'gitlab-tencent-sh-1.daydaymoney.com',
    }]);
    expect(JSON.stringify(forkPosts[0])).not.toMatch(/gid-source|gid-other|gid-owner/);

    await assertOpenedTaskDetailUrl(page, confirmModal);
  });

  test('T15 勾选两个模型：两次 POST 不同 agent_models，Idempotency-Key 同前缀', async ({ page }) => {
    test.setTimeout(120000);
    const PUBLIC_IP = '203.0.113.42';
    /** @type {any[]} */
    const forkPosts = [];
    /** @type {string[]} */
    const forkPostHeaders = [];
    await mockForkApi(page, { forkPosts, forkPostHeaders, clientIp: PUBLIC_IP });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const forkBtn = page.locator('#task-fork-btn');
    await expect(forkBtn).toBeVisible({ timeout: 15000 });
    await forkBtn.click();
    const confirmModal = page.getByTestId('fork-auto-run-confirm-modal');
    await expect(confirmModal).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('fork-copy-count-input')).toHaveCount(0);
    await page.getByTestId('fork-mode-auto-run').check();
    await expect(page.getByTestId('fork-agent-model-gpt-4.1')).toBeVisible({ timeout: 10000 });
    await page.getByTestId('fork-agent-model-gpt-4.1-mini').check();
    await page.getByTestId('fork-confirm-submit').click();
    await page.waitForTimeout(3000);

    expect(forkPosts).toHaveLength(2);
    expect(forkPosts.every((body) => body.fork_from === SOURCE_TASK_ID && body.auto_run === true)).toBe(true);
    expect(forkPosts.every((body) => body.fork_count === undefined)).toBe(true);
    const models = forkPosts.map((body) => body.agent_models?.[0]?.model).sort();
    expect(models).toEqual(['gpt-4.1', 'gpt-4.1-mini']);
    expect(forkPostHeaders).toHaveLength(2);
    const prefixes = forkPostHeaders.map((key) => String(key).split(':')[0]);
    const suffixes = forkPostHeaders.map((key) => String(key).split(':')[1]);
    expect(new Set(prefixes).size).toBe(1);
    expect(prefixes[0].length).toBeGreaterThan(8);
    expect(suffixes.sort()).toEqual(['1', '2']);

    await assertOpenedTaskDetailUrl(page, confirmModal);
  });
});
