// @ts-check
/**
 * E2E: 验证「工作面板」导航链接修复 — profile 页面场景
 *
 * 问题：用户在 /user/:id/profile/ 等非租户页面上，
 * Navbar 的「工作面板」链接 href="/" 导致点击后跳转到
 * 营销首页 Home.vue 而非工作空间 WorkPanel。
 *
 * 根因：currentTenant ref 初始化为 ''，localStorage 中的
 * lastActiveTenantId 在 API 调用完成后才写入，首帧渲染时
 * workPanelPath 回退到 '/'。
 *
 * 修复：
 *   1. Navbar.logic.vue — currentTenant 从 localStorage 同步初始化
 *   2. router.js — 已认证用户访问 / 时重定向到工作面板
 *
 * 运行方式：
 *   NODE_PATH=~/.npm/_npx/e41f203b7505f1fb/node_modules \
 *   node taskFE/tests/Navbar.workPanelLink.profile-page-fix.playwright.test.js
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

async function setupMeApiMock(page, userId, tenantId, opts = {}) {
  const {
    isSuperuser = false,
    companies = [{ id: tenantId, name: 'Test Company' }],
    currentCompany = { id: tenantId, name: 'Test Company' },
  } = opts;

  await page.route(`**/api/accounts/users/me/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: userId,
        username: 'test-user',
        is_superuser: isSuperuser,
        current_company: currentCompany,
        companies,
      }),
    });
  });

  await page.route('**/api/accounts/users/profile/', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        user_id: userId,
        personal_nickname: 'Test User',
        email: 'test@example.com',
      }),
    });
  });
}

async function run() {
  console.log(`🔗 Connecting to Chrome CDP at ${CDP_URL}...`);
  const browser = await chromium.connectOverCDP(CDP_URL);
  console.log(`✅ Connected. Browser: ${browser.version()}`);

  // ═══════════════════════════════════════════════════════════════
  // Test 1: localStorage 有租户 ID → 首帧渲染即指向正确工作面板
  // ═══════════════════════════════════════════════════════════════
  console.log('\n📋 Test 1: localStorage 回退 — 首帧渲染 href 正确 (核心修复验证)');

  const ctx1 = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page1 = await ctx1.newPage();

  // Set localStorage + userId cookie before navigation
  await page1.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page1.evaluate(({ uid, tid }) => {
    localStorage.setItem('lastActiveTenantId', tid);
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID, tid: TENANT_ID });

  await setupMeApiMock(page1, USER_ID, TENANT_ID);

  // Navigate to profile page (non-tenant route)
  await page1.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'load',
    timeout: 30000,
  });

  await page1.waitForSelector('a[data-testid="nav-work-panel"]', { timeout: 15000 });

  // Check link href IMMEDIATELY (before API response might update it)
  // This verifies that the synchronous localStorage init works
  const hrefInitial = await page1.evaluate(() => {
    const el = document.querySelector('a[data-testid="nav-work-panel"]');
    return el ? el.getAttribute('href') : null;
  });
  assert(
    hrefInitial && hrefInitial !== '/' && hrefInitial.includes(`/tenant/${TENANT_ID}/work-panel/`),
    `Test 1a: 首帧工作面板链接 href="${hrefInitial}" (期望 /tenant/${TENANT_ID}/work-panel/)`,
  );

  // Wait for API to complete and verify link is STILL correct
  await page1.waitForTimeout(2500);

  const hrefAfterApi = await page1.evaluate(() => {
    const el = document.querySelector('a[data-testid="nav-work-panel"]');
    return el ? el.getAttribute('href') : null;
  });
  assert(
    hrefAfterApi && hrefAfterApi !== '/' && hrefAfterApi.includes(`/tenant/${TENANT_ID}/work-panel/`),
    `Test 1b: API 完成后工作面板链接 href="${hrefAfterApi}" (期望 /tenant/${TENANT_ID}/work-panel/)`,
  );

  await ctx1.close();

  // ═══════════════════════════════════════════════════════════════
  // Test 2: SPA 内部导航 → 工作面板链接点击后正确跳转
  // ═══════════════════════════════════════════════════════════════
  console.log('\n📋 Test 2: 端到端点击流程 — profile 页点击「工作面板」导航到工作空间');

  const ctx2 = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page2 = await ctx2.newPage();

  await page2.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page2.evaluate(({ uid, tid }) => {
    localStorage.setItem('lastActiveTenantId', tid);
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID, tid: TENANT_ID });

  // 全站 API 兜底 mock（先注册、后注册的具体路由优先匹配）：防止 work-panel
  // 未显式 mock 的接口打到真实网关触发 forward-auth 401 重定向登录（test isolation）
  await page2.route('**/api/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await setupMeApiMock(page2, USER_ID, TENANT_ID);

  // Mock work panel API to prevent 404s during navigation
  await page2.route(`**/api/tenant/${TENANT_ID}**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ rhythms: [], sections: [], tasks: [] }),
    });
  });

  await page2.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'load',
    timeout: 30000,
  });

  await page2.waitForSelector('a[data-testid="nav-work-panel"]', { timeout: 15000 });
  await page2.waitForTimeout(1500);

  // Verify link href before clicking
  const hrefBeforeClick = await page2.evaluate(() => {
    const el = document.querySelector('a[data-testid="nav-work-panel"]');
    return el ? el.getAttribute('href') : null;
  });
  console.log(`  点击前 href="${hrefBeforeClick}"`);

  // Click the work panel link
  await page2.click('a[data-testid="nav-work-panel"]');
  await page2.waitForTimeout(3000);

  const urlAfterClick = page2.url();
  assert(
    urlAfterClick.includes('/work-panel/'),
    `Test 2a: 点击后 URL="${urlAfterClick}" (期望包含 /work-panel/)`,
  );
  assert(
    urlAfterClick.includes(`/tenant/${TENANT_ID}`),
    `Test 2b: 点击后 URL 包含正确租户 (URL="${urlAfterClick}")`,
  );

  await ctx2.close();

  // ═══════════════════════════════════════════════════════════════
  // Test 3: 超管无公司 — 显示「系统管理」，不显示「工作面板」
  // （有公司时允许并列公司下拉/工作面板，见 Navbar.ui 平台角色并列渲染）
  // ═══════════════════════════════════════════════════════════════
  console.log('\n📋 Test 3: 超管无公司导航栏 — 显示「系统管理」而非「工作面板」');

  const ctx3 = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page3 = await ctx3.newPage();

  await page3.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page3.evaluate(({ uid }) => {
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID });

  await setupMeApiMock(page3, USER_ID, TENANT_ID, {
    isSuperuser: true,
    companies: [],
    currentCompany: null,
  });

  await page3.goto(`${BASE_URL}/system-admin/`, {
    waitUntil: 'load',
    timeout: 30000,
  });

  await page3.waitForTimeout(2000);

  const hasSystemAdmin = await page3.evaluate(() => {
    return !!document.querySelector('a[href="/system-admin/"]');
  });
  const hasWorkPanel = await page3.evaluate(() => {
    return !!document.querySelector('a[data-testid="nav-work-panel"]');
  });

  assert(hasSystemAdmin, 'Test 3a: 超管导航栏有「系统管理」链接');
  assert(!hasWorkPanel, 'Test 3b: 超管导航栏无「工作面板」链接');

  await ctx3.close();

  // ═══════════════════════════════════════════════════════════════
  // Test 4: 多公司用户 — 工作面板位置渲染为公司切换下拉，
  // current_company 持久化到 localStorage
  // ═══════════════════════════════════════════════════════════════
  console.log('\n📋 Test 4: 多公司用户 — 工作面板为公司切换下拉，current_company 落 localStorage');

  const SECOND_TENANT = '999999999999999999';
  const ctx4 = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page4 = await ctx4.newPage();

  await page4.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page4.evaluate(({ uid, tid }) => {
    localStorage.setItem('lastActiveTenantId', tid);
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID, tid: TENANT_ID });

  // Mock /me/ with multiple companies — current is SECOND_TENANT
  await setupMeApiMock(page4, USER_ID, SECOND_TENANT, {
    companies: [
      { id: TENANT_ID, name: 'First Company' },
      { id: SECOND_TENANT, name: 'Current Company' },
    ],
    currentCompany: { id: SECOND_TENANT, name: 'Current Company' },
  });

  await page4.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'load',
    timeout: 30000,
  });

  await page4.waitForSelector('[data-testid="nav-company-switcher"]', { timeout: 15000 });
  await page4.waitForTimeout(2500);

  const hasSwitcher4 = await page4.evaluate(() => {
    return !!document.querySelector('[data-testid="nav-company-switcher"]');
  });
  const hasAnchor4 = await page4.evaluate(() => {
    return !!document.querySelector('a[data-testid="nav-work-panel"]');
  });
  const switcherLabel4 = await page4.evaluate(() => {
    const btn = document.querySelector('button[data-testid="nav-work-panel"]');
    return btn ? btn.textContent.trim() : '';
  });
  assert(hasSwitcher4, 'Test 4a: 多公司用户渲染公司切换下拉');
  assert(!hasAnchor4, 'Test 4b: 多公司用户不再渲染单公司「工作面板」<a> 链接');
  assert(
    switcherLabel4.includes('Current Company'),
    `Test 4c: 下拉触发按钮显示当前公司名（"${switcherLabel4}"）`,
  );

  // Verify localStorage was updated to current company
  const stored4 = await page4.evaluate(() => localStorage.getItem('lastActiveTenantId'));
  assert(
    stored4 === SECOND_TENANT,
    `Test 4d: localStorage lastActiveTenantId="${stored4}" (期望 ${SECOND_TENANT})`,
  );

  await ctx4.close();

  // ═══════════════════════════════════════════════════════════════
  // Test 5: 无 localStorage 场景 — 用户首次访问非租户页，
  // API 返回后可正确更新链接（验证异步回填逻辑）
  // ═══════════════════════════════════════════════════════════════
  console.log('\n📋 Test 5: 无 localStorage 时 API 回填 — 异步更新后链接正确');

  const ctx5 = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page5 = await ctx5.newPage();

  // Set userId cookie but NO localStorage tenant
  await page5.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page5.evaluate(({ uid }) => {
    localStorage.removeItem('lastActiveTenantId');
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID });

  await setupMeApiMock(page5, USER_ID, TENANT_ID);

  await page5.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'load',
    timeout: 30000,
  });

  await page5.waitForSelector('a[data-testid="nav-work-panel"]', { timeout: 15000 });

  // Before API completes, the link MIGHT show / as a last resort
  // (localStorage is empty, currentTenant starts as '')
  // After API completes, it should update to the correct work panel URL
  await page5.waitForTimeout(3000); // Allow API call to complete

  const href5 = await page5.evaluate(() => {
    const el = document.querySelector('a[data-testid="nav-work-panel"]');
    return el ? el.getAttribute('href') : null;
  });
  assert(
    href5 && href5 !== '/' && href5.includes(`/tenant/${TENANT_ID}/work-panel/`),
    `Test 5: API 回填后工作面板链接 href="${href5}" (期望 /tenant/${TENANT_ID}/work-panel/)`,
  );

  // Verify localStorage was set by API response handler
  const stored5 = await page5.evaluate(() => localStorage.getItem('lastActiveTenantId'));
  assert(
    stored5 === TENANT_ID,
    `Test 5b: API 调用后 localStorage lastActiveTenantId="${stored5}" (期望 ${TENANT_ID})`,
  );

  await ctx5.close();

  // ═══════════════════════════════════════════════════════════════
  // Test 6: 平台角色 + 多公司 — 「系统管理」与公司切换下拉并列，
  // 选择公司后进入对应租户工作面板（OPT-20260811-015）
  // ═══════════════════════════════════════════════════════════════
  console.log('\n📋 Test 6: 平台角色多公司 — 系统管理与公司切换下拉并列，切换进对应工作面板');

  const SECOND_TENANT_6 = '888888888888888888';
  const ctx6 = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page6 = await ctx6.newPage();

  await page6.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page6.evaluate(({ uid, tid }) => {
    localStorage.setItem('lastActiveTenantId', tid);
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID, tid: TENANT_ID });

  // 全站 API 兜底 mock（先注册、后注册的具体路由优先匹配）：防止切换后
  // work-panel 未显式 mock 的接口打到真实网关触发 forward-auth 401 重定向登录
  await page6.route('**/api/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  // Mock /me/: 平台角色(isSuperuser) + 两家公司，当前公司为第一家
  await setupMeApiMock(page6, USER_ID, TENANT_ID, {
    isSuperuser: true,
    companies: [
      { id: TENANT_ID, name: '公司A' },
      { id: SECOND_TENANT_6, name: '公司B' },
    ],
    currentCompany: { id: TENANT_ID, name: '公司A' },
  });

  // Mock 第二家公司工作面板 API，防止切换后导航 404
  await page6.route(`**/api/tenant/${SECOND_TENANT_6}**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ rhythms: [], sections: [], tasks: [] }),
    });
  });

  await page6.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'load',
    timeout: 30000,
  });

  await page6.waitForSelector('[data-testid="nav-company-switcher"]', { timeout: 15000 });
  await page6.waitForTimeout(1500);

  const hasSystemAdmin6 = await page6.evaluate(() => {
    return !!document.querySelector('a[href="/system-admin/"]');
  });
  const hasSwitcher6 = await page6.evaluate(() => {
    return !!document.querySelector('[data-testid="nav-company-switcher"]');
  });
  const hasWorkPanelAnchor6 = await page6.evaluate(() => {
    return !!document.querySelector('a[data-testid="nav-work-panel"]');
  });

  assert(hasSystemAdmin6, 'Test 6a: 平台角色多公司 — 「系统管理」链接存在');
  assert(hasSwitcher6, 'Test 6b: 平台角色多公司 — 公司切换下拉存在');
  assert(!hasWorkPanelAnchor6, 'Test 6c: 多公司时工作面板为下拉按钮而非 <a> 链接');

  // 展开下拉 → 断言两家公司 → 选择第二家
  await page6.click('button[data-testid="nav-work-panel"]');
  await page6.waitForSelector('[data-testid="nav-company-menu"]', { timeout: 10000 });
  const menuItemCount6 = await page6.evaluate(() => {
    return document.querySelectorAll('[data-testid="nav-company-menu"] [role="menuitem"]').length;
  });
  assert(menuItemCount6 === 2, `Test 6d: 公司下拉列出 2 家公司（实际 ${menuItemCount6}）`);

  // 选择非当前公司（data-current=false），真实 a[href] 全页导航到对应工作面板
  await Promise.all([
    page6.waitForURL((url) => url.pathname.includes(`/tenant/${SECOND_TENANT_6}/work-panel/`), { timeout: 15000 }),
    page6.click('[data-testid="nav-company-menu"] [role="menuitem"][data-current="false"]'),
  ]).catch(() => {});

  const url6 = page6.url();
  assert(
    url6.includes(`/tenant/${SECOND_TENANT_6}/work-panel/`),
    `Test 6e: 切换后 URL="${url6}" (期望 /tenant/${SECOND_TENANT_6}/work-panel/)`,
  );

  await ctx6.close();

  // ═══════════════════════════════════════════════════════════════
  // Summary
  // ═══════════════════════════════════════════════════════════════
  console.log(`\n${'='.repeat(50)}`);
  console.log(`📊 Results: ${passed} passed, ${failures} failed`);
  console.log(`${'='.repeat(50)}`);

  process.exit(failures > 0 ? 1 : 0);
}

run().catch((err) => {
  console.error('❌ Test harness error:', err.message);
  process.exit(1);
});
