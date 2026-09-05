/**
 * CDP 核验：公网 www+api 跨子域 GitLab SSO（userId Domain 共享 + authorize 续接）。
 * 由 Gitlab.daydaymoney-cross-subdomain-sso.playwright.test.sh 调用。
 */
import { chromium } from 'playwright';
import {
  buildOidcAuthorizeNextUrl,
  clickGitlabTaskAuthSso,
  isLoggedIntoGitLab,
  isMainSiteHomepage,
  openGitlabSignInFromProjects,
  readGitlabSsoPublicEnv,
} from './helpers/gitlabTaskAuthSsoE2e.js';

const CDP_URL = (process.env.CDP_URL || process.env.PW_CDP_URL || 'http://127.0.0.1:9222').replace(
  /\/$/,
  '',
);
const env = readGitlabSsoPublicEnv();
const OIDC_NEXT = buildOidcAuthorizeNextUrl(env.gatewayUrl, env.gitlabUrl);

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

/** 公网 SPA 登录（避免共享 helper 在 CDP 下偶发卡死）。 */
async function loginOnWww(page) {
  await page.goto(`${env.siteOrigin}/auth/login/`, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await page.waitForSelector('#email', { timeout: 90000 });
  if (!page.url().includes('/auth/login')) {
    return;
  }
  const tab = page.getByRole('button', { name: /邮箱\/密码/ }).first();
  if (await tab.isVisible().catch(() => false)) {
    await tab.click();
    await page.waitForTimeout(300);
  }
  for (const tid of ['login-accept-all', 'login-privacy-accept', 'login-license-accept']) {
    const el = page.getByTestId(tid);
    if (!(await el.isVisible().catch(() => false))) continue;
    if (!(await el.isChecked().catch(() => false))) {
      await el.check({ timeout: 5000 }).catch(() => {});
    }
  }
  await page.fill('#email', env.email);
  await page.fill('#password', env.password);
  const authPromise = page.waitForResponse(
    (r) => /\/api\/auth\/?$/.test(new URL(r.url()).pathname) && r.request().method() === 'POST',
    { timeout: 60000 },
  );
  await page.locator('form').first().evaluate((form) => form.requestSubmit());
  const authResp = await authPromise;
  assert(authResp.status() >= 200 && authResp.status() < 300, `登录 API ${authResp.status()}`);
  await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 60000 });
}

const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 15000 });
const context = browser.contexts()[0] || (await browser.newContext());
const page = await context.newPage();

try {
  assert(env.siteOrigin.includes('www.'), `siteOrigin 须为 www: ${env.siteOrigin}`);
  assert(env.gatewayUrl.includes('api.'), `gatewayUrl 须为 api: ${env.gatewayUrl}`);
  assert(
    new URL(env.siteOrigin).hostname !== new URL(env.gatewayUrl).hostname,
    'www 与 api 主机名必须不同',
  );
  assert(env.email && env.password, '缺少登录凭证');

  console.log('[cross-subdomain-sso] login', env.siteOrigin);
  await loginOnWww(page);
  console.log('[cross-subdomain-sso] after login', page.url());

  const apiCookies = await context.cookies(env.gatewayUrl);
  const userIdCookie = apiCookies.find((c) => c.name === 'userId');
  console.log(
    '[cross-subdomain-sso] api cookies userId=',
    userIdCookie
      ? { value: String(userIdCookie.value).slice(0, 8) + '…', domain: userIdCookie.domain }
      : null,
  );
  assert(userIdCookie, '登录后 api 子域应能看到 userId（共享 Domain）');
  assert(
    /^\.?daydaymoney\.com$/.test(userIdCookie.domain || ''),
    `Cookie Domain 应为 .example.com，实际: ${userIdCookie.domain}`,
  );

  const authorizePage = await context.newPage();
  const navUrls = [];
  authorizePage.on('framenavigated', (frame) => {
    if (frame === authorizePage.mainFrame()) navUrls.push(frame.url());
  });
  await authorizePage.goto(OIDC_NEXT, { waitUntil: 'domcontentloaded', timeout: 90000 });
  await authorizePage.waitForTimeout(2500);
  const authorizeUrl = authorizePage.url();
  console.log('[cross-subdomain-sso] authorize final', authorizeUrl, 'nav', navUrls.join(' -> '));
  assert(!authorizeUrl.includes('/auth/login'), `authorize 不应停在登录页: ${authorizeUrl}`);
  assert(
    authorizeUrl.includes('gitlab') || authorizeUrl.includes('code=') || authorizeUrl.includes('/users/'),
    `期望进入 GitLab/OIDC 会话，实际: ${authorizeUrl}`,
  );
  await authorizePage.close();

  console.log('[cross-subdomain-sso] projects → gitlab SSO');
  const gitlabPage = await openGitlabSignInFromProjects(page, context, env);
  console.log('[cross-subdomain-sso] gitlab entry', gitlabPage.url());
  assert(/gitlab/i.test(gitlabPage.url()), `应进入 GitLab: ${gitlabPage.url()}`);

  if (!isLoggedIntoGitLab(gitlabPage.url(), env.gitlabUrl)) {
    await clickGitlabTaskAuthSso(gitlabPage);
    await gitlabPage
      .waitForURL(
        (url) =>
          isLoggedIntoGitLab(url.toString(), env.gitlabUrl) ||
          isMainSiteHomepage(url.toString(), env.siteOrigin) ||
          url.toString().includes('/auth/login'),
        { timeout: 120000 },
      )
      .catch(async () => {
        await gitlabPage.waitForTimeout(5000);
      });
  }

  const finalUrl = gitlabPage.url();
  const bodyText = await gitlabPage.textContent('body').catch(() => '');
  console.log('[cross-subdomain-sso] SSO final', finalUrl);
  assert(!isMainSiteHomepage(finalUrl, env.siteOrigin), `不应跳回主站首页: ${finalUrl}`);
  assert(!finalUrl.includes('/auth/login'), `不应停留在登录页: ${finalUrl}`);
  assert(!/404 page not found/i.test(bodyText || ''), '不应出现 404');
  assert(!/Could not authenticate you from OpenIDConnect/i.test(bodyText || ''), 'OIDC 认证失败');
  assert(isLoggedIntoGitLab(finalUrl, env.gitlabUrl), `期望进入 GitLab，实际: ${finalUrl}`);

  console.log('PASS');
  process.exit(0);
} catch (err) {
  console.error('FAIL', err?.message || err);
  process.exit(1);
} finally {
  await page.close().catch(() => {});
}
