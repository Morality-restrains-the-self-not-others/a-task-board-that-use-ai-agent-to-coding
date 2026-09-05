// @ts-check
/**
 * 回归：推送并创建PR 成功后应出现 PR 按钮，点击打开审查页。
 * 对应 docs/intents/frontend/ztree_push_and_create_pr.*
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_ztree_pr_btn_mock';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = '20260713_pr_btn_layer';
const PR_URL = 'https://gitlab.daydaymoney.com/example-user/somanyad/-/merge_requests/42';

test.describe('TaskDetail zTree 推送后 PR 按钮', () => {
  test('推送成功后出现 PR 按钮，点击打开审查页', async ({ page }) => {
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
            title: 'Mock push create PR button',
            description: '',
            created_at: '2026-07-13T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            projects: [
              {
                stored_repo_address: 'https://gitlab.daydaymoney.com/example-user/somanyad.git',
                target_branch: 'feature/mock-pr-btn',
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
            ? { is_git: true, ahead: 1, no_upstream: false, upstream: 'origin/feature/mock-pr-btn' }
            : {
                is_git: true,
                ahead: 0,
                no_upstream: false,
                upstream: 'origin/feature/mock-pr-btn',
                last_pushed_count: 1,
                pr_html_url: PR_URL,
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
                created_at: '2026-07-13T06:14:44Z',
                command: 'mock push pr btn',
                job_status: 'completed',
                mind_state: 'idle_done',
                git_worktree_dirty: false,
                git_remote: gitRemote,
                queue_depth: 0,
              },
            ],
            jobs: [
              {
                id: 'job-mock-pr-btn',
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: 'mock push pr btn',
                created_at: '2026-07-13T06:15:00Z',
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
            github_pull_request: {
              html_url: PR_URL,
              number: 42,
              provider: 'gitlab',
            },
            github_oauth_multirepo: {
              repos: [
                {
                  rel_prefix: 'somanyad',
                  push_ok: true,
                  provider: 'gitlab',
                  github_slug: 'example-user/somanyad',
                  pr: { html_url: PR_URL, number: 42 },
                },
              ],
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

    await expect(ztreePanel.getByTestId('layer-ztree-pr-btn')).toHaveCount(0);

    const pushBtn = ztreePanel.getByTestId('layer-ztree-push-btn');
    await expect(pushBtn).toBeVisible();
    await expect(pushBtn).toHaveText('推送并创建PR');

    await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 30000 },
      ),
      pushBtn.click(),
    ]);

    const prBtn = ztreePanel.getByTestId('layer-ztree-pr-btn');
    await expect(prBtn).toBeVisible({ timeout: 15000 });
    await expect(ztreePanel.getByTestId('layer-ztree-push-btn')).toHaveCount(0);
    await expect(prBtn).toHaveAttribute('href', PR_URL);
    await expect(prBtn).toHaveAttribute('target', '_blank');
    await expect(prBtn).toHaveAttribute('rel', /noopener/);
  });
});
