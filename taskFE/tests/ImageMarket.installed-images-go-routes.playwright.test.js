// @ts-check
/**
 * 核验 installed-images 租户 API 经网关路由至 taskCloudService Go。
 *
 * - GET  list / catalog / dev-catalog 返回 JSON 数组（非 {installed_images:[]} 包装）
 * - POST resolve-target-architectures 缺 image_url 时返回 400
 *
 * 运行:
 *   cd taskFE
 *   npx playwright test -c playwright.verify.config.js \
 *     tests/ImageMarket.installed-images-go-routes.playwright.test.js --project=chromium
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import {
  playwrightApiOrigin,
  playwrightSiteOrigin,
  playwrightTenantCredentials,
} from './playwrightApiEnv.js';

const SITE_ORIGIN = playwrightSiteOrigin();
const API_ORIGIN = playwrightApiOrigin();
const { email: EMAIL, password: PASSWORD } = playwrightTenantCredentials();

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} apiRoot
 * @param {string} tenantId
 * @param {string} pathSuffix
 */
async function fetchInstalledImagesApi(page, apiRoot, tenantId, pathSuffix) {
  const url = `${apiRoot.replace(/\/$/, '')}/api/cloud/installed-images/tenant_id/${tenantId}${pathSuffix}`;
  return page.evaluate(
    async ({ fetchUrl }) => {
      const resp = await fetch(fetchUrl, { credentials: 'include' });
      const text = await resp.text();
      let body;
      try {
        body = JSON.parse(text);
      } catch {
        body = text;
      }
      return { status: resp.status, body };
    },
    { fetchUrl: url }
  );
}

test.describe('镜像市场 installed-images Go 路由', () => {
  test('list/catalog/dev-catalog 应返回 JSON 数组', async ({ page }) => {
    test.setTimeout(120000);

    await page.goto(`${SITE_ORIGIN}/auth/login/`);
    await page.waitForLoadState('domcontentloaded');
    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_ORIGIN,
    });
    await page.waitForURL(/\/tenant\//, { timeout: 30000 }).catch(() => {});
    await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});

    const tenantMatch = page.url().match(/\/tenant\/(\d+)/);
    expect(tenantMatch, '登录后应进入租户上下文').toBeTruthy();
    const tenantId = tenantMatch[1];

    for (const suffix of ['', 'catalog/', 'dev-catalog/']) {
      const result = await fetchInstalledImagesApi(page, API_ORIGIN, tenantId, suffix);
      expect(result.status, `${suffix || 'list'} 应 2xx`).toBeGreaterThanOrEqual(200);
      expect(result.status, `${suffix || 'list'} 不应 5xx`).toBeLessThan(500);
      expect(Array.isArray(result.body), `${suffix || 'list'} 响应应为 JSON 数组`).toBeTruthy();
      expect(result.body, `${suffix || 'list'} 不应为旧 stub 包装`).not.toHaveProperty('installed_images');
    }
  });

  test('resolve-target-architectures 缺 image_url 应 400', async ({ page }) => {
    test.setTimeout(120000);

    await page.goto(`${SITE_ORIGIN}/auth/login/`);
    await page.waitForLoadState('domcontentloaded');
    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_ORIGIN,
    });
    await page.waitForURL(/\/tenant\//, { timeout: 30000 }).catch(() => {});

    const tenantMatch = page.url().match(/\/tenant\/(\d+)/);
    expect(tenantMatch).toBeTruthy();
    const tenantId = tenantMatch[1];

    const result = await page.evaluate(
      async ({ apiRoot, tenantId }) => {
        const url = `${apiRoot}/api/cloud/installed-images/resolve-target-architectures/tenant_id/${tenantId}/`;
        const resp = await fetch(url, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({}),
        });
        const text = await resp.text();
        let body;
        try {
          body = JSON.parse(text);
        } catch {
          body = text;
        }
        return { status: resp.status, body };
      },
      { apiRoot: API_ORIGIN, tenantId }
    );

    expect(result.status, '缺 image_url 应 400').toBe(400);
    const detail = typeof result.body === 'object' && result.body ? result.body.detail : '';
    expect(String(detail)).toMatch(/image_url/i);
  });
});
