// @ts-check
/**
 * 回归：relayToTrae 启动日志按 task_id 隔离；Fork 后新任务页不应沿用源任务日志。
 */
import { test, expect } from '@playwright/test';

const TENANT_ID = '827923618468040704';
const WORKSPACE_ID = '827923618602258432';
const SOURCE_TASK_ID = '848180922808590336';
const FORKED_TASK_ID = '848180922808590337';
const SOURCE_TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${SOURCE_TASK_ID}/?relayToTrae=true`;
const FORKED_TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${FORKED_TASK_ID}/?relayToTrae=true`;
const SOURCE_LOG_LINE = 'relay-startup-log-for-source-task';

function buildRelayStatusBody(taskId) {
  if (taskId === SOURCE_TASK_ID) {
    return {
      running: true,
      online_service_up: true,
      active_task_id: SOURCE_TASK_ID,
      log_task_id: SOURCE_TASK_ID,
      logs: [SOURCE_LOG_LINE],
    };
  }
  return {
    running: true,
    online_service_up: true,
    active_task_id: SOURCE_TASK_ID,
    log_task_id: taskId,
    logs: [],
  };
}

function taskDetailTodoBody(taskId, forkFrom = null) {
  return {
    id: taskId,
    title: forkFrom ? `Fork of ${forkFrom}` : 'Relay source task',
    workspace_id: WORKSPACE_ID,
    owner: '1',
    assignees: [],
    comments: [],
    ai_comments: [],
    fork_from: forkFrom,
  };
}

async function installRelayApiMocks(context) {
  await context.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    const taskMatch = url.match(/\/todos\/(\d+)\//);
    const routeTaskId = taskMatch ? taskMatch[1] : '';

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}`) && method === 'GET' && routeTaskId) {
      const forkFrom = routeTaskId === FORKED_TASK_ID ? SOURCE_TASK_ID : null;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(taskDetailTodoBody(routeTaskId, forkFrom)),
      });
      return;
    }

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}`) && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: FORKED_TASK_ID }),
      });
      return;
    }

    if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'img-relay-1', name: 'Relay Image', version: '1.0' }]),
      });
      return;
    }

    if (url.includes('/relay-to-trae/env-prepare/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          env: {
            TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8001',
            BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
            ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
          },
        }),
      });
      return;
    }

    if (url.includes('/relay-to-trae/health/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'ok' }),
      });
      return;
    }

    if (url.includes('/relay-to-trae/register/') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'ok' }),
      });
      return;
    }

    if (url.includes('/relay-to-trae/status/') && method === 'GET') {
      const pathTaskMatch = url.match(/\/task\/(\d+)\/cloud\/compute\/relay-to-trae\/status\//);
      const queryTaskMatch = url.match(/[?&]task_id=(\d+)/);
      const statusTaskId = queryTaskMatch?.[1] || pathTaskMatch?.[1] || SOURCE_TASK_ID;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(buildRelayStatusBody(statusTaskId)),
      });
      return;
    }

    if (url.includes('/server-runtime-status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
      });
      return;
    }

    if (url.includes('/server-content/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
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
        body: JSON.stringify({ status: 'success', container_endpoint_registered: false }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

test.describe('TaskDetail relayToTrae fork startup logs', () => {
  test('Fork 后新标签页不应显示源任务的启动日志', async ({ page, context }) => {
    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await installRelayApiMocks(context);

    await page.goto(SOURCE_TASK_PATH);
    await page.waitForLoadState('domcontentloaded');
    await page.getByTestId('server-config-relay-direct-tab').click();
    await expect(page.getByText(SOURCE_LOG_LINE)).toBeVisible({ timeout: 10000 });

    const forkedPagePromise = context.waitForEvent('page');
    await page.locator('#task-fork-btn').click();
    await expect(page.getByTestId('fork-auto-run-confirm-modal')).toBeVisible({ timeout: 5000 });
    await page.getByTestId('fork-mode-fork-only').check();
    await page.getByTestId('fork-confirm-submit').click();
    const forkedPage = await forkedPagePromise;
    await forkedPage.waitForLoadState('domcontentloaded');
    await forkedPage.getByTestId('server-config-relay-direct-tab').click();
    await expect(forkedPage.getByText(SOURCE_LOG_LINE)).toHaveCount(0, { timeout: 10000 });
  });
});
