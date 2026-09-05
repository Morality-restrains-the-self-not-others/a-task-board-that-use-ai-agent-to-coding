/**
 * E2E: profile → 工作面板；/me/ 403 时弹窗确认，取消则留在工作面板（禁止静默回弹 profile）
 *
 * 运行（需 9222 CDP Chrome）：
 *   NODE_PATH=... node taskFE/tests/WorkPanel.access-denied-confirm-no-bounce.playwright.test.js
 */
const { chromium } = require('playwright');

const CDP_URL = process.env.PW_CDP_URL || 'http://127.0.0.1:9222';
const BASE_URL = process.env.PW_BASE_URL || 'http://127.0.0.1:4000';
const USER_ID = process.env.PW_USER_ID || '875216526544760832';
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
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();

  await page.goto(`${BASE_URL}/auth/login/`, { waitUntil: 'domcontentloaded' });
  await page.evaluate(({ uid, tid }) => {
    localStorage.setItem('lastActiveTenantId', tid);
    localStorage.setItem('currentUserId', uid);
    document.cookie = `userId=${uid}; path=/`;
  }, { uid: USER_ID, tid: TENANT_ID });

  // 兜底 API mock
  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    if (url.includes('/api/accounts/users/me/')) {
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({ detail: 'forbidden', trace_id: 'e2e-access-denied' }),
      });
      return;
    }
    if (url.includes('/api/accounts/users/profile/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user_id: USER_ID,
          personal_nickname: 'E2E User',
          companies: [{ id: TENANT_ID, name: 'E2E Co' }],
          current_company: { id: TENANT_ID, name: 'E2E Co' },
        }),
      });
      return;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(`${BASE_URL}/user/${USER_ID}/profile/`, {
    waitUntil: 'load',
    timeout: 30000,
  });

  // 注入登录态 UI：直接进工作面板验证弹窗路径（不依赖 Navbar /me/ 成功）
  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/`, {
    waitUntil: 'domcontentloaded',
    timeout: 30000,
  });

  // 等待确认弹窗（modalService）
  await page.waitForTimeout(2000);
  const modalVisible = await page.evaluate(() => {
    const text = document.body?.innerText || '';
    return text.includes('无法访问工作面板') || text.includes('留在本页') || text.includes('前往个人资料');
  });
  assert(modalVisible, 'Test1: /me/ 403 后出现确认弹窗（非静默回弹）');

  // 点「留在本页」/取消
  const cancelClicked = await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const cancel = buttons.find((b) => /留在本页|取消/.test(b.textContent || ''));
    if (cancel) {
      cancel.click();
      return true;
    }
    return false;
  });
  assert(cancelClicked, 'Test2: 可点击「留在本页」');

  await page.waitForTimeout(1500);
  const path = new URL(page.url()).pathname;
  assert(
    path.includes('/work-panel/'),
    `Test3: 取消后仍停留在工作面板（path=${path}）`,
  );
  assert(
    !/\/user\/\d+\/profile\//.test(path),
    'Test4: 未静默回弹到 profile',
  );

  await ctx.close();
  console.log(`\n📊 Results: ${passed} passed, ${failures} failed`);
  process.exit(failures > 0 ? 1 : 0);
}

run().catch((err) => {
  console.error('❌ harness error:', err.message);
  process.exit(1);
});
