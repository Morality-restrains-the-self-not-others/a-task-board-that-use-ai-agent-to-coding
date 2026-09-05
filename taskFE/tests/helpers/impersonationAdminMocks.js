// @ts-check
/** Shared mock fixtures for system-admin impersonation Playwright tests. */

export const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496'
export const ADMIN_USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488'
export const TARGET_USER_ID = process.env.PLAYWRIGHT_TARGET_USER_ID || '837920218451263488'
export const OTHER_USER_ID = process.env.PLAYWRIGHT_OTHER_USER_ID || '837920218451263499'
export const TARGET_USER_EMAIL = 'e2e-target@example.com'
export const OTHER_USER_EMAIL = 'e2e-other@example.com'
export const IMPERSONATE_REASON = '排查线上工单 T-playwright'
export const NESTED_IMPERSONATION_DETAIL = '已在模拟登录中，请先退出'

export function listUser(id, email, username) {
  return {
    id,
    username,
    email,
    phone: '',
    is_active: true,
    is_superuser: false,
    is_staff: false,
    is_tenant: false,
    is_tester: false,
    is_archived: false,
    date_joined: '2026-07-01T00:00:00Z',
    last_login: '2026-08-01T00:00:00Z',
    login_methods: [{ method_type: 'email', is_verified: true }],
    referrer_code: 'E2E001',
    tenant_companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} origin
 * @param {{
 *   users?: object[],
 *   onImpersonate?: (route: import('@playwright/test').Route, request: import('@playwright/test').Request) => Promise<void>,
 *   inboxResults?: object[],
 * }} [opts]
 */
export async function installImpersonationAdminMocks(page, origin, opts = {}) {
  const users = opts.users || [listUser(TARGET_USER_ID, TARGET_USER_EMAIL, 'e2e-target')]
  const inboxResults = opts.inboxResults || []

  await page.context().addCookies([
    { name: 'userId', value: ADMIN_USER_ID, url: origin },
    { name: 'sessionid', value: 'e2e-session-impersonate', url: origin },
    { name: 'csrftoken', value: 'e2e-csrf', url: origin },
  ])

  await page.route(/\/api\/.*/, async (route) => {
    const request = route.request()
    const url = request.url()
    const method = request.method()

    if (url.includes('/impersonate/') && method === 'POST') {
      if (opts.onImpersonate) {
        await opts.onImpersonate(route, request)
        return
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'imp_e2e_token',
          user: { id: TARGET_USER_ID, username: 'e2e-target' },
          redirect_url: '/onboarding/',
        }),
      })
      return
    }

    if (url.includes('/api/accounts/users/activate-session/') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          token: 'imp_e2e_token',
          user: { id: TARGET_USER_ID, username: 'e2e-target' },
        }),
      })
      return
    }

    if (url.includes('/api/auth/inbox/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ results: inboxResults }),
      })
      return
    }

    if (url.includes('/api/accounts/users/me/')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: ADMIN_USER_ID,
          username: 'e2e-admin',
          is_superuser: true,
          has_phone: true,
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
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

    if (url.includes('/api/system-admin/users/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ users, total: users.length }),
      })
      return
    }

    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
  })
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} email
 */
export async function openImpersonateModal(page, email) {
  const row = page.locator('tr', { hasText: email })
  await row.waitFor({ state: 'visible', timeout: 45000 })
  await row.getByTestId('user-row-impersonate-btn').click()
  await page.getByTestId('impersonate-reason-modal').waitFor({ state: 'visible', timeout: 15000 })
}
