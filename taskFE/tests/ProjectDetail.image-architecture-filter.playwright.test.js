// @ts-check
/**
 * E2E: 项目详情页 — 已安装镜像展示 CPU 架构；硬件配置默认筛选项与可用实例请求携带 image_architecture。
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
const MOCK_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '859671040643174400';
const MOCK_AUTH_ID = '862031128628060160';
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

const MOCK_PROJECT = {
  id: PROJECT_ID,
  name: 'E2E 镜像架构过滤',
  description: 'desc',
  tags: [],
  git_repos: [],
  container_image_id: MOCK_IMAGE_ID,
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
async function registerProjectArchitectureMocks(page) {
  /** @type {{ availableInstancesUrl?: string }} */
  const capture = {};
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

    if (url.includes('/installed-images/') && method === 'GET' && !url.includes('/regions/') && !url.includes('/catalog')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: MOCK_IMAGE_ID,
            name: 'task2app-trae',
            version: 'x86_64_2026-05-08',
            image_url: 'registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae',
            target_architectures: ['x86_64'],
          },
        ]),
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
      capture.availableInstancesUrl = url;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            instance_type: 'ecs.e2e.small',
            instance_type_id: 'ecs.e2e.small',
            cpu_cores: 2,
            memory_gb: 4,
            architecture: 'x86_64',
            instance_type_family: 'ecs.e2e',
          },
        ]),
      });
      return;
    }

    await route.continue({ headers });
  });
  return capture;
}

test.describe('ProjectDetail 镜像架构展示与过滤', () => {
  test('已安装镜像应展示架构，可用实例请求应携带 image_architecture 与 container_image_id', async ({ page }) => {
    test.setTimeout(120_000);
    const capture = await registerProjectArchitectureMocks(page);

    const instancesResponsePromise = page.waitForResponse(
      (r) =>
        r.url().includes('available-instances') &&
        r.request().method() === 'GET' &&
        r.url().includes('image_architecture=x86_64') &&
        r.status() === 200,
      { timeout: 60_000 },
    );

    await loginAndOpenDetail(page);

    await expect(page.getByTestId('project-container-image-display')).toContainText('task2app-trae');
    await expect(page.getByTestId('project-container-image-display')).toContainText('架构 x86_64');

    await expect(page.getByTestId('project-run-template-panel')).toBeVisible({ timeout: 30_000 });
    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel).toBeVisible({ timeout: 60_000 });

    await expect(page.getByTestId('image-architecture-display')).toHaveText('x86_64', { timeout: 30_000 });

    await instancesResponsePromise;
    expect(capture.availableInstancesUrl || '').toContain('image_architecture=x86_64');
    expect(capture.availableInstancesUrl || '').toContain(`container_image_id=${MOCK_IMAGE_ID}`);
  });
});
