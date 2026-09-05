// @ts-check
import { test, expect } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const portConfig = loadPortConfig();

const ACCESS_CODE = process.env.REGISTER_ACCESS_CODE || 'u824976301710503936';
const REGISTER_PASSWORD = process.env.REGISTER_TEST_PASSWORD;
test.skip(!REGISTER_PASSWORD, 'REGISTER_PASSWORD env required (no hardcoded fallback)');

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
async function waitForActivationEmailInStream(email, { timeoutMs = 20000 } = {}) {
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
      expect(envelope.data?.subject || '').toMatch(/激活/);
      expect(String(envelope.data?.html_message || envelope.data?.message || '')).toMatch(
        /\/auth\/activate\//,
      );
      return envelope;
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error(`超时：Redis stream ${stream} 未找到发往 ${email} 的 EMAIL_SENT 事件`);
}

test.describe('邮箱注册激活邮件', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前后端与 Redis 的全栈 E2E');

  test('带 accessCode 注册后发布 EMAIL_SENT 激活邮件事件', async ({ page }) => {
    const uniqueEmail = `pw-reg-${Date.now()}@example.com`;
    const registerUrl = `/auth/register/?accessCode=${encodeURIComponent(ACCESS_CODE)}`;

    await page.goto(registerUrl);
    await page.waitForLoadState('domcontentloaded');

    const checkboxCandidates = [
      page.getByTestId('register-privacy-accept'),
      page.getByRole('checkbox', { name: /隐私政策/ }),
      page.getByTestId('register-license-accept'),
      page.getByRole('checkbox', { name: /软件许可/ }),
      page.getByRole('checkbox', { name: /全部条款/ }),
    ];
    for (const checkbox of checkboxCandidates) {
      if (await checkbox.isVisible().catch(() => false)) {
        if (!(await checkbox.isChecked().catch(() => false))) {
          await checkbox.check();
        }
      }
    }

    await page.locator('#email').fill(uniqueEmail);
    await page.locator('#password').fill(REGISTER_PASSWORD);

    const registerResponsePromise = page.waitForResponse(
      (r) => r.url().includes('/api/accounts/users/email_register/') && r.request().method() === 'POST',
      { timeout: 60000 },
    );

    await page.locator('form').first().evaluate((form) => form.requestSubmit());

    const registerResp = await registerResponsePromise;
    expect(registerResp.status(), await registerResp.text().catch(() => '')).toBeLessThan(400);

    const envelope = await waitForActivationEmailInStream(uniqueEmail);
    expect(envelope.data.recipient_list).toContain(uniqueEmail);
  });
});
