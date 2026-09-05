// @ts-check
/**
 * OPT-20260823-023 / 030：系统管理模拟登录理由弹窗、收信箱、同目标重放跳转、嵌套 409 中文。
 * 纯 mock，无超管口令依赖。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import {
  TARGET_USER_ID,
  TARGET_USER_EMAIL,
  OTHER_USER_ID,
  OTHER_USER_EMAIL,
  IMPERSONATE_REASON,
  NESTED_IMPERSONATION_DETAIL,
  listUser,
  installImpersonationAdminMocks,
  openImpersonateModal,
} from './helpers/impersonationAdminMocks.js';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_IMPERSONATION_MOCK_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');

test('system-admin/users: 取消理由弹窗不发 impersonate POST', async ({ page }) => {
  test.setTimeout(120000);
  /** @type {string[]} */
  const posts = [];
  await installImpersonationAdminMocks(page, BASE_URL, {
    onImpersonate: async (route, request) => {
      posts.push(request.postData() || '');
      await route.fulfill({ status: 500, contentType: 'application/json', body: '{"detail":"should not POST"}' });
    },
  });

  await page.goto(`${BASE_URL}/system-admin/users/`);
  await page.waitForLoadState('domcontentloaded');
  await openImpersonateModal(page, TARGET_USER_EMAIL);
  await page.getByTestId('impersonate-reason-modal').getByRole('button', { name: '取消' }).click();
  await expect(page.getByTestId('impersonate-reason-modal')).toHaveCount(0);
  expect(posts).toHaveLength(0);
  await expect(page).toHaveURL(/\/system-admin\/users\/?/);
});

test('system-admin/users: 确认理由后进入目标身份且收信箱展示该理由', async ({ page }) => {
  test.setTimeout(120000);
  /** @type {string[]} */
  const posts = [];
  await installImpersonationAdminMocks(page, BASE_URL, {
    inboxResults: [
      {
        id: 'inbox-e2e-1',
        title: '管理员以你的身份登录',
        body: `系统管理员（用户 ID：admin）以你的身份登录了本平台。`,
        reason: IMPERSONATE_REASON,
        created_at: '2026-08-28T02:00:00Z',
      },
    ],
    onImpersonate: async (route, request) => {
      posts.push(request.postData() || '');
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'imp_e2e_token',
          user: { id: TARGET_USER_ID, username: 'e2e-target' },
          redirect_url: '/onboarding/',
        }),
      });
    },
  });

  await page.goto(`${BASE_URL}/system-admin/users/`);
  await page.waitForLoadState('domcontentloaded');
  await openImpersonateModal(page, TARGET_USER_EMAIL);
  await page.getByTestId('impersonate-reason-input').fill(IMPERSONATE_REASON);
  await page.getByTestId('impersonate-reason-confirm').click();
  await expect.poll(() => posts.length).toBe(1);
  expect(JSON.parse(posts[0]).reason).toBe(IMPERSONATE_REASON);
  await page.waitForURL(/\/onboarding\//, { timeout: 30000 });
  await page.waitForLoadState('domcontentloaded');

  await page.goto(`${BASE_URL}/profile/inbox/`);
  await page.waitForLoadState('domcontentloaded');
  await expect(page.getByTestId('inbox-message')).toBeVisible({ timeout: 30000 });
  await expect(page.getByTestId('inbox-message-reason')).toContainText(IMPERSONATE_REASON);
});

test('system-admin/users: 同一用户第二次确认登录仍离开用户列表', async ({ page }) => {
  test.setTimeout(120000);
  let postCount = 0;
  await installImpersonationAdminMocks(page, BASE_URL, {
    onImpersonate: async (route) => {
      postCount += 1;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'imp_e2e_token',
          user: { id: TARGET_USER_ID, username: 'e2e-target' },
          redirect_url: '/onboarding/',
        }),
      });
    },
  });

  await page.goto(`${BASE_URL}/system-admin/users/`);
  await page.waitForLoadState('domcontentloaded');
  await openImpersonateModal(page, TARGET_USER_EMAIL);
  await page.getByTestId('impersonate-reason-input').fill(IMPERSONATE_REASON);
  await page.getByTestId('impersonate-reason-confirm').click();
  await page.waitForURL(/\/onboarding\//, { timeout: 30000 });
  await page.waitForLoadState('domcontentloaded');

  await page.goto(`${BASE_URL}/system-admin/users/`, { waitUntil: 'domcontentloaded' });
  await openImpersonateModal(page, TARGET_USER_EMAIL);
  await page.getByTestId('impersonate-reason-input').fill(IMPERSONATE_REASON);
  await page.getByTestId('impersonate-reason-confirm').click();
  await expect.poll(() => postCount).toBe(2);
  await page.waitForURL(/\/onboarding\//, { timeout: 30000 });
  await expect(page.getByTestId('impersonate-reason-error')).toHaveCount(0);
});

test('system-admin/users: 嵌套模拟另一用户 409 中文且停在弹窗', async ({ page }) => {
  test.setTimeout(120000);
  await installImpersonationAdminMocks(page, BASE_URL, {
    users: [
      listUser(TARGET_USER_ID, TARGET_USER_EMAIL, 'e2e-target'),
      listUser(OTHER_USER_ID, OTHER_USER_EMAIL, 'e2e-other'),
    ],
    onImpersonate: async (route, request) => {
      const url = request.url();
      if (url.includes(`/${OTHER_USER_ID}/impersonate/`)) {
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({
            detail: NESTED_IMPERSONATION_DETAIL,
            error: NESTED_IMPERSONATION_DETAIL,
          }),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'imp_e2e_token',
          user: { id: TARGET_USER_ID, username: 'e2e-target' },
          redirect_url: '/onboarding/',
        }),
      });
    },
  });

  await page.goto(`${BASE_URL}/system-admin/users/`);
  await page.waitForLoadState('domcontentloaded');
  await openImpersonateModal(page, OTHER_USER_EMAIL);
  await page.getByTestId('impersonate-reason-input').fill(IMPERSONATE_REASON);
  await page.getByTestId('impersonate-reason-confirm').click();
  const err = page.getByTestId('impersonate-reason-error');
  await expect(err).toBeVisible({ timeout: 15000 });
  await expect(err).toHaveText(NESTED_IMPERSONATION_DETAIL);
  await expect(page).toHaveURL(/\/system-admin\/users\/?/);
  await expect(page.getByTestId('impersonate-reason-modal')).toBeVisible();
});
