// @ts-check
/**
 * 回归：购买页合卡后 GitLab 磁盘与流量共用一个区域下拉（OPT-20260823-015）。
 * - VIP1 打开 /tenant/:tid/billing/orders/create/
 * - `order-gitlab-resources` 卡片内同时含磁盘与流量输入
 * - `order-gitlab-region` 下拉数量为 1，且不存在旧 `order-gitlab-traffic-region` 双下拉
 *
 * 纯 mock，无真实账号依赖；参考 WorkspaceSettings.archive-modal.playwright.test.js。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';
const ORDER_URL = `${BASE_URL}/tenant/${TENANT_ID}/billing/orders/create/`;

test('billing/orders/create: GitLab 磁盘与流量共用唯一区域下拉', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-order-gitlab-region', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  await page.route(/\/api\/.*/, async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // 用户信息（Navbar 依赖 + 陈旧租户守卫校验 /me/ companies）
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
          current_workspace: { id: 'ws-e2e', name: '默认工作空间' },
        }),
      });
      return;
    }

    // 权限/角色
    if (url.includes('/api/auth/user-roles/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ roles: [] }) });
      return;
    }
    if (url.includes('/api/auth/user-permissions/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ tenant_perms: {} }) });
      return;
    }

    // 定价（VIP1 可见 GitLab 磁盘/流量单价）
    if (url.includes('/billing/order-pricing/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          pricing: {
            task_post: { price_yuan: 10, required_tier: 'normal' },
            gitlab_disk: { price_yuan: 5, required_tier: 'vip1' },
            gitlab_traffic: { price_yuan: 2, required_tier: 'vip1' },
          },
        }),
      });
      return;
    }

    // 会员等级 → VIP1
    if (url.includes('/billing/membership/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          membership: { tier: 'vip1', cumulative_consumption_yuan: 100 },
        }),
      });
      return;
    }

    // GitLab 区域列表
    if (url.includes('/api/billing/gitlab-regions/tenant_id/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          regions: [{ slug: 'cn-hongkong', name: '中国香港', is_active: true }],
        }),
      });
      return;
    }

    // 其余 API 一律返回中性 200（防止真实后端 401 → redirect_url 跳登录）
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(ORDER_URL);
  await page.waitForLoadState('domcontentloaded');

  // 等待合卡资源卡片渲染（VIP1）
  const resources = page.locator('[data-testid="order-gitlab-resources"]');
  await resources.waitFor({ state: 'visible', timeout: 45000 });

  // 磁盘与流量共用一张卡片
  await expect(resources.locator('[data-testid="order-gitlab-disk-gb"]')).toBeVisible();
  await expect(resources.locator('[data-testid="order-gitlab-traffic-gb"]')).toBeVisible();

  // 区域下拉唯一（共用），且旧的双下拉「流量区域」不复存在
  await expect(page.locator('[data-testid="order-gitlab-region"]')).toHaveCount(1);
  await expect(page.locator('[data-testid="order-gitlab-traffic-region"]')).toHaveCount(0);
});
