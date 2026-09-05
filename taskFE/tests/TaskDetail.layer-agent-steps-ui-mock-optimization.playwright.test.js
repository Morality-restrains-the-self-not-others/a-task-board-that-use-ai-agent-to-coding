// @ts-check
/**
 * TaskDetail zTree 代理步骤 UI 优化回归测试（mock）：
 *
 * OPT-20260719-029 (low): 仅含 tool_calls 的步骤展开后不出现 indigo pre（bash 命令行正文）
 * OPT-20260719-045 (low): 完成态步骤标题「步骤 N · ✅」后换行再接 command
 * OPT-20260719-047 (medium): 层图 job running 但 steps 全 completed 时刷新后不显示 running/执行中
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`
);
const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

/** 共享 mock 路由设置 */
async function setupCommonMocks(page, { execLogMock, layerGraphMock }) {
  await page.context().addCookies([
    { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
  ]);
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    if (url.includes('/container-job-execution-log/') && route.request().method() === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(execLogMock) });
      return;
    }
    if (url.includes('/container-layer-graph/') && route.request().method() === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(layerGraphMock) });
      return;
    }
    if (url.includes('/container-task-ui-context/') && route.request().method() === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', container_endpoint_registered: true }) });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

test.describe('TaskDetail agent-steps 优化回归（mock）', () => {
  /**
   * OPT-20260719-029: 仅含 tool_calls 的步骤展开后不出现 indigo pre（bash 命令行正文）。
   */
  test('OPT-029: 仅含 tool_calls 的步骤展开 no indigo pre', async ({ page }) => {
    await page.addInitScript(({ tenantId, workspaceId, taskId }) => {
      class MockEventSource {
        static CONNECTING = 0; static OPEN = 1; static CLOSED = 2;
        constructor(url) { this.url = String(url || ''); this.readyState = MockEventSource.OPEN; }
        close() { this.readyState = MockEventSource.CLOSED; }
      }
      window.EventSource = MockEventSource;
    }, { tenantId: TENANT_ID, workspaceId: WORKSPACE_ID, taskId: TASK_ID });

    const execLogMock = {
      status: 'success',
      job_id: JOB_ID,
      job_status: 'completed',
      steps: [{
        step_index: 1,
        step_status: 'completed',
        step_type: 'agent',
        content_type: 'application/json',
        tool_calls: [{ function: { name: 'bash', arguments: 'ls -la' } }],
        tool_results: [{ result: 'total 42' }],
      }],
    };

    const layerGraphMock = {
      layers: [{ id: LAYER_ID, layer_name: 'layer-1' }],
      jobs: [{ id: JOB_ID, status: 'completed', layer_id: LAYER_ID }],
    };

    await setupCommonMocks(page, { execLogMock, layerGraphMock });
    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待步骤卡片渲染
    await expect(page.locator('[data-testid="layer-agent-step-card"]').first()).toBeVisible({ timeout: 10_000 });

    // 展开步骤
    await page.locator('[data-testid="layer-agent-step-card-summary"]').first().click();
    await page.waitForTimeout(500);

    // 断言没有 indigo-themed <pre>（bash 命令行正文）
    const indigoPres = await page.locator('pre[class*="indigo"]').count();
    expect(indigoPres).toBe(0);
  });

  /**
   * OPT-20260719-045: 完成态步骤标题「步骤 N · ✅」后换行再接 command。
   */
  test('OPT-045: 完成态步骤标题含 ✅ 和换行', async ({ page }) => {
    await page.addInitScript(() => {
      class MockEventSource {
        static CONNECTING = 0; static OPEN = 1; static CLOSED = 2;
        constructor(url) { this.url = String(url || ''); this.readyState = MockEventSource.OPEN; }
        close() { this.readyState = MockEventSource.CLOSED; }
      }
      window.EventSource = MockEventSource;
    });

    const execLogMock = {
      status: 'success', job_id: JOB_ID, job_status: 'completed',
      steps: [{
        step_index: 1, step_status: 'completed', step_type: 'agent',
        content_type: 'text/plain',
        content: 'echo hello world',
        tool_calls: [{ function: { name: 'Bash', arguments: '{"command":"echo hello world"}' } }],
      }],
    };

    const layerGraphMock = {
      layers: [{ id: LAYER_ID, layer_name: 'layer-1' }],
      jobs: [{ id: JOB_ID, status: 'completed', layer_id: LAYER_ID }],
    };

    await setupCommonMocks(page, { execLogMock, layerGraphMock });
    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待步骤卡片渲染
    await expect(page.locator('[data-testid="layer-agent-step-card"]').first()).toBeVisible({ timeout: 10_000 });

    // 获取 summary 中 <p> 的 textContent
    const summary = page.locator('[data-testid="layer-agent-step-card-summary"]').first();
    const text = await summary.locator('p').first().textContent();
    expect(text).toContain('✅');
    expect(text).toContain('\n');
  });

  /**
   * OPT-20260719-047: 层图 job running 但 steps 全 completed 时刷新后不显示 running/执行中。
   */
  test('OPT-047: 层图 running 但步骤 completed 刷新后不显示执行中', async ({ page }) => {
    await page.addInitScript(() => {
      class MockEventSource {
        static CONNECTING = 0; static OPEN = 1; static CLOSED = 2;
        constructor(url) { this.url = String(url || ''); this.readyState = MockEventSource.OPEN; }
        close() { this.readyState = MockEventSource.CLOSED; }
      }
      window.EventSource = MockEventSource;
    });

    // 层图返回 running，但 exec-log 返回 completed
    const execLogMock = {
      status: 'success', job_id: JOB_ID, job_status: 'completed',
      steps: [
        { step_index: 1, step_status: 'completed', step_type: 'agent' },
        { step_index: 2, step_status: 'completed', step_type: 'agent' },
      ],
    };

    const layerGraphMock = {
      layers: [{ id: LAYER_ID, layer_name: 'layer-1' }],
      jobs: [{ id: JOB_ID, status: 'running', layer_id: LAYER_ID }],
    };

    await setupCommonMocks(page, { execLogMock, layerGraphMock });
    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 等待 exec-log 和 layer-graph 加载
    await expect(page.locator('[data-testid="layer-agent-step-card"]').first()).toBeVisible({ timeout: 10_000 });

    // 使用 toPass 等待前端 reconcile 更新
    await expect(async () => {
      // zTree 节点不应再显示 running
      const ztreeNodes = page.locator('[data-testid^="layer-ztree-node"]');
      const count = await ztreeNodes.count();
      for (let i = 0; i < count; i++) {
        const nodeText = await ztreeNodes.nth(i).textContent();
        expect(nodeText).not.toContain('running');
      }
    }).toPass({ timeout: 15_000 });

    // 代理步骤徽章应为「已结束」
    const badge = page.getByText('已结束').first();
    await expect(badge).toBeVisible({ timeout: 5_000 });
  });
});
