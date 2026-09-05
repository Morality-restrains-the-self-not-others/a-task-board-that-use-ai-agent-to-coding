// @ts-check
/**
 * OPT-20260823-056：管理端赠送页只改 VIP 不生成赠送订单；填数量才生成。
 * 纯 mock，无真实账号。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import {
  TENANT_ID,
  TENANT_NAME,
  installBillingAdminMocks,
  jsonOk,
} from './helpers/billingAdminMocks.js';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_BILLING_MOCK_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');

const GRANT_PATH = `/system-admin/grant-points/?tenant_id=${TENANT_ID}`;
const TENANTS = [{ id: TENANT_ID, name: TENANT_NAME, phone: '', email: 'e2e@example.com' }];

test('只设 VIP1 不填数量：preview 无赠送订单且 POST resources=[] grants=[]', async ({ page }) => {
  test.setTimeout(120000);
  /** @type {object[]} */
  const grantPosts = [];
  await installBillingAdminMocks(page, BASE_URL, async (route, request) => {
    const url = request.url();
    const method = request.method();
    if (url.includes('/tenant-options/')) {
      await jsonOk(route, TENANTS);
      return true;
    }
    if (url.includes('/gitlab-regions/')) {
      await jsonOk(route, { regions: [] });
      return true;
    }
    if (url.includes('/billing/membership/') && method === 'GET') {
      await jsonOk(route, { membership: { tier: 'normal', admin_tier_locked: false } });
      return true;
    }
    if (url.includes('admin_grant_points') && method === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      grantPosts.push({
        body,
        idempotencyHeader: request.headers()['idempotency-key'] || request.headers()['Idempotency-Key'] || '',
      });
      await jsonOk(route, {
        status: 'ok',
        grants: [],
        membership: { tier: 'vip1', admin_tier_locked: true },
      });
      return true;
    }
    return false;
  });

  await page.goto(`${BASE_URL}${GRANT_PATH}`);
  await expect(page.getByTestId('tenant-selected-summary')).toContainText(TENANT_NAME, { timeout: 20000 });
  await page.getByTestId('grant-membership-tier').selectOption('vip1');
  await expect(page.getByTestId('grant-submit-preview')).toContainText('不会生成赠送订单');
  await page.getByRole('button', { name: '确认赠送' }).click();
  await expect.poll(() => grantPosts.length).toBe(1);
  expect(grantPosts[0].body.resources).toEqual([]);
  expect(grantPosts[0].body.membership_tier).toBe('vip1');
  expect(grantPosts[0].body.idempotency_key).toBeTruthy();
  expect(grantPosts[0].idempotencyHeader).toBe(grantPosts[0].body.idempotency_key);
  await expect(page.getByRole('status')).toContainText('会员等级已设为VIP1');
});

test('填写数量后提交生成赠送订单且预览文案正确', async ({ page }) => {
  test.setTimeout(120000);
  /** @type {object[]} */
  const grantPosts = [];
  await installBillingAdminMocks(page, BASE_URL, async (route, request) => {
    const url = request.url();
    const method = request.method();
    if (url.includes('/tenant-options/')) {
      await jsonOk(route, TENANTS);
      return true;
    }
    if (url.includes('/gitlab-regions/')) {
      await jsonOk(route, { regions: [] });
      return true;
    }
    if (url.includes('/billing/membership/') && method === 'GET') {
      await jsonOk(route, { membership: { tier: 'normal', admin_tier_locked: false } });
      return true;
    }
    if (url.includes('admin_grant_points') && method === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      grantPosts.push(body);
      const qty = body.resources?.[0]?.quantity;
      await jsonOk(route, {
        status: 'ok',
        grants: qty ? [{ resource_type: 'task_post', quantity: qty }] : [],
      });
      return true;
    }
    return false;
  });

  await page.goto(`${BASE_URL}${GRANT_PATH}`);
  await expect(page.getByTestId('tenant-selected-summary')).toContainText(TENANT_NAME, { timeout: 20000 });
  await page.getByTestId('grant-quantity-input').fill('2');
  await expect(page.getByTestId('grant-submit-preview')).toContainText('将生成赠送订单');
  await expect(page.getByTestId('grant-submit-preview')).toContainText('任务帖 × 2');
  await page.getByRole('button', { name: '确认赠送' }).click();
  await expect.poll(() => grantPosts.length).toBe(1);
  expect(grantPosts[0].resources).toEqual([
    expect.objectContaining({ resource_type: 'task_post', quantity: 2 }),
  ]);
  await expect(page.getByRole('status')).toContainText('赠送成功');
  await expect(page.getByRole('status')).toContainText('任务帖 × 2');
});
