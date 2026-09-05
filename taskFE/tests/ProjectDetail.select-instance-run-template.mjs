// @ts-check
/**
 * 独立脚本：项目详情页从可用实例列表选节点保存模版（不依赖 channel:chrome）
 * SITE_BASE=http://183.250.1.132:4000 node taskFE/tests/ProjectDetail.select-instance-run-template.mjs
 */
import { chromium } from '@playwright/test';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';
import {
  loginViaGatewayApi,
  gotoAuthenticatedPath,
  readE2eOrigins,
} from './helpers/gatewayLoginE2e.js';
import { installApisixCorsWorkaround } from './helpers/remoteLoginE2e.js';

const portConfig = loadPortConfig();
const localHost = (host) => (host === 'localhost' ? '127.0.0.1' : host);
const BASE_URL =
  (process.env.SITE_BASE || process.env.BASE_URL || '').replace(/\/$/, '') ||
  portConfig.vue?.publicBaseUrl ||
  `http://${localHost(portConfig.vue.host)}:${portConfig.vue.port}`;
const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const PROJECT_ID = process.env.TEST_PROJECT_ID || '861581450509701120';
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD ,
};

async function main() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  const consoleErrors = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });

  let patchPayload = null;
  await page.route(`**/api/projects/tenant_id/${TENANT_ID}${PROJECT_ID}/`, async (route) => {
    if (route.request().method() === 'PATCH') {
      patchPayload = route.request().postDataJSON();
      const getRes = await route.fetch();
      const body = await getRes.json().catch(() => ({}));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...body,
          ...patchPayload,
          server_run_template: patchPayload?.server_run_template || body.server_run_template,
        }),
      });
      return;
    }
    await route.continue();
  });

  await installApisixCorsWorkaround(page);
  await loginViaGatewayApi(page, {
    email: CREDENTIALS.email,
    password: CREDENTIALS.password,
    siteOrigin: BASE_URL,
    gatewayOrigin: process.env.PLAYWRIGHT_GATEWAY_ORIGIN || readE2eOrigins().gatewayOrigin,
  });
  await gotoAuthenticatedPath(page, DETAIL_URL);
  await page.waitForLoadState('networkidle', { timeout: 90_000 }).catch(() => null);
  await page.waitForTimeout(5000);
  console.log('current url:', page.url());
  await page.screenshot({ path: '/tmp/project-detail-e2e.png', fullPage: true });

  const panelVisible = await page
    .locator('[data-testid="project-run-template-panel"]')
    .isVisible()
    .catch(() => false);
  console.log('panel visible:', panelVisible);
  if (!panelVisible) {
    const bodyText = await page.locator('body').innerText().catch(() => '');
    console.log('body snippet:', bodyText.slice(0, 500));
  }

  await page.waitForSelector('[data-testid="project-run-template-panel"]', { timeout: 60_000 });

  const platformSelect = page.locator('#cloud-platform-select');
  await page.waitForFunction(
    () => document.querySelector('#cloud-platform-select')?.options?.length > 0,
    null,
    { timeout: 90_000 },
  ).catch(() => null);

  const platformCount = await platformSelect.locator('option').count();
  console.log('cloud platform options:', platformCount);

  const regionDisabled = await page.locator('#region-select').isDisabled();
  const regionOptions = await page.locator('#region-select option').count();
  console.log('region disabled:', regionDisabled, 'options:', regionOptions);

  const instanceHeading = await page.locator('h4').filter({ hasText: '可用实例列表' }).innerText().catch(() => '');
  console.log('instance heading:', instanceHeading);

  await page.waitForResponse(
    (r) => r.url().includes('/cloud/available-instances/') && r.status() === 200,
    { timeout: 120_000 },
  ).catch((e) => console.log('no available-instances response:', e.message));

  const noInstances = await page.getByText('暂无可用实例').isVisible().catch(() => false);
  console.log('暂无可用实例 visible:', noInstances);

  const selectBtn = page
    .locator('[data-testid="server-hardware-config-panel"] button')
    .filter({ hasText: /^选择$/ })
    .first();

  if (await selectBtn.isVisible({ timeout: 15_000 }).catch(() => false)) {
    const instanceTypeText = await selectBtn
      .locator('xpath=ancestor::div[contains(@class,"border-gray-200")][1]')
      .locator('span.font-medium')
      .first()
      .innerText();
    console.log('selecting instance:', instanceTypeText);
    await selectBtn.click();
    await page.getByRole('button', { name: '保存运行模版' }).click();
    await page.waitForTimeout(3000);
    console.log('patch payload:', JSON.stringify(patchPayload, null, 2));
  } else {
    console.log('no select button found');
  }

  if (consoleErrors.length) {
    console.log('console errors:', consoleErrors.slice(0, 10));
  }

  await browser.close();
  if (!patchPayload) {
    process.exit(1);
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
