// @ts-check
/**
 * 回归：层变动文件列表滚动续拉分页。
 * - 首屏 has_more=true 时存在 load-more-sentinel
 * - 滚动触发续拉（container-layer-diff-parent-files），列表条目增长
 *
 * 纯 mock，无真实账号依赖。
 */
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
const PAGE1_SIZE = 40;
const PAGE2_SIZE = 20;
const TOTAL = PAGE1_SIZE + PAGE2_SIZE;

function makeChanges(start, count, kind = 'modified') {
  return Array.from({ length: count }, (_, i) => ({
    path: `src/file-${String(start + i).padStart(3, '0')}.ts`,
    kind,
    git_staged: false,
    git_unstaged: true,
    git_layer_diff_only: false,
  }));
}

const page1Changes = makeChanges(0, PAGE1_SIZE, 'modified');
const page2Changes = makeChanges(PAGE1_SIZE, PAGE2_SIZE, 'added');

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ loadMoreHits: string[] }} counters
 */
async function setupPageMocks(page, counters) {
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
          steps: { note: '', steps: [] },
          layer_changes: {
            layer_id: LAYER_ID,
            parent_layer_id: 'layer-0',
            same: false,
            truncated: false,
            change_count: TOTAL,
            changes: page1Changes,
            offset: 0,
            next_offset: PAGE1_SIZE,
            has_more: true,
            detail: '',
          },
        }),
      });
      return;
    }

    // 滚动续拉真实 API（不是 container-layer-changes）
    if (url.includes('/cloud/compute/container-layer-diff-parent-files/') && method === 'GET') {
      counters.loadMoreHits.push(url);
      const urlObj = new URL(url);
      const offset = parseInt(urlObj.searchParams.get('offset') || '0', 10);
      const changes = offset >= PAGE1_SIZE ? page2Changes : page1Changes;
      const hasMore = offset < PAGE1_SIZE;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          layer_id: LAYER_ID,
          parent_layer_id: 'layer-0',
          same: false,
          truncated: false,
          change_count: TOTAL,
          changes,
          offset,
          next_offset: hasMore ? PAGE1_SIZE : TOTAL,
          has_more: hasMore,
          detail: '',
        }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(TASK_DETAIL_PATH);
  await page.waitForLoadState('domcontentloaded');
}

/** @param {import('@playwright/test').Page} page */
async function openLayerChangesList(page) {
  const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
  await expect(ztreePanel).toBeVisible({ timeout: 30000 });
  await ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0).click();

  const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
  await expect(execPanel).toBeVisible({ timeout: 15000 });

  const layerChangesPanel = await openLayerFilesChangesTab(page);

  const changeSummary = layerChangesPanel.locator('summary').filter({ hasText: '个文件发生变动' }).first();
  await expect(changeSummary).toBeVisible({ timeout: 15000 });
  await changeSummary.click();

  const list = page.getByTestId('task-detail-layer-changes-list');
  await expect(list).toBeVisible({ timeout: 15000 });
  return list;
}

test.describe('TaskDetail 层变动文件列表滚动续拉', () => {
  test('首屏 has_more=true 时可见变动条目与 sentinel', async ({ page }) => {
    const counters = { loadMoreHits: /** @type {string[]} */ ([]) };
    await setupPageMocks(page, counters);

    const list = await openLayerChangesList(page);
    // 每条 li 含「路径」+「添加」两个 button，按 li 计数
    await expect(list.locator('ul li')).toHaveCount(PAGE1_SIZE, { timeout: 10000 });
    await expect(page.getByTestId('task-detail-layer-changes-load-more-sentinel')).toBeAttached({
      timeout: 5000,
    });
  });

  test('滚动触发续拉请求后列表条目增长', async ({ page }) => {
    const counters = { loadMoreHits: /** @type {string[]} */ ([]) };
    await setupPageMocks(page, counters);

    const list = await openLayerChangesList(page);
    await expect(list.locator('ul li')).toHaveCount(PAGE1_SIZE, { timeout: 10000 });

    // 滚动到底触发 remain<=48 的 load-more
    await list.evaluate((el) => {
      el.scrollTop = el.scrollHeight;
    });

    await expect
      .poll(() => counters.loadMoreHits.length, { timeout: 10000 })
      .toBeGreaterThan(0);

    await expect(list.locator('ul li')).toHaveCount(TOTAL, { timeout: 10000 });
    await expect(page.getByTestId('task-detail-layer-changes-load-more-sentinel')).toHaveCount(0);
  });
});
