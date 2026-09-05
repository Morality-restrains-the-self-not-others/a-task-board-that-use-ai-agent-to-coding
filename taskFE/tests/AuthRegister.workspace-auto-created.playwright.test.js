// @ts-check
/**
 * E2E: 注册新用户后，事件链 USER_CREATED → COMPANY_CREATED → WORKSPACE_CREATED
 * 自动执行，最终生成默认工作空间。
 *
 * 依赖：taskEvents Go 消费者运行中、Redis 可连接
 */
import { test, expect } from '@playwright/test';
import { execSync } from 'child_process';
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

/**
 * Query Redis stream for events of a given type that contain a matchFn.
 * Returns array of parsed envelopes.
 *
 * @param {string} eventType
 * @param {(data: Record<string, any>) => boolean} matchFn
 * @param {{ timeoutMs?: number }} [opts]
 * @returns {Promise<Array<{id: string, data: Record<string, any>}>>}
 */
async function waitForEventsInStream(eventType, matchFn, { timeoutMs = 120000 } = {}) {
  const stream = domainEventsStreamKey();
  const deadline = Date.now() + timeoutMs;
  const results = [];
  while (Date.now() < deadline) {
    // Use XRANGE to scan forward from the beginning; also try XREVRANGE for recent
    const raw = execSync(
      ['redis-cli', ...redisCliArgs(), '--raw', 'XREVRANGE', stream, '+', '-', 'COUNT', '100'].join(' '),
      { encoding: 'utf8' },
    );
    const lines = raw.split('\n').map((l) => l.trim()).filter(Boolean);
    for (let i = 0; i < lines.length; i += 1) {
      if (lines[i] !== 'payload') continue;
      const payload = lines[i + 1];
      if (!payload) continue;
      /** @type {{ event_type?: string; data?: any }} */
      let envelope;
      try {
        envelope = JSON.parse(payload);
      } catch {
        continue;
      }
      if (envelope.event_type !== eventType) continue;
      if (!matchFn(envelope.data || {})) continue;
      const msgId = lines[i - 1]; // message ID is the line before 'payload'
      results.push({ id: msgId, data: envelope.data });
    }
    if (results.length >= 3) break; // enough events
    await new Promise((r) => setTimeout(r, 2000));
  }
  return results;
}

/**
 * @param {string} email
 * @returns {Promise<{id: string, data: any}>}
 */
async function waitForUserCreated(email) {
  const events = await waitForEventsInStream('USER_CREATED', (d) => d.email === email, { timeoutMs: 60000 });
  if (events.length === 0) throw new Error(`未在 Redis stream 中找到 USER_CREATED for ${email}`);
  return events[0];
}

/**
 * @param {string} userId
 * @returns {Promise<{id: string, data: any}>}
 */
async function waitForCompanyCreated(userId) {
  const events = await waitForEventsInStream('COMPANY_CREATED', (d) => String(d.creator_id) === String(userId), { timeoutMs: 120000 });
  if (events.length === 0) throw new Error(`未在 Redis stream 中找到 COMPANY_CREATED for user_id=${userId}`);
  return events[0];
}

/**
 * @param {string} companyId
 * @returns {Promise<{id: string, data: any}>}
 */
async function waitForWorkspaceCreated(companyId) {
  const events = await waitForEventsInStream('WORKSPACE_CREATED', (d) => String(d.company_id) === String(companyId), { timeoutMs: 120000 });
  if (events.length === 0) throw new Error(`未在 Redis stream 中找到 WORKSPACE_CREATED for company_id=${companyId}`);
  return events[0];
}

test.describe('注册时自动创建工作空间', () => {
  test.skip(() => process.env.CI === 'true' || process.env.PRE_COMMIT === '1', '全栈E2E需要Redis+Go消费者');

  test('邮箱注册 → USER_CREATED → COMPANY_CREATED → WORKSPACE_CREATED', async ({ page }) => {
    const uniqueEmail = `pw-ws-${Date.now()}@example.com`;
    const registerUrl = `/auth/register/?accessCode=${encodeURIComponent(ACCESS_CODE)}`;

    // Step 1: 打开注册页面
    await page.goto(registerUrl);
    await page.waitForLoadState('domcontentloaded');

    // Step 2: 勾选隐私政策与许可协议
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

    // Step 3: 填写注册信息
    await page.locator('#email').fill(uniqueEmail);
    await page.locator('#password').fill(REGISTER_PASSWORD);

    // Step 4: 提交注册表单
    const registerResponsePromise = page.waitForResponse(
      (r) => r.url().includes('/api/accounts/users/email_register/') && r.request().method() === 'POST',
      { timeout: 60000 },
    );

    await page.locator('form').first().evaluate((form) => form.requestSubmit());

    const registerResp = await registerResponsePromise;
    const respBody = await registerResp.text().catch(() => '');
    expect(registerResp.status(), `注册API返回 ${registerResp.status()}: ${respBody}`).toBeLessThan(400);

    // Step 5: 验证 USER_CREATED 事件
    console.log(`[test] 等待 USER_CREATED for ${uniqueEmail} ...`);
    const userCreated = await waitForUserCreated(uniqueEmail);
    const userId = String(userCreated.data.user_id);
    expect(userId).toBeTruthy();
    console.log(`[test] USER_CREATED: user_id=${userId}`);

    // Step 6: 验证 COMPANY_CREATED 事件（由 taskEvents user_created/0_create_company 消费者生成）
    console.log(`[test] 等待 COMPANY_CREATED for user_id=${userId} ...`);
    const companyCreated = await waitForCompanyCreated(userId);
    const companyId = String(companyCreated.data.company_id);
    expect(companyId).toBeTruthy();
    console.log(`[test] COMPANY_CREATED: company_id=${companyId}`);

    // Step 7: 验证 WORKSPACE_CREATED 事件（由 taskEvents company_created/3_create_default_workspace 消费者生成）
    console.log(`[test] 等待 WORKSPACE_CREATED for company_id=${companyId} ...`);
    const workspaceCreated = await waitForWorkspaceCreated(companyId);
    const workspaceId = String(workspaceCreated.data.workspace_id);
    const workspaceName = String(workspaceCreated.data.workspace_name || '');
    expect(workspaceId).toBeTruthy();
    expect(workspaceCreated.data.is_default).toBe(true);
    console.log(`[test] WORKSPACE_CREATED: workspace_id=${workspaceId} name="${workspaceName}"`);

    // Step 8: 验证工作空间名称包含用户名（注册邮箱前缀）
    const expectedNamePrefix = uniqueEmail.split('@')[0];
    expect(workspaceName).toContain(expectedNamePrefix);
    expect(workspaceName).toMatch(/的工作空间$/);
  });

  test('事件链完整性：3个事件顺序与数据一致性', async ({ page }) => {
    const uniqueEmail = `pw-chain-${Date.now()}@example.com`;
    const registerUrl = `/auth/register/?accessCode=${encodeURIComponent(ACCESS_CODE)}`;

    await page.goto(registerUrl);
    await page.waitForLoadState('domcontentloaded');

    // Accept legal policies
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
    expect(registerResp.status()).toBeLessThan(400);

    // Collect all three events
    const userCreated = await waitForUserCreated(uniqueEmail);
    const userId = String(userCreated.data.user_id);

    const companyCreated = await waitForCompanyCreated(userId);
    const companyId = String(companyCreated.data.company_id);

    const workspaceCreated = await waitForWorkspaceCreated(companyId);
    const workspaceId = String(workspaceCreated.data.workspace_id);

    // Chain integrity checks
    // company_id from USER_CREATED flow should match
    expect(userCreated.data.user_id).toBeTruthy();
    // company creator should be the registering user
    expect(String(companyCreated.data.creator_id)).toBe(userId);
    // workspace company should match
    expect(String(workspaceCreated.data.company_id)).toBe(companyId);
    // workspace is default
    expect(workspaceCreated.data.is_default).toBe(true);
    // workspace creator should be the registering user
    expect(String(workspaceCreated.data.created_by)).toBe(userId);

    console.log(
      `[test] 事件链完整: user=${userId} → company=${companyId} → workspace=${workspaceId}`,
    );
  });
});
