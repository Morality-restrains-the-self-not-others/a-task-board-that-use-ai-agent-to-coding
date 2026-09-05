// @ts-check
/**
 * 冷启动全链路 E2E：无实例任务 → POST start-vm-auto → Kafka 链 → instance_id → workbench-link。
 * 默认使用专用任务 PLAYWRIGHT_COLD_START_TASK_ID（无 cloud 实例，勿与 warm 任务混用）。
 *
 * 运行（约 12 分钟，需真实阿里云授权与余额）：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * npx playwright test tests/TaskDetail.start-vm-auto-cold-start.playwright.test.js \
 *   --config=playwright.config.chromium.js
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import {
  SITE,
  COLD_START_TASK_ID,
  COLD_START_WORKSPACE_ID,
  postStartVmAuto,
  pollServerRuntimeStatus,
  waitForInstanceId,
  assertWorkbenchLinkForInstance,
  ensureNoRunningInstance,
} from './helpers/startVmAutoE2e.js';

test.describe('TaskDetail start-vm-auto cold start E2E', () => {
  test('冷启动：start-vm-auto 全链路产生 instance_id 且 workbench-link 可用', async ({ page, request }) => {
    test.setTimeout(900_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081',
    });

    const precheck = await pollServerRuntimeStatus(
      request,
      loginData.token,
      COLD_START_TASK_ID,
      COLD_START_WORKSPACE_ID,
    );
    expect(precheck.status, 'precheck server-runtime-status HTTP').toBe(200);

    if (process.env.PLAYWRIGHT_COLD_START_SKIP_CLEANUP !== '1') {
      await ensureNoRunningInstance(
        request,
        loginData.token,
        COLD_START_TASK_ID,
        COLD_START_WORKSPACE_ID,
      );
    } else {
      const existingInstance = String(precheck.body?.instance_id || '').trim();
      test.skip(
        !!existingInstance,
        `冷启动任务 ${COLD_START_TASK_ID} 已有 instance_id=${existingInstance}，请换 PLAYWRIGHT_COLD_START_TASK_ID 或先释放实例`,
      );
    }

    const start = await postStartVmAuto(
      request,
      loginData.token,
      COLD_START_TASK_ID,
      COLD_START_WORKSPACE_ID,
    );
    expect(start.status, `start-vm-auto body=${JSON.stringify(start.body).slice(0, 400)}`).toBe(200);
    expect(start.body?.status).toBe('success');
    expect(start.body?.event_id, 'async event_id').toBeTruthy();

    const instanceId = await waitForInstanceId(
      request,
      loginData.token,
      COLD_START_TASK_ID,
      COLD_START_WORKSPACE_ID,
    );
    expect(instanceId, 'cloud server instance_id after cold start-vm-auto chain').not.toBe('');

    await assertWorkbenchLinkForInstance(
      instanceId,
      request,
      loginData.token,
      COLD_START_TASK_ID,
      COLD_START_WORKSPACE_ID,
    );
  });
});
