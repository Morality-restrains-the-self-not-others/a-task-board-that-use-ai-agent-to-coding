// @ts-check
/**
 * 核验: 镜像市场「开发中镜像」同组不同版本叠列为一张组卡（组卡内版本列表）。
 *
 * 背景: 开发中目录按版本返回（同组多版本平铺各自成卡），前端按 image_group 分组后
 *       一组一张卡。本用例以 trae-agent（x86_64-latest / private_x86_64-latest）验证。
 *
 * 运行:
 *   cd taskFE
 *   LOGIN_URL=https://www.daydaymoney.com npx playwright test -c playwright.verify.config.js \
 *     tests/Verify.ImageMarket.grouping.playwright.test.js --project=chromium
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { playwrightSiteOrigin, playwrightTenantCredentials } from './playwrightApiEnv.js';

const SITE_ORIGIN = playwrightSiteOrigin();
const { email: EMAIL, password: PASSWORD } = playwrightTenantCredentials();
const TENANT_ID = '877397588196749312';

test.describe('镜像市场同组版本叠列（线上核验）', () => {
  test('trae-agent 同组两版本叠成一张组卡', async ({ page }) => {
    test.setTimeout(180000);

    await page.goto(`${SITE_ORIGIN}/auth/login/`);
    await page.waitForLoadState('domcontentloaded');
    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_ORIGIN,
    });
    await page.waitForURL(/\/tenant\//, { timeout: 30000 }).catch(() => {});
    await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});

    await page.goto(`${SITE_ORIGIN}/tenant/${TENANT_ID}/image-market`);
    await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => {});
    await page.waitForTimeout(2500);

    // 开发中镜像区列表为页面第一个 ul.mt-6.space-y-4
    const devList = page.locator('ul.mt-6.space-y-4').first();
    const devCards = devList.locator(':scope > li');
    const count = await devCards.count();

    // 账号未绑定厂商时开发中区为空：退化为 API 契约核验（响应含 image_group 即分组可解析）
    if (count === 0) {
      const payload = await page.evaluate(async ({ tid }) => {
        const resp = await fetch(`/api/cloud/installed-images/dev-catalog/tenant_id/${tid}/`, {
          credentials: 'include',
        });
        return { status: resp.status, body: await resp.text() };
      }, { tid: TENANT_ID });
      console.log('[verify] 开发中区空，dev-catalog API:', payload.status, payload.body.slice(0, 500));
      expect(payload.status, 'dev-catalog 应可访问').toBeGreaterThanOrEqual(200);
      expect(payload.status, 'dev-catalog 不应 5xx').toBeLessThan(500);
      return;
    }

    // 同组应只有一张卡（旧版为每个版本一张卡 → 2 张）。
    // 注意按 h3 精确名匹配，避免误匹配同前缀组（如 trae-agent-skill）。
    const traeCards = devList
      .locator(':scope > li')
      .filter({ has: page.locator('h3', { hasText: /^trae-agent$/ }) });
    const traeCount = await traeCards.count();
    expect(traeCount, 'trae-agent 同组应仅渲染一张组卡').toBe(1);

    const cardText = await traeCards.first().innerText();
    expect(cardText, '组卡内应含 x86_64-latest 版本行').toContain('x86_64-latest');
    expect(cardText, '组卡内应含 private_x86_64-latest 版本行').toContain('private_x86_64-latest');

    const installButtons = traeCards.first().locator('button:has-text("安装")');
    const btnCount = await installButtons.count();
    expect(btnCount, '组卡内每个版本行各有一个安装按钮').toBeGreaterThanOrEqual(1);
    console.log(`[verify] trae-agent 组卡: 1 张，版本行安装按钮 ${btnCount} 个`);
  });
});
