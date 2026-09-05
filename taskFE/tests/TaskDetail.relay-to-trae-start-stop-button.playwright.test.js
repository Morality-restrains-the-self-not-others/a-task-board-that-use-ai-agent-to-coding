// @ts-check
/**
 * 回归：relayToTrae 直接启动页面，点击「启动」后「停止」按钮应持续可见。
 *
 * 修复背景：startRelayToTrae 成功后未设置 relayToTraeOnlineServiceUp，
 * 导致 stop 按钮在启动完成后消失。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const SITE_ORIGIN = String(process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://localhost:4000').replace(/\/$/, '');
const TASK_ID = '843742455533076480';
const ACCESS_CODE = 'u827923618451263488';
const TASK_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?relayToTrae=true&accessCode=${ACCESS_CODE}`;

const RELAY_BASE = `/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}relay-to-trae`;

test('relayToTrae stop button should stay visible after start completes', async ({ page, context }) => {
  test.setTimeout(60000);

  await context.addCookies([
    { name: 'userId', value: 'e2e-user', url: `${SITE_ORIGIN}/` },
    { name: 'csrftoken', value: 'e2e-csrf', url: `${SITE_ORIGIN}/` },
  ]);

  let startCalled = false;

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // User profile
    if (url.includes('/api/user/') && url.includes('/profile/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'e2e-user',
          username: 'e2e-user',
          email: 'e2e@test.com',
        }),
      });
      return;
    }

    // Task detail
    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'RelayToTrae Test Task',
          description: 'E2E test for relayToTrae start/stop button',
          created_at: '2026-01-01T00:00:00Z',
          created_by: { username: 'e2e-user' },
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
          projects: [],
          parameters: { repo_clone_git_identities: {} },
        }),
      });
      return;
    }

    // relayToTrae health
    if (url.includes(`${RELAY_BASE}/health/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      return;
    }

    // relayToTrae register
    if (url.includes(`${RELAY_BASE}/register/`) && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    // relayToTrae status (not running initially)
    if (url.includes(`${RELAY_BASE}/status/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, online_service_up: false }),
      });
      return;
    }

    // relayToTrae env-prepare
    if (url.includes(`${RELAY_BASE}/env-prepare/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'ok',
          env: {
            TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
            BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8766',
            ACCESS_TOKEN: 'mock-token-xxx',
          },
        }),
      });
      return;
    }

    // relayToTrae start
    if (url.includes(`${RELAY_BASE}/start/`) && method === 'POST') {
      startCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'accepted',
          request_id: 'mock-request-id-for-stop-btn',
        }),
      });
      return;
    }

    // Catch-all: return empty JSON
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  // Collect page errors (ignore SSE connection errors which are expected in mock)
  const pageErrors = [];
  page.on('pageerror', (error) => {
    const msg = String(error?.message || error);
    if (msg.includes('text/event-stream') || msg.includes('SSE')) return;
    pageErrors.push(msg);
  });

  await page.goto(`${SITE_ORIGIN}${TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForTimeout(4000);

  // Verify the relayDirect tab is active and start button is visible
  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  await expect(startBtn).toBeVisible({ timeout: 15000 });
  await expect(startBtn).toBeEnabled({ timeout: 5000 });

  // Stop button should NOT be visible before start
  const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
  await expect(stopBtn).not.toBeVisible({ timeout: 3000 });

  // Click start
  await startBtn.click();

  // Wait for the start API to be called and the UI to update
  await expect(() => expect(startCalled).toBe(true)).toPass({ timeout: 10000 });

  // Wait a bit for the finally block to complete
  await page.waitForTimeout(2000);

  // THE KEY ASSERTION: stop button must be visible after start completes
  await expect(stopBtn).toBeVisible({ timeout: 10000 });

  // Start button should still be visible
  await expect(startBtn).toBeVisible({ timeout: 3000 });

  // No page errors
  expect(pageErrors, `Unexpected page errors: ${pageErrors.join(' | ')}`).toEqual([]);
});

test('relayToTrae start pending then stop should reset start button state', async ({ page, context }) => {
  test.setTimeout(60000);

  await context.addCookies([
    { name: 'userId', value: 'e2e-user', url: `${SITE_ORIGIN}/` },
    { name: 'csrftoken', value: 'e2e-csrf', url: `${SITE_ORIGIN}/` },
  ]);

  let registerRequestCount = 0;
  let stopCalled = false;

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes('/api/user/') && url.includes('/profile/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'e2e-user', username: 'e2e-user' }),
      });
      return;
    }

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'RelayToTrae Start Pending Stop',
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
        }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/health/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      return;
    }

    if (url.includes(`${RELAY_BASE}/register/`) && method === 'POST') {
      registerRequestCount += 1;
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    if (url.includes(`${RELAY_BASE}/status/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, online_service_up: false, orphan_port: false }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/env-prepare/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'ok',
          env: {
            TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
            BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8766',
            ACCESS_TOKEN: 'mock-token-xxx',
          },
        }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/token-init/`) && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'ok' }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/start/`) && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'accepted',
          request_id: 'mock-request-id',
        }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/stop/`) && method === 'POST') {
      stopCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'ok' }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(`${SITE_ORIGIN}${TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForTimeout(2000);

  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
  await expect(startBtn).toBeVisible({ timeout: 15000 });
  await startBtn.click();

  await expect(startBtn).toHaveText('启动中...');
  await expect(stopBtn).toBeVisible({ timeout: 10000 });
  await stopBtn.click();

  await expect(() => expect(stopCalled).toBe(true)).toPass({ timeout: 10000 });
  await expect(() => expect(registerRequestCount).toBeGreaterThan(0)).toPass({ timeout: 10000 });
  await expect(startBtn).toHaveText('启动', { timeout: 10000 });
  await expect(startBtn).toBeEnabled({ timeout: 10000 });
});

test('relayToTrae start should show mapped error message by error_code', async ({ page, context }) => {
  test.setTimeout(60000);

  await context.addCookies([
    { name: 'userId', value: 'e2e-user', url: `${SITE_ORIGIN}/` },
    { name: 'csrftoken', value: 'e2e-csrf', url: `${SITE_ORIGIN}/` },
  ]);

  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    if (url.includes('/api/user/') && url.includes('/profile/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'e2e-user', username: 'e2e-user' }),
      });
      return;
    }

    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'RelayToTrae Start Error Mapping',
          comments: [],
          ai_comments: [],
          assignees: [],
          workspace_id: WORKSPACE_ID,
        }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/health/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      return;
    }

    if (url.includes(`${RELAY_BASE}/register/`) && method === 'POST') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"ok"}' });
      return;
    }

    if (url.includes(`${RELAY_BASE}/status/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, online_service_up: false }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/env-prepare/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'ok',
          env: {
            TASK_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8765',
            BUSINESS_API_ENDPOINT_ORIGIN: 'http://127.0.0.1:8766',
            ACCESS_TOKEN: 'mock-token-xxx',
          },
        }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/token-init/`) && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'ok' }),
      });
      return;
    }

    if (url.includes(`${RELAY_BASE}/start/`) && method === 'POST') {
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'error',
          detail: 'URL 中的租户/工作空间/任务与令牌不匹配',
          error_code: 'TOKEN_SCOPE_MISMATCH',
          trace_id: 'trace-playwright-mismatch',
        }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(`${SITE_ORIGIN}${TASK_PATH}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForTimeout(2000);

  const startBtn = page.getByTestId('relay-to-trae-start-btn');
  await expect(startBtn).toBeVisible({ timeout: 15000 });
  await startBtn.click();

  await expect(page.getByText('任务上下文不一致，请刷新页面后重试')).toBeVisible({ timeout: 10000 });
  await expect(startBtn).toBeVisible({ timeout: 3000 });
  await expect(startBtn).toBeEnabled({ timeout: 3000 });
});
