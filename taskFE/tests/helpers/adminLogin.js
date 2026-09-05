// @ts-check
/**
 * 超管 E2E 登录：直接走 API 而非 AdminLogin 页面表单。
 *
 * 背景（OPT-20260901-023）：对 /auth/admin-login/ 填 #email 后 click「管理员登录」在
 * CDP/headless Chrome 上会挂死（>100s 无第二帧），表单路径依赖原生 submit 与前端
 * legal-consent / getElementById 交互。这里改用两段式 API，通常 2s 内完成登录：
 *   1) POST /api/auth/admin-login/ { username, password } → 200 { token, user }
 *   2) 页内 fetch POST /api/accounts/users/activate-session/（Authorization: Token …,
 *      body { user_id }）→ 服务端 Set-Cookie 落 HttpOnly userId+token 会话凭据，
 *      后续 page.goto('/system-admin/…') 认证才有效（forward-auth 依赖该 cookie）。
 *
 * 禁止改走客户入口 playwrightLoginWithLegalAccept / locator('#email').fill。
 *
 * @param {import('@playwright/test').Page} page
 * @param {{ email: string; password: string; baseURL?: string }} creds
 * @returns {Promise<{ userId: string; token: string; user: unknown }>}
 */
export async function playwrightAdminLoginViaApi(page, { email, password, baseURL } = {}) {
  if (!email || !password) {
    throw new Error('playwrightAdminLoginViaApi: email/password 必须来自 env，禁止硬编码回退');
  }
  // baseURL 缺省沿用 Playwright baseURL（同源 fetch 不受 CORS 影响）。
  const root = String(baseURL ?? '').trim().replace(/\/+$/, '');

  const login = await page.evaluate(
    async ({ root, email, password }) => {
      const res = await fetch(`${root}/api/auth/admin-login/`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest',
          Accept: 'application/json',
        },
        body: JSON.stringify({ username: email, password }),
      });
      let body = {};
      try {
        body = await res.json();
      } catch {
        /* 保留 status，下面按非 200 抛错 */
      }
      return { status: res.status, body };
    },
    { root, email, password },
  );

  if (login.status !== 200 || !login.body?.token || !login.body?.user?.id) {
    throw new Error(
      `playwrightAdminLoginViaApi: admin-login 返回 ${login.status}（期望 200 + token/user.id）` +
        ` trace_id=${login.body?.trace_id || ''} body=${JSON.stringify(login.body).slice(0, 200)}`,
    );
  }

  const token = login.body.token;
  const userId = String(login.body.user.id);

  const activated = await page.evaluate(
    async ({ root, token, userId }) => {
      const res = await fetch(`${root}/api/accounts/users/activate-session/`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          Authorization: `Token ${token}`,
        },
        body: JSON.stringify({ user_id: userId }),
      });
      let body = {};
      try {
        body = await res.json();
      } catch {
        /* 保留 status */
      }
      return { status: res.status, body };
    },
    { root, token, userId },
  );

  if (activated.status !== 200) {
    throw new Error(
      `playwrightAdminLoginViaApi: activate-session 返回 ${activated.status}（期望 200）` +
        ` body=${JSON.stringify(activated.body).slice(0, 200)}`,
    );
  }

  return {
    userId,
    token,
    user: activated.body?.user ?? login.body.user,
  };
}
