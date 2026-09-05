// @ts-check
/**
 * 生产环境：登录 daydaymoney → 打开任务详情 →（可选）点击「启动服务器」→
 * 轮询直到出现「打开容器开发页面」→ 新窗口打开 VS Code / code-server。
 *
 * 凭据与 URL 一律走环境变量，勿写入仓库。
 *
 * 云平台 AccessKey / 授权与镜像侧 SSH 公钥占位符等，请在「工作空间设置 → 云平台」
 * 与镜像 userdata（如 `${TASK2APP_SSH_PUBLIC_KEY}`）中配置；本脚本只负责浏览器侧自动化。
 *
 * 用法（在 task2app/playwright 目录）：
 *   export DAYDAYMONEY_LOGIN_EMAIL=...
 *   export DAYDAYMONEY_LOGIN_PASSWORD=...
 *   export DAYDAYMONEY_TASK_DETAIL_URL='http://www.daydaymoney.com/tenant/.../task-detail/.../'
 *   node ./scripts/daydaymoney-open-task-vscode.mjs
 *
 * 常用环境变量：
 *   DAYDAYMONEY_LOGIN_URL          默认 http://www.daydaymoney.com/auth/login/
 *   DAYDAYMONEY_TASK_DETAIL_URL    任务详情完整 URL
 *   DAYDAYMONEY_CLICK_START        设为 1 时，若「启动服务器」可点则点击（默认 1）
 *   STARTUP_TIMEOUT_MS         等待 VS Code 按钮出现，默认 1800000（30 分钟）
 *   HEADED                     设为 0 则无头；默认 1（有界面，便于调试）
 *   SLOW_MO_MS                 操作间隔毫秒，默认 0
 *   CODE_SERVER_PASSWORD       若新开的 code-server 页要求密码，自动填入
 *   KEEP_OPEN_MS               VS Code 页打开后保持脚本运行的时间，默认 600000（10 分钟）
 *   DAYDAYMONEY_PREP_HARDWARE      默认 1：切换「服务器硬件配置」Tab，尝试「应用上一次运行配置」并点选首个「选择」实例，以便「启动服务器」可点
 */
import { chromium } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from '../tests/playwrightTenantEnv.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const LOGIN_URL = process.env.DAYDAYMONEY_LOGIN_URL || 'http://www.daydaymoney.com/auth/login/';
const TASK_URL =
  process.env.DAYDAYMONEY_TASK_DETAIL_URL ||
  `http://www.daydaymoney.com/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/839037065709281280/`;
const email = process.env.DAYDAYMONEY_LOGIN_EMAIL || '';
const password = process.env.DAYDAYMONEY_LOGIN_PASSWORD || '';
const clickStart = process.env.DAYDAYMONEY_CLICK_START !== '0';
const prepHardware = process.env.DAYDAYMONEY_PREP_HARDWARE !== '0';
const startupTimeoutMs = Number(process.env.STARTUP_TIMEOUT_MS || 1_800_000);
const headed = process.env.HEADED !== '0';
const slowMoMs = Number(process.env.SLOW_MO_MS || 0);
const codeServerPassword = process.env.CODE_SERVER_PASSWORD || '';
const keepOpenMs = Number(process.env.KEEP_OPEN_MS || 600_000);

async function login(page) {
  const privacyWait = page
    .waitForResponse((r) => r.url().includes('/api/privacy-policy/') && r.request().method() === 'GET', {
      timeout: 45_000,
    })
    .catch(() => null);
  const licenseWait = page
    .waitForResponse((r) => r.url().includes('/api/license-agreement/') && r.request().method() === 'GET', {
      timeout: 45_000,
    })
    .catch(() => null);

  await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded', timeout: 90_000 });
  await Promise.all([privacyWait, licenseWait]);

  await page.waitForSelector('[data-testid="login-privacy-accept"], #email', {
    state: 'visible',
    timeout: 120_000,
  });

  const emailPwdTab = page.getByRole('button', { name: /邮箱\/密码|邮箱.*密码/ }).first();
  if (await emailPwdTab.isVisible().catch(() => false)) {
    await emailPwdTab.click();
    await page.waitForTimeout(400);
  }

  const privacy = page.getByTestId('login-privacy-accept');
  const license = page.getByTestId('login-license-accept');
  await privacy.waitFor({ state: 'visible', timeout: 60_000 });
  await license.waitFor({ state: 'visible', timeout: 60_000 });
  await privacy.check();
  await license.check();

  await page
    .waitForFunction(
      () => {
        const btn = document.querySelector('form button[type="submit"]');
        return btn && !btn.disabled;
      },
      null,
      { timeout: 25_000 },
    )
    .catch(() => null);

  const form = page.locator('form').first();
  await form.locator('#email').fill(email);
  await form.locator('#password').fill(password);
  await form.getByRole('button', { name: /^登录$/ }).click();

  await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 90_000 }).catch(() => {});
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(800);

  if (page.url().includes('/auth/login')) {
    throw new Error('登录失败：仍在登录页，请检查账号密码或页面结构是否变更');
  }
}

/**
 * @param {import('playwright').Page} page
 */
async function ensureHardwareTab(page) {
  const tab = page.getByRole('button', { name: '服务器硬件配置' });
  if (await tab.isVisible({ timeout: 5000 }).catch(() => false)) {
    await tab.click();
    await page.waitForTimeout(400);
  }
}

/**
 * 解除「未选实例」等导致的启动按钮禁用：应用历史配置 + 选第一条可用实例。
 *
 * @param {import('playwright').Page} page
 */
async function prepareHardwareForStart(page) {
  if (!prepHardware) return;

  await ensureHardwareTab(page);

  const applyPrev = page.getByRole('button', { name: '应用上一次运行配置' });
  if (await applyPrev.isVisible({ timeout: 8000 }).catch(() => false)) {
    console.log('点击「应用上一次运行配置」…');
    await applyPrev.click();
    await page.waitForTimeout(2500);
  }

  const hw = page.locator('#hardware-config-section');
  const pickFirst = hw.getByRole('button', { name: '选择' }).first();
  if (await pickFirst.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log('在可用实例列表中点击首个「选择」…');
    await pickFirst.click();
    await page.waitForTimeout(800);
  }
}

/**
 * @param {import('playwright').Page} page
 */
async function dumpDebugArtifacts(page) {
  const statusEl = page.locator('#server-startup-status');
  if (await statusEl.isVisible().catch(() => false)) {
    const t = await statusEl.innerText().catch(() => '');
    console.error('--- #server-startup-status ---\n', t.slice(0, 8000));
  }
  const shot = path.join(__dirname, '..', 'tests', 'test_results', 'daydaymoney-open-vscode-failure.png');
  try {
    fs.mkdirSync(path.dirname(shot), { recursive: true });
    await page.screenshot({ path: shot, fullPage: true });
    console.error('已保存页面截图:', shot);
  } catch (_) {}
}

/**
 * @param {import('playwright').Page} page
 */
async function maybeClickStartServer(page) {
  const btn = page.locator('#start-server-btn');
  const vscodeBtn = page.locator('#open-container-vscode-btn');

  if (await vscodeBtn.isVisible().catch(() => false)) {
    console.log('已有「打开容器开发页面」按钮，跳过启动点击');
    return;
  }
  if (!clickStart) {
    console.log('DAYDAYMONEY_CLICK_START=0，不点击启动，仅等待 VS Code 按钮');
    return;
  }

  await btn.waitFor({ state: 'visible', timeout: 120_000 }).catch(() => null);
  const visible = await btn.isVisible().catch(() => false);
  if (!visible) {
    console.log('未找到「启动服务器」按钮（可能已在启动中或 UI 未加载），继续等待 VS Code 入口');
    return;
  }

  const disabled = await btn.isDisabled().catch(() => true);
  const label = (await btn.innerText().catch(() => '')).trim();
  if (disabled || !label.includes('启动服务器')) {
    console.log(`启动按钮状态: disabled=${disabled}, label=${label} — 不重复点击，等待就绪`);
    return;
  }

  console.log('点击「启动服务器」…');
  await btn.click();
}

/**
 * @param {import('playwright').Page} vscodePage
 */
async function maybeFillCodeServerPassword(vscodePage) {
  if (!codeServerPassword) return;
  const pwd = vscodePage.locator('input[type="password"]').first();
  if (await pwd.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log('检测到 code-server 密码框，正在填入 CODE_SERVER_PASSWORD');
    await pwd.fill(codeServerPassword);
    const submit = vscodePage.getByRole('button', { name: /提交|登录|Sign in|Log in/i }).first();
    if (await submit.isVisible().catch(() => false)) await submit.click();
  }
}

async function main() {
  if (!email || !password) {
    console.error('请设置环境变量 DAYDAYMONEY_LOGIN_EMAIL 与 DAYDAYMONEY_LOGIN_PASSWORD');
    process.exit(2);
  }

  const browser = await chromium.launch({
    headless: !headed,
    slowMo: slowMoMs || undefined,
  });
  const context = await browser.newContext();
  const page = await context.newPage();

  page.on('console', (msg) => {
    if (process.env.DEBUG_PAGE_CONSOLE === '1') {
      console.log(`[page ${msg.type()}]`, msg.text());
    }
  });

  console.log('登录…');
  await login(page);

  console.log('打开任务详情:', TASK_URL);
  await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 120_000 });
  await page.waitForLoadState('domcontentloaded');

  await prepareHardwareForStart(page);
  await maybeClickStartServer(page);

  console.log(`等待「打开容器开发页面」出现（最长 ${Math.round(startupTimeoutMs / 60000)} 分钟）…`);
  const vscodeLink = page.locator('#open-container-vscode-btn');
  try {
    await vscodeLink.waitFor({ state: 'visible', timeout: startupTimeoutMs });
  } catch (e) {
    await dumpDebugArtifacts(page);
    throw e;
  }

  const href = await vscodeLink.getAttribute('href');
  if (!href || href === '') {
    throw new Error('VS Code 链接 href 为空');
  }
  console.log('容器开发 URL:', href);

  const popupPromise = page.waitForEvent('popup', { timeout: 30_000 });
  await vscodeLink.scrollIntoViewIfNeeded();
  await vscodeLink.click();
  const vscodePage = await popupPromise;

  await vscodePage.waitForLoadState('domcontentloaded', { timeout: 120_000 }).catch(() => {});
  await maybeFillCodeServerPassword(vscodePage);

  console.log('VS Code / code-server 页:', vscodePage.url());
  console.log(`保持浏览器 ${Math.round(keepOpenMs / 1000)} 秒（可改 KEEP_OPEN_MS），便于查看日志与调试…`);
  await vscodePage.waitForTimeout(keepOpenMs);

  await browser.close();
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
