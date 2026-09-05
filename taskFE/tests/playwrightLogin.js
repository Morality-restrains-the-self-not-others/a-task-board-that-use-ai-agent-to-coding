// @ts-check
/**
 * 登录页须勾选隐私与许可协议后「登录」才可点；与旧用例仅填邮箱密码不同。
 *
 * @param {import('@playwright/test').Page} page
 * @param {{ email: string; password: string; baseURL?: string; skipGoto?: boolean }} creds
 * @param {string} [creds.baseURL] 若设置（如 `http://localhost:4000`），则打开 `${baseURL}/auth/login/`；否则沿用 Playwright 配置的相对路径 `/auth/login/`（CDP 外接 Chrome 时常需传入）。
 * @param {boolean} [creds.skipGoto] 为 true 时假定已在登录页（由 waitForLoginLegalPolicies 打开）。
 */
export async function playwrightLoginWithLegalAccept(page, { email, password, baseURL, skipGoto = false }) {
  if (!email || !password) {
    throw new Error('playwrightLoginWithLegalAccept: email/password must come from env (no hardcoded fallback)');
  }
  const root = baseURL != null && String(baseURL).trim() !== '' ? String(baseURL).replace(/\/$/, '') : '';
  if (!skipGoto) {
    await page.goto(root ? `${root}/auth/login/` : '/auth/login/');
    await page.waitForLoadState('domcontentloaded');
  }

  const emailPwdTab = page.getByRole('button', { name: /邮箱\/密码|邮箱.*密码/ }).first();
  if (await emailPwdTab.isVisible().catch(() => false)) {
    await emailPwdTab.click();
    await page.waitForTimeout(300);
  }

  // 只勾 input[type=checkbox]（testid）。禁止点「《隐私政策》」按钮，那会打开条款弹层并把登录卡住。
  const checkboxCandidates = [
    page.getByTestId('login-accept-all'),
    page.getByTestId('login-privacy-accept'),
    page.getByTestId('login-license-accept'),
    page.getByTestId('login-terms-accept'),
  ];
  for (const checkbox of checkboxCandidates) {
    const visible = await checkbox.isVisible().catch(() => false);
    if (!visible) continue;
    const checked = await checkbox.isChecked().catch(() => false);
    if (!checked) {
      await checkbox.check({ force: true });
    }
  }
  const policyDialog = page.getByRole('heading', { name: '隐私政策条款' });
  if (await policyDialog.isVisible().catch(() => false)) {
    await page.getByRole('button', { name: '关闭' }).click().catch(() => {});
  }
  const legalHint = page.getByText('请阅读并勾选同意隐私政策后再登录', { exact: false });
  if (await legalHint.isVisible().catch(() => false)) {
    await page.getByRole('button', { name: '确定' }).click().catch(() => {});
  }

  await page.locator('#email').fill(email);
  await page.locator('#password').fill(password);
  const loginBtn = page.getByRole('button', { name: '登录' });
  const disabledBeforeSubmit = await loginBtn.isDisabled().catch(() => false);
  if (disabledBeforeSubmit) {
    for (const checkbox of checkboxCandidates) {
      const visible = await checkbox.isVisible().catch(() => false);
      if (!visible) continue;
      await checkbox.check({ force: true }).catch(() => {});
    }
  }
  for (let i = 0; i < 60; i++) {
    if (await loginBtn.isEnabled().catch(() => false)) break;
    await page.waitForTimeout(500);
  }
  const authResponsePromise = page
    .waitForResponse(
      (r) => /\/api\/auth\/?$/.test(new URL(r.url()).pathname) && r.request().method() === 'POST',
      { timeout: 120000 },
    )
    .catch(() => null);

  await page.locator('form').first().evaluate((form) => form.requestSubmit());

  const authResp = await authResponsePromise;
  if (authResp) {
    const status = authResp.status();
    if (status >= 400) {
      const body = (await authResp.text().catch(() => '')).slice(0, 400);
      const hint =
        status === 404 && body.includes('user not found')
          ? '（enrich-login 无法在 Django 侧解析 taskAuth 用户：请确认 saas-backend 的 TASKAUTH_BASE_URL 与网关 /api/auth/ 指向同一 task-auth 实例，且 auth.db 中 accounts_user 与 login_method.object_id 一致）'
          : status === 403 && body.includes('forbidden')
            ? '（多为 taskAuth djangoInternalApiBase 指向公网 api，enrich-login 被 APISIX deny-internal 拦截；应改为 http://127.0.0.1:8001）'
            : '';
      throw new Error(`登录 API 返回 ${status}：${body}${hint}`);
    }
  } else {
    const stuckLoggingIn = await page.getByRole('button', { name: /登录中/ }).isVisible().catch(() => false);
    if (stuckLoggingIn) {
      throw new Error(
        '登录请求 120s 内无 /api/auth 响应：请确认 Django :8001 未阻塞（其它长请求占满 runserver 单线程时会出现「登录中…」卡住）',
      );
    }
  }

  await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 120000 });
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(500);
}
