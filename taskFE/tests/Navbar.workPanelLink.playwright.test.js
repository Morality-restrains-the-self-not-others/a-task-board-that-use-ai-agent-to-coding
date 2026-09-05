// @ts-check
/**
 * E2E: 验证「工作面板」导航链接 — localStorage 回退修复
 *
 * 场景：普通用户在非租户页面（如 profile）上，
 * localStorage 中有 lastActiveTenantId 时，
 * 「工作面板」链接应指向正确租户而非 /。
 *
 * 运行方式：
 *   NODE_PATH=~/.npm/_npx/e41f203b7505f1fb/node_modules \
 *   node taskFE/tests/Navbar.workPanelLink.playwright.test.js
 */

const { chromium } = require('playwright');

const CDP_URL = process.env.PW_CDP_URL || 'http://127.0.0.1:9222';
const BASE_URL = process.env.PW_BASE_URL || 'http://127.0.0.1:4000';
const USER_ID = process.env.PW_USER_ID || '870705390352887808';
const TENANT_ID = process.env.PW_TENANT_ID || '850256677331562496';

let failures = 0;
let passed = 0;

function assert(condition, message) {
  if (condition) {
    passed++;
    console.log(`  ✅ ${message}`);
  } else {
    failures++;
    console.error(`  ❌ ${message}`);
  }
}

async function run() {
  console.log(`🔗 Connecting to Chrome CDP at ${CDP_URL}...`);
  const browser = await chromium.connectOverCDP(CDP_URL);
  console.log(`✅ Connected. Browser: ${browser.version()}`);

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await context.newPage();

  // ── Test 1: API returns company data → workPanelPath resolves correctly ──
  console.log('\n📋 Test 1: API 返回公司数据 → 工作面板链接指向正确租户');

  // Mock /me/ API
  await page.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID,
        username: 'test-user',
        is_superuser: false,
        current_company: { id: TENANT_ID, name: 'Test Company' },
        companies: [{ id: TENANT_ID, name: 'Test Company' }],
      }),
    });
  });

  // Allow all other requests through
  await page.route('**/api/accounts/users/profile/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user_id: USER_ID,
        personal_nickname: 'Test User',
        email: 'test@example.com',
      }),
    });
  });

  // Navigate to profile page (non-tenant page)
  await page.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'networkidle',
    timeout: 30000,
  });

  // Wait for Navbar to render with the work panel link
  await page.waitForSelector('a[data-testid="nav-work-panel"]', { timeout: 15000 });

  // Give Vue time to reactively update after API response
  await page.waitForTimeout(1500);

  // Check localStorage persistence (Navbar.logic should have written it)
  const stored = await page.evaluate(() => {
    return localStorage.getItem('lastActiveTenantId');
  });
  console.log(`  localStorage['lastActiveTenantId'] = ${stored}`);

  const href1 = await page.evaluate(() => {
    const el = document.querySelector('a[data-testid="nav-work-panel"]');
    return el ? el.getAttribute('href') : null;
  });
  assert(
    href1 && href1 !== '/' && href1.includes('/tenant/') && href1.includes('/work-panel/'),
    `Test 1: 工作面板链接 href="${href1}" (期望 /tenant/${TENANT_ID}/work-panel/)`,
  );

  // ── Test 2: localStorage fallback when API returns no companies ──
  console.log('\n📋 Test 2: API 返回空 companies → localStorage 回退');

  const page2 = await context.newPage();

  // Set cookies + localStorage before navigation
  await page2.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page2.evaluate(({ uid, tid }) => {
    localStorage.setItem('lastActiveTenantId', tid);
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID, tid: TENANT_ID });

  // Mock /me/ with NO companies (simulates orphaned user or API edge case)
  await page2.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID,
        username: 'no-company-user',
        is_superuser: false,
        companies: [],
      }),
    });
  });

  await page2.route('**/api/accounts/users/profile/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user_id: USER_ID,
        personal_nickname: 'No Company',
        email: 'nc@example.com',
      }),
    });
  });

  await page2.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'networkidle',
    timeout: 30000,
  });

  await page2.waitForSelector('a[data-testid="nav-work-panel"]', { timeout: 15000 });
  await page2.waitForTimeout(1500);

  const href2 = await page2.evaluate(() => {
    const el = document.querySelector('a[data-testid="nav-work-panel"]');
    return el ? el.getAttribute('href') : null;
  });
  assert(
    href2 && href2 !== '/' && href2.includes(`/tenant/${TENANT_ID}/work-panel/`),
    `Test 2: 工作面板链接 href="${href2}" (localStorage 回退 → /tenant/${TENANT_ID}/work-panel/)`,
  );

  // ── Test 3: Superuser navigation visibility ──
  console.log('\n📋 Test 3: 超管用户导航栏渲染「系统管理」');

  const page3 = await context.newPage();
  await page3.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page3.evaluate(({ uid }) => {
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID });

  // Mock /me/ as superuser
  await page3.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID,
        username: 'admin',
        is_superuser: true,
        companies: [],
      }),
    });
  });

  await page3.route('**/api/accounts/users/profile/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ user_id: USER_ID }),
    });
  });

  await page3.goto(`${BASE_URL}/system-admin/`, {
    waitUntil: 'networkidle',
    timeout: 30000,
  });

  await page3.waitForTimeout(2000);

  const hasSystemAdmin = await page3.evaluate(() => {
    return !!document.querySelector('a[href="/system-admin/"]');
  });
  const hasWorkPanel = await page3.evaluate(() => {
    return !!document.querySelector('a[data-testid="nav-work-panel"]');
  });

  assert(hasSystemAdmin, 'Test 3: 超管导航栏有「系统管理」链接');
  assert(!hasWorkPanel, 'Test 3: 超管导航栏无「工作面板」链接');

  // ── Summary ──
  console.log(`\n${'='.repeat(50)}`);
  console.log(`📊 Results: ${passed} passed, ${failures} failed`);

  // CDP disconnect: closes the page/browser handle, doesn't kill Chrome
  await context.close();

  process.exit(failures > 0 ? 1 : 0);
}

run().catch((err) => {
  console.error('❌ Test harness error:', err.message);
  process.exit(1);
});
