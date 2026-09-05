// @ts-check
/**
 * workbench-link 路由：网关经 taskCloudService 原生实现，不得 501 not-yet-ported。
 *
 * 运行（任务须已有 cloud_server_configs 与真实 instance_id 方可断言 200）：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * PLAYWRIGHT_TASK_ID=task_12590983282794675865 \
 * npx playwright test tests/TaskDetail.workbench-link-routing.playwright.test.js \
 *   --config=playwright.config.chromium.js
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12590983282794675865';

const ROUTING_FAILURE = 'compute action not yet ported to taskCloudService';

test.describe('TaskDetail workbench-link routing', () => {
  test('GET workbench-link uses Go native path (not 501 not-yet-ported)', async ({ page, request }) => {
    test.setTimeout(60_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    const apiUrl =
      `${GATEWAY}/api/cloud/compute/workbench-link/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}` +
      `/?task_id=${encodeURIComponent(TASK_ID)}`;
    const resp = await request.get(apiUrl, {
      headers: {
        Authorization: `Token ${loginData.token}`,
        Origin: SITE,
      },
    });

    const bodyText = await resp.text();
    expect(bodyText, 'response body').not.toContain(ROUTING_FAILURE);
    expect(resp.status(), `workbench-link status body=${bodyText.slice(0, 400)}`).not.toBe(501);

    let parsed = {};
    try {
      parsed = JSON.parse(bodyText);
    } catch {
      /* non-json acceptable for infra errors */
    }
    if (parsed.message) {
      expect(String(parsed.message)).not.toContain('compute action not yet ported');
    }

    if (resp.status() === 200) {
      expect(parsed.status).toBe('success');
      expect(String(parsed.workbench_url || '')).toContain('ecs-workbench.aliyun.com');
      expect(String(parsed.instance_id || '')).not.toBe('');
      expect(String(parsed.region || '')).not.toBe('');
    } else if (resp.status() === 404) {
      expect(parsed.message || '').toMatch(/未找到|服务器配置/);
    }
  });
});
