// @ts-check
/**
 * 核验镜像市场（Saas_Ai_Provider）对外接口可被主站后端同源逻辑访问。
 *
 * 背景：主站 GET .../installed-images/dev-catalog/ 会服务端请求
 * ``{aiProvider}/api/public/vendor-development-catalog/?saas_user_id=...``；
 * 若 port_config 仍指向 localhost:8010 而 API 与 Provider 分机部署，会得到 503。
 *
 * 默认探测 monorepo ``conf/`` 聚合的 ``aiProvider.allowedHost`` 之 origin（与团队约定一致）；
 * 可覆盖：``E2E_PROVIDER_ORIGIN=http://provider.daydaymoney.com``
 *
 *   cd task2app/playwright && npx playwright test -c playwright.verify.config.js \
 *     tests/ImageMarket.provider-dev-catalog-api.playwright.test.js
 */
import { test, expect } from '@playwright/test';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

function readProviderOriginFromPortConfig() {
  const j = loadPortConfig();
  const allowed = String((j.aiProvider || {}).allowedHost || '').trim();
  if (allowed.startsWith('http://') || allowed.startsWith('https://')) {
    const u = new URL(allowed);
    return u.origin;
  }
  const host = (j.aiProvider || {}).host || '127.0.0.1';
  const port = Number((j.aiProvider || {}).port) || 8010;
  return `http://${host}:${port}`;
}

test('镜像市场 GET vendor-development-catalog 应 200 且为 JSON 数组', async ({ request }) => {
  const origin = (process.env.E2E_PROVIDER_ORIGIN || '').trim() || readProviderOriginFromPortConfig();
  const url = `${origin.replace(/\/$/, '')}/api/public/vendor-development-catalog/?saas_user_id=1`;
  const res = await request.get(url, { timeout: 30000 });
  expect(res.status(), `请求失败: ${url}`).toBe(200);
  const body = await res.json();
  expect(Array.isArray(body)).toBeTruthy();
});
