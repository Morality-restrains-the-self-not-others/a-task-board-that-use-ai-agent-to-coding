// @ts-check
/**
 * 公网跨子域 SSO 回归：www 已登录 → api OIDC authorize → gitlab SSO，
 * 验证 Domain=.example.com 的 userId Cookie 可被 api 子域读取。
 *
 * 运行：
 *   bash taskFE/tests/Gitlab.daydaymoney-cross-subdomain-sso.playwright.test.sh
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import {
  buildOidcAuthorizeNextUrl,
  clickGitlabTaskAuthSso,
  isLoggedIntoGitLab,
  isMainSiteHomepage,
  openGitlabSignInFromProjects,
  readGitlabSsoPublicEnv,
} from './helpers/gitlabTaskAuthSsoE2e.js';

const env = readGitlabSsoPublicEnv();
const OIDC_NEXT = buildOidcAuthorizeNextUrl(env.gatewayUrl, env.gitlabUrl);

test.describe('daydaymoney www+api 跨子域 GitLab SSO @cross-subdomain-sso', () => {
  test('登录后 userId Cookie 对 api 子域可见，且 authorize 续接不回登录表单循环', async ({
    page,
    context,
  }) => {
    test.setTimeout(180000);
    test.skip(!env.email || !env.password, '请设置 PW_EMAIL / PW_PASSWORD（或 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD）');

    expect(env.siteOrigin, 'siteOrigin 须为 www 公网').toContain('www.');
    expect(env.gatewayUrl, 'gatewayUrl 须为 api 公网').toContain('api.');
    expect(new URL(env.siteOrigin).hostname).not.toBe(new URL(env.gatewayUrl).hostname);

    await playwrightLoginWithLegalAccept(page, {
      email: env.email,
      password: env.password,
      baseURL: env.siteOrigin,
    });

    const apiCookies = await context.cookies(env.gatewayUrl);
    const userIdCookie = apiCookies.find((c) => c.name === 'userId');
    expect(userIdCookie, '登录后 api 子域应能看到 userId（共享 Domain）').toBeTruthy();
    expect(userIdCookie?.domain || '', 'Cookie Domain 应为父域 .example.com').toMatch(/^\.?daydaymoney\.com$/);

    const authorizePage = await context.newPage();
    const navUrls = [];
    authorizePage.on('framenavigated', (frame) => {
      if (frame === authorizePage.mainFrame()) navUrls.push(frame.url());
    });
    await authorizePage.goto(OIDC_NEXT, { waitUntil: 'domcontentloaded', timeout: 90000 });
    await authorizePage.waitForTimeout(2000);

    const authorizeUrl = authorizePage.url();
    expect(authorizeUrl, 'authorize 不应停在主站登录页').not.toContain('/auth/login');
    expect(
      authorizeUrl.includes('gitlab') || authorizeUrl.includes('code=') || authorizeUrl.includes('/users/'),
      `期望进入 GitLab OIDC 回调/会话，实际: ${authorizeUrl} 导航: ${navUrls.join(' -> ')}`,
    ).toBeTruthy();
    await authorizePage.close();
  });

  test('浏览器：项目列表 → 代码仓库 → taskAuth SSO 首次成功（公网子域）', async ({ page, context }) => {
    test.setTimeout(240000);
    test.skip(!env.email || !env.password, '请设置 PW_EMAIL / PW_PASSWORD（或 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD）');

    await playwrightLoginWithLegalAccept(page, {
      email: env.email,
      password: env.password,
      baseURL: env.siteOrigin,
    });

    const gitlabPage = await openGitlabSignInFromProjects(page, context, env);
    expect(gitlabPage.url(), '应进入 GitLab').toMatch(/gitlab/i);

    const navigationUrls = [];
    gitlabPage.on('framenavigated', (frame) => {
      if (frame === gitlabPage.mainFrame()) navigationUrls.push(frame.url());
    });

    // 若已因上次会话登录，直接通过；否则点 SSO
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

    expect(isMainSiteHomepage(finalUrl, env.siteOrigin), `不应跳回主站首页: ${finalUrl}`).toBe(false);
    expect(finalUrl, '不应停留在主站登录页').not.toContain('/auth/login');
    expect(bodyText).not.toMatch(/404 page not found/i);
    expect(bodyText).not.toMatch(/Could not authenticate you from OpenIDConnect/i);
    expect(
      isLoggedIntoGitLab(finalUrl, env.gitlabUrl),
      `期望进入 GitLab，实际: ${finalUrl}，导航: ${navigationUrls.join(' -> ')}`,
    ).toBeTruthy();
  });
});
