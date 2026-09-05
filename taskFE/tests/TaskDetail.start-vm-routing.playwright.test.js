// @ts-check
/**
 * 核验「启动服务器」在已有 VPC/交换机/安全组场景走 start-vm 时，
 * 网关经 taskCloudService 原生编排（Phase 3i），不得再返回 501 not-yet-ported。
 *
 * 运行：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * npx playwright test tests/TaskDetail.start-vm-routing.playwright.test.js \
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

/** 已有网络资源的 start-vm 请求体骨架（需替换为租户真实 vpc/vswitch/sg） */
const START_VM_PAYLOAD = {
  task_id: TASK_ID,
  container_image_id: process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '859671040643174400',
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
  region_id: process.env.PLAYWRIGHT_REGION_ID || 'cn-qingdao',
  zone_id: process.env.PLAYWRIGHT_ZONE_ID || 'cn-qingdao-b',
  vpc_id: process.env.PLAYWRIGHT_VPC_ID || 'vpc-placeholder',
  vswitch_id: process.env.PLAYWRIGHT_VSWITCH_ID || 'vsw-placeholder',
  security_group_id: process.env.PLAYWRIGHT_SECURITY_GROUP_ID || 'sg-placeholder',
  cloud_platform_id: '1',
  authorization_id: process.env.PLAYWRIGHT_AUTHORIZATION_ID || '862031128628060160',
  selected_instance: 'ecs.c6.large',
  bandwidth: 5,
  bandwidth_charging_mode: 'PayByTraffic',
};

test.describe('TaskDetail start-vm routing', () => {
  test('POST start-vm uses Go native path (not 501 not-yet-ported)', async ({ page, request }) => {
    test.setTimeout(120_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    const apiUrl = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}start-vm/`;
    const resp = await request.post(apiUrl, {
      headers: {
        Authorization: `Token ${loginData.token}`,
        'Content-Type': 'application/json',
        Origin: SITE,
      },
      data: START_VM_PAYLOAD,
    });

    const bodyText = await resp.text();
    expect(bodyText, 'response body').not.toContain(ROUTING_FAILURE);
    expect(resp.status(), `start-vm status body=${bodyText.slice(0, 400)}`).not.toBe(501);

    let parsed = {};
    try {
      parsed = JSON.parse(bodyText);
    } catch {
      /* non-json acceptable for infra errors */
    }
    if (parsed.message) {
      expect(String(parsed.message)).not.toContain('compute action not yet ported');
    }
    // Go 原生路径成功受理时的固定文案（Django 遗留为「虚拟机启动请求已提交」）
    if (resp.status() === 200 && parsed.status === 'success') {
      expect(String(parsed.message)).toMatch(/启动虚拟机请求已提交/);
    }
  });
});
