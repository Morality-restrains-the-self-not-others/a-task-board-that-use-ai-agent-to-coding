// @ts-check
/**
 * stop-vm 路由：须经 taskCloudService Go 原生路径，不得 501。
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12590983282794675865';

const ROUTING_FAILURE = 'compute action not yet ported to taskCloudService';

test.describe('TaskDetail stop-vm routing', () => {
  test('POST stop-vm uses Go native path (not 501)', async ({ page, request }) => {
    test.setTimeout(60_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    const apiUrl = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}stop-vm/`;
    const resp = await request.post(apiUrl, {
      headers: {
        Authorization: `Token ${loginData.token}`,
        'Content-Type': 'application/json',
        Origin: SITE,
      },
      data: { task_id: TASK_ID },
    });

    const bodyText = await resp.text();
    expect(bodyText).not.toContain(ROUTING_FAILURE);
    expect(resp.status()).not.toBe(501);
  });
});
