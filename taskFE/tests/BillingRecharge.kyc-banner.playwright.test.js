// @ts-check
/**
 * OPT-20260722-030: 充值页 KYC banner 与系统管理用户 KYC drawer。
 *
 * 测试内容：
 * 1. 低等级用户（T0_unverified）在充值页看到警告型 banner（含限额/拦截信息）
 * 2. 中高等级用户（T1_basic）在充值页看到信息型 banner（显示单笔/日累计上限）
 * 3. 超管可在用户列表打开 KYC drawer，并看到 profile 字段（tier / status / AML）
 *
 * 使用 page.route mock KYC tier API，用 data-testid / data-alias 选择器断言。
 */
import { test, expect } from '@playwright/test';

const ORIGIN = 'http://localhost:4000';
const USER_ID = 'playwright-e2e-kyc';
const TENANT_ID = '834421734734905344';
const TEST_USER_ID = 'user-kyc-test-001';

/**
 * 设置通用用户认证 mock（userId cookie + /me/ API）。
 */
async function setupAuth(page, uid = USER_ID, isSuperuser = false) {
  await page.context().addCookies([{ name: 'userId', value: uid, url: ORIGIN }]);

  await page.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: uid,
        username: 'playwright-e2e-kyc',
        is_superuser: isSuperuser,
        current_company: { id: TENANT_ID, name: 'E2E Tenant' },
        companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
      }),
    });
  });
}

/**
 * 设置充值页必需的 mock API（不含 kyc，由测试用例动态指定）。
 */
async function setupRechargeMocks(page) {
  // 手机状态（sms_verified = true 让充值页正常渲染）
  await page.route(`**/billing/accounts/recharge_phone_status/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        has_phone: true,
        phone_masked: '139****0000',
        sms_verified: true,
        paypal_enabled: false,
        wechat_enabled: true,
        paypal_currency: 'USD',
      }),
    });
  });

  // 同意协议相关的 API
  await page.route(`**/api/license-agreement/public/recharge-consent/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ consent_id: 'mock-consent-id' }),
    });
  });

  // 余额
  await page.route(`**/api/tenant/${TENANT_ID}/billing/accounts/balance/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        points: 1000,
        available_balance: 1000,
        frozen_balance: 0,
      }),
    });
  });
}

/**
 * 设置 KYC API mock。
 * @param {import('@playwright/test').Page} page
 * @param {object} kycData - { tier, status, max_single_yuan, max_daily_yuan }
 */
async function setupKycMock(page, kycData) {
  await page.route(`**/api/accounts/me/kyc/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(kycData),
    });
  });
}

/**
 * 清理 KYC mock 以便重新设置。
 */
async function clearKycMock(page) {
  await page.unroute(`**/api/accounts/me/kyc/**`);
}

test.describe('Billing 充值页 KYC banner', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('T0 用户可见 KYC 警告型 banner', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    const origin = baseURL || ORIGIN;

    await setupAuth(page);
    await setupRechargeMocks(page);
    await setupKycMock(page, {
      tier: 'T0_unverified',
      status: 'none',
      max_single_yuan: 0,
      max_daily_yuan: 0,
    });

    await page.goto(`${origin}/tenant/${TENANT_ID}/billing/recharge/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    // KYC banner 区域可见
    const banner = page.locator('[data-alias="BillingRechargeKycBanner"]');
    await expect(banner).toBeVisible({ timeout: 15000 });

    // T0 警告文案
    await expect(banner.getByText('身份等级：T0 未验证')).toBeVisible();
    await expect(banner.getByText('完成手机验证前无法充值')).toBeVisible();
    console.log('[PASS] T0 用户可见 KYC banner 及「无法充值」警告');
  });

  test('T1 用户可见 KYC 信息型 banner（含限额）', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    const origin = baseURL || ORIGIN;

    await setupAuth(page);
    await setupRechargeMocks(page);

    // 先清除旧 mock，再设置新 mock
    await clearKycMock(page);
    await setupKycMock(page, {
      tier: 'T1_basic',
      status: 'approved',
      max_single_yuan: 5000,
      max_daily_yuan: 20000,
    });

    await page.goto(`${origin}/tenant/${TENANT_ID}/billing/recharge/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    const banner = page.locator('[data-alias="BillingRechargeKycBanner"]');
    await expect(banner).toBeVisible({ timeout: 15000 });

    // T1 等级标签
    await expect(banner.getByText('身份等级：T1 基础')).toBeVisible();
    // 限额信息
    await expect(banner.getByText('单笔上限')).toBeVisible();
    await expect(banner.getByText('日累计上限')).toBeVisible();
    // 具体数值
    await expect(banner.getByText('5000')).toBeVisible();
    await expect(banner.getByText('20000')).toBeVisible();
    console.log('[PASS] T1 用户可见 KYC banner 及正确的限额信息');
  });
});

test.describe('系统管理用户 KYC drawer', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('超管可打开 KYC drawer 并看到 profile 字段', async ({ page, baseURL }) => {
    test.setTimeout(60_000);
    const origin = baseURL || ORIGIN;

    const adminUid = 'admin-kyc-e2e';
    await setupAuth(page, adminUid, true);

    // mock 用户列表 API，返回一个可 KYC 的用户
    await page.route(`**/api/system-admin/users/**`, async (route) => {
      const url = route.request().url();
      // 只 mock GET 用户列表
      if (route.request().method() === 'GET' && url.includes('/users/')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            users: [
              {
                id: TEST_USER_ID,
                username: 'kyc-test-user',
                email: 'kyc-test@example.com',
                phone: '',
                is_active: true,
                is_superuser: false,
                is_staff: false,
                date_joined: '2026-01-01T00:00:00Z',
                login_methods: [{ method_type: 'email', is_verified: true }],
              },
            ],
            total: 1,
          }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    // mock KYC profile API
    await page.route(`**/api/system-admin/users/${TEST_USER_ID}/kyc/**`, async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            profile: {
              tier: 'T1_basic',
              status: 'approved',
              max_single_yuan: 5000,
              max_daily_yuan: 20000,
            },
            audit: [
              {
                id: 'audit-001',
                created_at: '2026-06-01T10:00:00Z',
                old_tier: 'T0_unverified',
                new_tier: 'T1_basic',
                old_status: 'none',
                new_status: 'approved',
                trigger_source: 'evaluate',
                reason_code: 'PHONE_VERIFIED',
                reason_detail: '手机验证完成',
              },
            ],
            latest_aml: {
              result: 'clear',
              checked_at: '2026-06-01T10:30:00Z',
            },
          }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(`${origin}/system-admin/users/`);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    // 用户列表中有 KYC 按钮
    const kycBtn = page.getByRole('button', { name: 'KYC' });
    await expect(kycBtn).toBeVisible({ timeout: 15000 });
    await kycBtn.click();
    await page.waitForTimeout(500);

    // KYC drawer 已打开
    const drawer = page.locator('[data-alias="SystemAdminUserKycDrawer"]');
    await expect(drawer).toBeVisible({ timeout: 15000 });

    // drawer 标题
    await expect(drawer.getByText('KYC 身份等级')).toBeVisible();
    // 用户 ID
    await expect(drawer.getByText(TEST_USER_ID)).toBeVisible();
    // 各等级说明（T0/T1/T2）
    const tierHelp = drawer.getByTestId('kyc-tier-help');
    await expect(tierHelp).toBeVisible();
    await expect(tierHelp.getByText('各等级说明')).toBeVisible();
    await expect(tierHelp.getByText(/T0 未验证/)).toBeVisible();
    await expect(tierHelp.getByText(/T1 基础/)).toBeVisible();
    await expect(tierHelp.getByText(/T2 增强/)).toBeVisible();

    // profile 字段：当前状态区
    await expect(drawer.getByText('当前状态')).toBeVisible();
    await expect(drawer.getByText('T1 基础')).toBeVisible();
    await expect(drawer.getByText('已通过')).toBeVisible();

    // AML 信息
    await expect(drawer.getByText('最近 AML')).toBeVisible();
    await expect(drawer.getByText('通过')).toBeVisible();

    // 审计时间线
    await expect(drawer.getByText('审计时间线')).toBeVisible();
    await expect(drawer.getByText('T0 未验证 → T1 基础')).toBeVisible();

    // 人工覆盖区可见
    await expect(drawer.getByText('人工覆盖')).toBeVisible();

    console.log('[PASS] KYC drawer 打开且 profile/AML/审计信息可见');
  });
});
