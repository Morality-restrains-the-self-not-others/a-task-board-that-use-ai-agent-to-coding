// @ts-check
/**
 * 核验：经「工作空间管理 → 机器节点」打开「设置默认配置」时，
 * GET 返回的 VPC/交换机/安全组若在列表中且不是第一项，
 * 表单应保留已保存的选项，而不是被「自动选第一项」覆盖。
 *
 * 本地运行需已启动前后端，并设置环境变量：
 *   E2E_EMAIL、E2E_PASSWORD
 * 可选：E2E_TENANT_ID（默认 824976301723086848）
 *
 * 通过 route mock 云 API，不依赖真实阿里云资源。
 */
import { test, expect } from '@playwright/test';

const MOCK_AUTH_ID = '888888888888888888';
const MOCK_WS_ID = 'ws-e2e-default-config';

test.describe('机器节点策略 — 默认配置弹窗回显', () => {
  test('加载已保存配置后应保留非首项的 VPC/交换机/安全组', async ({ page }) => {
    const email = process.env.E2E_EMAIL;
    const password = process.env.E2E_PASSWORD;
    const tenantId = process.env.E2E_TENANT_ID || '824976301723086848';

    test.skip(!email || !password, '未设置 E2E_EMAIL / E2E_PASSWORD，跳过需登录的用例');

    const registerCloudMocks = async () => {
      await page.route('**/*', async (route) => {
      const url = route.request().url();
      if (url.includes('/workspaces/') && route.request().method() === 'GET' && /\/workspaces\/?(\?|$)/.test(url)) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: MOCK_WS_ID,
              name: 'E2E默认配置空间',
              is_current: true,
              is_default: true,
              task_archive_tier: '7d',
              description: '',
            },
          ]),
        });
        return;
      }
      if (url.includes('/cloud/compute/workspace-machine-policy/') && route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            idle_recycle_minutes: 30,
            prefer_idle_reuse: true,
            enabled_authorization_ids: [],
          }),
        });
        return;
      }
      if (url.includes('/cloud/cloud-platform-authorizations/') && route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: MOCK_AUTH_ID,
              platform_type: 'aliyun',
              authorization_type: 'access_key',
              remark: 'e2e-mock',
              secret_id: 'mock-ak',
              created_at: '2020-01-01T00:00:00Z',
              is_active: true
            }
          ])
        });
        return;
      }
      if (url.includes('/cloud/oauth-tokens/') && route.request().method() === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }
      if (url.includes('/cloud/regions/') && route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'cn-hangzhou', name: '华东 1（杭州）' }
          ])
        });
        return;
      }
      if (
        url.includes('/cloud-platform/') &&
        url.includes('/cloud/regions/') &&
        route.request().method() === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'cn-hangzhou', name: '华东1（杭州）' },
            { id: 'cn-beijing', name: '华北2（北京）' }
          ])
        });
        return;
      }
      if (
        url.includes('/cloud/server-config-default/') &&
        route.request().method() === 'GET' &&
        url.includes(MOCK_AUTH_ID)
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: {
              region: 'cn-hangzhou',
              vpc_id: 'vpc-saved-002',
              vswitch_id: 'vsw-saved-002',
              security_group_id: 'sg-saved-002',
              payment_type: 'PostPaid',
              bandwidth_charging_mode: 'PayByTraffic',
              bandwidth: 5
            }
          })
        });
        return;
      }
      if (url.includes('/cloud/server-images/vpcs/') && route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { vpc_id: 'vpc-first-001', vpc_name: 'First VPC' },
            { vpc_id: 'vpc-saved-002', vpc_name: 'Saved VPC' }
          ])
        });
        return;
      }
      if (url.includes('/cloud/server-images/vswitches/') && route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'vsw-first-001', name: 'VSW First' },
            { id: 'vsw-saved-002', name: 'VSW Saved' }
          ])
        });
        return;
      }
      if (url.includes('/cloud/server-images/security-groups/') && route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'sg-first-001', name: 'SG First' },
            { id: 'sg-saved-002', name: 'SG Saved' }
          ])
        });
        return;
      }
      await route.continue();
    });
    };

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
    await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 25000 });
    await page.waitForLoadState('networkidle');

    await registerCloudMocks();

    await page.goto(`/tenant/${tenantId}/settings/task-panel/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(800);

    await page.locator('[data-alias="open-workspace-machine-policy"]').first().click();
    const policyModal = page.locator('[data-alias="workspace-machine-policy-modal"]');
    await expect(policyModal).toBeVisible({ timeout: 15000 });
    await policyModal.locator('[data-alias="open-default-server-config"]').first().click();

    const modal = page.locator('div.max-w-2xl').filter({ has: page.getByRole('heading', { name: '设置默认配置' }) });
    await expect(modal).toBeVisible({ timeout: 15000 });

    const selects = modal.locator('select');
    await expect(selects.nth(0).locator('option')).toHaveCount(3, { timeout: 15000 });
    await expect(selects.nth(0)).toHaveValue('cn-hangzhou', { timeout: 15000 });
    await expect(selects.nth(1)).toHaveValue('vpc-saved-002');
    await expect(selects.nth(2)).toHaveValue('vsw-saved-002');
    await expect(selects.nth(3)).toHaveValue('sg-saved-002');
    await expect(selects.nth(4)).toHaveValue('PostPaid');
    await expect(selects.nth(5)).toHaveValue('PayByTraffic');
  });
});
