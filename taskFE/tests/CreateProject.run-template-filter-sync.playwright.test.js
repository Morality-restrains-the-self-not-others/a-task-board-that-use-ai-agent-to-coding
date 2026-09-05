// @ts-check
/**
 * E2E: CreateProject 应用「快速应用默认模版」后实例筛选与选中实例对齐（OPT-20260812-019）
 *
 * 回归：cloud_server_config_defaults 硬件字段落库后，`defaultConfigToRunTemplate`
 * 曾把 hardware_config / filter_options 映射成空对象，导致选择默认模版后 CPU/内存/
 * 磁盘筛选与选中实例不恢复。本用例：
 *  1. mock `server-config-default` 列表（含 cpu_cores/memory_gb/instance_type/disk）
 *  2. 选择工作空间 + 默认模版
 *  3. 断言 filter cores/memory/disk 与 selected_instance 恢复
 *
 * 采用与 CreateProject.auto-clone-nested 一致的 mock 方案（假 Cookie + page.route
 * 拦截全部 /api/），不依赖真实登录；云平台/实例 mock 对齐
 * ProjectDetail.hardware-config-sync 的 registerCloudMocks。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL =
  process.env.BASE_URL ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;
const TENANT_ID = process.env.TEST_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.TEST_WORKSPACE_ID || 'ws-mock-1';
const MOCK_AUTH_ID = '862031128628060160';
const CREATE_PROJECT_PATH = `/tenant/${TENANT_ID}/create-project/`;

// 默认模版：2核8GB · ecs.g6.large · ESSD 数据盘
const MOCK_DEFAULT_TEMPLATE = {
  id: 'tpl-mock-1',
  platform: 'aliyun',
  region: 'cn-hongkong',
  zone_id: 'cn-hongkong-d',
  authorization_id: MOCK_AUTH_ID,
  cpu_cores: '2',
  memory_gb: '8',
  instance_type: 'ecs.g6.large',
  system_disk_category: 'cloud_essd',
  data_disk_category: 'cloud_essd',
  payment_type: 'PostPaid',
  bandwidth_charging_mode: 'PayByTraffic',
  bandwidth: 5,
};

/**
 * 安装 mock 路由：覆盖创建项目页 + 运行模版面板依赖的全部 /api/ 请求。
 * @param {import('@playwright/test').Page} page
 */
async function installApiMocks(page) {
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 租户守卫依赖 /me/：返回该 tenant 在公司列表内 → 放行
    if (url.includes('/api/accounts/users/me/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'mock-user',
          username: 'mock-user',
          current_company: { id: TENANT_ID, name: 'Mock 租户' },
          companies: [{ id: TENANT_ID, name: 'Mock 租户' }],
        }),
      });
      return;
    }

    // 工作空间下拉（创建项目页）
    if (url.includes('/api/projects/workspaces/tenant_id/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: WORKSPACE_ID, name: 'Mock 工作空间' }]),
      });
      return;
    }

    // 已安装镜像
    if (url.includes('/api/cloud/installed-images/tenant_id/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    // 快速应用默认模版列表：含 cpu/memory/instance_type/disk
    if (url.includes('/api/cloud/server-config-default/tenant_id/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [MOCK_DEFAULT_TEMPLATE] }),
      });
      return;
    }

    // 工作空间已绑定云平台
    if (url.includes(`/api/cloud/platforms/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/`) && method === 'GET') {
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
              platform_name: '阿里云',
              authorization_id: MOCK_AUTH_ID,
            },
          ],
        }),
      });
      return;
    }

    // 云平台默认配置
    if (url.includes('/api/cloud/platforms/default-config/tenant_id/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          default_configs: [
            {
              platform_type: 'aliyun',
              authorization_id: MOCK_AUTH_ID,
              remark: 'mock',
              config: { region: 'cn-hongkong', zone_id: 'cn-hongkong-d' },
            },
          ],
        }),
      });
      return;
    }

    // 云平台地域
    if (/\/cloud-platform\/[^/]+\/regions\/tenant_id\//.test(url) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'cn-hongkong', name: '中国香港' }]),
      });
      return;
    }

    // 云平台可用区
    if (/\/cloud-platform\/[^/]+\/zones\/tenant_id\//.test(url) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'cn-hongkong-d', name: '香港 D' }]),
      });
      return;
    }

    // VPC / 交换机 / 安全组
    if (url.includes('/cloud/server-images/vpcs/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ vpc_id: 'vpc-mock', vpc_name: 'Mock VPC' }]),
      });
      return;
    }
    if (url.includes('/cloud/server-images/vswitches/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { vswitch_id: 'vsw-mock', zone_id: 'cn-hongkong-d', vswitch_name: 'Mock VSW' },
        ]),
      });
      return;
    }
    if (url.includes('/cloud/server-images/security-groups/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'sg-mock', name: 'Mock SG' }]),
      });
      return;
    }

    // 可用实例列表：2核8GB ecs.g6.large（ESSD）
    if (/\/cloud-platform\/[^/]+\/(?:cloud\/)?available-instances\//.test(url) && method === 'GET') {
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

    // 实例详情 / 价格（选中实例展示依赖，返回空避免挂起）
    if (/\/cloud-platform\/[^/]+\/(?:instance-details|instance-price|bandwidth-limitation)\//.test(url)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', instances: [], data: [] }),
      });
      return;
    }

    // 其余 API 一律返回空对象，避免未 mock 请求挂起
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });
}

async function gotoCreateProject(page) {
  await page.goto(`${BASE_URL}${CREATE_PROJECT_PATH}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForSelector('#projectName', { timeout: 15000 });
  await page.waitForFunction(() => {
    const select = document.querySelector('#workspace');
    return select && select.options.length >= 1;
  }, { timeout: 15000 });
}

test.describe('CreateProject 默认模版 → 实例筛选对齐', () => {
  test('选择默认模版后 filter cores/memory/disk 与 selected_instance 恢复', async ({ page, context }) => {
    test.setTimeout(120_000);
    await context.addCookies([
      { name: 'userId', value: 'e2e-user', url: `${BASE_URL}/` },
      { name: 'csrftoken', value: 'e2e-csrf', url: `${BASE_URL}/` },
    ]);
    await installApiMocks(page);

    await gotoCreateProject(page);
    await page.selectOption('#workspace', { label: 'Mock 工作空间' });

    // 等待运行模版面板挂载并加载默认模版选项
    const templateSelect = page.locator('#project-run-template-select');
    await expect(templateSelect).toBeVisible({ timeout: 20_000 });
    await expect
      .poll(() => templateSelect.locator('option').count(), { timeout: 20_000 })
      .toBeGreaterThan(1);

    // 选择「快速应用默认模版」
    await templateSelect.selectOption({ label: 'aliyun · cn-hongkong' });

    // 断言硬件筛选与模版一致（2核8GB · ESSD 数据盘）
    const filterFields = page.locator('.hardware-config-filter-fields');
    await expect(filterFields).toBeVisible({ timeout: 20_000 });
    const cpuInput = page.getByPlaceholder('核心数');
    const memoryInput = page.getByPlaceholder('内存大小');
    await expect(cpuInput).toHaveValue('2', { timeout: 20_000 });
    await expect(memoryInput).toHaveValue('8', { timeout: 20_000 });
    await expect(filterFields.getByTestId('data-disk-category-select')).toHaveValue('cloud_essd');
    await expect(filterFields.locator('select').first()).toHaveValue('cloud_essd');

    // 断言选中实例恢复为 ecs.g6.large（可用实例列表出现「已选」）
    const hardwarePanel = page.locator('[data-testid="server-hardware-config-panel"]');
    await expect(hardwarePanel.getByText('ecs.g6.large', { exact: true })).toBeVisible({ timeout: 30_000 });
    await expect(hardwarePanel.getByRole('button', { name: '已选' })).toBeVisible({ timeout: 30_000 });
  });
});
