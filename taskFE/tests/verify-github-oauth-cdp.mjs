// @ts-check
/**
 * 通过 Chrome DevTools（默认 9222）控制**已有**标签页，完整走 GitHub App 用户授权并打印结果。
 * 使用 chrome-remote-interface（Playwright 的 connectOverCDP 在本机 Chrome 147 上对 browser 级 ws 易挂起）。
 *
 * 用法（密码走环境变量，勿写入仓库）：
 * CDP_PORT=9222 SITE_BASE=http://localhost:4000 \
 * GIT_SITE_OAUTH_PATH=/user/<id>/profile/git-site-oauth/ \
 * E2E_EMAIL=… E2E_PASSWORD='…' GH_EMAIL=… GH_PASSWORD='…' \
 * （工作目录 task2app/playwright）node taskFE/tests/verify-github-oauth-cdp.mjs
 *
 * 若需 **Playwright** `connectOverCDP` 版本并在结束后校验 gitOauth 库，见同目录
 * `verify-git-oauth-playwright-cdp.mjs`（`npm run verify:git-oauth-cdp`）。
 */
import CDP from 'chrome-remote-interface';

const PORT = Number(process.env.CDP_PORT || '9222', 10);
const BASE = process.env.SITE_BASE?.trim()?.replace(/\/$/, '') || 'http://localhost:4000';
/** 含前导 `/`，与路由一致，默认与「用户中心」资料下 git oauth 页一致 */
const GIT_SITE_PATH =
  (process.env.GIT_SITE_OAUTH_PATH || '/user/827923618451263488/profile/git-site-oauth/').trim() ||
  '/profile/git-site-oauth/';
const oauthPathNorm = GIT_SITE_PATH.startsWith('/') ? GIT_SITE_PATH : `/${GIT_SITE_PATH}`;
const LS_TTL_MS = Number(process.env.GITHUB_APP_RETURN_TTL_MS || '120000', 10) || 120000;
const APP_EMAIL = process.env.E2E_EMAIL || '';
const APP_PW = process.env.E2E_PASSWORD || '';
const GH_EMAIL = process.env.GH_EMAIL || '';
const GH_PW = process.env.GH_PASSWORD || '';

function requireEnv(name, v) {
  if (!v) {
    console.error(`缺少环境变量 ${name}`);
    process.exit(1);
  }
}

function ev(client, expression) {
  return client.Runtime.evaluate({ expression, awaitPromise: true, returnByValue: true });
}

/** @param {import('chrome-remote-interface').Client} client */
async function waitUrl(client, predicate, timeoutMs = 120000) {
  const t0 = Date.now();
  while (Date.now() - t0 < timeoutMs) {
    const { result } = await ev(client, 'location.href');
    const u = String(result.value || '');
    if (predicate(u)) return u;
    await new Promise((r) => setTimeout(r, 400));
  }
  const { result } = await ev(client, 'location.href');
  return String(result.value || '');
}

async function main() {
  requireEnv('E2E_EMAIL', APP_EMAIL);
  requireEnv('E2E_PASSWORD', APP_PW);
  requireEnv('GH_EMAIL', GH_EMAIL);
  requireEnv('GH_PASSWORD', GH_PW);

  const host = new URL(BASE).host;
  let targets = await CDP.List({ port: PORT });
  let pick =
    targets.find(
      (x) =>
        x.type === 'page' &&
        String(x.url || '').includes(host) &&
        String(x.url || '').includes('git-site-oauth'),
    ) ||
    targets.find((x) => x.type === 'page' && String(x.url || '').includes(host)) ||
    targets.find((x) => x.type === 'page' && /^https?:\/\//.test(String(x.url || '')));
  if (!pick?.id) {
    console.error('未在 CDP /json/list 中找到可操作的 http(s) 页面，请先打开', BASE);
    process.exit(1);
  }
  /** 避免误用其它 localhost 标签页（如任务详情），必要时新开 git-site-oauth */
  if (!String(pick.url || '').includes('git-site-oauth')) {
    console.log('当前标签非 git-site-oauth（', String(pick.url || '').slice(0, 100), '），新开标签页');
    const created = await CDP.New({
      port: PORT,
      url: `${BASE}${oauthPathNorm.endsWith('/') ? oauthPathNorm : `${oauthPathNorm}/`}`,
      width: 1280,
      height: 900,
    });
    targets = await CDP.List({ port: PORT });
    pick = targets.find((x) => x.type === 'page' && x.id === created.id) || {
      id: created.id,
      url: created.url,
    };
  }
  console.log('使用 CDP 目标标签:', pick.url?.slice(0, 120));

  const client = await CDP({ port: PORT, target: pick.id });
  const { Page, Runtime, Network } = client;
  await Page.enable();
  await Runtime.enable();
  await Network.enable();

  const logConnection = (url, body) => {
    if (String(url).includes('/api/accounts/github/app/connection')) {
      console.log('[network] connection', String(url).slice(-80), body?.slice?.(0, 400));
    }
  };
  Network.responseReceived(async (e) => {
    const u = e.response?.url || '';
    if (!u.includes('/api/accounts/github/app/connection')) return;
    try {
      const body = await Network.getResponseBody({ requestId: e.requestId }).catch(() => null);
      logConnection(u, body?.body);
    } catch {
      /* ignore */
    }
  });

  const siteLogin = async () => {
    await Page.navigate({ url: `${BASE}/auth/login/` });
    await Page.loadEventFired();
    await new Promise((r) => setTimeout(r, 800));
    const email = JSON.stringify(APP_EMAIL);
    const pw = JSON.stringify(APP_PW);
    await ev(
      client,
      `(async () => {
        const tab = document.querySelector('button') && [...document.querySelectorAll('button')].find(b => /邮箱/.test(b.textContent || ''));
        if (tab) tab.click();
        await new Promise(r => setTimeout(r, 300));
        const e = document.querySelector('#email');
        const p = document.querySelector('#password');
        if (!e || !p) throw new Error('login form missing');
        e.value = ${email};
        p.value = ${pw};
        const accept = document.querySelector('[data-testid="login-accept-all"]');
        if (accept) accept.click();
        document.querySelector('form')?.requestSubmit();
      })()`,
    );
    await new Promise((r) => setTimeout(r, 2500));
  };

  await siteLogin();

  const oauthLanding = `${BASE}${oauthPathNorm.endsWith('/') ? oauthPathNorm : `${oauthPathNorm}/`}`;
  console.log('[navigate] git oauth 页面:', oauthLanding);
  await Page.navigate({ url: oauthLanding });
  await Page.loadEventFired();
  await new Promise((r) => setTimeout(r, 2000));

  const apiConn = `${BASE}/api/accounts/github/app/connection/`;
  const before = await ev(
    client,
    `(async () => {
      const u = ${JSON.stringify(apiConn)};
      const r = await fetch(u, { headers: { Accept: 'application/json' }, credentials: 'include' });
      return { status: r.status, body: await r.json() };
    })()`,
  );
  console.log('[before OAuth] connection API:', JSON.stringify(before.result.value));

  const t0 = Date.now();
  /** 与 Vue 前端一致：写好 localStorage 后调用 start API；跳转用 Page.navigate（evaluate 内设 location.href 到外链可能不触发导航）。 */
  const started = await ev(
    client,
    `(async () => {
      try {
        const nextPath = window.location.pathname + (window.location.search || '');
        const a = new Uint8Array(16);
        crypto.getRandomValues(a);
        const returnKey = Array.from(a, (x) => x.toString(16).padStart(2, '0')).join('');
        const STORAGE_PREFIX = 'github_app_return:';
        localStorage.setItem(
          STORAGE_PREFIX + returnKey,
          JSON.stringify({ return_url: nextPath, expires_at: Date.now() + ${LS_TTL_MS} }),
        );
        const startUrl =
          '/api/git-oauth/github-app-start/?next=' +
          encodeURIComponent(nextPath) +
          '&return_key=' +
          encodeURIComponent(returnKey);
        const r = await fetch(startUrl, { credentials: 'include', headers: { Accept: 'application/json' } });
        const data = await r.json().catch(() => ({}));
        if (!r.ok) return JSON.stringify({ ok: false, status: r.status, detail: data.detail || data });
        if (!data.authorize_url) return JSON.stringify({ ok: false, reason: 'no_authorize_url', data });
        return JSON.stringify({ ok: true, authorize_url: String(data.authorize_url) });
      } catch (e) {
        return JSON.stringify({ ok: false, err: String(e && e.message ? e.message : e) });
      }
    })()`,
  );
  /** @type {{ ok?: boolean; authorize_url?: string; detail?: unknown }} */
  let sv = {};
  try {
    sv = JSON.parse(String(started.result.value ?? '{}'));
  } catch {
    sv = {};
  }
  console.log('[oauth start]', sv?.ok ? `ok, authorize URL host=${new URL(String(sv.authorize_url)).hostname}` : JSON.stringify(sv));
  if (!sv?.ok || !sv.authorize_url) {
    console.log('无法启动 GitHub 授权（参见 [oauth start]）');
    await client.close();
    return;
  }
  await Page.navigate({ url: sv.authorize_url });
  await Page.loadEventFired().catch(() => {});
  const immediate = await ev(client, `location.href`);
  console.log('[post Page.navigate]', String(immediate?.result?.value || ''));

  console.log(
    `--- 已在 GitHub 流程中（localStorage TTL=${LS_TTL_MS}ms：从发起 start 时起算，超限会 return_expired）---`,
  );
  let u =
    String((await ev(client, 'location.href')).result.value || '').trim() ||
    (await waitUrl(client, (x) => x.includes('github.com'), 15000));
  console.log('[github] URL:', u.slice(0, 280));

  for (let step = 0; step < 24; step++) {
    u = String((await ev(client, 'location.href')).result.value || '').trim();
    console.log(`[step ${step + 1}]`, u.slice(0, 200));

    if (u.includes(host) && (u.includes('/oauth/github/callback') || u.includes('/profile/git-site-oauth'))) {
      break;
    }

    if (u.includes('github.com/login') && !u.includes('/oauth/')) {
      const ge = JSON.stringify(GH_EMAIL);
      const gp = JSON.stringify(GH_PW);
      await ev(
        client,
        `(async () => {
          const login = document.querySelector('#login_field') || document.querySelector('input[name="login"]');
          const pw = document.querySelector('#password') || document.querySelector('input[name="password"]');
          if (login && pw) {
            login.value = ${ge};
            pw.value = ${gp};
            const s = document.querySelector('input[type="submit"][name="commit"]') || document.querySelector('input[type="submit"]');
            s?.click();
          }
        })()`,
      );
      await new Promise((r) => setTimeout(r, 2500));
      continue;
    }

    if (u.includes('/login/oauth/authorize')) {
      await ev(
        client,
        `(async () => {
          const primary =
            document.querySelector('button[type="submit"].Button--primary') ||
            document.querySelector('form button.Button--primary') ||
            [...document.querySelectorAll('button')].find((x) => /Authorize/i.test(x.textContent || ''));
          if (primary) primary.click();
        })()`,
      );
      await new Promise((r) => setTimeout(r, 3500));
      continue;
    }

    if (u.includes(host)) break;
    await new Promise((r) => setTimeout(r, 700));
  }

  await new Promise((r) => setTimeout(r, 2500));
  const finalU = (await ev(client, 'location.href')).result.value;
  console.log('\n--- 最终 URL ---\n', String(finalU));

  const after = await ev(
    client,
    `(async () => {
      const u = ${JSON.stringify(apiConn)};
      const r = await fetch(u, { headers: { Accept: 'application/json' }, credentials: 'include' });
      return { status: r.status, body: await r.json() };
    })()`,
  );
  console.log('[after OAuth] connection API:', JSON.stringify(after.result.value));

  const txt = await ev(client, 'document.body ? document.body.innerText.slice(0, 2500) : ""');
  console.log('\n--- 页面正文（截断）---\n', String(txt.result.value || ''));

  const sec = (Date.now() - t0) / 1000;
  console.log(`\n从点击「使用 GitHub 授权」到结束约 ${sec.toFixed(1)} 秒`);
  if (String(finalU).includes('github=no_refresh')) {
    console.log(
      '\n结论: 回调 URL 含 github=no_refresh —— GitHub 未返回 refresh_token。请在 GitHub App 中启用 user token rotation，见 GITHUB_APP_DEPLOYMENT.md。',
    );
  } else if (String(finalU).includes('github=return_expired')) {
    console.log('\n结论: 回跳信息超过 30 秒过期（githubAppReturnStorage TTL），请加快授权或延长 TTL。');
  } else if (after.result.value?.body?.connected) {
    console.log('\n结论: API 已报告 connected=true。');
  } else {
    console.log('\n结论: API 仍为未绑定；请结合 URL 中 github= 参数与上方文案排查。');
  }

  await client.close();
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
