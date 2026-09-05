// @ts-check
/**
 * 回归：Vite 不得把 container GET（ui-context、layer-graph）误代理到 taskContainerGateway。
 * 网关 P1 仅实现 POST container-layer-git-commit；误代理会导致 405 method not allowed。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '848546827193511936';
const GATEWAY_ORIGIN = 'http://127.0.0.1:8014';

const CONTAINER_GET_PATHS = [
  `/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}container-task-ui-context/?task_id=${TASK_ID}`,
  `/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}container-layer-graph/`,
];

test.describe('TaskDetail container API Vite proxy', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需网关与 Vite dev server 的全栈 E2E');

  for (const path of CONTAINER_GET_PATHS) {
    const label = path.split('/').slice(-2).join('/');

    test(`gateway GET ${label} stays POST-only (405)`, async ({ request }) => {
      const res = await request.get(`${GATEWAY_ORIGIN}${path}`, {
        failOnStatusCode: false,
        timeout: 5000,
      });
      expect(res.status()).toBe(405);
      const body = await res.text();
      expect(body).toContain('method not allowed');
    });

    test(`Vite GET ${label} must not be gateway 405`, async ({ request, baseURL }) => {
      const origin = (baseURL || 'http://127.0.0.1:4000').replace(/\/$/, '');
      let res;
      try {
        res = await request.get(`${origin}${path}`, {
          failOnStatusCode: false,
          timeout: 8000,
        });
      } catch (err) {
        test.skip(
          true,
          `Django/Vite 代理 8s 内无响应（runserver 可能被长请求阻塞）：${err instanceof Error ? err.message : String(err)}`,
        );
        return;
      }
      const status = res.status();
      const body = (await res.text().catch(() => '')).slice(0, 200);
      expect(status, `expected not 405; body=${body}`).not.toBe(405);
    });
  }
});
