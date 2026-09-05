/**
 * CDP 9222 验收：多仓 GitLab 推送不再报 terminal prompts disabled。
 *
 * 运行：
 *   node tests/TaskDetail.multirepo-gitlab-push-oauth-fix-cdp.mjs
 */
import { chromium } from '@playwright/test';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE_BASE = (process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:4000').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12939023414154091865';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;
const TASK_URL = `${SITE_BASE}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`;

async function loginIfNeeded(page) {
  if (!page.url().includes('/auth/login')) return;
  const emailTab = page.getByRole('tab', { name: /邮箱|密码/ });
  if (await emailTab.isVisible().catch(() => false)) await emailTab.click();
  for (const name of [/隐私|privacy/i, /许可|license/i]) {
    const cb = page.getByRole('checkbox', { name });
    if (await cb.isVisible().catch(() => false)) {
      const checked = await cb.isChecked().catch(() => false);
      if (!checked) await cb.check().catch(() => {});
    }
  }
  await page.locator('input[type="email"], input[name="email"]').first().fill(EMAIL);
  await page.locator('input[type="password"]').first().fill(PASSWORD);
  await page.getByRole('button', { name: /登录|Login/i }).first().click();
  await page.waitForURL((u) => !String(u).includes('/auth/login'), { timeout: 120000 });
}

async function firstEnabledButton(scope, name) {
  const buttons = scope.getByRole('button', { name });
  await buttons.first().waitFor({ state: 'visible', timeout: 120000 });
  const n = await buttons.count();
  for (let i = 0; i < n; i++) {
    const b = buttons.nth(i);
    if (await b.isEnabled().catch(() => false)) return b;
  }
  throw new Error(`no enabled button: ${name}`);
}

async function main() {
  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  const context = browser.contexts()[0] || (await browser.newContext());
  const page = await context.newPage();
  page.on('dialog', (d) => void d.accept());

  try {
    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await loginIfNeeded(page);
    if (page.url().includes('/auth/login')) {
      await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    }

    const ztree = page.getByTestId('comment-layer-ztree-panel');
    await ztree.waitFor({ state: 'visible', timeout: 180000 });

    const nameBtns = ztree.locator('button.break-words.flex-1.min-w-0');
    await nameBtns.first().waitFor({ state: 'visible', timeout: 120000 });
    const n = await nameBtns.count();
    for (let i = 0; i < n; i++) {
      const label = ((await nameBtns.nth(i).innerText()) || '').trim();
      if (label === '可写层' || /^可写层（/.test(label)) continue;
      await nameBtns.nth(i).click();
      break;
    }

    const pushBtn = await firstEnabledButton(ztree, '推送');
    const [pushResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 180000 },
      ),
      pushBtn.click(),
    ]);

    const post = JSON.parse(pushResp.request().postData() || '{}');
    const status = pushResp.status();
    const text = await pushResp.text().catch(() => '');
    let detail = text;
    try {
      const j = JSON.parse(text);
      detail = String(j.detail || text);
    } catch {
      /* keep raw */
    }

    console.log('[push-verify] prefer_container_remote=', post.prefer_container_remote);
    console.log('[push-verify] status=', status);
    console.log('[push-verify] detail=', detail.slice(0, 500));

    if (!post.prefer_container_remote) {
      throw new Error('expected prefer_container_remote=true for multi-repo task');
    }
    if (/could not read Username|terminal prompts disabled/i.test(detail)) {
      throw new Error(`still credential-prompt failure: ${detail.slice(0, 300)}`);
    }
    if (status >= 500) {
      throw new Error(`server error ${status}: ${detail.slice(0, 300)}`);
    }

    // Correlate gateway log briefly
    console.log('[push-verify] OK — no Username/terminal-prompts failure');
    process.exitCode = 0;
  } catch (e) {
    console.error('[push-verify] FAIL', e);
    process.exitCode = 1;
  } finally {
    await page.close().catch(() => {});
    // Do not close shared CDP browser
  }
}

main();
