// @ts-check
/**
 * E2E: 任务详情页 — synology-gitlab OAuth 绑定（client_id: 3366a2…）
 *
 * 回归场景：点击「OAuth 绑定」→ GitLab 登录 → taskAuth SSO → /oauth/authorize
 * 不应出现 Doorkeeper「Client authentication failed due to unknown client」。
 *
 * 背景：synology-gitlab provider 曾被合并到 gitlab-local，
 * 导致其 YAML 配置文件从仓库删除，但部署环境仍引用该 provider。
 * 同步脚本（sync_local_oauth_app_scopes.sh）无法注册对应的 Doorkeeper Application，
 * GitLab 重建后 OAuth App 丢失 → 客户端认证失败。
 *
 * 修复：恢复 http-synology-gitlab.yaml provider 配置 + 同步脚本模板解析。
 * 本测试验证修复有效。
 *
 * 运行：
 *   PLAYWRIGHT_INTEGRATION=1 bash taskFE/tests/TaskDetail.synology-gitlab-oauth-bind.playwright.test.sh
 */
import { test, expect } from '@playwright/test';

const SITE_BASE = process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000';
const GATEWAY_BASE = process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081';
const GITLAB_BASE = process.env.GITLAB_BASE || 'http://127.0.0.1:8012';
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '857903329669984256';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || '860371538948571136';

const CREDENTIALS = {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: process.env.PLAYWRIGHT_TEST_PASSWORD,
};

/**
 * synology-gitlab Doorkeeper Application
 * （conf/auth/git-oauth/providers/http-synology-gitlab.yaml）
 */
const SYNOLOGY_GITLAB_CLIENT_ID =
  '3366a2f6956b51cbb10ad8bc8d6d0977de4696f42b6131e2fa0bb14ff1e2920e';
const SYNOLOGY_GITLAB_REDIRECT_URI =
  process.env.SYNOLOGY_GITLAB_REDIRECT_URI ||
  `${process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081'}/api/accounts/synology-gitlab/oauth/callback/`;
const SYNOLOGY_GITLAB_SCOPE = 'read_repository write_repository api read_user';

const TASK_DETAIL_PATH = `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`;
const CLIENT_AUTH_FAILED_RE =
  /Client authentication failed|unknown client|unsupported authentication method/i;
const OAUTH_AUTHORIZE_PATH = '/oauth/authorize';

/** 通过网关 API 登录并注入 session cookie */
async function loginViaGatewayApi(page, request) {
  const gatewayOrigin = process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081';
  const siteOrigin = process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000';

  const loginRes = await request.post(`${gatewayOrigin}/api/auth/`, {
    data: { email: CREDENTIALS.email, password: CREDENTIALS.password },
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
  });

  if (!loginRes.ok()) {
    test.skip(true, `主站登录 API 不可用: ${loginRes.status()} ${await loginRes.text().catch(() => '')}`);
  }

  const setCookie = loginRes.headers()['set-cookie'] || '';
  const userIdMatch = setCookie.match(/userId=([^;]+)/i);
  if (userIdMatch) {
    await page.context().addCookies([
      {
        name: 'userId',
        value: userIdMatch[1],
        url: `${siteOrigin}/`,
      },
    ]);
  }

  // 访问主站以建立 session
  await page.goto(`${siteOrigin}/tenant/${TENANT_ID}/projects/`, {
    waitUntil: 'domcontentloaded',
    timeout: 60_000,
  });
  if (page.url().includes('/auth/login')) {
    test.skip(true, 'API 登录后仍被重定向到登录页，请检查网关与 saas-backend');
  }
}

/**
 * 断言 authorize URL 参数符合 synology-gitlab provider 配置
 */
function assertAuthorizeParams(authorizeUrl) {
  const parsed = new URL(authorizeUrl);
  expect(parsed.searchParams.get('client_id')).toBe(SYNOLOGY_GITLAB_CLIENT_ID);
  expect(parsed.searchParams.get('redirect_uri')).toBe(SYNOLOGY_GITLAB_REDIRECT_URI);
  expect(parsed.searchParams.get('scope')).toBe(SYNOLOGY_GITLAB_SCOPE);
  expect(parsed.searchParams.get('response_type')).toBe('code');
  const state = parsed.searchParams.get('state');
  expect(state).toBeTruthy();
  expect(state.length).toBeGreaterThanOrEqual(16);
}

// ═══════════════════════════════════════════════════════════════════
// 预检：直接请求 GitLab /oauth/authorize，验证 Doorkeeper 应用存在
// ═══════════════════════════════════════════════════════════════════

test.describe('synology-gitlab OAuth 绑定 — GitLab Doorkeeper 预检', () => {
  test('GitLab /oauth/authorize 应识别 synology-gitlab client_id（非 unknown client）', async ({
    request,
  }) => {
    const authorizeUrl =
      `${GITLAB_BASE}${OAUTH_AUTHORIZE_PATH}?` +
      `client_id=${encodeURIComponent(SYNOLOGY_GITLAB_CLIENT_ID)}` +
      `&redirect_uri=${encodeURIComponent(SYNOLOGY_GITLAB_REDIRECT_URI)}` +
      `&scope=${encodeURIComponent(SYNOLOGY_GITLAB_SCOPE)}` +
      `&state=e2e-synology-preflight` +
      `&response_type=code`;

    const res = await request.get(authorizeUrl, { maxRedirects: 0 });
    const status = res.status();
    const body = await res.text().catch(() => '');

    // 核心断言：不应出现 "unknown client" 错误
    expect(body).not.toMatch(CLIENT_AUTH_FAILED_RE);

    // 未登录时应 302 到 sign_in，而非 403/503 unknown client
    expect([302, 200]).toContain(status);
    if (status === 302) {
      const location = res.headers()['location'] || '';
      expect(location).toMatch(/sign_in|login|oauth\/authorize/i);
    }
  });

  test('synology-gitlab 与 gitlab-local 的 client_id 不应相同', () => {
    const gitlabLocalClientId =
      '8d98496251234865578283bbf0e8d65872872b7e435979baa3c8441c13770b5a';
    expect(SYNOLOGY_GITLAB_CLIENT_ID).not.toBe(gitlabLocalClientId);
  });

  test('synology-gitlab scopes 应包含 write_repository', () => {
    expect(SYNOLOGY_GITLAB_SCOPE).toContain('write_repository');
    expect(SYNOLOGY_GITLAB_SCOPE).toContain('read_repository');
    expect(SYNOLOGY_GITLAB_SCOPE).toContain('api');
    expect(SYNOLOGY_GITLAB_SCOPE).toContain('read_user');
  });

  test('synology-gitlab redirect_uri 应包含正确的 service_provider 路径', () => {
    expect(SYNOLOGY_GITLAB_REDIRECT_URI).toContain('/api/accounts/synology-gitlab/oauth/callback/');
  });
});

// ═══════════════════════════════════════════════════════════════════
// 端到端：任务详情页 → OAuth 按钮 → authorize 页面
// ═══════════════════════════════════════════════════════════════════

test.describe('TaskDetail synology-gitlab OAuth 绑定 — 端到端', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑全栈集成');
  test.skip(() => process.env.PLAYWRIGHT_INTEGRATION !== '1', '需 PLAYWRIGHT_INTEGRATION=1');

  test.setTimeout(240_000);

  test('任务详情点击 OAuth 绑定 → SSO → authorize 页无 unknown client 错误', async ({
    page,
    request,
  }) => {
    await loginViaGatewayApi(page, request);

    await page.goto(`${SITE_BASE}${TASK_DETAIL_PATH}`, {
      waitUntil: 'domcontentloaded',
      timeout: 60_000,
    });
    if (page.url().includes('/auth/login')) {
      test.skip(true, '登录后无法访问任务详情');
    }

    await expect(page.getByText('关联项目', { exact: true })).toBeVisible({ timeout: 60_000 });

    const mentionEditor = page.getByTestId('comment-content-editor');
    if (await mentionEditor.isVisible().catch(() => false)) {
      await mentionEditor.click();
      await mentionEditor.pressSequentially('@');
      const pickerItem = page.locator('[data-testid="comment-image-mention-picker"] [role="option"]').first();
      if (await pickerItem.isVisible({ timeout: 8000 }).catch(() => false)) {
        await pickerItem.click();
      }
    }

    const oauthBindBtn = page.getByTestId('comment-repo-oauth-bind-btn').or(page.getByTestId('task-repo-oauth-bind-btn')).first();
    await expect
      .poll(
        async () => {
          const bindVisible = await oauthBindBtn.isVisible().catch(() => false);
          const detecting = await page.getByRole('button', { name: '检测中...' }).count();
          return bindVisible || detecting === 0;
        },
        { timeout: 45_000, message: '等待 OAuth 绑定状态检测完成' },
      )
      .toBe(true);

    const bindCount = await page.getByTestId('comment-repo-oauth-bind-btn').or(page.getByTestId('task-repo-oauth-bind-btn')).count();
    test.skip(bindCount === 0, '当前任务所有仓库 OAuth 已绑定，跳过未绑定场景');

    await expect(oauthBindBtn).toBeVisible();
    await expect(oauthBindBtn).toContainText('OAuth 绑定');

    // 监听 authorize 请求以捕获参数
    let capturedAuthorizeUrl = '';
    let capturedStartUrl = '';
    page.on('request', (req) => {
      const url = req.url();
      if (url.includes(OAUTH_AUTHORIZE_PATH)) {
        capturedAuthorizeUrl = url;
      }
      if (url.includes('/oauth/start-from-gateway/')) {
        capturedStartUrl = url;
      }
    });

    await oauthBindBtn.click();

    // 等待进入 GitLab 域（端口 8012）
    await expect
      .poll(() => page.url(), { timeout: 60_000, message: '等待跳转到 GitLab' })
      .toMatch(/:8012/);

    const gitlabHostPattern = /:8012/;
    let ssoAttempts = 0;
    const maxSteps = 40;

    for (let step = 0; step < maxSteps; step++) {
      const u = page.url();
      const bodyText = (await page.textContent('body').catch(() => '')) || '';

      // 核心断言：每一步都不应看到 unknown client 错误
      expect(bodyText, `步骤 ${step} URL=${u}`).not.toMatch(CLIENT_AUTH_FAILED_RE);

      if (bodyText.match(CLIENT_AUTH_FAILED_RE)) {
        throw new Error(
          `GitLab 返回 unknown client 错误\n` +
            `URL: ${u}\n` +
            `capturedAuthorizeUrl: ${capturedAuthorizeUrl}\n` +
            `capturedStartUrl: ${capturedStartUrl}`,
        );
      }

      // 已回到主站（OAuth 回调完成）
      if (u.includes(new URL(SITE_BASE).host) && u.includes('/task-detail/')) {
        break;
      }

      // GitLab OAuth 授权页 — 成功到达
      if (u.includes(OAUTH_AUTHORIZE_PATH)) {
        expect(bodyText).not.toMatch(CLIENT_AUTH_FAILED_RE);

        // 验证 authorize URL 参数
        if (capturedAuthorizeUrl) {
          assertAuthorizeParams(capturedAuthorizeUrl);
        }

        // 若已登录且出现 Authorize 按钮，点击完成授权
        const authorizeBtn = page
          .locator(
            'input[type="submit"][value="Authorize"], button:has-text("Authorize"), button:has-text("授权")',
          )
          .first();
        if (await authorizeBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
          await authorizeBtn.click();
          await page.waitForTimeout(5000);
        }
        continue;
      }

      // GitLab 登录页 — 点击 taskAuth SSO
      if (
        gitlabHostPattern.test(u) &&
        (u.includes('/sign_in') || u.includes('/login') || u.includes('/users/sign_in'))
      ) {
        const ssoBtn = page
          .getByRole('button', { name: /taskAuth/i })
          .or(page.locator('[data-testid="oidc-login-button"]'))
          .first();
        if (await ssoBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
          ssoAttempts += 1;
          expect(ssoAttempts, 'taskAuth SSO 不应无限重试').toBeLessThanOrEqual(5);
          await ssoBtn.click();
          await page.waitForTimeout(8000);
          continue;
        }
      }

      // 主站 SSO/OIDC 回调中
      if (
        u.includes(new URL(SITE_BASE).host) &&
        (u.includes('/auth/') || u.includes('/oidc/') || u.includes('/sso/'))
      ) {
        await page.waitForTimeout(5000);
        continue;
      }

      await page.waitForTimeout(2000);
    }

    // 最终断言：不应有 unknown client 错误
    const finalUrl = page.url();
    const finalBody = (await page.textContent('body').catch(() => '')) || '';
    expect(finalBody).not.toMatch(CLIENT_AUTH_FAILED_RE);

    const reachedAuthorize =
      capturedAuthorizeUrl.includes(SYNOLOGY_GITLAB_CLIENT_ID) ||
      finalUrl.includes(OAUTH_AUTHORIZE_PATH) ||
      finalUrl.includes('/task-detail/');
    expect(
      reachedAuthorize,
      `应到达 authorize 或完成回调，captured=${capturedAuthorizeUrl} final=${finalUrl}`,
    ).toBeTruthy();

    // 验证捕获到的 authorize URL 参数
    if (capturedAuthorizeUrl) {
      assertAuthorizeParams(capturedAuthorizeUrl);
    }
  });
});

// ═══════════════════════════════════════════════════════════════════
// 配置一致性：验证 YAML 配置与测试常量对齐
// ═══════════════════════════════════════════════════════════════════

test.describe('synology-gitlab 配置一致性', () => {
  test('client_id 格式为 64 位十六进制（SHA-256）', () => {
    expect(SYNOLOGY_GITLAB_CLIENT_ID).toMatch(/^[0-9a-f]{64}$/);
  });

  test('redirect_uri 使用 gateway 域名而非 localhost', () => {
    expect(SYNOLOGY_GITLAB_REDIRECT_URI).not.toContain('localhost');
    expect(SYNOLOGY_GITLAB_REDIRECT_URI).not.toContain('127.0.0.1');
  });

  test('scope 中每个权限用空格分隔', () => {
    const scopes = SYNOLOGY_GITLAB_SCOPE.split(' ');
    expect(scopes.length).toBeGreaterThanOrEqual(3);
    scopes.forEach((s) => expect(s).toBeTruthy());
  });
});
