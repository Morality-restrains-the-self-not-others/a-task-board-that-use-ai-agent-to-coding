// @ts-check
/**
 * 核验「启动服务器」走 start-vm-auto 时，网关经 taskCloudService 原生编排（Phase 3i），
 * 不得再返回 501 compute action not yet ported。
 *
 * UI 点击链路见 TaskDetail.start-vm-auto-mock-ui.playwright.test.js（全 mock，更稳定）。
 *
 * 运行：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * npx playwright test tests/TaskDetail.start-vm-auto-routing.playwright.test.js \
 *   --config=playwright.config.chromium.js
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '857903329669984256';
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '860371538948571136';

const ROUTING_FAILURE = 'compute action not yet ported to taskCloudService';

/** 与任务页自动创建 VPC/交换机/安全组场景一致的请求体骨架 */
const START_VM_AUTO_PAYLOAD = {
  task_id: TASK_ID,
  container_image_id: process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '859671040643174400',
  hardware_config: {
    cpu_cores: '1',
    memory_gb: '1',
    storage_gb: '40',
    generation: '',
    instance_type: 'ecs.c6.large',
    spot_strategy: 'SpotAsPriceGo',
    spot_duration: 0,
    instance_charge_type: 'PostPaid',
    system_disk_category: 'cloud_essd',
    internet_max_bandwidth_out: 5,
  },
  region_id: 'cn-qingdao',
  zone_id: 'cn-qingdao-b',
  vpc_id: null,
  vswitch_id: null,
  auto_create_vswitch: true,
  cloud_platform_id: '1',
  authorization_id: process.env.PLAYWRIGHT_AUTHORIZATION_ID || '862031128628060160',
  filter_options: {
    cores: '2',
    memory: '4',
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

test.describe('TaskDetail start-vm-auto routing', () => {
  test('POST start-vm-auto uses Go native path (not 501 not-yet-ported)', async ({ page, request }) => {
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
    expect(bodyText, 'response body').not.toContain(ROUTING_FAILURE);
    expect(resp.status(), `start-vm-auto status body=${bodyText.slice(0, 400)}`).not.toBe(501);

    let parsed = {};
    try {
      parsed = JSON.parse(bodyText);
    } catch {
      /* non-json acceptable for infra errors */
    }
    if (parsed.message) {
      expect(String(parsed.message)).not.toContain('compute action not yet ported');
    }
  });
});
