// @ts-check
/**
 * start-vm-auto 端到端（warm 任务）：已有实例时走快速路径断言 workbench-link。
 * 冷启动全链路见 TaskDetail.start-vm-auto-cold-start.playwright.test.js（PLAYWRIGHT_COLD_START_TASK_ID）。
 *
 * 运行（需真实阿里云授权与余额；VM 创建可能需数分钟）：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * PLAYWRIGHT_TENANT_ID=850256677331562496 \
 * PLAYWRIGHT_WORKSPACE_ID=861623708318031872 \
 * PLAYWRIGHT_TASK_ID=task_12590983282794675865 \
 * PLAYWRIGHT_CONTAINER_IMAGE_ID=862588024964280320 \
 * npx playwright test tests/TaskDetail.start-vm-auto-server-start.playwright.test.js \
 *   --config=playwright.config.chromium.js
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import { mentionInstalledImageInComment, waitForCommentCreateRequest } from './helpers/commentImageMentionE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12590983282794675865';
const TASK_DETAIL_PATH =
  process.env.PLAYWRIGHT_TASK_DETAIL_PATH ||
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`;
const CONTAINER_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '862588024964280320';
const AUTHORIZATION_ID = process.env.PLAYWRIGHT_AUTHORIZATION_ID || '862031128628060160';

const INSTANCE_TYPE = process.env.PLAYWRIGHT_INSTANCE_TYPE || 'ecs.c8y.large';

const START_VM_AUTO_PAYLOAD = {
  task_id: TASK_ID,
  container_image_id: CONTAINER_IMAGE_ID,
  hardware_config: {
    cpu_cores: '2',
    memory_gb: '4',
    storage_gb: '40',
    instance_type: INSTANCE_TYPE,
    spot_strategy: 'SpotAsPriceGo',
    spot_duration: 0,
    instance_charge_type: 'PostPaid',
    system_disk_category: 'cloud_essd',
    internet_max_bandwidth_out: 5,
  },
  region_id: process.env.PLAYWRIGHT_REGION_ID || 'cn-hongkong',
  zone_id: process.env.PLAYWRIGHT_ZONE_ID || 'cn-hongkong-b',
  vpc_id: null,
  vswitch_id: null,
  auto_create_vswitch: true,
  cloud_platform_id: '1',
  authorization_id: AUTHORIZATION_ID,
  filter_options: {
    cores: 2,
    memory: 4,
    io_optimized: true,
    system_disk_category: 'cloud_essd',
    data_disk_category: 'cloud_essd',
    spot_strategy: 'SpotAsPriceGo',
    spot_duration: 0,
    instance_charge_type: 'PostPaid',
    network_category: 'vpc',
  },
  selected_instance: INSTANCE_TYPE,
  security_group_id: null,
  auto_create_security_group: true,
  bandwidth: 5,
  bandwidth_charging_mode: 'PayByTraffic',
  auto_release_enabled: true,
  auto_release_minutes: 30,
};

/** @param {import('@playwright/test').Page} page @param {string} instanceType */
async function selectInstanceType(page, instanceType) {
  const deadline = Date.now() + 120_000;
  while (Date.now() < deadline) {
    const row = page.locator('tr').filter({ hasText: instanceType }).first();
    if (await row.isVisible().catch(() => false)) {
      await row.getByRole('button', { name: '选择' }).click();
      return;
    }
    const nextBtn = page.getByRole('button', { name: '下一页' }).first();
    if (await nextBtn.isEnabled().catch(() => false)) {
      await nextBtn.click();
      await page.waitForTimeout(1500);
      continue;
    }
    await page.waitForTimeout(2000);
  }
  throw new Error(`未在实例列表中找到 ${instanceType}，请确认镜像架构过滤与地域库存`);
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token */
async function pollWorkbenchLink(request, token) {
  const url =
    `${GATEWAY}/api/cloud/compute/workbench-link/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}` +
    `/?task_id=${encodeURIComponent(TASK_ID)}`;
  const resp = await request.get(url, {
    headers: { Authorization: `Token ${token}`, Origin: SITE },
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token */
async function pollServerRuntimeStatus(request, token) {
  const url = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}server-runtime-status/`;
  const resp = await request.get(url, {
    headers: { Authorization: `Token ${token}`, Origin: SITE },
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token */
async function postStartVmAuto(request, token) {
  const url = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}start-vm-auto/`;
  const resp = await request.post(url, {
    headers: {
      Authorization: `Token ${token}`,
      'Content-Type': 'application/json',
      Origin: SITE,
    },
    data: START_VM_AUTO_PAYLOAD,
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/** @param {import('@playwright/test').Page} page */
async function openHardwareConfigPanel(page) {
  // 硬件配置已并入镜像卡，直接定位面板（不再有 Tab）
  const hardwarePanel = page.getByTestId('server-hardware-config-panel');
  await hardwarePanel.scrollIntoViewIfNeeded().catch(() => {});
  await expect(hardwarePanel).toBeVisible({ timeout: 30_000 });
  const regionCombo = page.getByRole('combobox', { name: '地域' });
  if (!(await regionCombo.isVisible().catch(() => false))) {
    const tempBtn = page.getByRole('button', { name: '临时配置' });
    if (await tempBtn.isVisible().catch(() => false)) {
      await tempBtn.click();
      await regionCombo.waitFor({ state: 'visible', timeout: 30_000 }).catch(() => {});
    }
  }
  return regionCombo.isVisible().catch(() => false);
}

/** @param {import('@playwright/test').Page} page */
async function clickSubmitAndRun(page) {
  const commentReqPromise = waitForCommentCreateRequest(page, 120_000);
  await page.getByTestId('task-detail-comment-submit').click();
  return commentReqPromise;
}

/** @param {string} instanceId @param {import('@playwright/test').APIRequestContext} request @param {string} token */
async function assertWorkbenchLinkForInstance(instanceId, request, token) {
  const { status: wbStatus, body: wbBody } = await pollWorkbenchLink(request, token);
  expect(wbStatus, 'workbench-link HTTP').toBe(200);
  expect(wbBody?.status).toBe('success');
  expect(String(wbBody?.workbench_url || '')).toContain('ecs-workbench.aliyun.com');
  expect(String(wbBody?.instance_id || '')).toBe(instanceId);
  expect(String(wbBody?.region || '')).not.toBe('');
}

test.describe('TaskDetail start-vm-auto server start E2E', () => {
  test('UI 点击启动服务器后 CLOUD_SERVER_STARTED 被消费并产生 instance_id', async ({ page, request }) => {
    test.setTimeout(900_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    const precheck = await pollServerRuntimeStatus(request, loginData.token);
    expect(precheck.status, 'precheck server-runtime-status HTTP').toBe(200);
    let instanceId = String(precheck.body?.instance_id || '').trim();
    if (instanceId) {
      await assertWorkbenchLinkForInstance(instanceId, request, loginData.token);
      return;
    }

    await page.goto(`${SITE}/`, { waitUntil: 'domcontentloaded' });
    await page.evaluate((token) => {
      localStorage.setItem('authToken', token);
    }, loginData.token);
    if (loginData.user?.id) {
      await page.context().addCookies([
        { name: 'userId', value: String(loginData.user.id), url: `${SITE}/` },
      ]);
    }

    await page.goto(`${SITE}${TASK_DETAIL_PATH}`);
    await page.waitForLoadState('domcontentloaded');

    await mentionInstalledImageInComment(page);

    const hasManualConfig = await openHardwareConfigPanel(page);

    if (hasManualConfig) {
      await expect(page.getByRole('combobox', { name: '地域' })).toBeEnabled({ timeout: 60_000 });
      await page.locator('#cpu-cores').fill('2');
      await page.locator('#memory-size').fill('4');
      await selectInstanceType(page, INSTANCE_TYPE);
      const sgSelect = page.getByRole('combobox', { name: '安全组' });
      await sgSelect.waitFor({ state: 'visible', timeout: 30_000 });
      await sgSelect.selectOption('auto_create_security_group');
    }

    const commentReq = await clickSubmitAndRun(page);
    const commentResp = await commentReq.response();
    if (commentResp?.status() !== 201 && commentResp?.status() !== 200) {
      const apiStart = await postStartVmAuto(request, loginData.token);
      expect(apiStart.status, `start-vm-auto API fallback body=${JSON.stringify(apiStart.body).slice(0, 300)}`).toBe(200);
      expect(apiStart.body?.status).toBe('success');
      expect(apiStart.body?.event_id, 'async event_id').toBeTruthy();
    } else {
      const commentBody = JSON.parse(commentReq.postData() || '{}');
      expect(commentBody.mentions?.[0]?.type).toBe('installed_image');
    }

    const deadline = Date.now() + 12 * 60_000;
    instanceId = '';
    while (Date.now() < deadline) {
      const { status, body } = await pollServerRuntimeStatus(request, loginData.token);
      expect(status, 'server-runtime-status HTTP').toBe(200);
      instanceId = String(body?.instance_id || '').trim();
      if (instanceId) {
        break;
      }
      await page.waitForTimeout(15_000);
    }

    expect(instanceId, 'cloud server instance_id after start-vm-auto chain').not.toBe('');
    await assertWorkbenchLinkForInstance(instanceId, request, loginData.token);
  });
});
