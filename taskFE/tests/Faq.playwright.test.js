// @ts-check
/**
 * E2E: 公网 /faq/ 无侧栏、无知识产权文案、联系邮箱已替换（OPT-20260830-007）
 *
 * Vitest 已覆盖 Faq.vue 组件。本用例打 hashed 入口 JS、SPA fallback 与
 * nginx try_files（静态 /faq/ 目录不得 403）。
 *
 * 运行：
 *   npx playwright test --config=playwright.verify.config.js tests/Faq.playwright.test.js
 */
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PW_BASE_URL ||
  process.env.BASE_URL ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');

const CONF_CONTACT = readFileSync(
  path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../conf/frontend/vue/config.yaml'),
  'utf8',
);
const CONTACT_EMAIL = (CONF_CONTACT.match(/^contactEmail:\s*(\S+)/m) || [])[1] || '';

test.describe('FAQ 公网页 OPT-20260830-007', () => {
  test('GET /faq/ 为 200，无侧栏/知识产权/账号文案，邮箱已替换，返回首页', async ({ page }) => {
    expect(CONTACT_EMAIL, 'conf/frontend/vue/config.yaml contactEmail').toMatch(/@/);

    const response = await page.goto(`${BASE_URL}/faq/`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    expect(response, '应拿到 /faq/ 导航响应').toBeTruthy();
    expect(response.status(), '/faq/ 不得 403（静态目录命中）或非 200').toBe(200);

    await expect(page.getByTestId('view-faq-page')).toBeVisible({ timeout: 30000 });
    await expect(page.getByTestId('faq-doc-nav')).toHaveCount(0);
    await expect(page.locator('aside')).toHaveCount(0);

    await expect(page.getByTestId('faq-doc-panel-usage')).toBeVisible();
    await expect(page.getByTestId('faq-doc-panel-billing')).toBeVisible();
    await expect(page.getByTestId('faq-doc-panel-account')).toHaveCount(0);

    const bodyText = await page.getByTestId('faq-doc-content').innerText();
    expect(bodyText).not.toContain('我们便不会主动提起诉讼');
    expect(bodyText).not.toContain('账号与登录');
    expect(bodyText).not.toContain('{{contactEmail}}');
    expect(bodyText).toContain(CONTACT_EMAIL);

    await page.getByTestId('faq-back-home-link').click();
    await expect(page.getByTestId('view-faq-page')).toHaveCount(0, { timeout: 15000 });
    expect(new URL(page.url()).pathname).toBe('/');
  });
});
