// @ts-check
/**
 * 回归：容器快照中子可写层带 parent_layer_id 且仅有 clone 任务时，
 * 评论区 zTree 为单串行列表：父层行与子层行按时间平铺，子层克隆指令仍可见。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '836132944159420416';
const LAYER_BOOT = 'layer-bootstrap-mock';
const LAYER_CHILD = 'layer-repo-child-mock';

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

test.describe('TaskDetail 克隆阶段 zTree 串行平铺（mock）', () => {
  test('子层 parent_layer_id 指向父层时子层行与父层行串行平铺', async ({ page }) => {
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
            title: 'Mock clone hierarchy',
            description: '',
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
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
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
            bootstrap_layer_id: LAYER_BOOT,
            layers: [
              {
                layer_id: LAYER_BOOT,
                parent_layer_id: null,
                created_at: '2026-01-01T00:00:00Z',
                command: '',
                job_status: 'completed',
                mind_state: 'idle_done',
                git_worktree_dirty: null,
              },
              {
                layer_id: LAYER_CHILD,
                parent_layer_id: LAYER_BOOT,
                created_at: '2026-01-01T00:01:00Z',
                command: '',
                job_status: 'running',
                mind_state: 'running',
                git_worktree_dirty: null,
              },
            ],
            jobs: [
              {
                id: 'job-clone-child',
                layer_id: LAYER_CHILD,
                status: 'running',
                command_kind: 'clone',
                command: 'git clone https://example.com/repo-a.git',
                created_at: '2026-01-01T00:01:01Z',
              },
            ],
          }),
        });
        return;
      }

      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto('http://localhost:4000' + TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const panel = page.getByTestId('comment-layer-ztree-panel');
    await expect(panel).toBeVisible({ timeout: 30000 });

    await expect(
      panel.getByRole('button', { name: /git clone|example\.com\/repo-a/i }),
    ).toBeVisible({ timeout: 15000 });

    const topLevelRows = panel.locator('ul.ztree-mimic > li');
    await expect(topLevelRows).toHaveCount(2);
  });
});
