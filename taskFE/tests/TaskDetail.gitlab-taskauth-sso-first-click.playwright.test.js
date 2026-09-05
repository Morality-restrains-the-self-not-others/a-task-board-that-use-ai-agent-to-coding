// @ts-check
/**
 * E2E 回归：主站已登录用户从 GitLab 点击 taskAuth SSO，第一次即完成 GitLab 登录。
 *
 * 复现路径：任务详情 → 代码仓库（或 GitLab 登录页）→ taskAuth SSO
 * 失败特征：第一次 SSO 跳回主站首页 http://*:4000/ ，需第二次才成功
 *
 * 运行示例（远程 127.0.0.1；若走 SOCKS 代理需禁用）：
 * ```bash
 * cd taskFE
 * NO_PROXY='*' HTTP_PROXY='' HTTPS_PROXY='' http_proxy='' https_proxy='' \
 *   PLAYWRIGHT_BASE_URL='http://127.0.0.1:4000' \
 *   PW_EMAIL='your@email.com' PW_PASSWORD='your-password' \
 *   npx playwright test -c playwright.verify.config.js \
 *     tests/TaskDetail.gitlab-taskauth-sso-first-click.playwright.test.js --project=chromium
 * ```
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { installApisixCorsWorkaround } from './helpers/remoteLoginE2e.js';
import {
  buildOidcAuthorizeNextUrl,
  clickGitlabTaskAuthSso,
  isLoggedIntoGitLab,
  isMainSiteHomepage,
  openGitlabSignInFromTaskDetail,
  readGitlabSsoEnv,
} from './helpers/gitlabTaskAuthSsoE2e.js';

const env = readGitlabSsoEnv();
const OIDC_NEXT = buildOidcAuthorizeNextUrl(env.gatewayUrl, env.gitlabUrl);

test.describe('GitLab taskAuth SSO 首次点击即成功 @gitlab-sso-first-click', () => {
  test.beforeEach(async ({ page }) => {
    await installApisixCorsWorkaround(page);
  });

  test('HTTP：已登录 session 访问 gateway /auth/login?next= 应续接 OIDC authorize', async ({ page }) => {
    test.skip(!env.email || !env.password, '请设置 PW_EMAIL / PW_PASSWORD（或 LOGIN_EMAIL / LOGIN_PASSWORD）');

    await playwrightLoginWithLegalAccept(page, {
      email: env.email,
      password: env.password,
      baseURL: env.siteOrigin,
    });

    const resumeRes = await page.request.get(`${env.gatewayUrl}/auth/login/?next=${encodeURIComponent(OIDC_NEXT)}`, {
      maxRedirects: 0,
    });

    expect(resumeRes.status(), '已登录用户应 302').toBe(302);
    const location = resumeRes.headers().location || '';
    const decoded = decodeURIComponent(location);
    const resumesOidcDirectly = location.includes('/api/oidc/authorize');
    const resumesViaFrontendLogin =
      location.includes('/auth/login') &&
      decoded.includes('/api/oidc/authorize');
    expect(
      resumesOidcDirectly || resumesViaFrontendLogin,
      `Location 应续接 OIDC（直连或经前端 login?next=），实际: ${location}`,
    ).toBeTruthy();
    expect(isMainSiteHomepage(location, env.siteOrigin), '不应跳回主站首页').toBe(false);
    if (resumesOidcDirectly) {
      const setCookie = resumeRes.headers()['set-cookie'] || '';
      expect(setCookie.toLowerCase(), '直连 OIDC 时应下发 userId cookie').toContain('userid=');
    }
  });

  test('浏览器：任务详情 → GitLab → taskAuth SSO 第一次即登录成功', async ({ page, context }) => {
    test.setTimeout(180000);
    test.skip(!env.email || !env.password, '请设置 PW_EMAIL / PW_PASSWORD（或 LOGIN_EMAIL / LOGIN_PASSWORD）');

    await playwrightLoginWithLegalAccept(page, {
      email: env.email,
      password: env.password,
      baseURL: env.siteOrigin,
    });

    const gitlabPage = await openGitlabSignInFromTaskDetail(page, context, env);
    expect(gitlabPage.url(), '应进入 GitLab 登录页').toMatch(/8012/);

    const navigationUrls = [];
    gitlabPage.on('framenavigated', (frame) => {
      if (frame === gitlabPage.mainFrame()) {
        navigationUrls.push(frame.url());
      }
    });

    await clickGitlabTaskAuthSso(gitlabPage);

    await gitlabPage.waitForURL(
      (url) => isLoggedIntoGitLab(url.toString()) || isMainSiteHomepage(url.toString(), env.siteOrigin),
      { timeout: 90000 },
    ).catch(async () => {
      await gitlabPage.waitForTimeout(5000);
    });

    const finalUrl = gitlabPage.url();
    const bodyText = await gitlabPage.textContent('body').catch(() => '');

    expect(isMainSiteHomepage(finalUrl, env.siteOrigin), `第一次 SSO 不应跳回主站首页，实际: ${finalUrl}`).toBe(false);
    expect(finalUrl, '第一次 SSO 不应停留在主站登录页').not.toContain('/auth/login');
    expect(bodyText).not.toMatch(/404 page not found/i);
    expect(bodyText).not.toMatch(/Could not authenticate you from OpenIDConnect/i);
    expect(isLoggedIntoGitLab(finalUrl, env.gitlabUrl), `期望首次 SSO 后进入 GitLab，实际 URL: ${finalUrl}，导航链: ${navigationUrls.join(' -> ')}`).toBeTruthy();
  });
});
