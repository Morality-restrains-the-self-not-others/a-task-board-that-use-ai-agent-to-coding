// @ts-check

/**
 * 远程 Vue + APISIX 登录 E2E 共用：剥离 X-Requested-With，收集登录后控制台 error。
 */

/**
 * @param {import('@playwright/test').Page} page
 */
export async function installApisixCorsWorkaround(page) {
  await page.route('**/*', async (route) => {
    const headers = { ...route.request().headers() };
    delete headers['x-requested-with'];
    await route.continue({ headers });
  });
}

/**
 * 登录页须先加载当前隐私/许可版本，否则 body 中 accepted_*_id 为空。
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} loginPath
 */
export async function waitForLoginLegalPolicies(page, loginPath = '/auth/login/') {
  await page.goto(loginPath);
  await page.waitForLoadState('domcontentloaded');
  await Promise.all([
    page
      .waitForResponse(
        (r) => r.url().includes('/api/privacy-policy/public/current/') && r.ok(),
        { timeout: 60000 },
      )
      .catch(() => null),
    page
      .waitForResponse(
        (r) => r.url().includes('/api/license-agreement/public/current/') && r.ok(),
        { timeout: 60000 },
      )
      .catch(() => null),
  ]);
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {{ errors: string[], markLoginDone: () => void }}
 */
export function trackConsoleErrorsAfterLogin(page) {
  const errors = [];
  let done = false;
  page.on('console', (msg) => {
    if (!done || msg.type() !== 'error') return;
    errors.push(msg.text());
  });
  page.on('pageerror', (err) => {
    if (!done) return;
    errors.push(`[pageerror] ${err.message}`);
  });
  return {
    errors,
    markLoginDone: () => {
      done = true;
    },
  };
}

/**
 * @param {string} root 无尾斜杠 base URL
 */
export function projectsUrlPattern(root) {
  const escaped = root.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return new RegExp(`${escaped}/projects/?$`);
}
