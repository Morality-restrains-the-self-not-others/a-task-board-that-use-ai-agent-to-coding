// @ts-check

/** @typedef {{ siteOrigin: string; gatewayUrl: string; gitlabUrl: string; email: string; password: string; taskDetailUrl: string; projectsUrl: string }} GitlabSsoEnv */

/**
 * @returns {GitlabSsoEnv}
 */
export function readGitlabSsoEnv() {
  const siteOrigin = (process.env.E2E_SITE_ORIGIN || process.env.PLAYWRIGHT_BASE_URL || 'http://183.250.1.132:4000').replace(/\/$/, '');
  const gatewayUrl = (process.env.GATEWAY_URL || 'http://183.250.1.132:18081').replace(/\/$/, '');
  const gitlabUrl = (process.env.GITLAB_URL || 'http://183.250.1.132:8012').replace(/\/$/, '');
  const email = process.env.PW_EMAIL || process.env.LOGIN_EMAIL || process.env.E2E_EMAIL || '';
  const password = process.env.PW_PASSWORD || process.env.LOGIN_PASSWORD || process.env.E2E_PASSWORD || '';
  const tenantId = process.env.E2E_TENANT_ID || '850256677331562496';
  const projectsUrl = process.env.E2E_PROJECTS_URL || `${siteOrigin}/tenant/${tenantId}/projects/`;
  const taskDetailUrl =
    process.env.TASK_DETAIL_URL ||
    `${siteOrigin}/tenant/${tenantId}/workspace/857903329669984256/task-detail/860865038854651904/?relayToTrae=true`;

  return { siteOrigin, gatewayUrl, gitlabUrl, email, password, taskDetailUrl, projectsUrl };
}

/**
 * 公网子域 SSOT（www / api / gitlab），用于跨子域 Cookie 回归。
 * @returns {GitlabSsoEnv}
 */
export function readGitlabSsoPublicEnv() {
  const siteOrigin = (process.env.E2E_SITE_ORIGIN || 'https://www.daydaymoney.com').replace(/\/$/, '');
  const gatewayUrl = (process.env.GATEWAY_URL || 'https://api.daydaymoney.com').replace(/\/$/, '');
  const gitlabUrl = (process.env.GITLAB_URL || 'https://gitlab.daydaymoney.com').replace(/\/$/, '');
  const email =
    process.env.PW_EMAIL ||
    process.env.LOGIN_EMAIL ||
    process.env.E2E_EMAIL ||
    process.env.PLAYWRIGHT_TEST_EMAIL ||
    '';
  const password =
    process.env.PW_PASSWORD ||
    process.env.LOGIN_PASSWORD ||
    process.env.E2E_PASSWORD ||
    process.env.PLAYWRIGHT_TEST_PASSWORD ||
    '';
  const tenantId = process.env.E2E_TENANT_ID || '850256677331562496';
  const projectsUrl = process.env.E2E_PROJECTS_URL || `${siteOrigin}/tenant/${tenantId}/projects/`;
  const taskDetailUrl =
    process.env.TASK_DETAIL_URL ||
    `${siteOrigin}/tenant/${tenantId}/workspace/857903329669984256/task-detail/860865038854651904/?relayToTrae=true`;

  return { siteOrigin, gatewayUrl, gitlabUrl, email, password, taskDetailUrl, projectsUrl };
}

/**
 * @param {string} gatewayUrl
 * @param {string} gitlabUrl
 */
export function buildOidcAuthorizeNextUrl(gatewayUrl, gitlabUrl) {
  const redirectUri = `${gitlabUrl}/users/auth/openid_connect/callback`;
  const params = new URLSearchParams({
    client_id: 'gitlab-git-service',
    response_type: 'code',
    scope: 'openid profile email',
    redirect_uri: redirectUri,
  });
  return `${gatewayUrl}/api/oidc/authorize?${params.toString()}`;
}

/**
 * @param {string} url
 * @param {string} siteOrigin
 */
export function isMainSiteHomepage(url, siteOrigin) {
  try {
    const parsed = new URL(url);
    const site = new URL(siteOrigin);
    const sameHost = parsed.hostname === site.hostname;
    const samePort = (parsed.port || (parsed.protocol === 'https:' ? '443' : '80'))
      === (site.port || (site.protocol === 'https:' ? '443' : '80'));
    return sameHost && samePort && (parsed.pathname === '/' || parsed.pathname === '');
  } catch {
    return false;
  }
}

/**
 * @param {string} url
 * @param {string} [gitlabUrl]
 */
export function isLoggedIntoGitLab(url, gitlabUrl) {
  try {
    const parsed = new URL(url);
    let isGitlabHost = false;
    if (gitlabUrl) {
      const expected = new URL(gitlabUrl);
      isGitlabHost = parsed.hostname === expected.hostname;
    } else {
      isGitlabHost =
        parsed.host.includes('8012') ||
        parsed.hostname.startsWith('gitlab.') ||
        parsed.hostname.includes('gitlab');
    }
    if (!isGitlabHost) return false;
    if (parsed.pathname.includes('/users/sign_in')) return false;
    if (parsed.pathname.includes('/users/auth/openid_connect/callback')) return false;
    return true;
  } catch {
    return false;
  }
}

/**
 * @param {import('@playwright/test').Page} gitlabPage
 */
export async function clickGitlabTaskAuthSso(gitlabPage) {
  const ssoButton = gitlabPage
    .getByRole('button', { name: /taskAuth/i })
    .or(gitlabPage.locator('input[type="submit"]').filter({ hasText: /taskAuth/i }))
    .first();
  await ssoButton.waitFor({ state: 'visible', timeout: 30000 });
  await ssoButton.click();
}

/**
 * 点击导航「代码仓库」。有区域时悬停/单击展开下拉再点第一项（真实 <a>）。
 * @param {import('@playwright/test').Page} page
 * @param {import('@playwright/test').BrowserContext} context
 * @returns {Promise<import('@playwright/test').Page|null>} 新标签，若无 popup 则为 null
 */
async function clickNavGitService(page, context) {
  const popupPromise = context.waitForEvent('page', { timeout: 20000 }).catch(() => null);
  const trigger = page.getByTestId('nav-git-service').or(page.locator('a').filter({ hasText: /代码仓库/ }).first());
  await trigger.waitFor({ state: 'visible', timeout: 30000 });
  // ≥1 区域时为 button+下拉：悬停或单击展开后再点区域项；0 区域为当前页 <a>
  await trigger.hover().catch(() => {});
  await trigger.click();
  const regionItem = page.getByTestId('nav-git-service-region').first();
  if (await regionItem.isVisible({ timeout: 2000 }).catch(() => false)) {
    await regionItem.click();
  }
  return popupPromise;
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {import('@playwright/test').BrowserContext} context
 * @param {GitlabSsoEnv} env
 */
export async function openGitlabSignInFromTaskDetail(page, context, env) {
  await page.goto(env.taskDetailUrl, { waitUntil: 'domcontentloaded', timeout: 60000 });
  if (page.url().includes('/auth/login')) {
    throw new Error('访问任务详情时被重定向到登录页，请先完成主站登录');
  }

  let gitlabPage = await clickNavGitService(page, context);
  if (!gitlabPage) {
    gitlabPage = page;
    await gitlabPage.goto(`${env.gitlabUrl}/users/sign_in`, { waitUntil: 'domcontentloaded' });
  } else {
    await gitlabPage.waitForLoadState('domcontentloaded');
  }

  return gitlabPage;
}

/**
 * 从项目列表页打开「代码仓库」→ GitLab（公网子域回归入口）。
 * @param {import('@playwright/test').Page} page
 * @param {import('@playwright/test').BrowserContext} context
 * @param {GitlabSsoEnv} env
 */
export async function openGitlabSignInFromProjects(page, context, env) {
  await page.goto(env.projectsUrl, { waitUntil: 'domcontentloaded', timeout: 90000 });
  if (page.url().includes('/auth/login')) {
    throw new Error('访问项目列表时被重定向到登录页，请先完成主站登录');
  }

  let gitlabPage = await clickNavGitService(page, context);
  if (!gitlabPage) {
    gitlabPage = page;
    await gitlabPage.waitForURL((url) => url.toString().includes('gitlab'), { timeout: 30000 }).catch(() => {});
  } else {
    await gitlabPage.waitForLoadState('domcontentloaded');
  }

  if (!String(gitlabPage.url()).includes('gitlab') && !String(gitlabPage.url()).includes('8012')) {
    await gitlabPage.goto(`${env.gitlabUrl}/users/sign_in`, { waitUntil: 'domcontentloaded' });
  }

  return gitlabPage;
}
