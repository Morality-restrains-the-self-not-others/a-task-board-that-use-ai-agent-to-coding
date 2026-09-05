// @ts-check
/**
 * 可选线上冒烟：核验登录页相关请求是否错误发往 api.daydaymoney.com。
 *
 * 默认跳过；运行：
 *   （工作目录 task2app/playwright）DAYDAYMONEY_LOGIN_VERIFY=1 npx playwright test -c playwright.config.js tests/daydaymoney-login-api-host.playwright.test.js
 *
 * 可选：`DAYDAYMONEY_LOGIN_URL=http://www.daydaymoney.com/auth/login/`
 */
import { test, expect } from '@playwright/test';

const ENABLED = process.env.DAYDAYMONEY_LOGIN_VERIFY === '1';
const LOGIN_URL = process.env.DAYDAYMONEY_LOGIN_URL || 'http://www.daydaymoney.com/auth/login/';

test.describe('daydaymoney 登录 API 域名（可选）', () => {
  test.skip(!ENABLED, '设置环境变量 DAYDAYMONEY_LOGIN_VERIFY=1 后运行');

  test('加载登录页时不应向 api.daydaymoney.com 发请求', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    const apiSubdomainUrls = [];
    page.on('request', (req) => {
      try {
        const u = new URL(req.url());
        if (u.hostname.includes('api.daydaymoney')) apiSubdomainUrls.push(req.url());
      } catch {
        /* ignore */
      }
    });

    await page.goto(LOGIN_URL, { waitUntil: 'domcontentloaded', timeout: 90000 });
    await page.waitForTimeout(1500);

    const apiBase = await page.evaluate(() => (window.config && window.config.API_BASE_URL) || '');
    expect(
      apiSubdomainUrls,
      `不应出现指向 api.daydaymoney 的请求，当前: ${apiSubdomainUrls.join('\n')}`
    ).toHaveLength(0);
    // 静态资源不经 Django spa 模板时可能仍带旧构建的 VITE_API_BASE_URL；以网络层为准，此处仅记录
    if (apiBase !== '') {
      console.warn('[daydaymoney] window.config.API_BASE_URL 非空:', apiBase);
    }

    await context.close();
  });
});
