// @ts-check
/**
 * 任务详情面板回归测试覆盖：
 *
 * OPT-20260719-039 (medium, subtree-status)：
 *   任务有未关闭下级交付物时，标记为已完成应弹窗报错并带 data-traceId。
 *
 * OPT-20260719-041 / F-102 (queued-schedule in comment composer)：
 *   评论「自动执行」卡含加入队列按钮与自动调度安排链接；无节奏模态。
 *
 * OPT-20260721-014 (queued-schedule join-queue)：
 *   入队后断言 queued-auto-run-leave 与「前方还有 N 个任务在等待」芯片；
 *   依赖选择器锁定「不等待前序」。
 *
 * OPT-20260719-020 的 PeopleGroups 测试已独立为 PeopleGroups.create-optimization-regression.playwright.test.js。
 *
 * ── 路由注册策略 ──
 * Playwright 按注册顺序匹配 route handler（先注册先匹配）。
 * 每个测试独立注册 page.route 的 api 通配 handler（先注册先匹配），
 * 先处理特定端点（if/return），最后兜底 route.fulfill。
 * 避免共享 catch-all handler 导致后续注册的 handler 无法生效。
 * 注意：块注释内禁止出现含斜杠星号的 glob 通配写法，会提前闭合注释。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

// ─── Mock Data ───────────────────────────────────────────────

const TASK_ID = '830423831930662912';

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

/**
 * 基础任务对象。
 * @param {Record<string, unknown>} overrides
 * @returns {Record<string, unknown>}
 */
function baseTask(overrides = {}) {
  return {
    id: TASK_ID,
    title: 'E2E Panels Regression Task',
    description: 'Regression test for detail panels',
    created_at: '2026-07-01T00:00:00Z',
    created_by: { username: 'e2e-user' },
    comments: [],
    ai_comments: [],
    container_agent_comments: [],
    assignees: [],
    workspace_id: WORKSPACE_ID,
    ...overrides,
  };
}

/**
 * 设置 cookie + MockEventSource（使页面不因 SSE 挂起）。
 */
async function setupPage(page) {
  await page.context().addCookies([
    { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
    { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
  ]);

  await page.addInitScript(() => {
    class MockEventSource {
      static CONNECTING = 0; static OPEN = 1; static CLOSED = 2;
      constructor(url) { this.url = String(url || ''); this.readyState = MockEventSource.OPEN; this.withCredentials = true; this.onmessage = null; this.onerror = null; this.onclose = null; this._listeners = { open: [] }; setTimeout(() => this._emitOpen(), 20); }
      addEventListener(type, cb) { if (!this._listeners[type]) this._listeners[type] = []; this._listeners[type].push(cb); }
      close() { this.readyState = MockEventSource.CLOSED; if (typeof this.onclose === 'function') this.onclose(); }
      _emitOpen() { const ev = { type: 'open' }; for (const cb of this._listeners.open || []) { try { cb(ev); } catch (_) {} } }
    }
    window.EventSource = MockEventSource;
  });
}

/**
 * 设置 task-detail 路由 handler（所有测试通用的端点）。
 * 返回一个 ({url, method}) => { handled: boolean } 维度的处理函数，
 * 未匹配时由调用方兜底 catch-all。
 *
 * 注意：每个 handler 内通过 include 匹配 URL 并 return 阻断后续逻辑。
 */
function makeTaskDetailApiHandler(taskOverrides = {}) {
  const task = baseTask(taskOverrides);

  return async function handleTaskDetailApi(route) {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 任务详情 GET
    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(task) });
      return true;
    }

    // 协作人员
    if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return true;
    }

    // 进度系统
    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}/${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          columns: [
            { id: 'col-open', name: '开放', wip_limit: 0 },
            { id: 'col-completed', name: '已完成', wip_limit: 0 },
          ],
        }),
      });
      return true;
    }

    // 进度状态选项
    if (url.includes('/workspaces/progress-status-options/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: 'col-open', name: '开放' },
          { id: 'col-completed', name: '已完成' },
        ]),
      });
      return true;
    }

    // 交付物类别
    if (url.includes('/deliverable-categories/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return true;
    }

    // 功能参数
    if (url.includes('/feature-params/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: {} }) });
      return true;
    }

    // cloud/compute: task-ui-context
    if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
      });
      return true;
    }

    // cloud/compute: server-start-history
    if (url.includes('/cloud/compute/server-start-history/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', records: [] }) });
      return true;
    }

    // cloud/compute: layer-graph
    if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ layers_root: '/workspace/layers', bootstrap_layer_id: 'layer-1', layers: [], jobs: [] }),
      });
      return true;
    }

    // subtree API（默认空子树）
    if (url.includes('/subtree/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ summary: null, nodes: [] }),
      });
      return true;
    }

    return false; // 未处理，由调用方兜底
  };
}

/**
 * 展开「任务辅助信息」面板，使 subtree-status / queued-schedule 可见。
 */
async function expandAuxInfo(page) {
  const toggle = page.getByTestId('task-aux-info-toggle');
  await expect(toggle).toBeVisible({ timeout: 15_000 });
  const ariaExpanded = await toggle.getAttribute('aria-expanded');
  if (ariaExpanded === 'false') {
    await toggle.click();
    await expect(page.getByTestId('task-aux-info-body')).toBeVisible({ timeout: 5_000 });
  }
}

// ─────── Test Suite: subtree-status ──────────────────────────

test.describe('OPT-20260719-039: TaskDetail subtree-status error traceId', () => {
  test('父任务改已完成遇开放子任务时断言错误文案与 data-traceId', async ({ page }) => {
    test.setTimeout(60_000);
    await setupPage(page);

    const MAIN_TRACE_ID = 'e2e-subtree-error-trace-20260719';
    let patchCount = 0;

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // subtree API → 有未关闭的子节点（须先于任务详情前缀匹配，subtree URL 以任务路径为前缀）
      if (url.includes('/subtree/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            summary: { total: 2, settled: 1 },
            nodes: [
              { id: 'child-1', title: '子任务A', completed: true, progress_column_name: '已完成' },
              { id: 'child-2', title: '子任务B', completed: false, progress_column_name: '开放' },
            ],
          }),
        });
        return;
      }

      // 任务详情 GET 带 parent_task 标识非顶层
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`)) {
        if (method === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(baseTask({
              parent_task: 'root-parent',
              progress_column_id: 'col-open',
              progress_column_name: '开放',
            })),
          });
          return;
        }
        // PATCH 进度 → 返回 400 含 traceId
        if (method === 'PATCH') {
          patchCount++;
          await route.fulfill({
            status: 400,
            contentType: 'application/json',
            headers: { 'X-Trace-Id': MAIN_TRACE_ID },
            body: JSON.stringify({
              error: '该交付物存在未关闭的下级交付物，请先完成所有下级交付物后再尝试关闭。',
              trace_id: MAIN_TRACE_ID,
            }),
          });
          return;
        }
      }

      // 兜底：通用 handler
      const handled = await makeTaskDetailApiHandler()(route);
      if (!handled) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      }
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');
    await expandAuxInfo(page);

    // 等待 subtree 面板出现
    const subtreePanel = page.getByTestId('task-subtree-status');
    await expect(subtreePanel).toBeVisible({ timeout: 15_000 });
    await expect(page.getByTestId('task-subtree-summary')).toContainText('已关闭 1/2');

    // 找到进度状态 select 并选择「已完成」
    const progressSelect = page.locator('#task-progress-status');
    await expect(progressSelect).toBeVisible({ timeout: 10_000 });
    await progressSelect.selectOption('col-completed');

    // 等待 PATCH 请求
    await expect.poll(() => patchCount, { timeout: 15_000, message: '应触发 PATCH 请求' }).toBeGreaterThanOrEqual(1);

    // 进度错误以内联段落展示（AuxInfoPanel progressStatusError），非全屏弹窗
    const errP = page.locator('p.text-xs.text-red-600', { hasText: '该交付物存在未关闭的下级交付物' });
    await expect(errP).toBeVisible({ timeout: 10_000 });

    // 断言内联错误段有 data-traceId 且值正确
    const traceIdAttr = await errP.getAttribute('data-traceId');
    expect(traceIdAttr).toBe(MAIN_TRACE_ID);

    // 确认 data-traceId 不是中文文案
    expect(traceIdAttr).not.toMatch(/[一-鿿]/);
  });
});

// ─────── Test Suite: queued-schedule toggle (OPT-041 / OPT-20260824-005) ─────────
// 排队调度设置已自任务详情拆出至「工作空间的排队调度」页面：任务详情仅保留
// 加入/离开队列 toggle 与页面入口链接，不再有节奏模态。

test.describe('OPT-20260719-041: TaskDetail queued-schedule toggle', () => {
  test('默认展示加入队列按钮与页面入口链接，无节奏模态', async ({ page }) => {
    test.setTimeout(60_000);
    await setupPage(page);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // 任务详情 GET → 顶层任务 + schedule_rhythm（legacy 字段仍在任务 JSON 中）
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(baseTask({
            parent_task: null,
            schedule_rhythm: {
              enabled: false,
              daily_start: '22:00',
              daily_end: '06:00',
              max_queued_machines: 1,
              auto_close: false,
            },
          })),
        });
        return;
      }

      // 兜底：通用 handler
      const handled = await makeTaskDetailApiHandler()(route);
      if (!handled) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      }
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 队列 toggle 在评论「自动执行」卡，无需展开辅助信息
    const joinBtn = page.getByTestId('queued-auto-run-join');
    await expect(joinBtn).toBeVisible({ timeout: 15_000 });
    await expect(joinBtn).toContainText('加入自动执行队列');

    const pageLink = page.getByTestId('queued-schedule-page-link');
    await expect(pageLink).toBeVisible({ timeout: 5_000 });
    await expect(pageLink).toContainText('自动调度安排');

    const depPicker = page.getByTestId('comment-execution-dependency-picker');
    await expect(depPicker.getByTestId('queued-auto-run-join')).toBeVisible({ timeout: 5_000 });
    await expect(depPicker.getByTestId('comment-dep-queue-serial-hint')).toContainText('一条一条执行');

    const pageLink = page.getByTestId('queued-schedule-page-link');
    await expect(pageLink).toBeVisible({ timeout: 5_000 });
    await expect(pageLink).toContainText('自动调度安排');

    // 节奏设置模态已移除
    await expect(page.getByTestId('task-queued-schedule-modal')).toHaveCount(0);
    await expect(page.getByTestId('schedule-rhythm-enabled')).toHaveCount(0);
  });
});

// ─────── Test Suite: queued-schedule join-queue (OPT-014) ────

test.describe('OPT-20260721-014: TaskDetail queued-schedule join-queue', () => {
  test('入队后 toggle 显示「离开队列」与状态 chip', async ({ page }) => {
    test.setTimeout(60_000);
    await setupPage(page);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      // 任务详情 GET → 已在队列中
      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(baseTask({
            parent_task: null,
            queued_auto_run: true,
            queued_ahead_count: 3,
            queued_auto_run_status: 'queued',
            queued_top_task_id: TASK_ID,
          })),
        });
        return;
      }

      // queue snapshot API
      if (url.includes('/queued-auto-run/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            members: [
              { task_id: 'task-a', title: '前方任务A', status: 'queued' },
              { task_id: TASK_ID, title: '当前任务', status: 'queued' },
              { task_id: 'task-c', title: '后方任务C', status: 'queued' },
            ],
            queued_slots_used: 3,
            max_queued_machines: 5,
            window_message: '22:00-06:00',
          }),
        });
        return;
      }

      // 兜底：通用 handler
      const handled = await makeTaskDetailApiHandler()(route);
      if (!handled) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      }
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    const leaveBtn = page.getByTestId('queued-auto-run-leave');
    await expect(leaveBtn).toBeVisible({ timeout: 15_000 });

    // 断言状态芯片文字「前方还有 3 个任务在等待」
    const statusChip = page.getByTestId('queued-auto-run-status-chip');
    await expect(statusChip).toBeVisible({ timeout: 5_000 });
    await expect(statusChip).toContainText('前方还有 3 个任务在等待');

    const depPicker = page.getByTestId('comment-execution-dependency-picker');
    await expect(depPicker.getByTestId('queued-auto-run-leave')).toBeVisible({ timeout: 5_000 });
    await expect(depPicker.getByTestId('comment-dep-queue-serial-hint')).toContainText('已加入自动执行队列');
    await expect(depPicker.getByTestId('comment-dep-independent')).toBeDisabled();

    // 队列列表已移至工作空间排队调度页面，任务详情不再渲染
    await expect(page.getByTestId('queued-auto-run-list')).toHaveCount(0);
  });
});

// ─────── Test Suite: PeopleGroups create (OPT-020) ──────────

