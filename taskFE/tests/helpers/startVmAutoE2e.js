// @ts-check

export const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://183.250.1.132:4000').replace(/\/$/, '');
export const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://183.250.1.132:18081').replace(/\/$/, '');
export const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';

/** 冷启动专用：无 cloud 实例的任务（勿与已有实例任务混用） */
export const COLD_START_TASK_ID =
  process.env.PLAYWRIGHT_COLD_START_TASK_ID || '860371538948571136';
export const COLD_START_WORKSPACE_ID =
  process.env.PLAYWRIGHT_COLD_START_WORKSPACE_ID || '857903329669984256';

/** 已有实例快速回归任务 */
export const WARM_TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12590983282794675865';
export const WARM_WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';

export const CONTAINER_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '862588024964280320';
export const AUTHORIZATION_ID = process.env.PLAYWRIGHT_AUTHORIZATION_ID || '862031128628060160';
export const INSTANCE_TYPE = process.env.PLAYWRIGHT_INSTANCE_TYPE || 'ecs.g6.large';

/** @param {string} taskId @param {string} workspaceId */
export function buildStartVmAutoPayload(taskId, workspaceId) {
  return {
    task_id: taskId,
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
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId */
export async function pollServerRuntimeStatus(request, token, taskId, workspaceId) {
  const url = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${workspaceId}/task_id/${taskId}server-runtime-status/`;
  const resp = await request.get(url, {
    headers: { Authorization: `Token ${token}`, Origin: SITE },
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId */
export async function pollWorkbenchLink(request, token, taskId, workspaceId) {
  const url =
    `${GATEWAY}/api/cloud/tenant_id/${TENANT_ID}/workspace_id/${workspaceId}` +
    `/cloud/compute/workbench-link/?task_id=${encodeURIComponent(taskId)}`;
  const resp = await request.get(url, {
    headers: { Authorization: `Token ${token}`, Origin: SITE },
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId */
export async function postStartVmAuto(request, token, taskId, workspaceId) {
  const url = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${workspaceId}start-vm-auto/`;
  const resp = await request.post(url, {
    headers: {
      Authorization: `Token ${token}`,
      'Content-Type': 'application/json',
      Origin: SITE,
    },
    data: buildStartVmAutoPayload(taskId, workspaceId),
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/** @param {string} instanceId @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId */
export async function assertWorkbenchLinkForInstance(instanceId, request, token, taskId, workspaceId) {
  const { expect } = await import('@playwright/test');
  const { status: wbStatus, body: wbBody } = await pollWorkbenchLink(request, token, taskId, workspaceId);
  expect(wbStatus, 'workbench-link HTTP').toBe(200);
  expect(wbBody?.status).toBe('success');
  expect(String(wbBody?.workbench_url || '')).toContain('ecs-workbench.aliyun.com');
  expect(String(wbBody?.instance_id || '')).toBe(instanceId);
  expect(String(wbBody?.region || '')).not.toBe('');
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId */
export async function postStopVm(request, token, taskId, workspaceId) {
  const url = `${GATEWAY}/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${workspaceId}stop-vm/`;
  const resp = await request.post(url, {
    headers: {
      Authorization: `Token ${token}`,
      'Content-Type': 'application/json',
      Origin: SITE,
    },
    data: { task_id: taskId },
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/**
 * Nightly 复跑：若冷启动任务已有实例则先 stop-vm，等待 instance_id 清空。
 * @param {import('@playwright/test').APIRequestContext} request
 * @param {string} token
 * @param {string} taskId
 * @param {string} workspaceId
 * @param {number} [timeoutMs]
 */
export async function ensureNoRunningInstance(request, token, taskId, workspaceId, timeoutMs = 8 * 60_000) {
  const { body } = await pollServerRuntimeStatus(request, token, taskId, workspaceId);
  const existing = String(body?.instance_id || '').trim();
  if (!existing) {
    return;
  }
  await postStopVm(request, token, taskId, workspaceId);
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const snap = await pollServerRuntimeStatus(request, token, taskId, workspaceId);
    if (!String(snap.body?.instance_id || '').trim()) {
      return;
    }
    await new Promise((r) => setTimeout(r, 15_000));
  }
  const { expect } = await import('@playwright/test');
  expect(existing, 'cold start task still has instance after stop-vm wait').toBe('');
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId */
export async function pollServerContent(request, token, taskId, workspaceId) {
  const url =
    `${GATEWAY}/api/cloud/tenant_id/${TENANT_ID}/workspace_id/${workspaceId}` +
    `/cloud/compute/server-content/?task_id=${encodeURIComponent(taskId)}`;
  const resp = await request.get(url, {
    headers: { Authorization: `Token ${token}`, Origin: SITE },
  });
  const body = await resp.json().catch(() => ({}));
  return { status: resp.status(), body };
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId @param {number} [timeoutMs] */
export async function waitForServerContentReady(request, token, taskId, workspaceId, timeoutMs = 20 * 60_000) {
  const { expect } = await import('@playwright/test');
  const deadline = Date.now() + timeoutMs;
  let last = { status: 0, body: {} };
  while (Date.now() < deadline) {
    last = await pollServerContent(request, token, taskId, workspaceId);
    if (last.status === 200 && last.body?.status === 'success') {
      return last;
    }
    await new Promise((r) => setTimeout(r, 20_000));
  }
  expect(
    last.status,
    `server-content not ready within timeout; last body=${JSON.stringify(last.body).slice(0, 400)}`
  ).toBe(200);
  return last;
}

/** @param {import('@playwright/test').APIRequestContext} request @param {string} token @param {string} taskId @param {string} workspaceId @param {number} [timeoutMs] */
export async function waitForInstanceId(request, token, taskId, workspaceId, timeoutMs = 12 * 60_000) {
  const { expect } = await import('@playwright/test');
  const deadline = Date.now() + timeoutMs;
  let instanceId = '';
  while (Date.now() < deadline) {
    const { status, body } = await pollServerRuntimeStatus(request, token, taskId, workspaceId);
    expect(status, 'server-runtime-status HTTP').toBe(200);
    instanceId = String(body?.instance_id || '').trim();
    if (instanceId) {
      return instanceId;
    }
    await new Promise((r) => setTimeout(r, 15_000));
  }
  return instanceId;
}
