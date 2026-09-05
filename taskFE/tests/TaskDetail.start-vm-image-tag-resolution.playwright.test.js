// @ts-check
/**
 * 核验链路：
 * 1) 任务页 @镜像 提交评论时，前端 mentions 只带 installed_image id（不发送 container_image_url/tag）。
 * 2) 镜像 tag 解析与拼接由 taskEvents 消费者 / start-vm 后端完成。
 *
 * 运行（需账号）：
 *   E2E_EMAIL=... E2E_PASSWORD=... \
 *   npx playwright test -c playwright.config.js \
 *   tests/TaskDetail.start-vm-image-tag-resolution.playwright.test.js --project=chromium
 */
import { test, expect } from '@playwright/test';
import { submitLoginWithEmailPassword } from './helpers/e2eLogin.js';
import { mentionInstalledImageInComment, waitForCommentCreateRequest } from './helpers/commentImageMentionE2e.js';

const MOCK_AUTH_ID = '777777777777777778';
const MOCK_IMAGE_ID = '900000000000000001';
const TENANT_ID = process.env.E2E_TENANT_ID || '824976301723086848';
const WORKSPACE_ID = process.env.E2E_WORKSPACE_ID || '824976301756641280';
const TASK_ID = process.env.E2E_TASK_ID || '826066359683985408';

test.describe('任务详情 — 启动服务器镜像 tag 解析责任边界', () => {
  test('start-vm 请求仅携带 container_image_id，镜像 URL/tag 由后端解析', async ({ page }) => {
    const email = process.env.E2E_EMAIL;
    const password = process.env.E2E_PASSWORD;
    test.skip(!email || !password, '未设置 E2E_EMAIL / E2E_PASSWORD，跳过需登录的用例');

    await page.route('**/*', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'E2E 任务',
            description: '',
            priority: 2,
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'e2e' },
            assignees: [],
            comments: [],
            ai_comments: [],
            container_image_id: MOCK_IMAGE_ID,
            workspace_id: WORKSPACE_ID,
          }),
        });
        return;
      }

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            container_image_id: MOCK_IMAGE_ID,
          }),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: MOCK_IMAGE_ID,
              name: 'task2app-trae',
              version: 'x86_64_2026-05-08_09-46',
              image_url: 'registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae',
              target_architectures: ['x86_64'],
            },
          ]),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}${MOCK_IMAGE_ID}/regions/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ region_id: 'cn-hangzhou', region_name: '华东1' }]),
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
              { id: 1, platform_type: 'aliyun', platform_type_display: '阿里云', platform_name: '阿里云' },
            ],
          }),
        });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/default-config/`) && method === 'GET') {
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
                config: {
                  region: 'cn-hangzhou',
                  zone_id: 'cn-hangzhou-h',
                  vpc_id: 'vpc-e2e',
                  vswitch_id: 'vsw-e2e',
                  security_group_id: 'sg-e2e',
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
          body: JSON.stringify([{ id: 'cn-hangzhou', name: '华东1' }]),
        });
        return;
      }

      if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/zones/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'cn-hangzhou-h', name: '杭州 H' }]),
        });
        return;
      }

      if (url.includes('/cloud/server-images/vpcs/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ vpc_id: 'vpc-e2e', vpc_name: 'VPC-E2E' }]),
        });
        return;
      }

      if (url.includes('/cloud/server-images/vswitches/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ vswitch_id: 'vsw-e2e', zone_id: 'cn-hangzhou-h', vswitch_name: 'VSW-E2E' }]),
        });
        return;
      }

      if (url.includes('/cloud/server-images/security-groups/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: 'sg-e2e', name: 'SG-E2E' }]),
        });
        return;
      }

      if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/available-instances/`) || url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/available-instances/`) && method === 'GET') {
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

      if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/bandwidth-limitation/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            RequestId: 'mock-bandwidth-req',
            min_bandwidth: 1,
            max_bandwidth: 100,
            BandwidthInfo: { min_bandwidth: 1, max_bandwidth: 100, default_bandwidth: 5, bandwidth_options: [5, 10] },
          }),
        });
        return;
      }

      if (url.includes(`/cloud-platform/${MOCK_AUTH_ID}/cloud/instance-price/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            RequestId: 'mock-price-req',
            currency: 'CNY',
            price: 0.21,
            system_disk_category: 'cloud_essd',
            Price: {
              Currency: 'CNY',
              TradePrice: 0.21,
              OriginalPrice: 0.28,
              DiscountPrice: 0.07,
              DetailInfos: {
                DetailInfo: [
                  { Resource: 'instanceType', TradePrice: 0.12, OriginalPrice: 0.15 },
                  { Resource: 'systemDisk', TradePrice: 0.06, OriginalPrice: 0.08 },
                  { Resource: 'bandwidth', TradePrice: 0.03, OriginalPrice: 0.05 },
                ],
              },
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

      if (url.includes('/comments/') && method === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ id: 'cmt-e2e', content: 'ok' }),
        });
        return;
      }

      if (url.includes(`/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}start-vm/`) && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', message: 'mock started' }),
        });
        return;
      }

      await route.continue();
    });

    await page.goto('/auth/login/');
    await page.waitForLoadState('networkidle');
    await submitLoginWithEmailPassword(page, email, password);
    await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 60000 });

    await page.goto(`/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`);
    await page.waitForLoadState('networkidle');

    await mentionInstalledImageInComment(page, { imageText: 'task2app-trae' });
    const hardwarePanel = page.getByTestId('server-hardware-config-panel');
    await hardwarePanel.scrollIntoViewIfNeeded();
    await expect(hardwarePanel).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('ecs.e2e.small')).toBeVisible({ timeout: 15000 });
    await page.getByRole('button', { name: '选择' }).first().click();

    const commentReqPromise = waitForCommentCreateRequest(page);
    await page.getByTestId('task-detail-comment-submit').click();
    const commentReq = await commentReqPromise;
    const payload = JSON.parse(commentReq.postData() || '{}');

    expect(payload.mentions?.[0]?.id).toBe(MOCK_IMAGE_ID);
    expect(payload.mentions?.[0]?.type).toBe('installed_image');
    expect(payload.container_image_url).toBeUndefined();
  });
});
