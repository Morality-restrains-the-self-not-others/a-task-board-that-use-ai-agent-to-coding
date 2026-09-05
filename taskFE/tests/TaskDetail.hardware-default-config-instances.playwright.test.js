// @ts-check
/**
 * 核验：任务详情「服务器硬件配置」在存在工作区默认配置时，应选中保存的地域/VPC/可用区并发起 available-instances 请求，
 * 且列表不应长期停留在「暂无可用实例」（在接口返回实例数据时）。
 *
 * 依赖本地前后端；需 E2E_EMAIL / E2E_PASSWORD。可选 E2E_TENANT_ID、E2E_WORKSPACE_ID、E2E_TASK_ID。
 * 通过 route mock 云 API，不依赖真实阿里云。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';

const MOCK_AUTH_ID = '777777777777777777';

test.describe('任务详情 — 默认云配置与可用实例', () => {
  test('有默认配置时应拉取可用实例列表', async ({ page }) => {
    const email = process.env.E2E_EMAIL;
    const password = process.env.E2E_PASSWORD;
    const tenantId = process.env.E2E_TENANT_ID || '824976301723086848';
    const workspaceId = process.env.E2E_WORKSPACE_ID || '824976301756641280';
    const taskId = process.env.E2E_TASK_ID || '826066359683985408';

    test.skip(!email || !password, '未设置 E2E_EMAIL / E2E_PASSWORD，跳过需登录的用例');

    await page.route('**/*', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes(`/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}${taskId}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: taskId,
            title: 'E2E 任务',
            description: '',
            priority: 2,
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'e2e' },
            assignees: [],
            comments: [],
            ai_comments: [],
            container_image_id: '',
            workspace_id: workspaceId,
          }),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${tenantId}${workspaceId}/cloud/platforms/`) && method === 'GET') {
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
              },
            ],
          }),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${tenantId}${workspaceId}/cloud/platforms/default-config/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            default_configs: [
              {
                platform_type: 'aliyun',
                platform_type_display: '阿里云',
                authorization_id: MOCK_AUTH_ID,
                remark: 'e2e',
                config: {
                  region: 'cn-hangzhou',
                  zone_id: 'cn-hangzhou-h',
                  vpc_id: 'vpc-saved-e2e',
                  vswitch_id: 'vsw-saved-e2e',
                  security_group_id: 'sg-saved-e2e',
                  payment_type: 'PostPaid',
                  bandwidth_charging_mode: 'PayByTraffic',
                  bandwidth: 5,
                },
              },
            ],
          }),
        });
        return;
      }

      if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/regions/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'cn-shanghai', name: '华东2' },
            { id: 'cn-hangzhou', name: '华东1' },
          ]),
        });
        return;
      }

      if (url.includes('/cloud/server-images/vpcs/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { vpc_id: 'vpc-other', vpc_name: 'Other' },
            { vpc_id: 'vpc-saved-e2e', vpc_name: 'Saved VPC' },
          ]),
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
              zone_id: 'cn-hangzhou-h',
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
          body: JSON.stringify([
            { id: 'sg-saved-e2e', name: 'Saved SG' },
          ]),
        });
        return;
      }

      if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/zones/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'cn-hangzhou-h', name: '杭州 H' },
          ]),
        });
        return;
      }

      if (
        url.includes(`/cloud-platform/${MOCK_AUTH_ID}/available-instances/`) || url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/available-instances/`) &&
        method === 'GET'
      ) {
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
                },
              ],
            },
          }),
        });
        return;
      }

      if (url.includes('/previous-server-config/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', server_config: null }),
        });
        return;
      }

      if (url.includes('/installed-images/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'error', message: 'mock no vm' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'error', message: 'mock' }),
        });
        return;
      }

      await route.continue();
    });

    const instancesPromise = page.waitForResponse(
      (r) =>
        r.url().includes('available-instances') &&
        r.request().method() === 'GET' &&
        r.status() === 200
    );

    await page.goto('/auth/login/');
    await page.waitForLoadState('networkidle');

    const emailPasswordTab = page.locator('text=邮箱').or(page.locator('button:has-text("邮箱")')).first();
    if (await emailPasswordTab.isVisible()) {
      await emailPasswordTab.click();
      await page.waitForTimeout(300);
    }

    await page.locator('#email').fill(email);
    await page.locator('#password').fill(password);
    await page.getByRole('button', { name: '登录' }).click();
    await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 60000 });
    await page.waitForLoadState('networkidle');

    await page.goto(
      taskDetailPathWithMockQuery(`/tenant/${tenantId}/workspace/${workspaceId}/task-detail/${taskId}/`)
    );

    await instancesPromise;

    await expect(page.locator('#region-select')).toHaveValue('cn-hangzhou');
    await expect(page.locator('#vpc-select')).toHaveValue('vpc-saved-e2e');
    await expect(page.getByText('暂无可用实例')).not.toBeVisible();
    await expect(page.getByText('ecs.e2e.small')).toBeVisible();
  });
});
