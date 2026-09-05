// @ts-check
/**
 * Goal：在指定任务详情页点「启动」→ 等待运行 → 点「打开控制台」并打开容器页面。
 * 经 CDP 9222 连接本机 Chrome。
 *
 * 用法：
 *   cd task2app/playwright && node taskFE/tests/TaskDetail.relay-start-open-console-cdp.mjs
 */
import { chromium } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').replace(/\/$/, '');
const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || process.env.PLAYWRIGHT_BASE_URL || 'http://183.250.1.132:4000').replace(
  /\/$/,
  '',
);
const TASK_URL =
  process.env.RELAY_TASK_URL ||
  `${SITE}/tenant/850256677331562496/workspace/861623708318031872/task-detail/task_12590983282794675865/?relayToTrae=true`;
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD ;
const START_TIMEOUT_MS = Number(process.env.RELAY_START_TIMEOUT_MS || 180000);
const CONSOLE_TIMEOUT_MS = Number(process.env.RELAY_CONSOLE_TIMEOUT_MS || 120000);

function log(msg, extra) {
  const ts = new Date().toISOString();
  if (extra !== undefined) {
    console.log(`[${ts}] ${msg}`, extra);
  } else {
    console.log(`[${ts}] ${msg}`);
  }
}

async function ensureImageSelected(page) {
  const select = page.locator('#detail-image');
  await select.waitFor({ state: 'visible', timeout: 60000 });
  const value = await select.inputValue().catch(() => '');
  if (value) {
    log(`镜像已选: ${value}`);
    return value;
  }
  const options = select.locator('option:not([disabled])');
  const count = await options.count();
  for (let i = 0; i < count; i++) {
    const opt = options.nth(i);
    const v = await opt.getAttribute('value');
    if (v) {
      await select.selectOption(v);
      log(`已选择镜像: ${v}`);
      await page.waitForTimeout(800);
      return v;
    }
  }
  throw new Error('页面上没有可选镜像');
}

async function waitForConsoleLink(page, { afterMs = 0 } = {}) {
  // 仅认「直接启动」面板链接，避免点到任务详情区残留的「打开容器页面」
  const link = page.getByTestId('relay-to-trae-open-console');
  const deadline = Date.now() + CONSOLE_TIMEOUT_MS;
  const notBefore = Date.now() + afterMs;
  let lastHref = '';
  while (Date.now() < deadline) {
    if (Date.now() < notBefore) {
      await page.waitForTimeout(500);
      continue;
    }
    if (await link.isVisible().catch(() => false)) {
      const href = await link.getAttribute('href');
      if (
        href &&
        href.trim() &&
        !/\/task-detail\//i.test(href) &&
        !/:4000\b/.test(href) &&
        !/127\.0\.0\.1|localhost/i.test(href)
      ) {
        // 再等一轮刷新，避免点到启动瞬间残留的旧 token URL
        if (href === lastHref) {
          // probe that the URL answers (container up)
          try {
            const resp = await page.request.get(href, { timeout: 5000, maxRedirects: 0 });
            const status = resp.status();
            if (status > 0 && status < 500) {
              return { link, href: href.trim() };
            }
            log(`控制台 URL 尚未就绪 status=${status}，继续等待…`, href);
          } catch (e) {
            log('控制台 URL 探测失败，继续等待…', String(e?.message || e).slice(0, 120));
          }
        }
        lastHref = href;
      } else if (href) {
        log('控制台链接仍不可达或为 loopback，继续等待…', href);
      }
    }
    const refresh = page.getByTestId('relay-to-trae-status-refresh');
    if (await refresh.isVisible().catch(() => false)) {
      await refresh.click().catch(() => {});
    }
    await page.waitForTimeout(2000);
  }
  throw new Error(`等待「打开容器页面」超时（${CONSOLE_TIMEOUT_MS}ms）；须为可达 server_url 而非 task-detail/loopback`);
}

async function main() {
  log(`CDP=${CDP_URL}`);
  log(`TASK_URL=${TASK_URL}`);

  let browser;
  try {
    browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  } catch (e) {
    console.error(`无法连接 CDP ${CDP_URL}:`, e?.message || e);
    process.exitCode = 1;
    return;
  }

  const context = browser.contexts()[0] || (await browser.newContext());
  const page = context.pages().find((p) => !p.url().startsWith('devtools://')) || (await context.newPage());

  const result = {
    ok: false,
    imageId: '',
    startClicked: false,
    consoleHref: '',
    consolePageTitle: '',
    consolePageUrl: '',
    consoleStatus: 0,
    error: '',
  };

  try {
    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    if (page.url().includes('/auth/login')) {
      log('需要登录…');
      await playwrightLoginWithLegalAccept(page, { email: EMAIL, password: PASSWORD, baseURL: SITE });
      await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    }
    await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});

    const relayTab = page.getByTestId('server-config-relay-direct-tab');
    if (await relayTab.isVisible().catch(() => false)) {
      await relayTab.click();
      log('已切换到「直接启动」Tab');
    }

    result.imageId = await ensureImageSelected(page);

    // 强制业务端点为当前站点可达 IP，避免表单残留 127.0.0.1
    const host = new URL(SITE).hostname;
    if (host && host !== 'localhost' && host !== '127.0.0.1') {
      const bizInput = page.getByTestId('relay-to-trae-env-BUSINESS_API_ENDPOINT_ORIGIN');
      if (await bizInput.isVisible().catch(() => false)) {
        await bizInput.fill(`http://${host}:8765`);
        log(`已设置 BUSINESS_API_ENDPOINT_ORIGIN=http://${host}:8765`);
      }
    }

    // 若已在运行，先停再启，保证走一遍启动链路
    const stopBtn = page.getByTestId('relay-to-trae-stop-btn');
    if (await stopBtn.isVisible().catch(() => false)) {
      log('检测到已在运行，先停止…');
      await stopBtn.click();
      await page.getByTestId('relay-to-trae-start-btn').waitFor({ state: 'visible', timeout: 60000 });
      await page.waitForTimeout(2000);
    }

    const startBtn = page.getByTestId('relay-to-trae-start-btn');
    await startBtn.waitFor({ state: 'visible', timeout: 30000 });
    // 等待按钮可点（OAuth/镜像门禁）
    const enableDeadline = Date.now() + 60000;
    while (Date.now() < enableDeadline) {
      if (await startBtn.isEnabled().catch(() => false)) break;
      const msg = await page.locator('[data-testid="relay-to-trae-start-btn"]').locator('xpath=..').innerText().catch(() => '');
      log('启动按钮仍禁用，等待…', msg.slice(0, 200));
      await page.waitForTimeout(2000);
    }
    if (!(await startBtn.isEnabled())) {
      throw new Error('启动按钮一直禁用，无法点击');
    }

    log('点击「启动」…');
    await startBtn.click();
    result.startClicked = true;

    // 等待停止按钮出现或控制台链接出现（accepted 异步）
    await Promise.race([
      stopBtn.waitFor({ state: 'visible', timeout: START_TIMEOUT_MS }),
      page.getByRole('link', { name: /打开容器页面|打开控制台/ }).waitFor({ state: 'visible', timeout: START_TIMEOUT_MS }),
    ]).catch(() => {});

    log('等待「打开容器页面」（server_url）…');
    // 给 docker pull/run + register-reachability 留时间，避免点到旧 token
    const { link, href } = await waitForConsoleLink(page, { afterMs: 8000 });
    result.consoleHref = href;
    log(`控制台链接: ${href}`);

    const consolePagePromise = context.waitForEvent('page', { timeout: 30000 }).catch(() => null);
    await link.click();
    let consolePage = await consolePagePromise;
    if (!consolePage) {
      // 同页导航或已有页
      consolePage = context.pages().find((p) => p.url().includes('/ui/') || p.url() === href) || null;
    }
    if (!consolePage) {
      // 直接打开 href
      consolePage = await context.newPage();
      await consolePage.goto(href, { waitUntil: 'domcontentloaded', timeout: 60000 });
    } else {
      await consolePage.waitForLoadState('domcontentloaded', { timeout: 60000 }).catch(() => {});
    }

    result.consolePageUrl = consolePage.url();
    result.consolePageTitle = await consolePage.title().catch(() => '');
    const resp = await consolePage.goto(href, { waitUntil: 'domcontentloaded', timeout: 60000 }).catch(() => null);
    result.consoleStatus = resp?.status?.() || 0;
    result.consolePageUrl = consolePage.url();
    result.consolePageTitle = await consolePage.title().catch(() => '');

    // 容器页至少应返回可渲染内容（非空白错误页）
    const bodyText = await consolePage.locator('body').innerText().catch(() => '');
    log(`容器页 title=${result.consolePageTitle} url=${result.consolePageUrl} status=${result.consoleStatus}`);
    log(`容器页正文预览: ${bodyText.slice(0, 300).replace(/\s+/g, ' ')}`);

    if (!result.consolePageUrl || /无法访问|ERR_|404|502|503/.test(bodyText.slice(0, 500))) {
      throw new Error(`容器页面打开异常: url=${result.consolePageUrl} body=${bodyText.slice(0, 200)}`);
    }

    result.ok = true;
    log('✅ 完成：已启动并打开容器控制台页面');
  } catch (err) {
    result.error = String(err?.message || err);
    console.error('失败:', result.error);
    process.exitCode = 1;
  } finally {
    console.log('\n=== RESULT_JSON ===');
    console.log(JSON.stringify(result, null, 2));
    // 不关闭 CDP 浏览器，保留用户可见标签
  }
}

await main();
