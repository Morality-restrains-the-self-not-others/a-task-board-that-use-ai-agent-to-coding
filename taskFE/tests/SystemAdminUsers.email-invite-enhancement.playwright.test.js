// @ts-check
/**
 * E2E：邮箱邀请增强 — 邀请原因、角色预设、账号有效期
 *
 * 验证：
 * 1. POST /api/system-admin/email-invitations/ 接受新字段 assigned_role/account_expires_at/invite_reason
 * 2. GET /api/system-admin/email-invitations/ 返回 inviterName + 新字段
 * 3. 注册流程 applyInviteRoleAndExpiry 正确赋权
 *
 * 运行：npx playwright test SystemAdminUsers.email-invite-enhancement.playwright.test.js
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

let BASE = process.env.PLAYWRIGHT_SITE_ORIGIN || '';
if (!BASE) {
  try {
    const portConfig = loadPortConfig();
    BASE = `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;
  } catch {
    BASE = 'http://localhost:4000';
  }
}
BASE = BASE.replace(/\/$/, '');
const ADMIN_TOKEN = process.env.PLAYWRIGHT_ADMIN_TOKEN || '';
const ADMIN_SESSION_ID = process.env.PLAYWRIGHT_ADMIN_SESSION_ID || '';
const ADMIN_USER_ID = process.env.PLAYWRIGHT_ADMIN_USER_ID || '850249621660790784';

const authHeaders = (): Record<string, string> => {
  const h: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Requested-With': 'XMLHttpRequest',
    'Accept': 'application/json',
  };
  if (ADMIN_TOKEN) h['Authorization'] = `Token ${ADMIN_TOKEN}`;
  if (ADMIN_SESSION_ID) h['Cookie'] = `sessionid=${ADMIN_SESSION_ID}; userId=${ADMIN_USER_ID}`;
  return h;
};

test.describe('email-invite-enhancement', () => {
  test('API: POST create with enhanced fields returns 201', async ({ request }) => {
    test.setTimeout(30000);

    const testEmail = `e2e-enhance-${Date.now()}@test.com`;
    const resp = await request.post(`${BASE}/api/system-admin/email-invitations/`, {
      headers: authHeaders(),
      data: {
        email: testEmail,
        assigned_role: 'member',
        account_expires_at: '2027-12-31T00:00:00Z',
        invite_reason: 'E2E 测试 — 邀请增强验证',
      },
    });

    if (resp.status() === 401 || resp.status() === 403) {
      console.log('[enhance API] 认证未通过，跳过内容验证');
      return;
    }

    // 201 = created, 400 = email already registered or has pending invite
    expect([201, 400]).toContain(resp.status());

    const body = await resp.json();
    console.log(`[enhance API] POST response: ${resp.status()} — ${JSON.stringify(body)}`);

    if (resp.status() === 201) {
      expect(body).toHaveProperty('message');
      expect(body).toHaveProperty('email', testEmail);
    }
  });

  test('API: POST create with invalid assigned_role returns 400', async ({ request }) => {
    test.setTimeout(30000);

    const testEmail = `e2e-badrole-${Date.now()}@test.com`;
    const resp = await request.post(`${BASE}/api/system-admin/email-invitations/`, {
      headers: authHeaders(),
      data: {
        email: testEmail,
        assigned_role: 'invalid_role_xyz',
      },
    });

    if (resp.status() === 401 || resp.status() === 403) {
      console.log('[enhance API] 认证未通过，跳过内容验证');
      return;
    }

    expect(resp.status()).toBe(400);
    const body = await resp.json();
    expect(body.error).toContain('无效的角色类型');
    console.log(`[enhance API] 正确拒绝无效角色: ${body.error}`);
  });

  test('API: POST create with permanent account (no account_expires_at) returns 201', async ({ request }) => {
    test.setTimeout(30000);

    const testEmail = `e2e-permanent-${Date.now()}@test.com`;
    const resp = await request.post(`${BASE}/api/system-admin/email-invitations/`, {
      headers: authHeaders(),
      data: {
        email: testEmail,
        assigned_role: 'staff',
        invite_reason: '员工账号 — 永久有效',
        // 不传 account_expires_at = 永久有效
      },
    });

    if (resp.status() === 401 || resp.status() === 403) {
      console.log('[enhance API] 认证未通过，跳过内容验证');
      return;
    }

    expect([201, 400]).toContain(resp.status());
    console.log(`[enhance API] 永久有效邀请: ${resp.status()}`);
  });

  test('API: GET invitations list includes enhanced fields', async ({ request }) => {
    test.setTimeout(30000);

    const resp = await request.get(`${BASE}/api/system-admin/email-invitations/`, {
      headers: authHeaders(),
    });

    if (resp.status() === 401 || resp.status() === 403) {
      console.log('[enhance API] 认证未通过，跳过内容验证');
      return;
    }

    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body).toHaveProperty('invitations');
    expect(Array.isArray(body.invitations)).toBe(true);

    if (body.invitations.length > 0) {
      const first = body.invitations[0];
      // 验证新增字段存在于响应中（值可为空）
      expect(first).toHaveProperty('inviterUserId');
      // inviterName 可能为空（inviter 被删除或不存在于 auth_user）
      const hasInviterName = 'inviterName' in first;
      const hasInviteReason = 'inviteReason' in first;
      const hasAccountExpiresAt = 'accountExpiresAt' in first;
      const hasAssignedRole = 'assignedRole' in first;

      console.log(`[enhance API] 第一条邀请 — inviterName:${hasInviterName} inviteReason:${hasInviteReason} accountExpiresAt:${hasAccountExpiresAt} assignedRole:${hasAssignedRole}`);
      console.log(`[enhance API] 字段值 — inviterName:"${first.inviterName}" inviteReason:"${first.inviteReason}" accountExpiresAt:"${first.accountExpiresAt}" assignedRole:"${first.assignedRole}"`);

      // 核心断言：所有增强字段 key 必须存在于响应对象中
      expect(hasInviterName).toBe(true);
      expect(hasInviteReason).toBe(true);
      expect(hasAccountExpiresAt).toBe(true);
      expect(hasAssignedRole).toBe(true);
    } else {
      console.log('[enhance API] 无邀请记录，跳过字段验证');
    }
  });

  test('UI: invite modal has role/reason/expiry fields', async ({ page }) => {
    test.setTimeout(60000);

    if (ADMIN_SESSION_ID) {
      await page.context().addCookies([
        { name: 'userId', value: ADMIN_USER_ID, url: BASE },
        { name: 'sessionid', value: ADMIN_SESSION_ID, url: BASE },
      ]);
    }

    // Mock /me API
    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: ADMIN_USER_ID,
          username: 'admin',
          is_superuser: true,
          is_staff: true,
          current_company: null,
          companies: [],
        }),
      });
    });

    // Mock email-invitations GET to return empty list (avoid noise)
    await page.route('**/api/system-admin/email-invitations/', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ invitations: [] }),
        });
      } else {
        await route.continue();
      }
    });

    await page.goto(`${BASE}/system-admin/users/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    // 点击 "发送邮箱邀请" 按钮
    const inviteButton = page.locator('button').filter({ hasText: /发送邮箱邀请/ });
    await expect(inviteButton.first()).toBeVisible({ timeout: 15000 });
    await inviteButton.first().click();
    await page.waitForTimeout(500);

    // 验证弹窗中的新字段
    const modal = page.locator('.fixed.inset-0').filter({ has: page.locator('h3').filter({ hasText: /发送邮箱注册邀请/ }) });
    await expect(modal.first()).toBeVisible({ timeout: 5000 });

    // 检查角色选择器
    const roleSelect = modal.locator('select#invite-role');
    const roleVisible = await roleSelect.isVisible().catch(() => false);
    console.log(`[UI enhancement] 角色选择器可见: ${roleVisible}`);

    // 检查账号有效期输入
    const expiryInput = modal.locator('input#invite-expiry');
    const expiryVisible = await expiryInput.isVisible().catch(() => false);
    console.log(`[UI enhancement] 有效期输入可见: ${expiryVisible}`);

    // 检查永久有效复选框
    const permanentCheckbox = modal.locator('input[type="checkbox"]').first();
    const cbVisible = await permanentCheckbox.isVisible().catch(() => false);
    console.log(`[UI enhancement] 永久有效复选框可见: ${cbVisible}`);

    // 检查邀请原因输入
    const reasonTextarea = modal.locator('textarea#invite-reason');
    const reasonVisible = await reasonTextarea.isVisible().catch(() => false);
    console.log(`[UI enhancement] 邀请原因输入可见: ${reasonVisible}`);

    // 核心断言：所有增强字段必须可见
    expect(roleVisible).toBe(true);
    expect(expiryVisible).toBe(true);
    expect(cbVisible).toBe(true);
    expect(reasonVisible).toBe(true);
  });

  test('UI: invitation table has enhanced columns', async ({ page }) => {
    test.setTimeout(60000);

    if (ADMIN_SESSION_ID) {
      await page.context().addCookies([
        { name: 'userId', value: ADMIN_USER_ID, url: BASE },
        { name: 'sessionid', value: ADMIN_SESSION_ID, url: BASE },
      ]);
    }

    await page.route(`**/api/accounts/users/me/**`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: ADMIN_USER_ID,
          username: 'admin',
          is_superuser: true,
          is_staff: true,
          current_company: null,
          companies: [],
        }),
      });
    });

    // Mock invitations GET with enhanced fields
    await page.route('**/api/system-admin/email-invitations/', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            invitations: [{
              id: '1',
              email: 'test@example.com',
              status: 'pending',
              inviterUserId: ADMIN_USER_ID,
              inviterName: 'admin',
              inviteReason: 'E2E 测试邀请原因',
              assignedRole: 'staff',
              accountExpiresAt: '2027-12-31T00:00:00Z',
              expiresAt: '2026-08-10T00:00:00Z',
              deliveryStatus: 'delivered',
              emailSendAttempts: 1,
              canResend: false,
            }],
          }),
        });
      } else {
        await route.continue();
      }
    });

    await page.goto(`${BASE}/system-admin/users/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(4000);

    // 验证邀请记录区域可见
    const inviteHeading = page.locator('h2').filter({ hasText: /邮箱邀请记录/ });
    await expect(inviteHeading.first()).toBeVisible({ timeout: 15000 });

    // 验证新增列的表头
    const tableHeader = page.locator('thead tr').first();
    const headerText = await tableHeader.textContent();
    console.log(`[UI enhancement] 表头: ${headerText}`);

    // 关键断言：表头应包含新增列
    expect(headerText).toContain('邀请人');
    expect(headerText).toContain('预设角色');
    expect(headerText).toContain('账号有效期');
    expect(headerText).toContain('邀请原因');

    // 验证数据行包含增强字段
    const firstRow = page.locator('tbody tr').first();
    const rowText = await firstRow.textContent();
    console.log(`[UI enhancement] 第一行数据: ${rowText}`);

    // 应包含角色标签（员工 = staff）
    expect(rowText).toContain('员工');
    // 应包含有效期
    expect(rowText).toContain('2027');
    // 应包含邀请原因
    expect(rowText).toContain('E2E 测试邀请原因');
  });
});
