// @ts-check
/** Shared mocks for billing invoice + grant-points Playwright (OPT-20260823-055/056/065). */

export const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496'
export const ADMIN_USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488'
export const ORDER_ID = process.env.PLAYWRIGHT_BILLING_ORDER_ID || '901000000000000001'
export const INVOICE_APP_ID = 'inv-app-e2e-1'
export const TENANT_NAME = 'E2E Tenant'

export function paidOrderBase() {
  return {
    id: ORDER_ID,
    order_number: 'ORD-E2E-1',
    status: 'paid',
    total_yuan: '10.00',
    items: [{
      id: 'it-1',
      resource_type: 'task_post',
      quantity: 1,
      unit_price_yuan: '10.00',
      subtotal_yuan: '10.00',
      region: '',
    }],
    invoices: [],
    invoice_application: null,
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} origin
 * @param {(route: import('@playwright/test').Route, request: import('@playwright/test').Request) => Promise<boolean>} onApi
 */
export async function installBillingAdminMocks(page, origin, onApi) {
  await page.context().addCookies([
    { name: 'userId', value: ADMIN_USER_ID, url: origin },
    { name: 'sessionid', value: 'e2e-session-billing', url: origin },
    { name: 'csrftoken', value: 'e2e-csrf', url: origin },
  ])

  await page.route(/\/api\/.*/, async (route) => {
    const request = route.request()
    if (await onApi(route, request)) return

    const url = request.url()
    if (url.includes('/api/accounts/users/me/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: ADMIN_USER_ID,
          username: 'e2e-admin',
          is_superuser: true,
          has_phone: true,
          current_company: { id: TENANT_ID, name: TENANT_NAME },
          companies: [{ id: TENANT_ID, name: TENANT_NAME }],
          current_workspace: { id: 'ws-e2e', name: '默认工作空间' },
        }),
      })
      return
    }
    if (url.includes('/api/auth/user-roles/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          roles: [{ role: 'super_admin', level: 1, company_id: null }],
        }),
      })
      return
    }
    if (url.includes('/api/auth/user-permissions/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ tenant_perms: {} }),
      })
      return
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
  })
}

/** @param {import('@playwright/test').Route} route */
export async function jsonOk(route, body) {
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}
