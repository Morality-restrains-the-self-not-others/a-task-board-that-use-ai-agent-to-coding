// @ts-check
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { openLayerFilesChangesTab } from './openLayerFilesChangesTab.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '837978129569890304';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

test.describe('TaskDetail 文件变动刷新应同步刷新项目文件树', () => {
  test('文件变动区域刷新后，项目文件树应自动重新拉取', async ({ page }) => {
    let filesApiCallCount = 0;
    let layerChangesApiCallCount = 0;
    let fileListVersion = 1;

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
            bootstrap_layer_id: LAYER_ID,
            layers: [
              {
                layer_id: LAYER_ID,
                parent_layer_id: null,
                created_at: '2026-01-01T00:00:00Z',
                command: 'mock command',
                job_status: 'completed',
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

      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        layerChangesApiCallCount += 1;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: { id: JOB_ID, status: 'completed', layer_id: LAYER_ID, command: 'echo mock', output: 'job-output' },
            steps: { note: '', steps: [] },
            layer_changes: {
              layer_id: LAYER_ID,
              change_count: layerChangesApiCallCount,
              changes: [
                { path: `new-file-${layerChangesApiCallCount}.py`, kind: 'added', git_unstaged: true },
              ],
              truncated: false,
            },
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-files/') && method === 'GET') {
        filesApiCallCount += 1;
        const files = [];
        for (let i = 1; i <= fileListVersion; i++) {
          files.push({ path: `file-${i}.txt`, size: 10, mtime: '2026-01-01T00:00:00' });
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            truncated: false,
            files: files,
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

    const fileTreeRoot = page.getByTestId('comment-layer-ztree-project-file-tree');
    await expect(fileTreeRoot).toBeAttached({ timeout: 15000 });

    await expect
      .poll(() => filesApiCallCount, { timeout: 10000 })
      .toBeGreaterThan(0);

    const initialFileCount = filesApiCallCount;

    const layerChangesPanel = await openLayerFilesChangesTab(page);

    const refreshButton = layerChangesPanel.getByRole('button', { name: '刷新' });
    await expect(refreshButton).toBeVisible({ timeout: 10000 });

    fileListVersion = 2;
    await refreshButton.click();

    await expect
      .poll(() => layerChangesApiCallCount, { timeout: 10000 })
      .toBeGreaterThan(1);

    await expect
      .poll(() => filesApiCallCount, { timeout: 10000 })
      .toBeGreaterThan(initialFileCount);
  });
});
