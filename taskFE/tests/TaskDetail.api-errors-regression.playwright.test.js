// @ts-check
/**
 * 任务详情页：核心 cloud/compute 读接口不应再出现 404/501 路由缺失。
 * 环境相关失败（容器未启动、阿里云代理不可用）不计入断言。
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '857903329669984256';
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || '860371538948571136';

const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`;

/** 路由/实现缺失类错误（须为 0） */
const ROUTING_FAILURE_SNIPPETS = [
  'workspace cloud route not found',
  'compute action not yet ported to taskCloudService',
  '"path":"cloud"',
];

/** 可接受的环境/基础设施类错误（容器未起、阿里云代理、server-content 拉取失败等） */
const ENV_FAILURE_PATTERNS = [
  /proxyconnect tcp/i,
  /拉取服务器内容失败/i,
  /connection refused/i,
  /connect: connection refused/i,
];

test.describe('TaskDetail API routing regression', () => {
  test('task detail page: no cloud route 404/501 on core compute reads', async ({ page }) => {
    test.setTimeout(120_000);

    const routingFailures = [];

    page.on('response', async (resp) => {
      const status = resp.status();
      if (status < 400) return;
      const url = resp.url();
      if (!url.includes('/api/') && !url.includes(GATEWAY)) return;
      let body = '';
      try {
        body = await resp.text();
      } catch {
        /* ignore */
      }
      const isRouting = ROUTING_FAILURE_SNIPPETS.some((s) => body.includes(s));
      const isEnv = ENV_FAILURE_PATTERNS.some((re) => re.test(body));
      if (isRouting && !isEnv) {
        routingFailures.push({ status, url, body: body.slice(0, 300) });
      }
    });

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    await page.goto(`${SITE}/`, { waitUntil: 'domcontentloaded' });
    await page.evaluate((token) => {
      localStorage.setItem('authToken', token);
    }, loginData.token);
    if (loginData.user?.id) {
      await page.context().addCookies([
        { name: 'userId', value: String(loginData.user.id), url: `${SITE}/` },
      ]);
    }

    await page.goto(`${SITE}${TASK_DETAIL_PATH}`, { waitUntil: 'domcontentloaded', timeout: 60_000 });
    expect(page.url()).toContain('/task-detail/');
    await page.waitForTimeout(12_000);

    expect(routingFailures, JSON.stringify(routingFailures, null, 2)).toEqual([]);
  });
});
