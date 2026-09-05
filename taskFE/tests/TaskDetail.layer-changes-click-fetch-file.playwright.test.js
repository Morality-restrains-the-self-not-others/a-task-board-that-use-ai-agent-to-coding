// @ts-check
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { openLayerFilesChangesTab } from './openLayerFilesChangesTab.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

test.describe('TaskDetail layer changes 文件预览', () => {
  test('点击变更文件后应发起文件内容请求并显示右侧预览', async ({ page }) => {
    const fileContentHits = [];

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
            title: 'Mock Task',
            description: 'Mock task detail',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
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
            bootstrap_layer_id: LAYER_ID,
            layers: [
              {
                layer_id: LAYER_ID,
                parent_layer_id: null,
                created_at: '2026-01-01T00:00:00Z',
                command: 'mock command',
                job_status: 'completed',
                git_worktree_dirty: false,
                mind_state: 'idle_done',
                queue_depth: 0,
              },
            ],
            jobs: [
              {
                id: JOB_ID,
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: 'echo mock',
                created_at: '2026-01-01T00:00:01Z',
                output: 'job-output',
              },
            ],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layer_id: LAYER_ID, text: 'clone ok' }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: {
              id: JOB_ID,
              status: 'completed',
              layer_id: LAYER_ID,
              command: 'echo mock',
              output: 'job-output',
            },
            steps: {
              note: '',
              steps: [],
            },
            layer_changes: {
              layer_id: LAYER_ID,
              parent_layer_id: 'parent-1',
              same: false,
              truncated: false,
              change_count: 2,
              changes: [
                { path: 'src/a.py', kind: 'modified' },
                { path: 'README.md', kind: 'added' },
              ],
            },
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-file-content/') && method === 'GET') {
        fileContentHits.push(url);
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            path: 'src/a.py',
            kind: 'text',
            truncated: false,
            text: 'print("hello from layer")',
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });
    await ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0).click();

    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect(execPanel).toBeVisible({ timeout: 15000 });

    const layerChangesPanel = await openLayerFilesChangesTab(page);

    const changeSummary = layerChangesPanel.locator('summary').filter({ hasText: '个文件发生变动' }).first();
    await expect(changeSummary).toBeVisible({ timeout: 15000 });
    await changeSummary.click();

    const firstFileBtn = layerChangesPanel.locator('ul li button').first();
    await expect(firstFileBtn).toBeVisible({ timeout: 15000 });
    await firstFileBtn.click();

    await expect
      .poll(() => fileContentHits.length, { timeout: 10000 })
      .toBeGreaterThan(0);

    await expect(layerChangesPanel.getByText('文件内容预览')).toBeVisible({ timeout: 10000 });
    await expect(layerChangesPanel.getByText('print("hello from layer")')).toBeVisible({ timeout: 10000 });
  });
});
