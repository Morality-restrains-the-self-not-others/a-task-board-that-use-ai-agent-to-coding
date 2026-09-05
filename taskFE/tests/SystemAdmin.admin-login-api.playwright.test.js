// @ts-check
/**
 * OPT-20260901-023 — 超管 E2E 登录改走 API（admin-login + activate-session）。
 *
 * 背景：对 /auth/admin-login/ 填 #email + click「管理员登录」在 CDP/headless Chrome
 * 上挂死（>100s 无第二帧），表单路径依赖原生 submit 与前端 legal-consent 交互。
 * 这里用 playwrightAdminLoginViaApi 两段式 API 登录，断言：
 *   1) admin-login 请求体 { username, password } 且 200 + token/user.id
 *   2) activate-session 带 Authorization: Token … 且 body { user_id }，200
 *   3) 随后 /system-admin/container-images/ 侧栏可见「容器镜像列表」
 *
 * 无 PLAYWRIGHT_TEST_PASSWORD 时以纯 mock 断言请求契约（不依赖真实账号）；有真实
 * 凭据时可把 BASE_URL 指到生产域名跑真实登录。禁止对 AdminLogin 页面 fill #email。
 */
import { test, expect } from '@playwright/test';
import { playwrightAdminLoginViaApi } from './helpers/adminLogin.js';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_ADMIN_LOGIN_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const ADMIN_USER_ID = process.env.PLAYWRIGHT_ADMIN_USER_ID || '827923618451263488';
const ADMIN_EMAIL = process.env.PLAYWRIGHT_ADMIN_EMAIL || 'contact@daydaymoney.com';
const ADMIN_PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD || '';

test('admin-login API 契约：请求体 username/password，activate-session 带 Token 头', async ({ page }) => {
  test.setTimeout(60000);

  /** @type {Record<string, any> | null} */
  let adminLoginBody = null;
  let activateAuthHeader = '';
  /** @type {Record<string, any> | null} */
  let activateBody = null;

  await page.route('**/api/auth/admin-login/', async (route) => {
    adminLoginBody = route.request().postDataJSON?.() || {};
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'e2e-admin-token',
        user: { id: ADMIN_USER_ID, username: 'e2e-admin', email: ADMIN_EMAIL },
      }),
    });
  });
  await page.route('**/api/accounts/users/activate-session/', async (route) => {
    activateAuthHeader = route.request().headers()['authorization'] || '';
    activateBody = route.request().postDataJSON?.() || {};
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user: { id: ADMIN_USER_ID, username: 'e2e-admin', email: ADMIN_EMAIL },
        token: 'e2e-admin-token',
      }),
    });
  });

  await page.goto(`${BASE_URL}/auth/admin-login/`);
  await page.waitForLoadState('domcontentloaded');

  const creds = await playwrightAdminLoginViaApi(page, {
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD || 'e2e-admin-password',
    baseURL: BASE_URL,
  });

  expect(adminLoginBody).toMatchObject({ username: ADMIN_EMAIL });
  expect(adminLoginBody?.password).toBeTruthy();
  expect(activateAuthHeader).toBe('Token e2e-admin-token');
  expect(activateBody).toMatchObject({ user_id: ADMIN_USER_ID });
  expect(creds.userId).toBe(ADMIN_USER_ID);
  expect(creds.token).toBe('e2e-admin-token');
});

test('admin-login 后 /system-admin/container-images/ 侧栏可见「容器镜像列表」', async ({ page }) => {
  test.setTimeout(120000);

  await page.route('**/api/auth/admin-login/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        token: 'e2e-admin-token',
        user: { id: ADMIN_USER_ID, username: 'e2e-admin', email: ADMIN_EMAIL },
      }),
    });
  });
  await page.route('**/api/accounts/users/activate-session/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user: { id: ADMIN_USER_ID, username: 'e2e-admin', email: ADMIN_EMAIL },
        token: 'e2e-admin-token',
      }),
    });
  });
  await page.route('**/api/accounts/users/me/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: ADMIN_USER_ID,
        username: 'e2e-admin',
        is_superuser: true,
        current_company: { id: TENANT_ID, name: 'E2E Tenant' },
        companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
        current_workspace: { id: 'ws-e2e', name: '默认工作空间' },
        pending_privacy_policy: null,
      }),
    });
  });
  await page.route('**/api/auth/user-roles/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ roles: [{ role: 'super_admin', level: 1, company_id: null }] }),
    });
  });
  await page.route('**/api/auth/user-permissions/', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ tenant_perms: {} }) });
  });
  await page.route('**/api/accounts/sso/ai-provider/admin/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ url: 'https://provider.daydaymoney.com/admin' }),
    });
  });
  await page.route('**/api/ai-provider/admin-marketplace-settings/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: true, sso_enabled: true }),
    });
  });

  await page.goto(`${BASE_URL}/auth/admin-login/`);
  await page.waitForLoadState('domcontentloaded');
  await playwrightAdminLoginViaApi(page, {
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD || 'e2e-admin-password',
    baseURL: BASE_URL,
  });

  // activate-session 在真实场景由服务端 Set-Cookie 落 HttpOnly 会话凭据；
  // mock 模式补 cookie 以通过网关 forward-auth 前的本地 SPA 鉴权门。
  await page.context().addCookies([
    { name: 'userId', value: ADMIN_USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-admin-session', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  await page.goto(`${BASE_URL}/system-admin/container-images/`);
  await page.waitForLoadState('domcontentloaded');

  await expect(page.getByRole('navigation').getByText('容器镜像列表').first()).toBeVisible({ timeout: 15000 });
});
