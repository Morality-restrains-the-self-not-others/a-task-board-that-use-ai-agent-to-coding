// @ts-check
/**
 * 轻量：route mock 验证任务详情「仓库列表与克隆身份」面板加载项目仓库、
 * 选择身份后 PATCH repo-clone-git-identities。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

const REPO_URL = 'git@github.com:mock-org/mock-repo.git';
const GIT_IDENTITY_ID = 'git-ident-e2e-1';
const GITHUB_USER_ID = 999001;
const GITHUB_LOGIN = 'playwright-gh';

test.describe('TaskDetail 仓库克隆身份面板（mock）', () => {
  test('展示仓库行且选择身份后发出 PATCH', async ({ page }) => {
    /** @type {{ repo_clone_git_identities: Record<string, string> }} */
    let taskParameters = { repo_clone_git_identities: {} };
    /** @type {import('@playwright/test').Request[]} */
    const patchRequests = [];

    await page.addInitScript(({ tenantId, workspaceId, taskId, jobId }) => {
      class MockEventSource {
        static CONNECTING = 0;
        static OPEN = 1;
        static CLOSED = 2;

        constructor(url) {
          this.url = String(url || '');
          this.readyState = MockEventSource.OPEN;
          this.withCredentials = true;
          this.onmessage = null;
          this.onerror = null;
          this.onclose = null;
          this._listeners = { open: [] };

          setTimeout(() => {
            this._emitOpen();
            const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
            if (!this.url.includes(expected)) return;
            this._emitMessage({
              status: 'container_job_stream',
              job_id: jobId,
              phase: 'chunk',
              message: 'mock sse',
            });
          }, 60);
        }

        addEventListener(type, cb) {
          if (!this._listeners[type]) this._listeners[type] = [];
          this._listeners[type].push(cb);
        }

        close() {
          this.readyState = MockEventSource.CLOSED;
          if (typeof this.onclose === 'function') {
            this.onclose();
          }
        }

        _emitOpen() {
          const ev = { type: 'open' };
          for (const cb of this._listeners.open || []) {
            try {
              cb(ev);
            } catch (_) {}
          }
        }

        _emitMessage(payload) {
          if (typeof this.onmessage !== 'function') return;
          try {
            this.onmessage({ data: JSON.stringify(payload) });
          } catch (_) {}
        }
      }

      window.EventSource = MockEventSource;
    }, { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID, jobId: 'job-mock' });

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/task-detail/${TASK_ID}/mock-trae-online-log`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: 'mock snapshot\n', in_progress: false }),
        });
        return;
      }

      if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET' && url.includes('workspace_id=')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-mock-1',
              name: 'Mock 项目',
              git_repos: [REPO_URL],
            },
          ]),
        });
        return;
      }

      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            identities: [
              {
                id: GIT_IDENTITY_ID,
                display_name: 'E2E 身份',
                is_default: true,
              },
            ],
          }),
        });
        return;
      }

      if (
        url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/repo-clone-git-identities`) &&
        method === 'PATCH'
      ) {
        patchRequests.push(req);
        try {
          const body = req.postDataJSON();
          const incoming = body?.repo_clone_git_identities;
          if (incoming && typeof incoming === 'object') {
            taskParameters = {
              ...taskParameters,
              repo_clone_git_identities: {
                ...(taskParameters.repo_clone_git_identities || {}),
                ...incoming,
              },
            };
          }
        } catch (_) {
          /* ignore */
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ok: true,
            repo_clone_git_identities: taskParameters.repo_clone_git_identities || {},
          }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Task',
            description: 'Mock',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [
              { project_id: 'proj-mock-1', repo_index: 0, base_branch: 'main' },
            ],
            parameters: { ...taskParameters },
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
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
            container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
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
            bootstrap_layer_id: 'layer-1',
            layers: [],
            jobs: [],
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const panel = page.getByTestId('task-repo-clone-identity-panel');
    await expect(panel).toBeVisible({ timeout: 30000 });
    await expect(panel.getByText(REPO_URL)).toBeVisible({ timeout: 10000 });

    const rowSelect = panel.locator('select').first();
    await expect(rowSelect).toBeVisible({ timeout: 10000 });
    await rowSelect.selectOption(GIT_IDENTITY_ID);

    await expect.poll(() => patchRequests.length, { timeout: 15000 }).toBeGreaterThanOrEqual(1);
    const last = patchRequests[patchRequests.length - 1];
    const posted = last.postDataJSON();
    expect(posted.repo_clone_git_identities[REPO_URL]).toBe(GIT_IDENTITY_ID);
  });

  test('页面加载时自动选用公司默认 Git 身份并 PATCH', async ({ page }) => {
    /** @type {import('@playwright/test').Request[]} */
    const patchRequests = [];

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/task-detail/${TASK_ID}/mock-trae-online-log`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        });
        return;
      }

      if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET' && url.includes('workspace_id=')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-mock-1',
              name: 'Mock 项目',
              git_repos: [REPO_URL],
            },
          ]),
        });
        return;
      }

      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            identities: [
              {
                id: GIT_IDENTITY_ID,
                display_name: 'E2E 默认身份',
                git_user_name: 'default-user',
                git_user_email: 'default@example.com',
                is_default: true,
              },
            ],
          }),
        });
        return;
      }

      if (
        url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/repo-clone-git-identities`) &&
        method === 'PATCH'
      ) {
        patchRequests.push(req);
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ok: true,
            repo_clone_git_identities: { [REPO_URL]: GIT_IDENTITY_ID },
          }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Task',
            description: 'Mock',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [{ project_id: 'proj-mock-1', repo_index: 0, base_branch: 'main' }],
            parameters: { repo_clone_git_identities: {} },
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
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
            container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
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
            bootstrap_layer_id: 'layer-1',
            layers: [],
            jobs: [],
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const panel = page.getByTestId('task-repo-clone-identity-panel');
    await expect(panel).toBeVisible({ timeout: 30000 });

    const rowSelect = panel.locator('select').first();
    await expect(rowSelect).toHaveValue(GIT_IDENTITY_ID, { timeout: 15000 });

    await expect.poll(() => patchRequests.length, { timeout: 15000 }).toBeGreaterThanOrEqual(1);
    const posted = patchRequests[0].postDataJSON();
    expect(posted.repo_clone_git_identities[REPO_URL]).toBe(GIT_IDENTITY_ID);
  });

  test('选择身份后点击重新克隆时 POST 携带 repo_clone_git_identity_id', async ({ page }) => {
    /** @type {import('@playwright/test').Request[]} */
    const reclonePosts = [];

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/task-detail/${TASK_ID}/mock-trae-online-log`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        });
        return;
      }

      if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET' && url.includes('workspace_id=')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-mock-1',
              name: 'Mock 项目',
              git_repos: [REPO_URL],
            },
          ]),
        });
        return;
      }

      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            identities: [
              {
                id: GIT_IDENTITY_ID,
                display_name: 'E2E 身份',
                is_default: true,
              },
            ],
          }),
        });
        return;
      }

      if (
        url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/repo-clone-git-identities`) &&
        method === 'PATCH'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ok: true,
            repo_clone_git_identities: { [REPO_URL]: GIT_IDENTITY_ID },
          }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Task',
            description: 'Mock',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [
              { project_id: 'proj-mock-1', repo_index: 0, base_branch: 'main' },
            ],
            parameters: {
              repo_clone_git_identities: { [REPO_URL]: GIT_IDENTITY_ID },
            },
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
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
            container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
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
            bootstrap_layer_id: 'layer-1',
            layers: [],
            jobs: [],
          }),
        });
        return;
      }

      if (url.includes('/cloud/repo-reclone/') && method === 'POST') {
        reclonePosts.push(req);
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ok: true, result: { status: 'ok' } }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const panel = page.getByTestId('task-repo-clone-identity-panel');
    await expect(panel).toBeVisible({ timeout: 30000 });
    const rowSelect = panel.locator('select').first();
    await rowSelect.selectOption(GIT_IDENTITY_ID);

    const recloneBtn = panel.getByRole('button', { name: '重新克隆' });
    await expect(recloneBtn).toBeEnabled({ timeout: 15000 });
    await recloneBtn.click();

    await expect.poll(() => reclonePosts.length, { timeout: 15000 }).toBeGreaterThanOrEqual(1);
    const body = reclonePosts[reclonePosts.length - 1].postDataJSON();
    expect(body.repo_url).toBe(REPO_URL);
    expect(body.repo_clone_git_identity_id).toBe(GIT_IDENTITY_ID);
  });

  test('回跳 query 含 github=ok 时，PR 授权账号下拉应自动刷新', async ({ page }) => {
    let githubStatusHit = 0;

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/task-detail/${TASK_ID}/mock-trae-online-log`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: '', in_progress: false }),
        });
        return;
      }

      if (url.includes(`/api/projects/tenant_id/${TENANT_ID}`) && method === 'GET' && url.includes('workspace_id=')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'proj-mock-1',
              name: 'Mock 项目',
              git_repos: [REPO_URL],
            },
          ]),
        });
        return;
      }

      if (url.includes('/api/user/e2e-user/profile/git-identities/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            identities: [
              {
                id: GIT_IDENTITY_ID,
                display_name: 'E2E 身份',
                is_default: true,
              },
            ],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/github-credential-status/') && method === 'GET') {
        githubStatusHit += 1;
        const connected = githubStatusHit >= 3;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            github_app_connected: connected,
            github_connections: connected
              ? [{ connected: true, github_user_id: GITHUB_USER_ID, github_login: GITHUB_LOGIN }]
              : [],
            repo_bindings: [],
          }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Task',
            description: 'Mock',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [
              { project_id: 'proj-mock-1', repo_index: 0, base_branch: 'main' },
            ],
            parameters: { repo_clone_git_identities: {} },
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
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
            container_page_url: 'http://127.0.0.1:18080/ui/mock-token',
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
            bootstrap_layer_id: 'layer-1',
            layers: [],
            jobs: [],
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const panel = page.getByTestId('task-repo-clone-identity-panel');
    await expect(panel).toBeVisible({ timeout: 30000 });
    const row = panel.locator('li', { hasText: REPO_URL }).first();
    await expect(row).toBeVisible({ timeout: 10000 });
    const githubAccountSelect = row.locator('select').nth(1);
    await expect(githubAccountSelect).toBeVisible({ timeout: 10000 });

    await expect(
      githubAccountSelect.locator(`option[value="${GITHUB_USER_ID}"]`),
    ).toContainText(`@${GITHUB_LOGIN}`, { timeout: 15000 });
    expect(githubStatusHit).toBeGreaterThanOrEqual(3);
  });
});
