// @ts-check
/**
 * 回归：评论区对已绑定仓只做一次 AccessToken 探测（OPT-20260822-032，测试意图 T18）。
 * T18c（OPT-20260829-024）：probe `network_status=unreachable` → data-kind=unreachable；
 * 内网 skipped_intranet 保持 bound。
 *
 * 纯 mock，无真实账号依赖；结构参考 TaskDetail.comment-execution-git-oauth-summary。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { addE2eSession } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_comment_git_oauth_probe_mock';
const COMMENT_ID = 'comment-oauth-probe-001';
const LINKED_PROJECT_ID = 'proj_comment_git_oauth_probe';
const LINKED_REPO_URL = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad';
const GIT_IDENTITY_ID = 'gid-comment-probe-001';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`;

/**
 * @param {import('@playwright/test').Page} page
 * @param {object} probeBody
 */
async function installProbeRoutes(page, probeBody) {
  let probeCount = 0;
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'Comment Git OAuth Probe',
          description: 'Testing probe once',
          created_at: '2026-07-22T00:00:00Z',
          created_by: { username: 'mock' },
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
          projects: [{ project_id: LINKED_PROJECT_ID, repo_index: 0, base_branch: 'main' }],
        }),
      });
      return;
    }

    if (url.includes(`/api/tasks/${TASK_ID}/comments/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [
            {
              id: COMMENT_ID,
              content: '帮我改一下这个仓库',
              created_at: '2026-07-22T10:00:00Z',
              created_by: { username: '测试用户' },
              repo_identities: [
                { repo_url: LINKED_REPO_URL, git_identity_id: GIT_IDENTITY_ID },
              ],
            },
          ],
          next_cursor: null,
          has_more: false,
        }),
      });
      return;
    }

    if (url.includes('/api/ai-comment/task-detail/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ results: [] }) });
      return;
    }

    if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: LINKED_PROJECT_ID, name: 'somanyad', git_repos: [LINKED_REPO_URL] },
        ]),
      });
      return;
    }

    if (url.includes('/api/git-oauth/user-app-connection/') && method === 'GET') {
      const hasProbe = url.includes('probe_access_token=1');
      if (hasProbe) {
        probeCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(probeBody),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ connected: true }),
        });
      }
      return;
    }

    if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', container_endpoint_registered: true, container_page_url: 'http://127.0.0.1:18080/ui/dev-local-token' }),
      });
      return;
    }

    if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layers_root: '/workspace/layers', bootstrap_layer_id: '', layers: [], jobs: [] }) });
      return;
    }

    if (url.includes('comment-container-bindings') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bindings: [] }) });
      return;
    }
    if (url.includes('comment-container-bindings') && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
      return;
    }

    if (url.includes('/comments/') && method === 'PATCH') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
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

    if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layer_id: '', text: '' }) });
      return;
    }

    if (url.includes('workspace-runtime-indicators') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', indicators: [] }) });
      return;
    }
    if (url.includes('workspace-machine-summary') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }) });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
  return () => probeCount;
}

test.describe('TaskDetail 评论区 AccessToken 一次探测（T18）', () => {
  test('DB 已绑定 → 只探测一次 → 无效后摘要改未绑定', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-comment-probe' });
    const getProbeCount = await installProbeRoutes(page, { connected: false, access_token_valid: false });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await expect(page.getByText('帮我改一下这个仓库')).toBeVisible({ timeout: 15000 });
    const execDetails = page.getByTestId('comment-execution-details');
    await expect(execDetails).toBeVisible({ timeout: 15000 });

    const oauthChip = page.locator('[data-testid="comment-execution-git-oauth"]');
    await expect(oauthChip).toBeVisible({ timeout: 20000 });
    await expect(oauthChip).toHaveAttribute('data-kind', 'unbound', { timeout: 20000 });
    await expect(oauthChip).toContainText('未绑定');
    expect(getProbeCount()).toBe(1);

    const bind = page.locator('[data-testid="comment-execution-git-oauth-bind"]');
    await expect(bind).toBeVisible({ timeout: 10000 });
    const href = await bind.getAttribute('href');
    expect(String(href || '')).toContain('start-from-gateway');
    expect(String(href || '')).toContain(encodeURIComponent(LINKED_REPO_URL));
  });

  test('T18c probe network_status=unreachable → data-kind=unreachable 且无 bind href', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-comment-probe-unreach' });
    await installProbeRoutes(page, { connected: true, network_status: 'unreachable' });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText('帮我改一下这个仓库')).toBeVisible({ timeout: 15000 });

    const oauthChip = page.locator('[data-testid="comment-execution-git-oauth"]');
    await expect(oauthChip).toBeVisible({ timeout: 20000 });
    await expect(oauthChip).toHaveAttribute('data-kind', 'unreachable', { timeout: 20000 });
    await expect(oauthChip).toContainText('网络不可达');
    await expect(page.locator('[data-testid="comment-execution-git-oauth-bind"]')).toHaveCount(0);
  });

  test('T18c 内网 skipped_intranet 保持 data-kind=bound', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-comment-probe-intranet' });
    await installProbeRoutes(page, { connected: true, network_status: 'skipped_intranet' });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText('帮我改一下这个仓库')).toBeVisible({ timeout: 15000 });

    const oauthChip = page.locator('[data-testid="comment-execution-git-oauth"]');
    await expect(oauthChip).toBeVisible({ timeout: 20000 });
    await expect(oauthChip).toHaveAttribute('data-kind', 'bound', { timeout: 20000 });
    await expect(oauthChip).toContainText('已绑定');
  });
});
