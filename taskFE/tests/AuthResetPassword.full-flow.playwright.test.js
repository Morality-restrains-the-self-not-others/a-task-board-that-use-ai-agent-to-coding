// @ts-check
import { test, expect } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const portConfig = loadPortConfig();

const RESET_EMAIL = process.env.PW_RESET_EMAIL || '';
const NEW_PASSWORD = process.env.PW_RESET_NEW_PASSWORD || 'Test654321@';

const AUTH_DB = path.resolve(__dirname, '../../../../db/task-auth/auth.sqlite3');

function domainEventsStreamKey() {
  const prefix = String(portConfig.domainEvents?.redis?.streamKeyPrefix || 'domain-events:').replace(/:$/, '');
  return `${prefix}:all`;
}

function redisCliArgs() {
  const redisCfg = portConfig.domainEvents?.redis || {};
  const host = redisCfg.host || '127.0.0.1';
  const port = redisCfg.port || 6379;
  return ['-h', host, '-p', String(port), '-n', String(redisCfg.db ?? 0)];
}

/** @param {string} email */
function fetchResetTokenFromDb(email) {
  const sql = `SELECT password_reset_token FROM accounts_login_method WHERE identifier='${email.replace(/'/g, "''")}' AND method_type='email' LIMIT 1;`;
  const raw = execSync(`sqlite3 "${AUTH_DB}" "${sql}"`, { encoding: 'utf8' }).trim();
  return raw || null;
}

/** @param {string} email @param {{ timeoutMs?: number }} [opts] */
async function waitForResetToken(email, { timeoutMs = 20000 } = {}) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const token = fetchResetTokenFromDb(email);
    if (token) return token;
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`超时：auth.db 未找到 ${email} 的 password_reset_token`);
}

/** @param {string} email */
async function waitForPasswordResetEmailEvent(email, { timeoutMs = 20000 } = {}) {
  const stream = domainEventsStreamKey();
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const raw = execSync(
      ['redis-cli', ...redisCliArgs(), '--raw', 'XREVRANGE', stream, '+', '-', 'COUNT', '40'].join(' '),
      { encoding: 'utf8' },
    );
    const lines = raw.split('\n').map((l) => l.trim()).filter(Boolean);
    for (let i = 0; i < lines.length; i += 1) {
      if (lines[i] !== 'payload') continue;
      const payload = lines[i + 1];
      if (!payload) continue;
      let envelope;
      try {
        envelope = JSON.parse(payload);
      } catch {
        continue;
      }
      if (envelope.event_type !== 'EMAIL_SENT') continue;
      const recipients = envelope.data?.recipient_list || [];
      if (!recipients.includes(email)) continue;
      if (!String(envelope.data?.subject || '').includes('密码重置')) continue;
      const body = String(
        envelope.data?.html_message || envelope.data?.message || envelope.data?.context?.reset_url || '',
      );
      expect(body.length).toBeGreaterThan(0);
      return envelope;
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`超时：Redis stream ${stream} 未找到发往 ${email} 的密码重置 EMAIL_SENT`);
}

test.describe('密码重置完整流程', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端、Redis 与 taskAuth DB 的全栈 E2E');

  test('请求重置 → DB 取 token → 设置新密码 → 可用新密码登录', async ({ page }) => {
    test.skip(!RESET_EMAIL, '设置 PW_RESET_EMAIL 后运行（本地核验示例：PW_RESET_EMAIL=you@example.com PW_RESET_NEW_PASSWORD=…）');

    test.setTimeout(120000);

    page.on('dialog', (dialog) => dialog.accept());

    await page.goto('/auth/reset-password-request/');
    await page.waitForLoadState('domcontentloaded');

    await page.locator('#email').fill(RESET_EMAIL);

    const resetRequestPromise = page.waitForResponse(
      (r) => r.url().includes('/api/accounts/users/send_password_reset_link/') && r.request().method() === 'POST',
      { timeout: 60000 },
    );

    await page.locator('form').first().evaluate((form) => form.requestSubmit());

    const resetRequestResp = await resetRequestPromise;
    expect(resetRequestResp.status(), await resetRequestResp.text().catch(() => '')).toBeLessThan(400);

    const [token, emailEvent] = await Promise.all([
      waitForResetToken(RESET_EMAIL),
      waitForPasswordResetEmailEvent(RESET_EMAIL),
    ]);
    expect(token).toBeTruthy();
    expect(emailEvent.data?.template_name || emailEvent.data?.message).toBeTruthy();

    await page.goto(`/auth/reset-password/${token}/`);
    await page.waitForURL(/\/auth\/reset-password\/\?token=/, { timeout: 15000 });

    await expect(page.locator('h2')).toContainText('设置新密码');

    // Token 验证现在在页面加载时立即触发（onMounted），不再等用户提交表单
    // 设置监听器在导航之前捕获请求
    const getUserInfoPromise = page.waitForResponse(
      (r) => r.url().includes(`/api/accounts/users/get-reset-user-info/${token}/`) && r.request().method() === 'GET',
      { timeout: 60000 },
    );

    // 等待 token 验证完成
    const getUserInfoResp = await getUserInfoPromise;
    expect(getUserInfoResp.status()).toBeLessThan(400);

    await page.locator('#password').fill(NEW_PASSWORD);
    await page.locator('#password_confirm').fill(NEW_PASSWORD);

    const resetSubmitPromise = page.waitForResponse(
      (r) => r.url().includes(`/api/accounts/users/reset-password-with-link/${token}/`) && r.request().method() === 'POST',
      { timeout: 60000 },
    );

    await page.locator('form').first().evaluate((form) => form.requestSubmit());

    const resetSubmitResp = await resetSubmitPromise;
    expect(resetSubmitResp.status(), await resetSubmitResp.text().catch(() => '')).toBeLessThan(400);

    await page.waitForURL(/\/auth\/login\/?/, { timeout: 30000 });

    expect(fetchResetTokenFromDb(RESET_EMAIL)).toBeFalsy();

    const checkboxCandidates = [
      page.getByTestId('login-accept-all'),
      page.getByRole('checkbox', { name: /全部条款/ }),
      page.getByTestId('login-privacy-accept'),
      page.getByRole('checkbox', { name: /隐私政策/ }),
      page.getByTestId('login-license-accept'),
      page.getByRole('checkbox', { name: /软件许可/ }),
    ];
    for (const checkbox of checkboxCandidates) {
      if (await checkbox.isVisible().catch(() => false)) {
        if (!(await checkbox.isChecked().catch(() => false))) {
          await checkbox.check();
        }
      }
    }

    await page.locator('#email').fill(RESET_EMAIL);
    await page.locator('#password').fill(NEW_PASSWORD);

    const loginBtn = page.getByRole('button', { name: '登录' });
    for (let i = 0; i < 30; i += 1) {
      if (await loginBtn.isEnabled().catch(() => false)) break;
      await page.waitForTimeout(500);
    }

    const loginPromise = page.waitForResponse(
      (r) => r.url().includes('/api/auth') && r.request().method() === 'POST',
      { timeout: 60000 },
    );

    await loginBtn.click();

    const loginResp = await loginPromise;
    expect(loginResp.status(), await loginResp.text().catch(() => '')).toBeLessThan(400);

    await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 60000 });
  });
});
