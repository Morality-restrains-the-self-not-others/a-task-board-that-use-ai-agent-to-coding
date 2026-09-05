// @ts-check
/**
 * start-vm-auto 在请求体含 authorization_id 时不应再误报「未配置云平台授权」。
 * 根因：CPA 已迁至 taskCloudService SQLite，Django persist 须带 company_id 查 Go facade。
 *
 * 运行：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * npx playwright test tests/TaskDetail.start-vm-auto-authorization.playwright.test.js \
 *   --config=playwright.config.chromium.js
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12590983282794675865';
const AUTHORIZATION_ID = process.env.PLAYWRIGHT_AUTHORIZATION_ID || '862031128628060160';
const CONTAINER_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '862412768836349952';

const START_VM_AUTO_PAYLOAD = {
  task_id: TASK_ID,
  container_image_id: CONTAINER_IMAGE_ID,
  hardware_config: {
    cpu_cores: '1',
    memory_gb: '1',
    storage_gb: '40',
    instance_type: 'ecs.c6.large',
    spot_strategy: 'SpotAsPriceGo',
    spot_duration: 0,
    instance_charge_type: 'PostPaid',
    system_disk_category: 'cloud_essd',
    internet_max_bandwidth_out: 5,
  },
  region_id: 'cn-hongkong',
  zone_id: 'cn-hongkong-b',
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
  selected_instance: 'ecs.c6.large',
  security_group_id: null,
  auto_create_security_group: true,
  bandwidth: 5,
  bandwidth_charging_mode: 'PayByTraffic',
  auto_release_enabled: true,
  auto_release_minutes: 30,
};

test.describe('TaskDetail start-vm-auto authorization', () => {
  test('POST start-vm-auto with authorization_id does not return missing-auth error', async ({ page, request }) => {
    test.setTimeout(120_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    const apiUrl = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}start-vm-auto/`;
    const resp = await request.post(apiUrl, {
      headers: {
        Authorization: `Token ${loginData.token}`,
        'Content-Type': 'application/json',
        Origin: SITE,
      },
      data: START_VM_AUTO_PAYLOAD,
    });

    const bodyText = await resp.text();
    let parsed = {};
    try {
      parsed = JSON.parse(bodyText);
    } catch {
      /* non-json */
    }

    expect(bodyText, 'response body').not.toContain('未配置云平台授权');
    if (parsed.message) {
      expect(String(parsed.message)).not.toBe('未配置云平台授权');
    }
    expect(resp.status(), `start-vm-auto status body=${bodyText.slice(0, 500)}`).not.toBe(400);
    if (parsed.status === 'success') {
      expect(parsed).toHaveProperty('event_id');
    }
  });
});
