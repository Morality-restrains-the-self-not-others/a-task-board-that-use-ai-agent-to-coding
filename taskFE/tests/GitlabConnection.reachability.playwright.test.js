// @ts-check
/**
 * OPT-20260829-023：自建 GitLab 可达性 down + 勾选内网 skipped_intranet。
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';
import { addE2eSession, matchTenantShellApi } from './playwrightE2eSession.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const PATH = `/tenant/${TENANT_ID}/settings/gitlab-connection/`;

test.describe('GitLab 连接可达性（OPT-20260829-023）', () => {
  test('未勾选内网显示 gitlab-reachability-down；勾选后 data-status=skipped_intranet', async ({ page }) => {
    test.setTimeout(120000);
    await addE2eSession(page, { session: 'e2e-session-gitlab-reach' });

    await page.route('**/api/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();
      const shell = matchTenantShellApi(url, method, TENANT_ID);
      if (shell) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(shell) });
        return;
      }
      if (url.includes('/reachability/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'unreachable' }),
        });
        return;
      }
      if (url.includes('/api/git-oauth/tenant-connection/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            configured: true,
            base_url: 'https://gitlab.daydaymoney.com',
            client_id: 'app-id',
            remark: 'mock',
            intranet: false,
            redirect_uri: 'https://app.example/callback',
            provider_key: 'gitlab:example',
          }),
        });
        return;
      }
      if (url.includes('/api/billing/gitlab-regions/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ regions: [] }),
        });
        return;
      }
      if (url.includes('/api/billing/gitlab-resources/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            provisioning_status: 'not_purchased',
            available_regions: [],
          }),
        });
        return;
      }
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    await page.goto(PATH);
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByTestId('gitlab-reachability-down')).toBeVisible({ timeout: 20000 });
    await expect(page.getByTestId('gitlab-self-hosted-reachability')).toHaveAttribute('data-status', 'unreachable');

    await page.getByTestId('gitlab-intranet-checkbox').check();
    await expect(page.getByTestId('gitlab-self-hosted-reachability')).toHaveAttribute('data-status', 'skipped_intranet');
    await expect(page.getByTestId('gitlab-reachability-intranet')).toBeVisible();
  });
});
