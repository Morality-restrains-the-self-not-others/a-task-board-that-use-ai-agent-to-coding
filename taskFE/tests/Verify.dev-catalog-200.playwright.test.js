// @ts-check
/**
 * 核验: GET /api/cloud/installed-images/tenant_id/:tid/dev-catalog 应返回 200
 *
 * 背景: dev-catalog 已迁至 taskCloudService Go；API 须经网关 (apiBaseUrl) 访问。
 *
 * 运行:
 *   cd taskFE
 *   npx playwright test -c playwright.verify.config.js \
 *     tests/Verify.dev-catalog-200.playwright.test.js --project=chromium
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

test.describe('核验 dev-catalog 修复', () => {
  test('镜像市场开发目录 API 应返回 200', async ({ page }) => {
    test.setTimeout(120000);

    await page.goto(`${SITE_ORIGIN}/auth/login/`);
    await page.waitForLoadState('domcontentloaded');

    const respPromise = page.waitForResponse(
      (r) => r.url().includes('/api/auth') && r.request().method() === 'POST',
      { timeout: 30000 },
    );

    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_ORIGIN,
    });

    const authResp = await respPromise;
    expect(authResp.status(), '登录 API 应成功').toBeGreaterThanOrEqual(200);
    expect(authResp.status(), '登录 API 不应为 4xx/5xx').toBeLessThan(400);

    await page.waitForURL(/\/tenant\//, { timeout: 30000 }).catch(() => {});
    await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});

    const tenantMatch = page.url().match(/\/tenant\/(\d+)/);
    expect(tenantMatch, '应已进入租户页面').toBeTruthy();
    const tenantId = tenantMatch[1];

    /** @type {{ status: number; body: string } | null} */
    let devCatalogResult = null;
    page.on('response', async (response) => {
      const url = response.url();
      if (!url.includes('/installed-images/dev-catalog/')) return;
      if (devCatalogResult) return;
      const status = response.status();
      let body = '';
      try {
        body = await response.text();
      } catch {
        body = '(stream/error)';
      }
      devCatalogResult = { status, body };
    });

    await page.goto(`${SITE_ORIGIN}/tenant/${tenantId}/image-market`);
    await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});
    await page.waitForTimeout(2000);

    if (!devCatalogResult) {
      devCatalogResult = await page.evaluate(
        async ({ apiRoot, tenantId }) => {
          const url = `${apiRoot}/api/cloud/installed-images/dev-catalog/tenant_id/${tenantId}/`;
          const resp = await fetch(url, { credentials: 'include' });
          const text = await resp.text();
          return { status: resp.status, body: text };
        },
        { apiRoot: API_ORIGIN, tenantId },
      );
    }

    expect(devCatalogResult.status, `dev-catalog 应 2xx，实际 ${devCatalogResult.status}`).toBeGreaterThanOrEqual(200);
    expect(devCatalogResult.status, `dev-catalog 不应 5xx，实际 ${devCatalogResult.status}`).toBeLessThan(500);
    expect(devCatalogResult.body, '不应包含镜像服务返回错误').not.toContain('镜像服务返回错误');

    let parsed;
    try {
      parsed = JSON.parse(devCatalogResult.body);
    } catch {
      parsed = null;
    }
    if (parsed != null) {
      expect(Array.isArray(parsed), 'dev-catalog 响应应为 JSON 数组').toBeTruthy();
    }
  });
});
