// @ts-check
/** 工作面板打开「创建任务」时，相关 /api 请求不应 404（需 PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD） */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const ACCESS_CODE = 'u824976301710503936';

test('work-panel create-task modal: no API 404', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  /** @type {string[]} */
  const api404 = [];

  page.on('response', (response) => {
    const url = response.url();
    if (!url.includes('/api/')) return;
    const status = response.status();
    if (status !== 404) return;
    if (
      url.includes('/favicon') ||
      url.includes('.well-known') ||
      url.includes('/manifest')
    ) {
      return;
    }
    api404.push(`${status} ${url}`);
  });

  await playwrightLoginWithLegalAccept(page, { email, password });
  const q = new URLSearchParams({ accessCode: ACCESS_CODE });
  await page.goto(`/tenant/${TENANT_ID}/work-panel/?${q.toString()}`);
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2500);

  await page.locator('#create-task-btn').click();
  await page.locator('#create-task-modal').waitFor({ state: 'visible', timeout: 15000 });
  await page.waitForTimeout(4000);

  expect(api404, `Unexpected 404:\n${api404.join('\n')}`).toEqual([]);
});

test('work-panel create-task: upstream Git errors return 200 + error body (not HTTP 400)', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  /** @type {{ status: number; url: string; bodyError?: string }[]} */
  const branchResponses = [];

  page.on('response', async (response) => {
    const url = response.url();
    if (!/\/projects\/[^/]+\/branches\/?(\?|$)/.test(url)) return;
    /** @type {{ error?: string }} */
    let parsed = {};
    try {
      parsed = await response.json();
    } catch {
      parsed = {};
    }
    branchResponses.push({
      status: response.status(),
      url,
      bodyError: typeof parsed.error === 'string' ? parsed.error : undefined,
    });
  });

  await playwrightLoginWithLegalAccept(page, { email, password });
  const q = new URLSearchParams({ accessCode: ACCESS_CODE });
  await page.goto(`/tenant/${TENANT_ID}/work-panel/?${q.toString()}`);
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2500);

  await page.locator('#create-task-btn').click();
  await page.locator('#create-task-modal').waitFor({ state: 'visible', timeout: 15000 });
  await page.waitForTimeout(6000);

  for (const r of branchResponses) {
    const isUpstreamStyle =
      r.bodyError &&
      (r.bodyError.includes('GitHub') ||
        r.bodyError.includes('gitlab') ||
        r.bodyError.includes('Bitbucket') ||
        r.bodyError.includes('分支'));
    if (!isUpstreamStyle) continue;
    expect(
      r.status,
      `上游 Git 不可用时应返回 200 并在 JSON 中附带 error，而非 HTTP 400：${r.url} bodyError=${r.bodyError}`
    ).toBe(200);
  }
});

test('work-panel create-task: branches API uses repo_url query (not repo_index)', async ({ page }) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD for this test');

  /** @type {import('@playwright/test').Request[]} */
  const branchRequests = [];

  page.on('request', (req) => {
    if (req.method() !== 'GET') return;
    const url = req.url();
    if (!url.includes('/api/')) return;
    if (!/\/projects\/[^/]+\/branches\/?(\?|$)/.test(url)) return;
    branchRequests.push(req);
  });

  await playwrightLoginWithLegalAccept(page, { email, password });
  const q = new URLSearchParams({ accessCode: ACCESS_CODE });
  await page.goto(`/tenant/${TENANT_ID}/work-panel/?${q.toString()}`);
  await page.waitForLoadState('networkidle', { timeout: 45000 }).catch(() => {});
  await page.waitForTimeout(2500);

  await page.locator('#create-task-btn').click();
  await page.locator('#create-task-modal').waitFor({ state: 'visible', timeout: 15000 });
  await page.waitForTimeout(6000);

  for (const req of branchRequests) {
    const u = new URL(req.url());
    expect(u.searchParams.has('repo_url'), `Expected repo_url in ${req.url()}`).toBeTruthy();
    expect(
      u.searchParams.has('repo_index'),
      `Should not rely on repo_index in ${req.url()}`
    ).toBeFalsy();
    const repoUrl = u.searchParams.get('repo_url') || '';
    expect(repoUrl.length, 'repo_url should be non-empty').toBeGreaterThan(0);
  }
});
