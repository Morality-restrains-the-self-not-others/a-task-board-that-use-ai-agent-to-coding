// @ts-check
/**
 * 回归：评论执行细节摘要回显 Git OAuth（OPT-20260822-029，测试意图 T17）。
 * - 任务关联 GitLab 仓且评论带 repo_identities 时，执行细节 summary 同一行
 *   显示 `comment-execution-git-oauth` 与 `comment-execution-git-identity`
 * - 连接接口返回未绑定时：`comment-execution-git-oauth` data-kind=unbound、
 *   文案含「未绑定」，且 `comment-execution-git-oauth-bind` 为真实
 *   `<a href>`（含 start-from-gateway 与 repo_url）
 *
 * 纯 mock，无真实账号依赖；参考 TaskDetail.comment-execution-details-dom 与
 * TaskDetail.fork-popup（user-app-connection 路由）。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_comment_git_oauth_mock';
const COMMENT_ID = 'comment-oauth-001';
const LINKED_PROJECT_ID = 'proj_comment_git_oauth';
const LINKED_REPO_URL = 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad';
const GIT_IDENTITY_ID = 'gid-comment-001';
const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`;

test.describe('TaskDetail 评论执行细节 Git OAuth 摘要（T17）', () => {
  test('未绑定时 summary 展示「未绑定」且含真实 bind href', async ({ page }) => {
    test.setTimeout(120000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'sessionid', value: 'e2e-session-comment-git-oauth', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // 任务详情：一条带 repo_identities 的评论
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Comment Git OAuth Summary',
            description: 'Testing execution git-oauth summary',
            created_at: '2026-07-22T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [
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
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [{ project_id: LINKED_PROJECT_ID, repo_index: 0, base_branch: 'main' }],
          }),
        });
        return;
      }

      // 人类评论 feed（任务详情仅返回任务，评论区单独拉取）
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

      // AI 评论 feed（空）
      if (url.includes('/api/ai-comment/task-detail/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ results: [] }) });
        return;
      }

      // 关联项目详情：git_repos 含 GitLab 仓
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

      // Git OAuth 连接状态：未绑定
      if (url.includes('/api/git-oauth/user-app-connection/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ connected: false }),
        });
        return;
      }

      // container-task-ui-context
      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            container_endpoint_registered: true,
            container_page_url: 'http://127.0.0.1:18080/ui/dev-local-token',
          }),
        });
        return;
      }

      // container-layer-graph
      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layers_root: '/workspace/layers', bootstrap_layer_id: '', layers: [], jobs: [] }),
        });
        return;
      }

      // comment-container-bindings GET/POST
      if (url.includes('comment-container-bindings') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bindings: [] }) });
        return;
      }
      if (url.includes('comment-container-bindings') && method === 'POST') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
        return;
      }

      // 评论 PATCH
      if (url.includes('/comments/') && method === 'PATCH') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
        return;
      }

      // workspace-collaborators
      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      // progress-system
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
        return;
      }

      // 容器日志
      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ layer_id: '', text: '' }) });
        return;
      }

      // workspace-runtime-indicators / workspace-machine-summary
      if (url.includes('workspace-runtime-indicators') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', indicators: [] }) });
        return;
      }
      if (url.includes('workspace-machine-summary') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }) });
        return;
      }

      // 兜底
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2500);

    // 评论正文可见
    await expect(page.getByText('帮我改一下这个仓库')).toBeVisible({ timeout: 15000 });

    // 等待执行细节渲染
    const execDetails = page.getByTestId('comment-execution-details');
    await expect(execDetails).toBeVisible({ timeout: 15000 });

    // Git OAuth 芯片：未绑定（等待连接检查完成，loading → unbound）
    const oauthChip = page.locator('[data-testid="comment-execution-git-oauth"]');
    await expect(oauthChip).toBeVisible({ timeout: 20000 });
    await expect(oauthChip).toHaveAttribute('data-kind', 'unbound', { timeout: 20000 });
    await expect(oauthChip).toContainText('未绑定');

    // Git 身份与 OAuth 同一 summary 行
    const identityBadge = page.getByTestId('comment-execution-git-identity');
    await expect(identityBadge).toBeVisible({ timeout: 10000 });

    // 未绑定 bind href 为真实链接：含 start-from-gateway 与 repo_url
    const bind = page.locator('[data-testid="comment-execution-git-oauth-bind"]');
    await expect(bind).toBeVisible({ timeout: 10000 });
    const href = await bind.getAttribute('href');
    expect(String(href || '')).toContain('start-from-gateway');
    expect(String(href || '')).toContain(encodeURIComponent(LINKED_REPO_URL));
  });

  test('已绑定 + 层快照 Permission denied 时 data-kind=bound_no_write', async ({ page }) => {
    test.setTimeout(120000);
    const githubUrl = 'https://github.com/ruandao/helloworld.git';
    const pushError = 'remote: Permission to ruandao/helloworld.git denied to alice.';
    const taskId = 'task_comment_git_oauth_nowrite';
    const commentId = 'comment-oauth-nowrite-001';
    const projectId = 'proj_comment_git_oauth_nowrite';
    const path = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${taskId}/?accessCode=u824976301710503936`;

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://127.0.0.1:4000/' },
      { name: 'sessionid', value: 'e2e-session-comment-git-oauth-nw', url: 'http://127.0.0.1:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://127.0.0.1:4000/' },
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'sessionid', value: 'e2e-session-comment-git-oauth-nw', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${taskId}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: taskId,
            title: 'Comment Git OAuth no-write',
            description: 'bound_no_write',
            created_at: '2026-07-22T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [{ project_id: projectId, repo_index: 0, base_branch: 'main' }],
          }),
        });
        return;
      }

      if (url.includes(`/api/tasks/${taskId}/comments/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            results: [
              {
                id: commentId,
                content: '帮我改一下这个仓库',
                created_at: '2026-07-22T10:00:00Z',
                created_by: { username: '测试用户' },
                repo_identities: [{ repo_url: githubUrl, git_identity_id: 'gid-nw-1' }],
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
          body: JSON.stringify([{ id: projectId, name: 'helloworld', git_repos: [githubUrl] }]),
        });
        return;
      }

      if (url.includes('/api/git-oauth/user-app-connection/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            connected: true,
            bind_status: 'active',
            connections: [{ connected: true, provider: 'github', service_provider: 'default' }],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            container_endpoint_registered: true,
            container_page_url: 'http://127.0.0.1:18080/ui/dev-local-token',
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: 'layer-nw-1',
            layers: [
              {
                layer_id: 'layer-nw-1',
                created_at: '2026-07-22T10:01:00Z',
                git_remote: { last_push_error: pushError, last_push_error_trace_id: 'tid-perm-nw' },
              },
            ],
            jobs: [],
          }),
        });
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
      if (url.includes('workspace-runtime-indicators') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', indicators: [] }) });
        return;
      }
      if (url.includes('workspace-machine-summary') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(path);
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText('帮我改一下这个仓库')).toBeVisible({ timeout: 15000 });
    const oauthChip = page.locator('[data-testid="comment-execution-git-oauth"]');
    await expect(oauthChip).toBeVisible({ timeout: 20000 });
    await expect(oauthChip).toHaveAttribute('data-kind', 'bound_no_write', { timeout: 20000 });
    await expect(oauthChip).toContainText('无写权限');
    const bind = page.locator('[data-testid="comment-execution-git-oauth-bind"]');
    await expect(bind).toBeVisible({ timeout: 10000 });
    const href = await bind.getAttribute('href');
    expect(String(href || '')).toContain('github-start-from-gateway');
    expect(String(href || '')).toContain(encodeURIComponent(githubUrl));
  });
});
