// @ts-check
/**
 * 回归：跨架构保存镜像硬拦截（OPT-20260821-006）。
 * - 运行模版实例为 x86（ecs.e2e.small）+ 已安装镜像含 arm64 镜像时
 * - 选中 arm64 镜像保存 → 后端 PATCH 400 → 页面报错带 data-traceId，
 *   文案含「镜像要求」与「实例系统支持」
 * - 选中同架构 x86 镜像保存 → PATCH 200 → 编辑态关闭，无报错
 *
 * 纯 mock，无真实账号依赖；参考 ProjectDetail.image-architecture-filter.playwright.test.js。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '857903329669984256';
const PROJECT_ID = process.env.PLAYWRIGHT_PROJECT_ID || '861581450509701120';
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';
const X86_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '859671040643174400';
const ARM_IMAGE_ID = '859671040643174401';
const MOCK_AUTH_ID = '862031128628060160';
const DETAIL_URL = `${BASE_URL}/tenant/${TENANT_ID}/projects/${PROJECT_ID}/`;
const ERROR_TRACE_ID = 'e2e-cross-arch-400-trace';

const MOCK_PROJECT = {
  id: PROJECT_ID,
  name: 'E2E 跨架构拦截',
  description: 'desc',
  tags: [],
  git_repos: [],
  container_image_id: X86_IMAGE_ID,
  container_image: 'task2app-trae',
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

const INSTALLED_IMAGES = [
  {
    id: X86_IMAGE_ID,
    name: 'task2app-trae',
    version: 'x86_64_2026-05-08',
    image_url: 'registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae',
    target_architectures: ['x86_64'],
  },
  {
    id: ARM_IMAGE_ID,
    name: 'task2app-trae-arm',
    version: 'arm64_2026-05-08',
    image_url: 'registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae-arm',
    target_architectures: ['arm64'],
  },
];

/** @param {import('@playwright/test').Page} page */
async function registerMocks(page) {
  await page.route(/\/api\/.*/, async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    const headers = { ...route.request().headers() };
    delete headers['x-requested-with'];

    // 用户信息（Navbar + 陈旧租户守卫 /me/ companies）
    if (url.includes(`/api/accounts/users/me/`)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'e2e',
          has_phone: true,
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WORKSPACE_ID, name: '默认工作空间' },
        }),
      });
      return;
    }
    if (url.includes('/api/auth/user-roles/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ roles: [] }) });
      return;
    }
    if (url.includes('/api/auth/user-permissions/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ tenant_perms: {} }) });
      return;
    }

    // 保存镜像 PATCH：arm 镜像 → 400 硬拦截；x86 镜像 → 200
    if (url.includes(`/api/projects/${PROJECT_ID}/tenant_id/${TENANT_ID}/`) && method === 'PATCH') {
      const body = route.request().postDataJSON?.() || {};
      const selectedId = String(body?.container_image_id || '').trim();
      if (selectedId === ARM_IMAGE_ID) {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          headers: { 'X-Trace-Id': ERROR_TRACE_ID },
          body: JSON.stringify({
            detail: '镜像要求的 CPU 架构（arm64）与实例系统支持的 CPU 架构（x86_64）不匹配，请在运行模版中改选实例后再保存',
            trace_id: ERROR_TRACE_ID,
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ...MOCK_PROJECT, container_image_id: selectedId }),
        });
      }
      return;
    }

    // 项目详情
    if (url.includes('/api/') && url.includes(`/projects/${PROJECT_ID}`) && !url.includes('/branches') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_PROJECT),
      });
      return;
    }

    // 已安装镜像（x86 + arm）
    if (url.includes('/installed-images/') && method === 'GET' && !url.includes('/regions/') && !url.includes('/catalog')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(INSTALLED_IMAGES),
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
          platforms: [{ id: 1, platform_type: 'aliyun', platform_type_display: '阿里云', authorization_id: MOCK_AUTH_ID, remark: 'e2e', platform_name: '阿里云' }],
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
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([{ id: 'cn-hongkong', name: '中国香港' }]) });
      return;
    }
    if (url.includes('/cloud/server-images/vpcs/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([{ vpc_id: 'vpc-saved-e2e', vpc_name: 'Saved VPC' }]) });
      return;
    }
    if (url.includes('/cloud/server-images/vswitches/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([{ vswitch_id: 'vsw-saved-e2e', zone_id: 'cn-hongkong-b', vswitch_name: 'Saved VSW' }]) });
      return;
    }
    if (url.includes('/cloud/server-images/security-groups/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([{ id: 'sg-saved-e2e', name: 'Saved SG' }]) });
      return;
    }
    if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/zones/`) && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([{ id: 'cn-hongkong-b', name: '香港 B' }]) });
      return;
    }
    if (/\/cloud-platform\/[^/]+\/(?:cloud\/)?available-instances\//.test(url) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ instance_type: 'ecs.e2e.small', instance_type_id: 'ecs.e2e.small', cpu_cores: 2, memory_gb: 4, architecture: 'x86_64', instance_type_family: 'ecs.e2e' }]),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

async function openEditMode(page) {
  await page.goto(DETAIL_URL);
  await page.waitForLoadState('domcontentloaded');
  const display = page.locator('[data-testid="project-container-image-display"]');
  await display.waitFor({ state: 'visible', timeout: 45000 });
  await display.click();
  const select = page.locator('[data-testid="project-container-image-select"]');
  await select.waitFor({ state: 'visible', timeout: 15000 });
  return select;
}

test('跨架构保存镜像：arm 镜像 → 400 且报错带 data-traceId', async ({ page }) => {
  test.setTimeout(120000);
  await page.addInitScript(() => {
    try { sessionStorage.setItem('privacy_reconsent_dismissed_at', String(Date.now())); } catch { /* ignore */ }
  });
  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-image-arch', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);
  await registerMocks(page);

  const select = await openEditMode(page);

  // 选中 arm64 镜像
  await select.selectOption(ARM_IMAGE_ID);

  // 点击保存 → 后端 400 硬拦截（前端 saveBlockedHint 仅提示，保存按钮不拦截）
  await page.locator('[data-testid="project-container-image-save"]').click();

  const err = page.locator('[data-testid="project-inline-edit-error"]');
  await err.waitFor({ state: 'visible', timeout: 15000 });
  await expect(err).toContainText('镜像要求');
  await expect(err).toContainText('实例系统支持');
  await expect(err).toHaveAttribute('data-traceId', ERROR_TRACE_ID);
});

test('同架构保存镜像：x86 镜像 → 200 且编辑态关闭', async ({ page }) => {
  test.setTimeout(120000);
  await page.addInitScript(() => {
    try { sessionStorage.setItem('privacy_reconsent_dismissed_at', String(Date.now())); } catch { /* ignore */ }
  });
  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-image-arch', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);
  await registerMocks(page);

  const select = await openEditMode(page);

  // 选中 x86 镜像（与模版实例同架构）
  await select.selectOption(X86_IMAGE_ID);

  // 无前端阻断提示
  await expect(page.locator('[data-testid="project-image-arch-save-hint"]')).toHaveCount(0);

  // 点击保存 → 后端 200 → 编辑态关闭
  await page.locator('[data-testid="project-container-image-save"]').click();

  await expect(page.locator('[data-testid="project-container-image-edit"]')).toHaveCount(0, { timeout: 15000 });
  await expect(page.locator('[data-testid="project-inline-edit-error"]')).toHaveCount(0);
});
