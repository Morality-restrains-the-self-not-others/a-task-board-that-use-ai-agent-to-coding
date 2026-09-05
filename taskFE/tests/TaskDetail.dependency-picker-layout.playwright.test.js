// @ts-check
/**
 * 回归：TaskDetailCommentComposer 中 showDependencyPicker=true 时，
 * 评论执行依赖选择器（CommentExecutionDependencyPicker）应始终位于文本输入框之前。
 *
 * 对应：TaskDetailCommentComposer.vue — CommentExecutionDependencyPicker 在 textarea 之前渲染
 *
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_dep_picker_mock';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

test.describe('TaskDetailCommentComposer 依赖选择器布局', () => {
  test('依赖选择器（dependency picker）在 DOM 顺序上位于 textarea 之前', async ({ page }) => {
    test.setTimeout(120000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // 任务详情：简单任务，无评论
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Dependency Picker Layout Test',
            description: '',
            created_at: '2026-07-22T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
          }),
        });
        return;
      }

      // container-task-ui-context
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

      // container-layer-graph
      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: '',
            layers: [],
            jobs: [],
          }),
        });
        return;
      }

      // comment-container-bindings GET + POST
      if (url.includes('comment-container-bindings') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bindings: [] }) });
        return;
      }
      if (url.includes('comment-container-bindings') && method === 'POST') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
        return;
      }

      // workspace-collaborators
      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      // progress-system
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
        });
        return;
      }

      // 容器日志
      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layer_id: '', text: '' }),
        });
        return;
      }

      // workspace-runtime-indicators
      if (url.includes('workspace-runtime-indicators') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', indicators: [] }) });
        return;
      }

      // workspace-machine-summary
      if (url.includes('workspace-machine-summary') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }) });
        return;
      }

      // installed-images（composer loadAtModeContext 调用）
      if (url.includes('/installed-images/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      // workspace detail（composer loadAtModeContext 调用：GET /api/projects/workspaces/tenant_id/{tid}/{wid}）
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ id: WORKSPACE_ID, name: '测试空间', container_image_at_mode_enabled: false }),
        });
        return;
      }

      // 兜底
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    // 等待评论 composer 渲染
    // TaskDetailCommentComposer 根节点有 class .task-detail-comment-composer
    const composer = page.locator('.task-detail-comment-composer');
    await expect(composer).toBeVisible({ timeout: 20000 });

    // 依赖选择器应可见（showDependencyPicker 默认为 true）
    const depPicker = page.getByTestId('comment-execution-dependency-picker');
    await expect(depPicker).toBeVisible({ timeout: 10000 });

    // textarea 应可见（id=comment-content）
    const textarea = page.locator('#comment-content');
    await expect(textarea).toBeVisible({ timeout: 5000 });

    // 断言 DOM 顺序：依赖选择器在 textarea 之前
    // compareDocumentPosition: 2=DOCUMENT_POSITION_PRECEDING, 4=DOCUMENT_POSITION_FOLLOWING
    const order = await page.evaluate(() => {
      const picker = document.querySelector('[data-testid="comment-execution-dependency-picker"]');
      const ta = document.querySelector('#comment-content');
      if (!picker || !ta) return -1;
      return picker.compareDocumentPosition(ta);
    });
    // picker 在 textarea 前 → result & 4（FOLLOWING）应为 0（即 textarea 不在 picker 前）→ picker 在 textarea 前
    expect(order & 4).toBe(0, '依赖选择器应位于 textarea 之前');
  });
});
