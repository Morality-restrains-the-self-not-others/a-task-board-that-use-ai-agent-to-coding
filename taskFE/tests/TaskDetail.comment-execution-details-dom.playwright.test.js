// @ts-check
/**
 * 回归：评论 Feed 中 [data-testid=comment-execution-details] 应位于评论气泡
 * [data-testid=comment-bubble] 内，且不是 div.space-y-2 的直接子节点
 * （防止错误的 DOM 嵌套导致布局异常）。
 *
 * 对应：TaskDetailConversationFeed.vue slot → TaskDetailCommentsSection.vue #execution-details
 *       TaskDetailCommentExecutionDetails.vue
 *
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = 'task_comment_dom_mock';
const COMMENT_ID = 'comment-user-001';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

test.describe('TaskDetail 评论 Feed execution-details DOM 结构', () => {
  test('execution-details 位于评论气泡 comment-bubble 内，非 div.space-y-2 的直接子节点', async ({ page }) => {
    test.setTimeout(120000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // 任务详情：包含一条用户评论
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Comment DOM Test',
            description: 'Testing execution-details DOM nesting',
            created_at: '2026-07-22T00:00:00Z',
            created_by: { username: 'mock' },
            comments: [
              {
                id: COMMENT_ID,
                content: '写一个 hello world',
                created_at: '2026-07-22T10:00:00Z',
                created_by: { username: '测试用户' },
              },
            ],
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

      // comment-container-bindings GET
      if (url.includes('comment-container-bindings') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ bindings: [] }),
        });
        return;
      }

      // comment-container-bindings POST（ensure + advance 共用 — 注意 /advance/ 子路径也会匹配）
      // ensure：POST /comment-container-bindings/?task_id=...
      // advance：POST /comment-container-bindings/advance/?task_id=...
      if (url.includes('comment-container-bindings') && method === 'POST') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
        return;
      }

      // 评论 PATCH（syncBindingsForComments 内部会 PATCH execution_mode）
      // URL：/api/tasks/{taskId}/comments/{commentId}/tenant_id/{tid}
      if (url.includes('/comments/') && method === 'PATCH') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
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

      // 兜底
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    // 等待 comments-container 和评论渲染
    const commentsContainer = page.locator('#comments-container');
    await expect(commentsContainer).toBeVisible({ timeout: 20000 });

    // 至少一句评论正文可见
    await expect(page.getByText('写一个 hello world')).toBeVisible({ timeout: 10000 });

    // 等待 execution-details 渲染（需等待 bindings 同步完成后组件挂载）
    const execDetails = page.getByTestId('comment-execution-details');
    await expect(execDetails).toBeVisible({ timeout: 15000 });

    // 断言 1：execution-details 位于评论气泡 comment-bubble 内
    const insideBubble = await page.evaluate(() => {
      const el = document.querySelector('[data-testid="comment-execution-details"]');
      if (!el) return false;
      return Boolean(el.closest('[data-testid="comment-bubble"]'));
    });
    expect(insideBubble).toBe(true);

    // 断言 2：execution-details 不是 div.space-y-2 的直接子节点
    const notDirectChildOfSpaceY2 = await page.evaluate(() => {
      const el = document.querySelector('[data-testid="comment-execution-details"]');
      if (!el) return false;
      const parent = el.parentElement;
      if (!parent) return true;
      return !(parent.className || '').includes('space-y-2');
    });
    expect(notDirectChildOfSpaceY2).toBe(true);
  });
});
