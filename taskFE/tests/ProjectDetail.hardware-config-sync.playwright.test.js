// @ts-check
/**
 * E2E: 项目详情页 — 保存运行模版时 hardware_config 应与所选实例/过滤器一致
 *
 * 回归：选择 2核8GB 过滤器并选中 ecs.g6.large 后，PATCH 请求中 hardware_config
 * 曾错误保留默认 1核1GB，与 filter_options / selected_instance 不一致。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { gotoAuthenticatedPath } from './helpers/gatewayLoginE2e.js';
import { waitForLoginLegalPolicies } from './helpers/remoteLoginE2e.js';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const portConfig = loadPortConfig();

const BASE_URL =
  (process.env.SITE_BASE || process.env.BASE_URL || '').replace(/\/$/, '') ||
  portConfig.vue?.publicBaseUrl ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;

const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.TEST_WORKSPACE_ID || '861623708318031872';
const PROJECT_ID = process.env.TEST_PROJECT_HARDWARE_SYNC_ID || 'proj_-5910410547773912751';
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const MOCK_AUTH_ID = '862031128628060160';

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const MOCK_PROJECT = {
  id: PROJECT_ID,
  name: 'E2E hardware_config 同步',
  description: 'desc',
  tags: ['somanyad'],
  git_repos: [],
  container_image_id: '862588024964280320',
  container_image: 'trae0630',
  workspaces: [WORKSPACE_ID],
  server_run_template: {
    template_id: '',
    label: 'ljy0808-阿里云 · cn-hongkong',
    platform: 'aliyun',
    authorization_id: MOCK_AUTH_ID,
    cloud_platform_id: '1',
    region: 'cn-hongkong',
    zone_id: 'cn-hongkong-d',
    vpc_id: '',
    vswitch_id: '',
    security_group_id: '',
    payment_type: 'PostPaid',
    bandwidth_charging_mode: 'PayByTraffic',
    bandwidth: 5,
    hardware_config: { cpu_cores: '1', memory_gb: '1', storage_gb: '40' },
    filter_options: {
      cores: 2,
      memory: 8,
      io_optimized: true,
      system_disk_category: 'cloud_essd',
      data_disk_category: 'cloud_essd',
      spot_strategy: 'SpotAsPriceGo',
      image_architecture: 'x86_64',
      spot_duration: 0,
      instance_charge_type: 'PostPaid',
      network_category: 'vpc',
    },
    selected_instance: 'ecs.g6.large',
  },
  company: TENANT_ID,
  created_at: '2026-07-07T11:29:24Z',
  updated_at: '2026-07-08T06:59:24Z',
};

/** @param {import('@playwright/test').Page} page */
async function loginAndOpenDetail(page) {
  await waitForLoginLegalPolicies(page, `${BASE_URL}/auth/login/`);
  await playwrightLoginWithLegalAccept(page, {
    email: CREDENTIALS.email,
    password: CREDENTIALS.password,
    baseURL: BASE_URL,
    skipGoto: true,
  });
  await gotoAuthenticatedPath(page, DETAIL_URL);
  await page.waitForLoadState('networkidle').catch(() => null);
}

/** @param {import('@playwright/test').Page} page */
async function registerCloudMocks(page, patchState, options = {}) {
  const { project = MOCK_PROJECT } = options;
  await page.route('**/*', async (route) => {
    const headers = { ...route.request().headers() };
    delete headers['x-requested-with'];

    const url = route.request().url();
    const method = route.request().method();

    if (url.includes('/api/') && url.includes(`/projects/${PROJECT_ID}`) && !url.includes('/branches')) {
      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(project),
        });
        return;
      }
      if (method === 'PATCH') {
        patchState.payload = route.request().postDataJSON();
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...project,
            ...patchState.payload,
            server_run_template:
              patchState.payload?.server_run_template || project.server_run_template,
          }),
        });
        return;
      }
    }

    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}?ids=`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: WORKSPACE_ID, name: '默认工作空间' }]),
      });
      return;
    }

    if (/\/api\/tenant\/[^/]+\/workspaces\/$/.test(url.split('?')[0]) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
      return;
    }

    if (url.includes(`/api/cloud/server-config-default/tenant_id/${TENANT_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '861845264907628544',
              platform: 'aliyun',
              region: 'cn-hongkong',
              zone_id: 'cn-hongkong-d',
              authorization_id: MOCK_AUTH_ID,
            },
          ],
        }),
      });
      return;
    }

    if (
      url.includes(`/workspaces/${WORKSPACE_ID}/cloud/platforms/`) &&
      !url.includes('default-config') &&
      method === 'GET'
    ) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          platforms: [
            {
              id: 1,
              platform_type: 'aliyun',
              platform_type_display: '阿里云',
              remark: 'ljy0808',
              platform_name: '阿里云',
              authorization_id: MOCK_AUTH_ID,
            },
          ],
        }),
      });
      return;
    }

    if (url.includes(`/workspaces/${WORKSPACE_ID}/cloud/platforms/default-config/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          default_configs: [
            {
              platform_type: 'aliyun',
              authorization_id: MOCK_AUTH_ID,
              remark: 'ljy0808',
              config: {
                region: 'cn-hongkong',
                zone_id: 'cn-hongkong-d',
              },
            },
          ],
        }),
      });
      return;
    }

    if (/\/cloud-platform\/[^/]+\/cloud\/regions\//.test(url) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'cn-hongkong', name: '中国香港' }]),
      });
      return;
    }

    if (url.includes('/cloud/server-images/vpcs/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ vpc_id: 'vpc-e2e', vpc_name: 'E2E VPC' }]),
      });
      return;
    }

    if (url.includes('/cloud/server-images/vswitches/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            vswitch_id: 'vsw-e2e',
            zone_id: 'cn-hongkong-d',
            vswitch_name: 'E2E VSW',
          },
        ]),
      });
      return;
    }

    if (url.includes('/cloud/server-images/security-groups/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'sg-e2e', name: 'E2E SG' }]),
      });
      return;
    }

    if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/zones/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'cn-hongkong-d', name: '香港 D' }]),
      });
      return;
    }

    if (/\/cloud-platform\/[^/]+/(?:cloud/)?available-instances\//.test(url) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          InstanceTypes: {
            InstanceType: [
              {
                InstanceTypeId: 'ecs.g6.large',
                CpuCoreCount: 2,
                MemorySize: 8,
                Status: 'Available',
                StorageTypes: ['cloud_essd'],
              },
            ],
          },
        }),
      });
      return;
    }

    await route.continue({ headers });
  });
}

test.describe('ProjectDetail hardware_config 与实例规格同步（Mock API）', () => {
  test('选择 2核8GB 过滤器并选中 ecs.g6.large 保存时 hardware_config 应同步为 2/8', async ({ page }) => {
    test.setTimeout(120_000);
    const patchState = { payload: null };

    await registerCloudMocks(page, patchState);

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('#cloud-platform-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.locator('#region-select')).toHaveValue('cn-hongkong', { timeout: 60_000 });

    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.g6.large', { exact: true })).toBeVisible({ timeout: 30_000 });

    await page.locator('#cpu-cores').fill('2');
    await page.locator('#memory-size').fill('8');

    const selectBtn = hardwarePanel.getByRole('button', { name: '选择' }).first();
    await selectBtn.click();
    await expect(hardwarePanel.getByRole('button', { name: '已选' })).toBeVisible();

    await page.getByRole('button', { name: '保存运行模版' }).click();

    await expect.poll(() => patchState.payload, { timeout: 20_000 }).not.toBeNull();
    const tpl = patchState.payload.server_run_template;

    expect(tpl.selected_instance || tpl.hardware_config?.instance_type).toBe('ecs.g6.large');
    expect(String(tpl.hardware_config?.cpu_cores)).toBe('2');
    expect(String(tpl.hardware_config?.memory_gb)).toBe('8');
    expect(tpl.filter_options?.cores).toBe(2);
    expect(tpl.filter_options?.memory).toBe(8);
  });

  test('已保存 selected_instance 加载后应自动恢复「已选」状态', async ({ page }) => {
    test.setTimeout(120_000);
    const patchState = { payload: null };

    await registerCloudMocks(page, patchState);

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('#cloud-platform-select')).toBeEnabled({ timeout: 60_000 });

    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.g6.large', { exact: true })).toBeVisible({ timeout: 30_000 });
    await expect(hardwarePanel.getByRole('button', { name: '已选' })).toBeVisible({ timeout: 30_000 });

    await page.getByRole('button', { name: '保存运行模版' }).click();

    await expect.poll(() => patchState.payload, { timeout: 20_000 }).not.toBeNull();
    const tpl = patchState.payload.server_run_template;

    expect(tpl.selected_instance || tpl.hardware_config?.instance_type).toBe('ecs.g6.large');
    expect(String(tpl.hardware_config?.cpu_cores)).toBe('2');
    expect(String(tpl.hardware_config?.memory_gb)).toBe('8');
  });

  test('hardware_config 为占位 1/1 时重新保存应使用 filter_options 纠正 CPU/内存', async ({ page }) => {
    test.setTimeout(120_000);
    const patchState = { payload: null };

    await registerCloudMocks(page, patchState);

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('#cloud-platform-select')).toBeEnabled({ timeout: 60_000 });

    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.g6.large', { exact: true })).toBeVisible({ timeout: 30_000 });

    await page.getByRole('button', { name: '保存运行模版' }).click();

    await expect.poll(() => patchState.payload, { timeout: 20_000 }).not.toBeNull();
    const tpl = patchState.payload.server_run_template;

    expect(String(tpl.hardware_config?.cpu_cores)).toBe('2');
    expect(String(tpl.hardware_config?.memory_gb)).toBe('8');
    expect(tpl.filter_options?.cores).toBe(2);
    expect(tpl.filter_options?.memory).toBe(8);
  });
});
