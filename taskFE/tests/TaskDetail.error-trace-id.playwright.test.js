// @ts-check
/**
 * 验证请求报错 DOM 元素上 data-traceId 属性按元规则正确挂载。
 * @see .ai/01_project_constraints/24_frontend_error_data_trace_id.md
 *
 * 策略：模拟带 X-Trace-Id 响应头的失败 API → 等待错误展示 → 断言 data-traceId 值。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TEST_TRACE_ID = 'e2e-test-trace-id-20260718';

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`
);

test.describe('TaskDetail error data-traceId attribute', () => {
  test('budgets API 500 response should surface error with X-Trace-Id on DOM', async ({ page }) => {
    test.setTimeout(60_000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-test-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    /** @type {string[]} */
    const traceIdHeadersSeen = [];

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // 任务详情基本信息
      if (url.includes(`/todos/${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'E2E TraceId Test Task',
            description: 'Verify data-traceId on error display',
            created_at: '2026-07-01T00:00:00Z',
            created_by: { username: 'e2e-user' },
            comments: [],
            ai_comments: [],
            container_agent_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            branch_strategy: {},
          }),
        });
        return;
      }

      // 功能参数：启用 LLM 预算面板
      if (url.includes('/feature-params/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: { llm_budget_enabled: true },
          }),
        });
        return;
      }

      // 预算接口：模拟 500 错误并注入 X-Trace-Id
      if (url.includes('/model-budgets/') && method === 'GET') {
        traceIdHeadersSeen.push(TEST_TRACE_ID);
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          headers: {
            'X-Trace-Id': TEST_TRACE_ID,
          },
          body: JSON.stringify({ message: '模拟预算加载失败' }),
        });
        return;
      }

      // 协作人员
      if (url.includes('/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: '[]',
        });
        return;
      }

      // 进度系统
      if (url.includes('/progress-system/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
        });
        return;
      }

      // 交付物类别
      if (url.includes('/deliverable-categories/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: '[]',
        });
        return;
      }

      // 默认剩余：返回空对象
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(TASK_DETAIL_PATH, { waitUntil: 'domcontentloaded', timeout: 30_000 });

    // 等待错误文案出现
    const errorEl = page.locator('text=模拟预算加载失败').first();
    await expect(errorEl).toBeVisible({ timeout: 25_000 });

    // 断言 data-traceId 属性值正确
    const traceIdEl = page.locator('[data-traceId]').first();
    await expect(traceIdEl).toBeVisible({ timeout: 5_000 });
    const actualTraceId = await traceIdEl.getAttribute('data-traceId');
    expect(actualTraceId).toBe(TEST_TRACE_ID);

    // 确认测试框架发出的 traceId 请求头至少出现一次
    expect(traceIdHeadersSeen.length).toBeGreaterThanOrEqual(1);
  });
});
