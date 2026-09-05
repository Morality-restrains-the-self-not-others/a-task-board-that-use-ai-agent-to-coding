// @ts-check
/**
 * 核验：点击层变更中的二进制文件项 → 预览区展示 binary-props 而非文本 pre。
 *
 * OPT-20260720-017
 * 运行：
 *   PW_CDP_URL=http://127.0.0.1:9223 PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 *   npx playwright test --config=playwright.config.chromium.js \
 *   --grep="binary-props"
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { openLayerFilesChangesTab } from './openLayerFilesChangesTab.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID =
  process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);

const LAYER_ID = 'layer-binary-1';
const JOB_ID = 'job-binary-1';
const BINARY_PATH = 'build/output.bin';
const BINARY_SIZE_BYTES = 1048576;
const BINARY_MTIME = '2026-07-01T12:34:56Z';

test.describe('TaskDetail layer changes binary props UI', () => {
  test('点击含 NUL/kind=binary 的变动项，断言 binary-props 可见且无文本 pre', async ({ page }) => {
    // 注入登录态 cookie，绕过网关鉴权
    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // ── 任务详情 ──
      if (
        url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) &&
        method === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Binary Task',
            description: 'Mock task for binary preview',
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

      // ── 协作成员 ──
      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      // ── progress system ──
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
        });
        return;
      }

      // ── container task UI context ──
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

      // ── 层图（使用与工作测试一致的 response shape） ──
      if (
        url.includes('/cloud/compute/container-layer-graph/') &&
        method === 'GET'
      ) {
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

      // ── Clone log ──
      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layer_id: LAYER_ID, text: 'clone ok' }),
        });
        return;
      }

      // ── Job execution log（含 layer_changes，包含二进制文件） ──
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
              output: '',
            },
            steps: { note: '', steps: [] },
            layer_changes: {
              layer_id: LAYER_ID,
              parent_layer_id: 'parent-1',
              same: false,
              truncated: false,
              change_count: 3,
              changes: [
                { path: 'src/a.py', kind: 'modified' },
                { path: BINARY_PATH, kind: 'binary' },
                { path: 'README.md', kind: 'added' },
              ],
            },
          }),
        });
        return;
      }

      // ── 文件内容（二进制文件返回 kind=binary + 元数据，无 text 字段） ──
      if (url.includes('/cloud/compute/container-layer-file-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layer_id: LAYER_ID,
            path: BINARY_PATH,
            kind: 'binary',
            truncated: false,
            name: 'output.bin',
            size: BINARY_SIZE_BYTES,
            mtime: BINARY_MTIME,
          }),
        });
        return;
      }

      // ── 兜底 ──
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待层图面板出现
    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 30000 });

    // 点击第一个层节点展开变更列表
    await ztreePanel.locator('button.break-words.flex-1.min-w-0').nth(0).click();

    // 等待执行日志面板出现
    const execPanel = page.getByTestId('comment-layer-ztree-exec-log-panel');
    await expect(execPanel).toBeVisible({ timeout: 15000 });

    // 展开 layer changes summary
    const layerChangesPanel = await openLayerFilesChangesTab(page);
    const changeSummary = layerChangesPanel.locator('summary').filter({ hasText: '个文件发生变动' }).first();
    await expect(changeSummary).toBeVisible({ timeout: 15000 });
    await changeSummary.click();

    // 在层变更列表中点击二进制文件（build/output.bin）
    const binaryChangeEntry = layerChangesPanel.locator('ul li button').filter({ hasText: 'output.bin' });
    await expect(binaryChangeEntry).toBeVisible({ timeout: 10000 });
    await binaryChangeEntry.click();

    // 断言：二进制属性预览区域可见
    const binaryProps = page.getByTestId('layer-change-preview-binary-props');
    await expect(binaryProps).toBeVisible({ timeout: 10000 });

    // 断言：二进制预览中不含文本 pre（二进制不应展示文本内容）
    await expect(binaryProps.locator('pre')).toHaveCount(0);

    // 断言：关键字段可见
    await expect(binaryProps.locator('text=二进制（不展示内容）')).toBeVisible();
    await expect(binaryProps.locator(`text=${BINARY_PATH}`)).toBeVisible();
  });
});
