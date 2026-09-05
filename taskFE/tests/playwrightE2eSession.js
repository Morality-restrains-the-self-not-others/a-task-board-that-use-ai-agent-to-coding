// @ts-check
/** Playwright mock 会话：双 origin Cookie + localStorage currentUserId。 */

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ userId?: string, session?: string }} [opts]
 */
export async function addE2eSession(page, opts = {}) {
  const userId = opts.userId || 'e2e-user';
  const session = opts.session || 'e2e-session';
  await page.addInitScript((id) => {
    try {
      localStorage.setItem('currentUserId', id);
    } catch {
      /* ignore */
    }
  }, userId);
  const origins = ['http://127.0.0.1:4000/', 'http://localhost:4000/'];
  /** @type {import('@playwright/test').Cookie[]} */
  const cookies = [];
  for (const url of origins) {
    cookies.push(
      { name: 'userId', value: userId, url },
      { name: 'sessionid', value: session, url },
      { name: 'csrftoken', value: 'e2e-csrf', url },
    );
  }
  await page.context().addCookies(cookies);
}

/**
 * @param {string} tenantId
 */
export function mockMeBody(tenantId) {
  return {
    id: 'e2e-user',
    username: 'e2e',
    is_superuser: true,
    phone_verified: true,
    companies: [{ id: tenantId, name: 'E2E Co' }],
    current_company: { id: tenantId, name: 'E2E Co' },
  };
}

/**
 * @param {string} tenantId
 * @param {string[]} extra
 */
export function mockPermsBody(tenantId, extra = []) {
  return {
    tenant_perms: {
      [tenantId]: [
        'member:manage',
        'feedback:view',
        'company:manage',
        'company:view',
        'task:view',
        'task:manage',
        'project:view',
        'project:manage',
        'cloud:view',
        'cloud:manage',
        'workspace:manage',
        'billing:view',
        ...extra,
      ],
    },
  };
}

/**
 * Common tenant-page APIs so Navbar/Sidebar do not bounce to onboarding.
 * @param {string} url
 * @param {string} method
 * @param {string} tenantId
 * @returns {object|null} JSON body or null if not handled
 */
export function matchTenantShellApi(url, method, tenantId) {
  if (method !== 'GET') return null;
  if (url.includes('/api/accounts/users/me/')) return mockMeBody(tenantId);
  if (url.includes('/api/accounts/users/profile/')) return { user_id: 'e2e-user', id: 'e2e-user' };
  if (url.includes('/api/auth/user-permissions/')) return mockPermsBody(tenantId);
  if (url.includes('/api/auth/user-roles/')) {
    return { roles: [{ role: 'tenant_admin', company_id: tenantId, level: 1 }] };
  }
  if (url.includes(`/api/tenant/${tenantId}/accounts/companies/current/`)) {
    return { member_is_active: true, member_is_admin: true, member_is_creator: true };
  }
  return null;
}
