// @ts-check
/**
 * 回归：容器 jobs 已全部终态，但 layers[].job_status / mind_state 仍为 running 时，
 * zTree 层级行应与任务一致，不得长期显示 running（与 GET 执行日志 completed 对齐）。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '836107536731832320';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = '836107536731832320-layer';

test.describe('TaskDetail zTree 层状态与 jobs 对齐', () => {
  test('多条任务终态时层级行不显示 stale running', async ({ page }) => {
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
            title: 'Mock reconcile',
            description: '',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock' },
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
                command: 'mock',
                job_status: 'running',
                mind_state: 'running',
                git_worktree_dirty: false,
                queue_depth: 0,
              },
            ],
            jobs: [
              {
                id: 'job-mock-a',
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: 'step a',
                created_at: '2026-01-01T00:00:01Z',
              },
              {
                id: 'job-mock-b',
                layer_id: LAYER_ID,
                status: 'completed',
                command_kind: 'trae',
                command: 'step b',
                created_at: '2026-01-01T00:00:02Z',
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

    const nodeButtons = ztreePanel.locator('button.break-words.flex-1.min-w-0');
    await expect(nodeButtons.first()).toBeVisible({ timeout: 15000 });
    const count = await nodeButtons.count();
    let layerLabel = '';
    for (let i = 0; i < count; i++) {
      const label = ((await nodeButtons.nth(i).innerText()) || '').trim();
      if (label === '可写层' || /^可写层（/.test(label)) continue;
      layerLabel = label;
      break;
    }
    expect(layerLabel.length).toBeGreaterThan(0);
    expect(layerLabel.toLowerCase()).not.toMatch(/^\s*running\s/);
    expect(layerLabel.toLowerCase()).toMatch(/^\s*completed\s/);
  });
});
