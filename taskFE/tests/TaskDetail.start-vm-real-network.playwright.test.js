// @ts-check
/**
 * start-vm 全链路（真实 VPC/交换机/安全组）：先经网关拉取云资源，再 POST start-vm。
 * 断言 Go 原生路径受理（非 501），不要求 VM 真正创建成功（避免云侧库存/余额阻塞 CI）。
 *
 * 运行：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * npx playwright test tests/TaskDetail.start-vm-real-network.playwright.test.js \
 *   --config=playwright.config.chromium.js
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '857903329669984256';
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '860371538948571136';
const REGION_CANDIDATES = (
  process.env.PLAYWRIGHT_REGION_ID
    ? [process.env.PLAYWRIGHT_REGION_ID]
    : ['cn-hongkong', 'cn-qingdao', 'cn-hangzhou', 'ap-southeast-1']
);
const AUTHORIZATION_ID = process.env.PLAYWRIGHT_AUTHORIZATION_ID || '862031128628060160';
const CONTAINER_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '859671040643174400';

const ROUTING_FAILURE = 'compute action not yet ported to taskCloudService';

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} path */
async function gatewayGet(request, token, path) {
  const resp = await request.get(`${GATEWAY}${path}`, {
    headers: { Authorization: `Token ${token}`, Origin: SITE, Accept: 'application/json' },
  });
  const text = await resp.text();
  let data = [];
  try {
    data = JSON.parse(text);
  } catch {
    data = [];
  }
  return { status: resp.status(), data, text };
}

/**
 * @param {unknown} raw
 * @returns {unknown[]}
 */
function normalizeListPayload(raw) {
  if (Array.isArray(raw)) return raw;
  if (raw && typeof raw === 'object') {
    const o = /** @type {Record<string, unknown>} */ (raw);
    if (Array.isArray(o.results)) return o.results;
    if (Array.isArray(o.data)) return o.data;
    if (Array.isArray(o.vpcs)) return o.vpcs;
  }
  return [];
}

/**
 * @param {unknown} data
 * @returns {{ vpcId: string } | null}
 */
function pickVpcId(data) {
  const vpcs = normalizeListPayload(data).filter(
    (v) => v && typeof v === 'object' && (v.vpc_id || v.id || v.VpcId),
  );
  const vpc = vpcs.find((v) => {
    const id = String(v.vpc_id || v.id || v.VpcId);
    return id && id !== 'auto_create_vpc';
  });
  if (!vpc || typeof vpc !== 'object') return null;
  return { vpcId: String(vpc.vpc_id || vpc.id || vpc.VpcId) };
}

/**
 * @param {Record<string, unknown>} loginData
 * @returns {string}
 */
function resolveAuthToken(loginData) {
  return String(loginData?.token || loginData?.key || '').trim();
}

/**
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} token
 * @param {string} workspaceId
 * @returns {Promise<{ regionId: string, vpcId: string, vswitchId: string, securityGroupId: string, zoneId: string } | null>}
 */
async function resolveNetworkFromWorkspaceDefault(request, token, workspaceId) {
  const res = await gatewayGet(
    request,
    token,
    `/api/projects/workspaces/tenant_id/${TENANT_ID}${workspaceId}/cloud/platforms/default-config/`,
  );
  if (res.status !== 200 || !res.data || typeof res.data !== 'object') {
    return null;
  }
  const configs = Array.isArray(res.data.default_configs) ? res.data.default_configs : [];
  const cfg = configs.find((c) => c && typeof c === 'object' && c.config && c.config.vpc_id);
  if (!cfg || typeof cfg !== 'object') return null;
  const config = /** @type {Record<string, string>} */ (cfg.config || {});
  const vpcId = String(config.vpc_id || '').trim();
  if (!vpcId || vpcId === 'auto_create_vpc') return null;
  return {
    regionId: String(config.region || config.region_id || '').trim(),
    vpcId,
    vswitchId: String(config.vswitch_id || '').trim(),
    securityGroupId: String(config.security_group_id || '').trim(),
    zoneId: String(config.zone_id || '').trim(),
  };
}

/**
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} token
 * @param {string} workspaceId
 * @param {string} taskId
 */
async function resolveNetworkFromPreviousConfig(request, token, workspaceId, taskId) {
  const res = await gatewayGet(
    request,
    token,
    `/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${workspaceId}previous-server-config/?task_id=${encodeURIComponent(taskId)}`,
  );
  if (res.status !== 200 || !res.data || typeof res.data !== 'object') {
    return null;
  }
  const body = /** @type {Record<string, unknown>} */ (res.data);
  const prev = body.previous_config || body.config || body;
  if (!prev || typeof prev !== 'object') return null;
  const config = /** @type {Record<string, string>} */ (prev);
  const vpcId = String(config.vpc_id || '').trim();
  if (!vpcId || vpcId === 'auto_create_vpc') return null;
  return {
    regionId: String(config.region || config.region_id || '').trim(),
    vpcId,
    vswitchId: String(config.vswitch_id || '').trim(),
    securityGroupId: String(config.security_group_id || '').trim(),
    zoneId: String(config.zone_id || '').trim(),
  };
}

test.describe('TaskDetail start-vm real network E2E', () => {
  test('discover VPC/SG from cloud API then POST start-vm on Go native path', async ({ page, request }) => {
    test.setTimeout(180_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });
    const token = resolveAuthToken(loginData);
    expect(token, 'login token').toBeTruthy();

    let regionId = '';
    let vpcId = process.env.PLAYWRIGHT_VPC_ID || '';
    let vswitchId = process.env.PLAYWRIGHT_VSWITCH_ID || '';
    let securityGroupId = process.env.PLAYWRIGHT_SECURITY_GROUP_ID || '';
    let zoneId = process.env.PLAYWRIGHT_ZONE_ID || '';

    if (!vpcId) {
      for (const candidate of REGION_CANDIDATES) {
        const vpcRes = await gatewayGet(
          request,
          token,
          `/api/cloud/server-images/vpcs/tenant_id/${TENANT_ID}/?region_id=${candidate}&authorization_id=${AUTHORIZATION_ID}`,
        );
        if (vpcRes.status !== 200) continue;
        if (vpcRes.data && typeof vpcRes.data === 'object' && vpcRes.data.status === 'error') continue;
        const picked = pickVpcId(vpcRes.data);
        if (picked?.vpcId) {
          regionId = candidate;
          vpcId = picked.vpcId;
          break;
        }
      }
    } else {
      regionId = REGION_CANDIDATES[0];
    }

    if (!vpcId) {
      const fromDefault = await resolveNetworkFromWorkspaceDefault(request, token, WORKSPACE_ID);
      if (fromDefault?.vpcId) {
        regionId = fromDefault.regionId || regionId;
        vpcId = fromDefault.vpcId;
        vswitchId = vswitchId || fromDefault.vswitchId;
        securityGroupId = securityGroupId || fromDefault.securityGroupId;
        zoneId = zoneId || fromDefault.zoneId;
      }
    }

    if (!vpcId) {
      const fromPrev = await resolveNetworkFromPreviousConfig(request, token, WORKSPACE_ID, TASK_ID);
      if (fromPrev?.vpcId) {
        regionId = fromPrev.regionId || regionId;
        vpcId = fromPrev.vpcId;
        vswitchId = vswitchId || fromPrev.vswitchId;
        securityGroupId = securityGroupId || fromPrev.securityGroupId;
        zoneId = zoneId || fromPrev.zoneId;
      }
    }

    if (!vpcId) {
      const apiUrl = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}start-vm/`;
      const resp = await request.post(apiUrl, {
        headers: {
          Authorization: `Token ${token}`,
          'Content-Type': 'application/json',
          Origin: SITE,
        },
        data: {
          task_id: TASK_ID,
          container_image_id: CONTAINER_IMAGE_ID,
          region_id: REGION_CANDIDATES[0],
          vpc_id: 'vpc-e2e-probe',
          vswitch_id: 'vsw-e2e-probe',
          security_group_id: 'sg-e2e-probe',
          authorization_id: AUTHORIZATION_ID,
        },
      });
      const bodyText = await resp.text();
      expect(bodyText).not.toContain(ROUTING_FAILURE);
      expect(resp.status()).not.toBe(501);
      expect([200, 400, 402, 502]).toContain(resp.status());
      return;
    }

    if (!vswitchId || !securityGroupId) {
      const vswRes = await gatewayGet(
        request,
        token,
        `/api/cloud/server-images/vswitches/tenant_id/${TENANT_ID}/?region_id=${regionId}&vpc_id=${vpcId}&authorization_id=${AUTHORIZATION_ID}`,
      );
      expect(vswRes.status).toBe(200);
      const vswitches = normalizeListPayload(vswRes.data);
      const vswitch = vswitches.find((v) => v && typeof v === 'object' && (v.id || v.vswitch_id || v.VSwitchId));
      if (!vswitchId) {
        vswitchId = String(vswitch?.id || vswitch?.vswitch_id || vswitch?.VSwitchId || '');
      }

      const sgRes = await gatewayGet(
        request,
        token,
        `/api/cloud/server-images/security-groups/tenant_id/${TENANT_ID}/?region_id=${regionId}&vpc_id=${vpcId}&authorization_id=${AUTHORIZATION_ID}`,
      );
      expect(sgRes.status).toBe(200);
      const groups = normalizeListPayload(sgRes.data);
      const sg = groups.find((g) => g && typeof g === 'object' && (g.id || g.security_group_id || g.SecurityGroupId));
      if (!securityGroupId) {
        securityGroupId = String(sg?.id || sg?.security_group_id || sg?.SecurityGroupId || '');
      }
      if (!zoneId && vswitch) {
        zoneId = String(vswitch?.zone_id || vswitch?.ZoneId || `${regionId}-b`);
      }
    }

    expect(vswitchId, 'need vswitch in VPC').not.toBe('');
    expect(securityGroupId, 'need security group in VPC').not.toBe('');
    if (!zoneId) {
      zoneId = `${regionId}-b`;
    }

    const payload = {
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
      region_id: regionId,
      zone_id: zoneId,
      vpc_id: vpcId,
      vswitch_id: vswitchId,
      security_group_id: securityGroupId,
      cloud_platform_id: '1',
      authorization_id: AUTHORIZATION_ID,
      selected_instance: 'ecs.c6.large',
      bandwidth: 5,
      bandwidth_charging_mode: 'PayByTraffic',
    };

    const apiUrl = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}start-vm/`;
    const resp = await request.post(apiUrl, {
      headers: {
        Authorization: `Token ${token}`,
        'Content-Type': 'application/json',
        Origin: SITE,
      },
      data: payload,
    });

    const bodyText = await resp.text();
    expect(bodyText, 'response body').not.toContain(ROUTING_FAILURE);
    expect(resp.status(), `start-vm status body=${bodyText.slice(0, 500)}`).not.toBe(501);

    let parsed = {};
    try {
      parsed = JSON.parse(bodyText);
    } catch {
      /* allow non-json infra errors */
    }
    if (parsed.message) {
      expect(String(parsed.message)).not.toContain('compute action not yet ported');
    }
    if (resp.status() === 200 && parsed.status === 'success') {
      expect(String(parsed.message)).toMatch(/启动虚拟机请求已提交/);
      expect(parsed.event_id, 'async event_id on acceptance').toBeTruthy();
    }
  });
});
