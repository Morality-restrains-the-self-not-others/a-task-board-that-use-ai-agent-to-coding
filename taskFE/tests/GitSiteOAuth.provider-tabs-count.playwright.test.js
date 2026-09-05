// @ts-check
/**
 * 校验 git-site-oauth 设置页展示的 OAuth 站点数量与 monorepo conf/（YAML） 的 gitOauth 条目一致。
 *
 * 需：前端 Vite + Django 已启动；PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const readGitOauthCatalogFromPortConfig = () => {
  try {
    const raw = loadPortConfig();
    const gitOauth = raw?.gitOauth;
    if (!gitOauth || typeof gitOauth !== 'object' || Array.isArray(gitOauth)) {
      return [];
    }
    return Object.values(gitOauth)
      .filter((item) => item && typeof item === 'object')
      .map((item) => ({
        provider: String(item.provider || '').trim().toLowerCase(),
        service_provider: String(item.service_provider || 'default').trim().toLowerCase(),
        website: String(item.target?.website || item.website || '').trim(),
      }))
      .filter((item) => item.provider);
  } catch {
    return [];
  }
};

const expectedLabel = (entry) => {
  if (entry.provider === 'github') return 'GitHub';
  try {
    const host = new URL(entry.website).host;
    return `GitLab (${host})`;
  } catch {
    return `GitLab (${entry.service_provider})`;
  }
};

test.describe('Git 网站 OAuth 多站点切换', () => {
  test('设置页应展示 port_config 中全部 gitOauth service_provider', async ({ page }) => {
    const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
    const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
    test.skip(!email || !password, '设置 PLAYWRIGHT_TEST_EMAIL 与 PLAYWRIGHT_TEST_PASSWORD');

    const expected = readGitOauthCatalogFromPortConfig();
    test.skip(expected.length < 2, 'conf/ YAML 中 gitOauth 条目不足，跳过');

    await playwrightLoginWithLegalAccept(page, { email, password });
    const uid = await page.evaluate(() => {
      const m = document.cookie.match(/(?:^|;\s*)userId=([^;]+)/);
      return m ? decodeURIComponent(m[1]) : '';
    });
    test.skip(!uid, '登录后未获得 userId cookie');

    await page.goto(`/user/${uid}/profile/git-site-oauth/`, { waitUntil: 'networkidle' });

    const view = page.locator('[data-alias="view-user-git-site-oauth"]');
    await expect(view).toBeVisible();

    for (const entry of expected) {
      await expect(view.getByRole('button', { name: expectedLabel(entry), exact: true })).toBeVisible();
    }

    const providerButtons = view.locator('main button').filter({
      hasText: /^(GitHub|GitLab \(.+\))$/,
    });
    await expect(providerButtons).toHaveCount(expected.length);
  });
});
