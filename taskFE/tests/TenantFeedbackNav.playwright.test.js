// @ts-check
/**
 * OPT-20260830-002：租户侧栏「意见与建议」toggle + 真实 target=_blank 外链。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';
import { addE2eSession, matchTenantShellApi } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const FEEDBACK_URL = 'https://example.com/feedback-group-a';
const PATH = `/tenant/${TENANT_ID}/people/invite/`;

test.describe('租户侧栏意见与建议（OPT-20260830-002）', () => {
  test('tenant-feedback-toggle 展开后含真实 a[target=_blank]', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-feedback-nav' });

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();
      const shell = matchTenantShellApi(url, method, TENANT_ID);
      if (shell) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(shell) });
        return;
      }
      if (url.includes(`/api/tenant/${TENANT_ID}/billing/feedback-links/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            groups: [{
              id: 'g1',
              name: '产品反馈',
              links: [{ id: 'l1', title: '提交建议', url: FEEDBACK_URL }],
            }],
          }),
        });
        return;
      }
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: [{ id: 'ws1', name: 'WS1' }] }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(PATH);
    await page.waitForLoadState('domcontentloaded');
    const toggle = page.getByTestId('tenant-feedback-toggle');
    await expect(toggle).toBeVisible({ timeout: 20000 });
    await toggle.click();
    const link = page.getByTestId('tenant-feedback-link');
    await expect(link).toBeVisible({ timeout: 10000 });
    await expect(link).toHaveAttribute('target', '_blank');
    await expect(link).toHaveAttribute('href', FEEDBACK_URL);
  });
});
