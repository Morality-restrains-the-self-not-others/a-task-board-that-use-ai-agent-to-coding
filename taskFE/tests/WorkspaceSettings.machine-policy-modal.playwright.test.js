// @ts-check
/**
 * E2E：settings/task-panel 机器节点策略模态（Mock session + policy API）
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
const WS_ID = 'ws-e2e-machine-policy';

test('settings/task-panel: machine policy modal open and save', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-machine-policy', url: BASE_URL },
  ]);

  await page.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID,
        username: 'e2e',
        current_company: { id: TENANT_ID, name: 'E2E Tenant' },
        companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
      }),
    });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (route) => {
    const url = route.request().url();
    const method = route.request().method();
    const wsDetail = {
      id: WS_ID,
      name: 'E2E机器策略空间',
      is_current: true,
      is_default: true,
      task_archive_tier: '7d',
      description: '',
      container_image_at_mode_enabled: true,
    };
    if (method === 'GET' && /\/workspaces\/?(\?|$)/.test(url)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([wsDetail]),
      });
      return;
    }
    if ((method === 'GET' || method === 'PATCH') && url.includes(`/workspaces/${WS_ID}`)) {
      const body = method === 'PATCH' ? route.request().postDataJSON() || {} : {};
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...wsDetail, ...body }),
      });
      return;
    }
    await route.continue();
  });

  await page.route('**/cloud/compute/workspace-machine-policy/**', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          idle_recycle_minutes: 5,
          prefer_idle_reuse: true,
          enabled_authorization_ids: [],
        }),
      });
      return;
    }
    if (method === 'PUT') {
      const body = route.request().postDataJSON() || {};
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', ...body }),
      });
      return;
    }
    await route.continue();
  });

  await page.route('**/cloud/cloud-platform-authorizations/**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: 'auth-e2e', platform_type: 'aliyun', remark: 'E2E CPA' }]),
      });
      return;
    }
    await route.continue();
  });

  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/settings/task-panel/`);
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(2500);

  const openBtn = page.locator('[data-alias="open-workspace-machine-policy"]').first();
  await expect(openBtn).toBeVisible({ timeout: 45000 });
  await openBtn.click();

  const modal = page.locator('[data-alias="workspace-machine-policy-modal"]');
  await expect(modal).toBeVisible({ timeout: 15000 });
  await expect(modal.getByRole('heading', { name: '机器节点策略' })).toBeVisible();
  await expect(modal.locator('[data-alias="default-server-config-section"]')).toBeVisible();
  await expect(modal.getByText('默认服务器启动配置', { exact: true })).toBeVisible();
  await expect(modal.locator('[data-alias="open-default-server-config"]').first()).toBeVisible();

  const minutesInput = modal.locator('input[type="number"]');
  await expect(minutesInput).toHaveValue('5');
  await minutesInput.fill('25');
  await expect(modal.getByText('优先在闲置机器节点上启动新容器')).toHaveCount(0);

  const putPromise = page.waitForResponse(
    (res) =>
      res.url().includes('/cloud/compute/workspace-machine-policy/') &&
      res.request().method() === 'PUT',
    { timeout: 30000 },
  );
  await modal.getByRole('button', { name: '保存' }).click();
  const putRes = await putPromise;
  expect(putRes.ok()).toBeTruthy();
  await expect(modal).toBeHidden({ timeout: 15000 });
});
