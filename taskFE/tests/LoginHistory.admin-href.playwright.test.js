// @ts-check
/**
 * 超管用户行「登录历史」为真实 <a href>；隐私门先点「同意并继续」
 * （OPT-20260825-036）。纯 mock，无真实账号。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { isRfc1918Ipv4 } from './helpers/loginHistoryIp.js';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_LOGIN_HISTORY_MOCK_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const ADMIN_USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';
const TARGET_USER_ID = process.env.PLAYWRIGHT_TARGET_USER_ID || '837920218451263488';
const TARGET_USER_EMAIL = 'e2e-target@example.com';
const HISTORY_HREF = `/system-admin/users/${TARGET_USER_ID}/login-history/`;

test('isRfc1918Ipv4 识别 Docker 网桥与公网地址', () => {
  expect(isRfc1918Ipv4('172.26.0.1')).toBe(true);
  expect(isRfc1918Ipv4('172.16.0.1')).toBe(true);
  expect(isRfc1918Ipv4('172.31.255.255')).toBe(true);
  expect(isRfc1918Ipv4('172.15.0.1')).toBe(false);
  expect(isRfc1918Ipv4('10.0.0.1')).toBe(true);
  expect(isRfc1918Ipv4('192.168.1.1')).toBe(true);
  expect(isRfc1918Ipv4('203.0.113.9')).toBe(false);
  expect(isRfc1918Ipv4('8.8.8.8')).toBe(false);
  expect(isRfc1918Ipv4('—')).toBe(false);
});

test('system-admin/users: 隐私门同意后登录历史为 <a href> 并可打开', async ({ page }) => {
  test.setTimeout(120000);
  let consented = false;

  await page.context().addCookies([
    { name: 'userId', value: ADMIN_USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-login-history', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  await page.route(/\/api\/.*/, async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    if (url.includes('/api/privacy-policy/consent/') && method === 'POST') {
      consented = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ok: true }),
      });
      return;
    }

    if (url.includes('/api/accounts/users/me/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: ADMIN_USER_ID,
          username: 'e2e-admin',
          is_superuser: true,
          has_phone: true,
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: 'ws-e2e', name: '默认工作空间' },
          pending_privacy_policy: consented
            ? null
            : {
                id: 'pp-e2e-1',
                version: '2026-08-28',
                title: 'E2E 隐私政策',
                content: 'e2e privacy body',
              },
        }),
      });
      return;
    }

    if (url.includes('/api/auth/user-roles/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          roles: [{ role: 'super_admin', level: 1, company_id: null }],
        }),
      });
      return;
    }

    if (url.includes('/api/auth/user-permissions/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ tenant_perms: {} }),
      });
      return;
    }

    if (url.includes('/login-history/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results: [
            {
              id: 'lh-1',
              logged_in_at: '2026-08-28T01:00:00.000000Z',
              client_ip: '203.0.113.9',
              entry: 'customer',
              entry_label: '用户入口',
              method_type: 'email',
              method_label: '邮箱',
              outcome: 'success',
              outcome_label: '成功',
            },
          ],
          total: 1,
          limit: 20,
          offset: 0,
        }),
      });
      return;
    }

    if (url.includes('/api/system-admin/users/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          users: [
            {
              id: TARGET_USER_ID,
              username: 'e2e-target',
              email: TARGET_USER_EMAIL,
              phone: '',
              is_active: true,
              is_superuser: false,
              is_staff: false,
              is_tenant: false,
              is_tester: false,
              is_archived: false,
              date_joined: '2026-07-01T00:00:00Z',
              last_login: '2026-08-01T00:00:00Z',
              login_methods: [{ method_type: 'email', is_verified: true }],
              referrer_code: 'E2E001',
              tenant_companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
            },
          ],
          total: 1,
        }),
      });
      return;
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(`${BASE_URL}/system-admin/users/`);
  await page.waitForLoadState('domcontentloaded');

  const agree = page.getByRole('button', { name: '同意并继续' });
  await expect(agree).toBeVisible({ timeout: 45000 });
  await agree.click();
  await expect(agree).toBeHidden({ timeout: 20000 });

  const row = page.locator('tr', { hasText: TARGET_USER_EMAIL });
  await row.waitFor({ state: 'visible', timeout: 45000 });
  const link = row.getByTestId('user-row-login-history');
  await expect(link).toBeVisible();
  await expect(link).toHaveAttribute('href', HISTORY_HREF);
  const tag = await link.evaluate((el) => el.tagName);
  expect(tag).toBe('A');

  await link.click();
  await expect(page.getByTestId('system-admin-user-login-history')).toBeVisible({ timeout: 30000 });
  await expect(page).toHaveURL(new RegExp(`/system-admin/users/${TARGET_USER_ID}/login-history/`));
  await expect(page.getByTestId('login-history-row').first()).toContainText('用户入口');
});
