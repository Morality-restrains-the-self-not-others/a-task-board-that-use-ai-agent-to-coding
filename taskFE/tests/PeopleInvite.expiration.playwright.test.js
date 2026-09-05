// @ts-check
/**
 * OPT-20260829-026：邀请页「复制邀请链接」有效期下拉含 90/180/365，默认 90。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';
import { addE2eSession, matchTenantShellApi } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const PATH = `/tenant/${TENANT_ID}/people/invite/`;

test.describe('PeopleInvite 有效期下拉（OPT-20260829-026）', () => {
  test('复制邀请链接：invite-expiration-days 含 90/180/365 且默认 90', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-invite-exp' });

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();
      const shell = matchTenantShellApi(url, method, TENANT_ID);
      if (shell) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(shell) });
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
      if (url.includes('/api/auth/roles/company_id/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(PATH);
    await page.waitForLoadState('domcontentloaded');

    await page.locator('input[type="radio"][value="link"]').check();
    const select = page.getByTestId('invite-expiration-days');
    await expect(select).toBeVisible({ timeout: 15000 });
    await expect(select).toHaveValue('90');
    const values = await select.locator('option').evaluateAll((opts) => opts.map((o) => o.getAttribute('value')));
    expect(values).toEqual(expect.arrayContaining(['90', '180', '365']));
  });
});
