// @ts-check
/**
 * 回归：推送成功后 zTree 应显示「N 个提交已推送」并隐藏推送按钮。
 * 对应 docs/intents/frontend/ztree_push_ahead_after_success.*
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_ztree_push_ahead_mock';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = '20260712_061444_1ecb8d';

test.describe('TaskDetail zTree 推送成功后 ahead 文案', () => {
  test('推送成功后显示已推送并隐藏推送按钮', async ({ page }) => {
    test.setTimeout(120000);
    page.on('dialog', (d) => void d.accept());

    let graphGeneration = 0;

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock push ahead',
            description: '',
            created_at: '2026-07-12T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [
              {
                stored_repo_address: 'https://gitlab.daydaymoney.com/example-user/somanyad.git',
                target_branch: 'feature/mock-push-ahead',
              },
            ],
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
            container_page_url: 'http://127.0.0.1:18080/ui/dev-local-token',
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        const ahead = graphGeneration === 0 ? 1 : 0;
        const gitRemote =
          ahead > 0
            ? { is_git: true, ahead: 1, no_upstream: false, upstream: 'origin/feature/mock-push-ahead' }
            : {
                is_git: true,
                ahead: 0,
                no_upstream: false,
                upstream: 'origin/feature/mock-push-ahead',
                last_pushed_count: 1,
              };
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: LAYER_ID,
            layers: [
              {
                layer_id: LAYER_ID,
                parent_layer_id: null,
                created_at: '2026-07-12T06:14:44Z',
                command: 'mock push ahead',
                job_status: 'completed',
                mind_state: 'idle_done',
                git_worktree_dirty: false,
                git_remote: gitRemote,
                queue_depth: 0,
              },
            ],
            jobs: [
              {
                id: 'job-mock-push',
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: 'mock push ahead',
                created_at: '2026-07-12T06:15:00Z',
              },
            ],
          }),
        });
        return;
      }

      if (url.includes('container-layer-git-push') && method === 'POST') {
        graphGeneration = 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ok: true,
            git_remote: { is_git: true, ahead: 0, no_upstream: false },
            github_oauth_multirepo: {
              repos: [{ rel_prefix: 'somanyad', push_ok: true, provider: 'gitlab' }],
            },
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layer_id: LAYER_ID, text: '' }),
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

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });

    const aheadLabel = ztreePanel.getByTestId('layer-ztree-push-ahead-label');
    await expect(aheadLabel).toBeVisible({ timeout: 15000 });
    await expect(aheadLabel).toHaveText('1 个提交可推送');

    const pushBtn = ztreePanel.getByTestId('layer-ztree-push-btn');
    await expect(pushBtn).toBeVisible();
    await expect(pushBtn).toBeEnabled();

    await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 30000 },
      ),
      pushBtn.click(),
    ]);

    await expect(aheadLabel).toHaveText('1 个提交已推送', { timeout: 15000 });
    await expect(ztreePanel.getByTestId('layer-ztree-push-btn')).toHaveCount(0);
  });
});
