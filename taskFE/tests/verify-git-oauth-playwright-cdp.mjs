// @ts-check
/**
 * 使用 Playwright `chromium.connectOverCDP` 连接本机 Chrome（默认 9222），在
 * `http://localhost:4000/user/<id>/profile/git-site-oauth/` 完成 GitHub App 授权，
 * 再通过 gitOauth 内部 HTTP 校验 `GithubAppUserCredential` 是否包含该 task2app 用户。
 *
 * 前置：Chrome 已 `--remote-debugging-port=9222` 启动；前端 :4000、Django、gitOauth 可访问；
 * 登录账号的 cookie `userId` 必须与 URL 中的用户 ID 一致（本人资料页才可点「使用 GitHub 授权」）。
 *
 * 工作目录：`task2app/playwright`
 *
 * ```bash
 * CDP_URL=http://127.0.0.1:9222 SITE_BASE=http://localhost:4000 \
 * GIT_SITE_OAUTH_PATH=/user/827923618451263488/profile/git-site-oauth/ \
 * E2E_EMAIL='…' E2E_PASSWORD='…' GH_EMAIL='…' GH_PASSWORD='…' \
 * node taskFE/tests/verify-git-oauth-playwright-cdp.mjs
 * ```
 *
 * 可选：`GITOAUTH_VERIFY_BASE` 覆盖 conf 聚合的 `addressing.gitoauth`（OPT-20260806-053:
 * Django 退役，原 django.gitoauth 迁移至 _addressing.addresses.gitoauth）；
 * `GITOAUTH_BRIDGE_JWT_SECRET` 覆盖桥接密钥（默认读 port_config 的 `task2appSsoJwtSecret`）。
 */
import { chromium } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const CDP_URL = (process.env.CDP_URL || 'http://127.0.0.1:9222').trim().replace(/\/$/, '');
const SITE_BASE = (process.env.SITE_BASE || 'http://localhost:4000').trim().replace(/\/$/, '');
const GIT_SITE_PATH =
  (
    process.env.GIT_SITE_OAUTH_PATH ||
    '/user/827923618451263488/profile/git-site-oauth/'
  ).trim() || '/profile/git-site-oauth/';
const oauthPathNorm = GIT_SITE_PATH.startsWith('/') ? GIT_SITE_PATH : `/${GIT_SITE_PATH}`;
const oauthLanding = `${SITE_BASE}${oauthPathNorm.endsWith('/') ? oauthPathNorm : `${oauthPathNorm}/`}`;

const APP_EMAIL = process.env.E2E_EMAIL || '';
const APP_PW = process.env.E2E_PASSWORD || '';
const GH_EMAIL = process.env.GH_EMAIL || '';
const GH_PW = process.env.GH_PASSWORD || '';

function readPortConfig() {
  try {
    const raw = loadPortConfig();
    // OPT-20260806-053: Django 退役，django.gitoauth 已移除 →
    // 迁移到 _addressing.addresses.gitoauth（conf/base.yaml subdomains.gitoauth）
    const addressing = raw?._addressing?.addresses;
    const gitoauthHost =
      addressing && typeof addressing.gitoauth === 'string' ? addressing.gitoauth.trim() : '';
    const scheme = addressing?.scheme || 'https';
    const gitoauthBase = gitoauthHost ? `${scheme}://${gitoauthHost}` : '';
    const bridge =
      typeof raw.task2appSsoJwtSecret === 'string' ? raw.task2appSsoJwtSecret.trim() : '';
    return { gitoauthBase: gitoauthBase.replace(/\/$/, ''), bridge };
  } catch {
    return { gitoauthBase: '', bridge: '' };
  }
}

function parseProfileUserId() {
  const fromEnv = (process.env.TASK2APP_PROFILE_USER_ID || '').trim();
  if (fromEnv) return fromEnv;
  const m = oauthPathNorm.match(/\/user\/(\d+)\//);
  return m ? m[1] : '';
}

function requireEnv(name, v) {
  if (!v) {
    console.error(`缺少环境变量 ${name}`);
    process.exit(1);
  }
}

/**
 * @param {string} base
 * @param {string} secret
 * @param {string} userIdStr
 */
async function assertGitOauthHasCredential(base, secret, userIdStr) {
  const uid = Number(userIdStr);
  if (!Number.isFinite(uid) || uid <= 0) {
    throw new Error(`无效 task2app 用户 ID: ${userIdStr}`);
  }
  const url = `${base.replace(/\/$/, '')}/api/internal/github/oauth/user-credential/user-ids/`;
  const r = await fetch(url, {
    headers: {
      Accept: 'application/json',
      'X-GitOauth-Bridge-Secret': secret,
    },
  });
  const text = await r.text();
  if (!r.ok) {
    throw new Error(`gitOauth user-ids HTTP ${r.status}: ${text.slice(0, 500)}`);
  }
  let j;
  try {
    j = JSON.parse(text);
  } catch {
    throw new Error(`gitOauth user-ids 非 JSON: ${text.slice(0, 200)}`);
  }
  const ids = j?.user_ids;
  if (!Array.isArray(ids)) {
    throw new Error(`gitOauth 响应缺少 user_ids: ${text.slice(0, 300)}`);
  }
  const has = ids.map((x) => Number(x)).includes(uid);
  if (!has) {
    throw new Error(
      `gitOauth 库中未找到 task2app_user_id=${uid}；当前 user_ids 前 50 个: ${JSON.stringify(ids.slice(0, 50))}`,
    );
  }
  console.log(`[gitOauth] user-ids 校验通过：已包含 task2app_user_id=${uid}。`);

  const accessUrl = `${base.replace(/\/$/, '')}/api/internal/github/oauth/access-for-user/`;
  const ar = await fetch(accessUrl, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'X-GitOauth-Bridge-Secret': secret,
    },
    body: JSON.stringify({ user_id: uid }),
  });
  const atext = await ar.text();
  if (ar.status === 404) {
    throw new Error(`gitOauth access-for-user 404（无 refresh 或凭据损坏）: ${atext.slice(0, 300)}`);
  }
  if (!ar.ok) {
    throw new Error(`gitOauth access-for-user HTTP ${ar.status}: ${atext.slice(0, 500)}`);
  }
  let aj;
  try {
    aj = JSON.parse(atext);
  } catch {
    throw new Error(`access-for-user 非 JSON: ${atext.slice(0, 200)}`);
  }
  const tok = typeof aj?.access_token === 'string' ? aj.access_token.trim() : '';
  if (!tok) {
    throw new Error(`gitOauth access-for-user 未返回 access_token: ${atext.slice(0, 300)}`);
  }
  console.log('[gitOauth] access-for-user 换票成功（已拿到 access_token，长度 %d）。', tok.length);
}

async function githubOAuthLoop(page, siteHost) {
  const max = 28;
  for (let step = 0; step < max; step++) {
    const u = page.url();

    if (
      u.includes(siteHost) &&
      (u.includes('/oauth/github/callback') || u.includes('/profile/git-site-oauth'))
    ) {
      return u;
    }

    if (u.includes('github.com/login') && !u.includes('/oauth/')) {
      await page.locator('#login_field, input[name="login"]').first().fill(GH_EMAIL);
      await page.locator('#password, input[name="password"]').first().fill(GH_PW);
      await page.locator('input[type="submit"][name="commit"], input[type="submit"]').first().click();
      await page.waitForTimeout(2500);
      continue;
    }

    if (u.includes('github.com') && u.includes('/login/oauth/authorize')) {
      const primary = page
        .locator('button[type="submit"].Button--primary, form button.Button--primary')
        .first();
      const named = page.getByRole('button', { name: /Authorize/i }).first();
      if (await primary.isVisible().catch(() => false)) {
        await primary.click();
      } else if (await named.isVisible().catch(() => false)) {
        await named.click();
      } else {
        await page.locator('form').first().evaluate((form) => form.requestSubmit());
      }
      await page.waitForTimeout(3500);
      continue;
    }

    if (u.includes(siteHost)) {
      return u;
    }

    await page.waitForTimeout(700);
  }
  return page.url();
}

async function main() {
  requireEnv('E2E_EMAIL', APP_EMAIL);
  requireEnv('E2E_PASSWORD', APP_PW);
  requireEnv('GH_EMAIL', GH_EMAIL);
  requireEnv('GH_PASSWORD', GH_PW);

  const profileUserId = parseProfileUserId();
  if (!profileUserId) {
    console.error('无法从 GIT_SITE_OAUTH_PATH 解析用户 ID，请设置 TASK2APP_PROFILE_USER_ID');
    process.exit(1);
  }

  const siteHost = new URL(SITE_BASE).host;

  let browser;
  try {
    browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
  } catch (e) {
    console.error(
      `无法连接 CDP ${CDP_URL}：`,
      e?.message || e,
      '\n请先启动 Chrome（例如仓库根目录 task2app/runDebugChrome.sh，--remote-debugging-port=9222）。',
    );
    process.exit(1);
  }

  const contexts = browser.contexts();
  if (!contexts.length) {
    console.error('CDP 浏览器无可用 context');
    process.exit(1);
  }
  const context = contexts[0];
  const page = await context.newPage();

  try {
    await playwrightLoginWithLegalAccept(page, {
      email: APP_EMAIL,
      password: APP_PW,
      baseURL: SITE_BASE,
    });

    console.log('[navigate]', oauthLanding);
    await page.goto(oauthLanding, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.locator('[data-alias="view-user-git-site-oauth"]').waitFor({
      state: 'visible',
      timeout: 30000,
    });

    const warn = page.locator('.text-amber-900').first();
    if (await warn.isVisible().catch(() => false)) {
      const t = await warn.innerText().catch(() => '');
      if (t.includes('本人')) {
        throw new Error(
          '页面提示非本人资料：URL 中的用户 ID 须与当前登录账号一致。请使用对应账号的 E2E_EMAIL / E2E_PASSWORD。',
        );
      }
    }

    const useGithub = page.getByRole('button', { name: /使用 GitHub 授权/ });
    const reAuth = page.getByRole('button', { name: /重新授权/ });
    if (await useGithub.isVisible().catch(() => false)) {
      await useGithub.click();
    } else if (await reAuth.isVisible().catch(() => false)) {
      await reAuth.click();
    } else {
      throw new Error('未找到「使用 GitHub 授权」或「重新授权」按钮（可能仍在加载或非本人页）');
    }

    await page.waitForURL(/github\.com/, { timeout: 120000 });

    const finalU = await githubOAuthLoop(page, siteHost);
    console.log('[final url]', finalU.slice(0, 260));

    await page.waitForTimeout(2000);

    const connUrl = `${SITE_BASE}/api/accounts/github/app/connection/`;
    const connRes = await page.request.get(connUrl, { headers: { Accept: 'application/json' } });
    const connBody = await connRes.json().catch(() => ({}));
    console.log('[task2app] connection API', connRes.status(), JSON.stringify(connBody));

    const { gitoauthBase: cfgBase, bridge: cfgBridge } = readPortConfig();
    const gitoauthBase = (process.env.GITOAUTH_VERIFY_BASE || cfgBase || '').trim().replace(/\/$/, '');
    const bridgeSecret = (
      process.env.GITOAUTH_BRIDGE_JWT_SECRET ||
      process.env.TASK2APP_SSO_JWT_SECRET ||
      cfgBridge ||
      ''
    ).trim();

    if (!gitoauthBase || !bridgeSecret) {
      throw new Error(
        '无法校验 gitOauth：请配置 addressing.gitoauth 与 task2appSsoJwtSecret，或设置 GITOAUTH_VERIFY_BASE、GITOAUTH_BRIDGE_JWT_SECRET',
      );
    }

    await assertGitOauthHasCredential(gitoauthBase, bridgeSecret, profileUserId);

    if (String(finalU).includes('github=no_refresh')) {
      console.warn(
        '[warn] URL 含 github=no_refresh；GitHub 未返回 refresh_token，但凭据接口若仍成功请检查 App 设置。',
      );
    }

    console.log('\n完成：主站 connection 与 gitOauth 凭据 / 换票校验均已执行。');
  } finally {
    await page.close().catch(() => {});
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
