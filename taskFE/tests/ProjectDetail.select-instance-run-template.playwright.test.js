// @ts-check
/**
 * E2E: 项目详情页 — 从可用实例列表选择节点并保存为运行模版（Mock 云 API）
 *
 * 覆盖修复：运行模版模式下应用已保存配置后应拉取可用实例、可选节点并 PATCH 保存。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import {
  loginViaGatewayApi,
  gotoAuthenticatedPath,
} from './helpers/gatewayLoginE2e.js';
import { waitForLoginLegalPolicies } from './helpers/remoteLoginE2e.js';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const portConfig = loadPortConfig();

const BASE_URL =
  (process.env.SITE_BASE || process.env.BASE_URL || '').replace(/\/$/, '') ||
  portConfig.vue?.publicBaseUrl ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;

const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.TEST_WORKSPACE_ID || '857903329669984256';
const PROJECT_ID = process.env.TEST_PROJECT_ID || '861581450509701120';
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const MOCK_AUTH_ID = '862031128628060160';
const STALE_AUTH_ID = '861830261815164928';

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const MOCK_PROJECT = {
  id: PROJECT_ID,
  name: 'E2E 项目详情选实例模版',
  description: 'desc',
  tags: [],
  git_repos: [],
  container_image_id: '',
  container_image: '',
  workspaces: [WORKSPACE_ID],
  server_run_template: {
    template_id: '861845264907628544',
    label: 'aliyun · cn-hongkong',
    platform: 'aliyun',
    authorization_id: MOCK_AUTH_ID,
    cloud_platform_id: '1',
    region: 'cn-hongkong',
    zone_id: 'cn-hongkong-b',
    vpc_id: 'vpc-saved-e2e',
    vswitch_id: 'vsw-saved-e2e',
    security_group_id: 'sg-saved-e2e',
    hardware_config: { cpu_cores: '2', memory_gb: '4', storage_gb: '40', instance_type: 'ecs.e2e.small' },
    selected_instance: 'ecs.e2e.small',
    filter_options: {
      cores: '2',
      memory: '4',
      system_disk_category: 'cloud_essd',
      data_disk_category: 'cloud_essd',
      spot_strategy: 'SpotAsPriceGo',
    },
  },
  company: TENANT_ID,
  created_at: '2026-07-05T12:00:00Z',
  updated_at: '2026-07-05T12:00:00Z',
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
  const { platformsWithoutAuthId = false, project = MOCK_PROJECT, requestCounter = null } = options
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
              zone_id: 'cn-hongkong-b',
              vpc_id: 'vpc-saved-e2e',
              vswitch_id: 'vsw-saved-e2e',
              security_group_id: 'sg-saved-e2e',
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
      if (requestCounter) requestCounter.platforms += 1
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
              remark: 'e2e',
              platform_name: '阿里云',
              ...(platformsWithoutAuthId ? {} : { authorization_id: MOCK_AUTH_ID }),
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
              remark: 'e2e',
              config: {
                region: 'cn-hongkong',
                zone_id: 'cn-hongkong-b',
                vpc_id: 'vpc-saved-e2e',
                vswitch_id: 'vsw-saved-e2e',
                security_group_id: 'sg-saved-e2e',
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
        body: JSON.stringify([{ vpc_id: 'vpc-saved-e2e', vpc_name: 'Saved VPC' }]),
      });
      return;
    }

    if (url.includes('/cloud/server-images/vswitches/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            vswitch_id: 'vsw-saved-e2e',
            zone_id: 'cn-hongkong-b',
            vswitch_name: 'Saved VSW',
          },
        ]),
      });
      return;
    }

    if (url.includes('/cloud/server-images/security-groups/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'sg-saved-e2e', name: 'Saved SG' }]),
      });
      return;
    }

    if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/zones/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'cn-hongkong-b', name: '香港 B' }]),
      });
      return;
    }

    if (/\/cloud-platform\/[^/]+\/cloud\/available-instances\//.test(url) && method === 'GET') {
      if (requestCounter) requestCounter.availableInstances += 1
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          InstanceTypes: {
            InstanceType: [
              {
                InstanceTypeId: 'ecs.e2e.small',
                CpuCoreCount: 2,
                MemorySize: 4,
                Status: 'Available',
                StorageTypes: ['cloud_essd'],
              },
              {
                InstanceTypeId: 'ecs.e2e.large',
                CpuCoreCount: 4,
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

test.describe('ProjectDetail 可用实例选模版（Mock API）', () => {
  test('页面加载时 workspace cloud/platforms 仅请求一次', async ({ page }) => {
    test.setTimeout(120_000);
    const requestCounter = { platforms: 0 };
    await registerCloudMocks(page, { payload: null }, { requestCounter });

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('#cloud-platform-select')).toBeEnabled({ timeout: 60_000 });
    await page.waitForTimeout(1500);

    expect(requestCounter.platforms).toBe(1);
  });

  test('已保存运行模版加载时应自动恢复已选实例', async ({ page }) => {
    test.setTimeout(120_000);

    await registerCloudMocks(page, { payload: null });

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('#cloud-platform-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.getByText('暂无可用实例')).not.toBeVisible({ timeout: 30_000 });

    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.e2e.small', { exact: true })).toBeVisible({ timeout: 30_000 });
    await expect(hardwarePanel.getByRole('button', { name: '已选' })).toBeVisible({ timeout: 30_000 });
  });

  test('已保存运行模版时应展示可用实例、可选新节点并 PATCH 保存', async ({ page }) => {
    test.setTimeout(120_000);
    const patchState = { payload: null };

    await registerCloudMocks(page, patchState);

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });

    await expect(page.locator('#cloud-platform-select')).toBeVisible();
    await expect(page.locator('#cloud-platform-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.locator('#cloud-platform-select')).not.toHaveValue('');
    await expect(page.locator('#region-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.locator('#vpc-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.locator('#zone-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.locator('#region-select')).toHaveValue('cn-hongkong', { timeout: 60_000 });
    await expect(page.locator('#vpc-select')).toHaveValue('vpc-saved-e2e');

    await expect(page.getByText('暂无可用实例')).not.toBeVisible({ timeout: 30_000 });

    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.e2e.small', { exact: true })).toBeVisible();
    await expect(hardwarePanel.getByText('ecs.e2e.large', { exact: true })).toBeVisible();

    const selectLargeBtn = hardwarePanel.getByRole('button', { name: '选择' }).nth(1);
    await selectLargeBtn.click();
    await expect(hardwarePanel.getByRole('button', { name: '已选' })).toBeVisible();

    await page.getByRole('button', { name: '保存运行模版' }).click();

    await expect.poll(() => patchState.payload, { timeout: 20_000 }).not.toBeNull();
    const tpl = patchState.payload.server_run_template;
    expect(tpl.region).toBe('cn-hongkong');
    expect(tpl.selected_instance || tpl.hardware_config?.instance_type).toBe('ecs.e2e.large');
    expect(String(tpl.hardware_config?.cpu_cores)).toBe('4');
    expect(String(tpl.hardware_config?.memory_gb)).toBe('8');
  });

  test('无已保存模版时选择预设应同步服务器硬件配置', async ({ page }) => {
    test.setTimeout(120_000);

    const emptyProject = {
      ...MOCK_PROJECT,
      server_run_template: {},
    };

    const patchState = { payload: null };
    await registerCloudMocks(page, patchState, { project: emptyProject });

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('#project-run-template-select')).toBeVisible();

    await page.locator('#project-run-template-select').selectOption('861845264907628544');

    await expect(page.locator('#cloud-platform-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.locator('#cloud-platform-select')).not.toHaveValue('');
    await expect(page.locator('#region-select')).toHaveValue('cn-hongkong', { timeout: 60_000 });
    await expect(page.locator('#vpc-select')).toHaveValue('vpc-saved-e2e', { timeout: 60_000 });
    await expect(page.locator('#zone-select')).toHaveValue('cn-hongkong-b:vsw-saved-e2e', { timeout: 60_000 });
    await expect(page.getByText('暂无可用实例')).not.toBeVisible({ timeout: 30_000 });
  });

  test('已保存运行模版（含过期 authorization_id）加载时应触发可用实例 API 并展示地域', async ({ page }) => {
    test.setTimeout(120_000);
    const requestCounter = { platforms: 0, availableInstances: 0 };

    const staleAuthProject = {
      ...MOCK_PROJECT,
      server_run_template: {
        ...MOCK_PROJECT.server_run_template,
        authorization_id: STALE_AUTH_ID,
      },
    };

    await registerCloudMocks(page, { payload: null }, {
      project: staleAuthProject,
      requestCounter,
    });

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('#region-select')).toBeEnabled({ timeout: 60_000 });
    await expect(page.locator('#region-select')).toHaveValue('cn-hongkong', { timeout: 60_000 });
    await expect(page.locator('#vpc-select')).toHaveValue('vpc-saved-e2e', { timeout: 60_000 });
    await expect(page.locator('#zone-select')).toHaveValue('cn-hongkong-b:vsw-saved-e2e', { timeout: 60_000 });

    await expect.poll(() => requestCounter.availableInstances, { timeout: 30_000 }).toBeGreaterThan(0);
    await expect(page.getByText('暂无可用实例')).not.toBeVisible({ timeout: 30_000 });

    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.e2e.small', { exact: true })).toBeVisible();
  });

  test('云平台 API 无 authorization_id 时选择预设仍应同步', async ({ page }) => {
    test.setTimeout(120_000);

    const emptyProject = { ...MOCK_PROJECT, server_run_template: {} };
    const patchState = { payload: null };

    await registerCloudMocks(page, patchState, {
      platformsWithoutAuthId: true,
      project: emptyProject,
    });

    await loginAndOpenDetail(page);
    await page.locator('#project-run-template-select').selectOption('861845264907628544');
    await expect(page.locator('#region-select')).toHaveValue('cn-hongkong', { timeout: 60_000 });
    await expect(page.getByText('应用默认模版失败')).not.toBeVisible({ timeout: 5_000 });
  });
});
