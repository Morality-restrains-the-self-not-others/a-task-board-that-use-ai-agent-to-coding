// @ts-check
/**
 * 回归：有 git 且工作区干净时，zTree 仍应显示「提交」按钮（禁用），
 * 且文件变动摘要可见；对应 docs/intents/frontend/ztree_submit_visible_with_file_changes.*
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { openLayerFilesChangesTab } from './openLayerFilesChangesTab.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_ztree_submit_visible_mock';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = '20260712_151647_9dea2e';
const JOB_ID = 'e116ee72-41bc-4c50-87e7-0c0bf3ca11de';

test.describe('TaskDetail zTree 提交按钮可见性', () => {
  test('工作区干净时仍显示禁用的提交按钮，且文件变动列表有摘要', async ({ page }) => {
    test.setTimeout(120000);

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
            title: 'Mock submit visible',
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
                target_branch: 'feature/mock-submit',
              },
            ],
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
            bootstrap_layer_id: LAYER_ID,
            layers: [
              {
                layer_id: LAYER_ID,
                parent_layer_id: null,
                created_at: '2026-07-12T15:16:47Z',
                command: '用 lisp 写 hello',
                job_status: 'completed',
                mind_state: 'idle_done',
                git_worktree_dirty: false,
                git_remote: {
                  is_git: true,
                  ahead: 1,
                  no_upstream: false,
                  upstream: 'origin/master',
                  current_branch: 'feature/mock-submit',
                },
                queue_depth: 0,
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: '用 lisp 写 hello',
                created_at: '2026-07-12T15:17:00Z',
              },
            ],
          }),
        });
        return;
      }

      if (url.includes('container-job-execution-log') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job_id: JOB_ID,
            layer_changes: {
              layer_id: LAYER_ID,
              same: false,
              truncated: false,
              change_count: 1,
              changes: [
                {
                  path: 'somanyad/hello.lisp',
                  kind: 'added',
                  git_staged: false,
                  git_unstaged: false,
                  git_layer_diff_only: true,
                },
              ],
            },
            steps: [],
            output: '',
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

    const submitBtn = ztreePanel.getByTestId('layer-ztree-submit-btn');
    await expect(submitBtn).toBeVisible({ timeout: 15000 });
    await expect(submitBtn).toBeDisabled();
    await expect(submitBtn).toHaveAttribute('title', /暂无未提交变更/);

    // 选中层节点后，文件变动摘要应出现
    await ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0).click();
    const changesPanel = await openLayerFilesChangesTab(page);
    await expect(changesPanel).toContainText(/个文件发生变动/);
  });
});
