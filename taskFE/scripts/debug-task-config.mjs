// @ts-check
import { chromium } from '@playwright/test';
import { loginViaGatewayApi } from '../tests/helpers/gatewayLoginE2e.js';

const SITE = 'http://183.250.1.132:4000';
const GATEWAY = 'http://183.250.1.132:18081';
const TENANT = '850256677331562496';
const WS = '861623708318031872';
const TASK = 'task_12590983282794675865';

const browser = await chromium.launch();
const page = await browser.newPage();
const loginPassword = process.env.PLAYWRIGHT_TEST_PASSWORD;
if (!loginPassword) {
  throw new Error('PLAYWRIGHT_TEST_PASSWORD is required; do not hardcode secrets');
}
const login = await loginViaGatewayApi(page, {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: loginPassword,
  siteOrigin: SITE,
  gatewayOrigin: GATEWAY,
});

const headers = { Authorization: `Token ${login.token}`, Origin: SITE };

const taskResp = await page.request.get(`${GATEWAY}/api/tenant/${TENANT}/workspace/${WS}/todos/${TASK}/`, { headers });
console.log('TASK', taskResp.status(), (await taskResp.text()).slice(0, 2000));

const prevResp = await page.request.get(`${GATEWAY}/api/tenant/${TENANT}/workspace/${WS}/task/${TASK}/cloud/compute/previous-server-config/`, { headers });
console.log('PREV_CONFIG', prevResp.status(), (await prevResp.text()).slice(0, 2000));

const projectId = JSON.parse(await taskResp.text())?.project_id;
if (projectId) {
  const projResp = await page.request.get(`${GATEWAY}/api/tenant/${TENANT}/projects/${projectId}/`, { headers });
  const projText = await projResp.text();
  console.log('PROJECT', projResp.status(), projText.slice(0, 3000));
  try {
    const tpl = JSON.parse(projText)?.server_run_template;
    console.log('RUN_TEMPLATE', JSON.stringify(tpl, null, 2));
  } catch {
    /* ignore */
  }
}

await browser.close();
