// @ts-check
/**
 * E2E: 项目详情页 — 可用实例列表应展示 cpu_cores / memory_gb（Go 扁平数组响应格式）
 *
 * 回归：taskCloudService available-instances 曾返回 cpu_cores/memory_gb 全为 0。
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
const WORKSPACE_ID = process.env.TEST_WORKSPACE_ID || '857903329669984256';
const PROJECT_ID = process.env.TEST_PROJECT_ID || '861581450509701120';
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const MOCK_AUTH_ID = '862031128628060160';

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const MOCK_PROJECT = {
  id: PROJECT_ID,
  name: 'E2E 可用实例 CPU 内存展示',
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
async function registerCloudMocks(page) {
  await page.route('**/*', async (route) => {
    const headers = { ...route.request().headers() };
    delete headers['x-requested-with'];

    const url = route.request().url();
    const method = route.request().method();

    if (url.includes('/api/') && url.includes(`/projects/${PROJECT_ID}`) && !url.includes('/branches') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_PROJECT),
      });
      return;
    }

    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}?ids=`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E Workspace' }]),
      });
      return;
    }

    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/`) && method === 'GET') {
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
              authorization_id: MOCK_AUTH_ID,
              remark: 'e2e',
              platform_name: '阿里云',
            },
          ],
        }),
      });
      return;
    }

    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/default-config/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', default_configs: [] }),
      });
      return;
    }

    if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/regions/`) && method === 'GET') {
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
          { vswitch_id: 'vsw-saved-e2e', zone_id: 'cn-hongkong-b', vswitch_name: 'Saved VSW' },
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

    if (/\/cloud-platform\/[^/]+/(?:cloud/)?available-instances\//.test(url) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            instance_type: 'ecs.e2e.small',
            instance_type_id: 'ecs.e2e.small',
            cpu_cores: 2,
            memory_gb: 4,
            instance_type_family: 'ecs.e2e',
          },
          {
            instance_type: 'ecs.e2e.large',
            instance_type_id: 'ecs.e2e.large',
            cpu_cores: 4,
            memory_gb: 8,
            instance_type_family: 'ecs.e2e',
          },
        ]),
      });
      return;
    }

    await route.continue({ headers });
  });
}

test.describe('ProjectDetail 可用实例 CPU/内存展示', () => {
  test('扁平数组响应应展示非零核心数与内存，且请求携带 Cores/Memory', async ({ page }) => {
    test.setTimeout(120_000);
    await registerCloudMocks(page);

    const instancesResponsePromise = page.waitForResponse(
      (r) =>
        r.url().includes('available-instances') &&
        r.request().method() === 'GET' &&
        r.url().includes('Cores=2') &&
        r.url().includes('Memory=4') &&
        r.status() === 200,
      { timeout: 60_000 },
    );

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel).toBeVisible({ timeout: 60_000 });

    await page.locator('#cpu-cores').fill('2');
    await page.locator('#memory-size').fill('4');
    const instancesResponse = await instancesResponsePromise;
    expect(instancesResponse.url()).toContain('Cores=2');
    expect(instancesResponse.url()).toContain('Memory=4');

    await expect(hardwarePanel.getByText('ecs.e2e.small', { exact: true })).toBeVisible({ timeout: 60_000 });
    await expect(hardwarePanel.getByText('2核', { exact: true })).toBeVisible();
    await expect(hardwarePanel.getByText('4GB', { exact: true })).toBeVisible();
    await expect(hardwarePanel.getByText('0核')).not.toBeVisible();
    await expect(hardwarePanel.getByText('0GB')).not.toBeVisible();
  });

  test('分页响应应通过 instance-details 补全当前页 CPU/内存', async ({ page }) => {
    test.setTimeout(120_000);

    await page.route('**/*', async (route) => {
      const headers = { ...route.request().headers() };
      delete headers['x-requested-with'];
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes('/api/') && url.includes(`/projects/${PROJECT_ID}`) && !url.includes('/branches') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_PROJECT),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}?ids=`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E Workspace' }]),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            platforms: [{
              id: 1,
              platform_type: 'aliyun',
              platform_type_display: '阿里云',
              authorization_id: MOCK_AUTH_ID,
              remark: 'e2e',
              platform_name: '阿里云',
            }],
          }),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/default-config/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', default_configs: [] }),
        });
        return;
      }

      if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/regions/`) && method === 'GET') {
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
            { vswitch_id: 'vsw-saved-e2e', zone_id: 'cn-hongkong-b', vswitch_name: 'Saved VSW' },
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

      if (/\/cloud-platform\/[^/]+/(?:cloud/)?available-instances\//.test(url) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            instance_types: ['ecs.e2e.small', 'ecs.e2e.large'],
            pagination: {
              page: 1,
              page_size: 10,
              total_pages: 1,
              total_instances: 2,
            },
          }),
        });
        return;
      }

      if (url.includes('/cloud/instance-details/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              instance_type: 'ecs.e2e.small',
              cpu_cores: 2,
              memory_gb: 4,
              instance_type_family: 'ecs.e2e',
              status: 'available',
            },
            {
              instance_type: 'ecs.e2e.large',
              cpu_cores: 4,
              memory_gb: 8,
              instance_type_family: 'ecs.e2e',
              status: 'available',
            },
          ]),
        });
        return;
      }

      await route.continue({ headers });
    });

    const detailsPromise = page.waitForResponse(
      (r) => r.url().includes('/cloud/instance-details/') && r.request().method() === 'GET' && r.status() === 200,
      { timeout: 60_000 },
    );

    await loginAndOpenDetail(page);
    await detailsPromise;

    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.e2e.small', { exact: true })).toBeVisible({ timeout: 60_000 });
    await expect(hardwarePanel.getByText('2核', { exact: true })).toBeVisible();
    await expect(hardwarePanel.getByText('4GB', { exact: true })).toBeVisible();
  });
});
