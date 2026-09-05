// @ts-check
/**
 * 任务详情 — @镜像 提交评论带临时硬件模版（不依赖真实阿里云）。
 * 硬件卡不再 POST start-vm；启机由评论事件消费者完成。
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { mentionInstalledImageInComment, waitForCommentCreateRequest } from './helpers/commentImageMentionE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || process.env.SITE_BASE || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');

const MOCK_AUTH_ID = process.env.PLAYWRIGHT_AUTHORIZATION_ID || '862031128628060160';
const MOCK_IMAGE_ID = process.env.PLAYWRIGHT_CONTAINER_IMAGE_ID || '859671040643174400';
const MOCK_USER_ID = process.env.PLAYWRIGHT_MOCK_USER_ID || '827923618451263488';
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '857903329669984256';
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '860371538948571136';

const ROUTING_FAILURE = 'compute action not yet ported to taskCloudService';

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`,
);

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ noAutoRegion?: boolean }} [opts] noAutoRegion：地域列表返回空，
 *   临时硬件不齐（无地域）→ 面板显示无法运行提示，发评不带 server_run_template。
 */
async function installStartVmAutoMocks(page, opts = {}) {
  const { noAutoRegion = false } = opts;
  /** @type {{ availableInstancesUrl?: string }} */
  const capture = {};
  await page.route('**/*', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 用户信息：mock 当前租户/工作区，避免真实单租户账号与测试 tenant 不匹配跳转 onboarding
    if (url.includes('/api/accounts/users/me/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: MOCK_USER_ID,
          username: 'e2e',
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WORKSPACE_ID, name: 'E2E WS' },
        }),
      });
      return;
    }

    if (url.includes(`/todos/${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'start-vm-auto e2e',
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

    if (url.includes('/installed-images/') && method === 'GET' && !url.includes('/regions/')) {
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

    if (url.includes('/installed-images/') && url.includes('/regions/') && method === 'GET') {
      if (noAutoRegion) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ region_id: 'cn-hangzhou', region_name: '华东1', id: 'cn-hangzhou', name: '华东1' }]),
      });
      return;
    }

    if (url.includes('/cloud/platforms/') && method === 'GET' && !url.includes('default-config')) {
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

    if (url.includes('/cloud/platforms/default-config/') && method === 'GET') {
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

    if (url.includes('/cloud-platform/') && url.includes('/regions/') && method === 'GET') {
      // 实际 URL 形如 /api/cloud/cloud-platform/<authId>/regions/...（无 /cloud/regions/ 前缀）
      if (noAutoRegion) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'cn-hangzhou', name: '华东1' }]),
      });
      return;
    }

    if (url.includes('/cloud-platform/') && url.includes('/zones/') && method === 'GET') {
      // 实际 URL 形如 /api/cloud/cloud-platform/<authId>/zones/...（无 /cloud/zones/ 前缀）
      if (noAutoRegion) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }
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

    if (url.includes('/cloud-platform/') && (url.includes('/available-instances/') || url.includes('/cloud/available-instances/')) && method === 'GET') {
      capture.availableInstancesUrl = url;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          InstanceTypes: {
            InstanceType: [
              { InstanceTypeId: 'ecs.e2e.small', CpuCoreCount: 2, MemorySize: 4, Status: 'Available' },
            ],
          },
        }),
      });
      return;
    }

    if (url.includes('/cloud-platform/') && url.includes('/cloud/bandwidth-limitation/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          RequestId: 'mock-bandwidth',
          min_bandwidth: 1,
          max_bandwidth: 100,
          BandwidthInfo: { min_bandwidth: 1, max_bandwidth: 100, default_bandwidth: 5, bandwidth_options: [5, 10] },
        }),
      });
      return;
    }

    if (url.includes('/cloud-platform/') && url.includes('/cloud/instance-price/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          RequestId: 'mock-price',
          currency: 'CNY',
          price: 0.21,
          system_disk_category: 'cloud_essd',
          Price: { Currency: 'CNY', TradePrice: 0.21, OriginalPrice: 0.28 },
        }),
      });
      return;
    }

    if (url.includes('/cloud-platform/') && url.includes('/cloud/instance-details/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ InstanceTypes: { InstanceType: [] } }),
      });
      return;
    }

    if (url.includes('/previous-server-config/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ previous: null }),
      });
      return;
    }

    if (url.includes('/server-runtime-status/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'no_config', instance_id: '', public_ip: '' }),
      });
      return;
    }

    if (url.includes('/cloud/compute/start-vm-auto/') && method === 'POST') {
      const raw = req.postData() || '{}';
      if (raw.includes(ROUTING_FAILURE)) {
        await route.fulfill({ status: 501, contentType: 'application/json', body: raw });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', message: 'mock start-vm-auto accepted' }),
      });
      return;
    }

    if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    if (url.includes('/progress-system/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
      return;
    }

    if (url.includes('/container-task-ui-context/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', container_endpoint_registered: false, container_page_url: '' }),
      });
      return;
    }

    if (url.includes('/container-layer-graph/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ layers_root: '/', bootstrap_layer_id: 'l1', layers: [], jobs: [] }),
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

    if (url.includes('/api/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
      return;
    }

    await route.continue();
  });
  return capture;
}

test.describe('TaskDetail start-vm-auto mock UI', () => {
  test('@镜像 后调整临时硬件并提交评论，body 含 server_run_template', async ({ page }) => {
    test.setTimeout(180_000);
    const capture = await installStartVmAutoMocks(page);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.E2E_EMAIL || process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.E2E_PASSWORD || process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    await page.goto(`${SITE}/`, { waitUntil: 'domcontentloaded' });
    await page.evaluate((token) => {
      localStorage.setItem('authToken', token);
    }, loginData.token);
    if (loginData.user?.id) {
      await page.context().addCookies([
        { name: 'userId', value: String(loginData.user.id), url: `${SITE}/` },
      ]);
    }

    await page.goto(`${SITE}${TASK_DETAIL_PATH}`);
    await page.waitForLoadState('domcontentloaded');

    await mentionInstalledImageInComment(page, { imageText: 'task2app-trae' });
    await expect(page.getByTestId('comment-composer-mentioned-image')).toContainText('task2app-trae');
    const hardwarePanel = page.getByTestId('server-hardware-config-panel');
    await hardwarePanel.scrollIntoViewIfNeeded();
    await expect(hardwarePanel).toBeVisible({ timeout: 20_000 });
    await expect(page.locator('#start-server-btn')).toHaveCount(0);
    await expect(page.getByTestId('start-server-disabled-reason')).toHaveCount(0);
    await expect(page.getByRole('combobox', { name: '地域' })).toBeEnabled({ timeout: 20_000 });
    const tempConfigBtn = page.getByTestId('open-temporary-hardware-config-btn');
    if (await tempConfigBtn.count()) {
      await tempConfigBtn.click();
    }
    await expect(page.getByTestId('image-architecture-display')).toHaveText('x86_64');
    await expect(page.getByText('ecs.e2e.small')).toBeVisible({ timeout: 20_000 });
    expect(capture.availableInstancesUrl || '').toContain('image_architecture=x86_64');
    await page.getByRole('button', { name: '选择' }).first().click();

    const sgSelect = page.getByRole('combobox', { name: '安全组' });
    await sgSelect.waitFor({ state: 'visible', timeout: 15_000 });
    await sgSelect.selectOption('auto_create_security_group');

    const commentReqPromise = waitForCommentCreateRequest(page);
    await page.getByTestId('task-detail-comment-submit').click();
    const commentReq = await commentReqPromise;
    const payload = JSON.parse(commentReq.postData() || '{}');

    expect(payload.mentions?.[0]?.type).toBe('installed_image');
    expect(payload.mentions?.[0]?.id).toBe(MOCK_IMAGE_ID);
    expect(payload.server_run_template).toBeTruthy();
    expect(String(payload.server_run_template.cloud_platform_id || '')).not.toBe('');
    expect(String(payload.server_run_template.region || '')).not.toBe('');
    expect(JSON.stringify(payload)).not.toContain(ROUTING_FAILURE);
  });

  test('@镜像 后临时硬件不齐仍可发评，body 无 server_run_template', async ({ page }) => {
    test.setTimeout(180_000);
    // noAutoRegion：地域列表为空 → 临时硬件配置不齐，无法启动运行
    const capture = await installStartVmAutoMocks(page, { noAutoRegion: true });

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.E2E_EMAIL || process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.E2E_PASSWORD || process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    await page.goto(`${SITE}/`, { waitUntil: 'domcontentloaded' });
    await page.evaluate((token) => {
      localStorage.setItem('authToken', token);
    }, loginData.token);
    if (loginData.user?.id) {
      await page.context().addCookies([
        { name: 'userId', value: String(loginData.user.id), url: `${SITE}/` },
      ]);
    }

    await page.goto(`${SITE}${TASK_DETAIL_PATH}`);
    await page.waitForLoadState('domcontentloaded');

    await mentionInstalledImageInComment(page, { imageText: 'task2app-trae' });
    await expect(page.getByTestId('comment-composer-mentioned-image')).toContainText('task2app-trae');
    const hardwarePanel = page.getByTestId('server-hardware-config-panel');
    await hardwarePanel.scrollIntoViewIfNeeded();
    await expect(hardwarePanel).toBeVisible({ timeout: 20_000 });
    await expect(page.locator('#start-server-btn')).toHaveCount(0);

    // 临时配置不齐（未选实例 / 无地域）→ 出现无法运行提示
    const tempConfigBtn = page.getByTestId('open-temporary-hardware-config-btn');
    if (await tempConfigBtn.count()) {
      await tempConfigBtn.click();
    }
    const hint = page.getByTestId('comment-composer-hardware-run-hint');
    await expect(hint).toBeVisible({ timeout: 20_000 });
    await expect(hint).toContainText('无法启动运行');
    expect(capture.availableInstancesUrl || '').toBe('');

    // 仍可提交评论：comments POST 发出、无 server_run_template、请求未被前端吃掉
    const commentReqPromise = waitForCommentCreateRequest(page);
    await page.getByTestId('task-detail-comment-submit').click();
    const commentReq = await commentReqPromise;
    const payload = JSON.parse(commentReq.postData() || '{}');

    expect(payload.mentions?.[0]?.type).toBe('installed_image');
    expect(payload.mentions?.[0]?.id).toBe(MOCK_IMAGE_ID);
    expect(payload.server_run_template).toBeUndefined();
    expect(JSON.stringify(payload)).not.toContain(ROUTING_FAILURE);
  });
});
