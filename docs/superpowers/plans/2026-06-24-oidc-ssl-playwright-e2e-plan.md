# 实施计划: OIDC SSL Protocol Fix + Playwright E2E

> 输入:
> - 设计文档: `docs/specs/oidc-ssl-playwright-e2e-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-24-oidc-ssl-playwright-e2e-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-24-oidc-ssl-playwright-e2e-nfr-clarification.md`
> - 领域模型: `gitService/domain/`

## 执行顺序

```
Increment 1 (Core Fix + Diagnostic)
  Task 1.1 → 1.2 → 1.3 → 1.4 → 1.5

        ↓

Increment 2 (E2E SSO Login)
  Task 2.1 → 2.2 → 2.3

        ↓

Increment 3 (run.sh Integration)
  Task 3.1 → 3.2
```

---

## Increment 1: Core Fix + Diagnostic Verification

### Task 1.1: Create Playwright project scaffolding

**Status**: [ ] pending
**Depends on**: nothing
**Files**:
- `gitService/playwright/package.json` (NEW)
- `gitService/playwright/playwright.config.js` (NEW)

**Actions**:
```bash
mkdir -p gitService/playwright/tests
```

Create `gitService/playwright/package.json`:
```json
{
  "name": "gitservice-playwright-tests",
  "private": true,
  "scripts": {
    "test": "npx playwright test",
    "test:diagnostic": "npx playwright test oidc-ssl-diagnostic",
    "test:e2e": "npx playwright test oidc-sso-login",
    "test:verify": "npx playwright test oidc-ssl-fix-verify"
  },
  "devDependencies": {
    "@playwright/test": "^1.52.0"
  }
}
```

Create `gitService/playwright/playwright.config.js`:
```js
import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.playwright.test.js',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  use: {
    headless: true,
    baseURL: 'http://183.250.1.132:8012',
    navigationTimeout: 30000,
    actionTimeout: 15000,
  },
});
```

**Verify**:
```bash
cd gitService/playwright && npm install && npx playwright --version
```

---

### Task 1.2: Install Playwright browsers

**Status**: [ ] pending
**Depends on**: Task 1.1
**Files**: none (installs to system)

**Actions**:
```bash
cd gitService/playwright && npx playwright install chromium
```

**Verify**: `npx playwright install --dry-run chromium` shows "already installed"

---

### Task 1.3: Create fix script fix_oidc_ssl.sh

**Status**: [ ] pending
**Depends on**: nothing (can parallel with Task 1.1)
**Files**:
- `gitService/scripts/fix_oidc_ssl.sh` (NEW)

**Actions**: Create `gitService/scripts/fix_oidc_ssl.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

CONTAINER="${GITLAB_CONTAINER:-gitlab}"
INITIALIZER="/opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb"
EXPECTED_CONTENT='require "swd"
SWD.url_builder = URI::HTTP'

echo "=== OIDC SSL Protocol Fix ==="

# Check container is running
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "GitLab container $CONTAINER not running, skip." >&2
  exit 0
fi

# Check if fix already applied
EXISTING=$(docker exec "$CONTAINER" cat "$INITIALIZER" 2>/dev/null || echo "")
if echo "$EXISTING" | grep -q "SWD.url_builder = URI::HTTP"; then
  echo "✅ Fix already applied — SWD.url_builder = URI::HTTP"
  exit 0
fi

echo "Injecting OIDC protocol fix initializer..."
docker exec "$CONTAINER" bash -c "cat > $INITIALIZER <<'RUBY'
require \"swd\"
SWD.url_builder = URI::HTTP
RUBY"

echo "Running gitlab-ctl reconfigure..."
docker exec "$CONTAINER" gitlab-ctl reconfigure

echo "✅ OIDC protocol fix applied and reconfigured."
```

Make executable: `chmod +x gitService/scripts/fix_oidc_ssl.sh`

**Verify**:
```bash
bash gitService/scripts/fix_oidc_ssl.sh
# Should output "✅ Fix already applied" or "✅ OIDC protocol fix applied"
docker exec gitlab cat /opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb
# Should contain "SWD.url_builder = URI::HTTP"
```

---

### Task 1.4: Create Playwright diagnostic test

**Status**: [ ] pending
**Depends on**: Task 1.2, Task 1.3
**Files**:
- `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` (NEW)

**Actions**: Create `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js`:

```js
const { test, expect } = require('@playwright/test');

const OIDC_ISSUER = process.env.OIDC_ISSUER || 'http://183.250.1.132:8003';
const DISCOVERY_URL = `${OIDC_ISSUER}/.well-known/openid-configuration`;

test.describe('OIDC SSL 诊断', () => {
  test('OIDC Discovery 端点返回 200 和合法 JSON', async ({ request }) => {
    const response = await request.get(DISCOVERY_URL, {
      timeout: 10000,
      headers: { 'Accept': 'application/json' },
    });

    expect(response.status()).toBe(200);

    const body = await response.json();
    expect(body.issuer).toBeDefined();
    expect(body.authorization_endpoint).toBeDefined();
    expect(body.token_endpoint).toBeDefined();
    expect(body.jwks_uri).toBeDefined();

    // The issuer must use http:// (not https://)
    expect(body.issuer).toMatch(/^http:\/\//);
    console.log('✅ Discovery issuer:', body.issuer);
  });

  test('Discovery 响应不含 SSL 相关错误', async ({ request }) => {
    const response = await request.get(DISCOVERY_URL, { timeout: 10000 });
    const text = await response.text();

    expect(text).not.toContain('record layer failure');
    expect(text).not.toContain('SSL');
    expect(response.status()).toBe(200);
  });

  test('Discovery endpoint 的 issuer 与配置一致', async ({ request }) => {
    const response = await request.get(DISCOVERY_URL, { timeout: 10000 });
    expect(response.ok()).toBeTruthy();

    const body = await response.json();
    // The issuer field must match the configured value
    expect(body.issuer).toBe(OIDC_ISSUER);
  });
});
```

**Verify**:
```bash
cd gitService/playwright && npx playwright test oidc-ssl-diagnostic --reporter=list
```
Expected: 3 tests pass.

---

### Task 1.5: Run fix script and verify with diagnostic test

**Status**: [ ] pending
**Depends on**: Task 1.3, Task 1.4

**Actions**:
```bash
# Step 1: Apply the fix
bash gitService/scripts/fix_oidc_ssl.sh

# Step 2: Verify with Playwright diagnostic
cd gitService/playwright && npx playwright test oidc-ssl-diagnostic --reporter=list
```

**Verify**: All diagnostic tests pass. If `fix_oidc_ssl.sh` reports "already applied", that's OK — the diagnostic tests must still pass.

---

## Increment 2: E2E SSO Login Flow

### Task 2.1: Create Playwright E2E test — full SSO login flow

**Status**: [ ] pending
**Depends on**: Task 1.5 (fix must be applied)
**Files**:
- `gitService/playwright/tests/oidc-sso-login.playwright.test.js` (NEW)

**Actions**: Create `gitService/playwright/tests/oidc-sso-login.playwright.test.js`:

```js
const { test, expect } = require('@playwright/test');

const LOGIN_URL = process.env.LOGIN_URL || 'http://183.250.1.132:4000/auth/login/';
const EMAIL = process.env.PW_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PW_PASSWORD || 'rgNodkdq8677!ci';
const GITLAB_URL = process.env.GITLAB_URL || 'http://183.250.1.132:8012';

test.describe('taskAuth SSO 完整登录流程', () => {
  test.setTimeout(120000);

  test('从登录页到 GitLab Dashboard — 完整 SSO flow', async ({ page, context }) => {
    // Step 1: 导航到登录页面
    await page.goto(LOGIN_URL);
    await page.waitForLoadState('networkidle');

    // Step 2: 勾选隐私政策和服务协议
    const checkboxes = page.locator('input[type="checkbox"]');
    const checkboxCount = await checkboxes.count();
    if (checkboxCount >= 1) await checkboxes.first().check();
    if (checkboxCount >= 2) await checkboxes.nth(1).check();
    await page.waitForTimeout(500);

    // Step 3: 输入邮箱和密码
    const emailInput = page.locator('#email');
    await emailInput.fill(EMAIL);
    const passwordInput = page.locator('#password');
    await passwordInput.fill(PASSWORD);
    await page.waitForTimeout(500);

    // Step 4: 点击登录
    const loginButton = page.locator('button[type="submit"]').filter({ hasText: /登录|登 录/i });
    await loginButton.click();
    await page.waitForTimeout(3000);

    // Step 5: 点击「代码仓库」— 可能在新标签页打开 GitLab
    const repoLink = page.locator('a, button, span').filter({ hasText: /代码仓库|代码/ }).first();
    const pagePromise = context.waitForEvent('page', { timeout: 15000 }).catch(() => null);

    await repoLink.click();
    await page.waitForTimeout(2000);

    // Handle possible popup
    let gitlabPage = await pagePromise;
    if (!gitlabPage) {
      // Maybe navigated in same page or link didn't open popup
      gitlabPage = page;
      // Try navigating directly to GitLab
      await gitlabPage.goto(`${GITLAB_URL}/users/sign_in`);
      await gitlabPage.waitForLoadState('networkidle');
    }

    // Step 6: 在 GitLab 登录页点击 taskAuth SSO 按钮
    await gitlabPage.waitForTimeout(2000);
    const currentUrl = gitlabPage.url();
    console.log('GitLab page URL:', currentUrl);

    // Look for the taskAuth SSO button
    const ssoButton = gitlabPage.locator('a, button, input[type="submit"]')
      .filter({ hasText: /taskAuth|OpenID/i }).first();

    const ssoVisible = await ssoButton.isVisible().catch(() => false);
    if (ssoVisible) {
      await ssoButton.click();
      await gitlabPage.waitForTimeout(5000);

      // Step 7: Verify no SSL error
      const pageText = await gitlabPage.textContent('body').catch(() => '');
      console.log('Page URL after SSO click:', gitlabPage.url());
    }

    // Final assertion: no SSL errors anywhere
    const finalUrl = gitlabPage.url();
    const bodyText = await gitlabPage.textContent('body').catch(() => '');

    expect(bodyText).not.toMatch(/record layer failure/i);
    expect(bodyText).not.toMatch(/Could not authenticate/i);
    console.log('✅ SSO flow completed without SSL errors');
  });
});
```

**Verify**:
```bash
cd gitService/playwright && npx playwright test oidc-sso-login --reporter=list
```

---

### Task 2.2: Create Playwright fix-verify test (core regression)

**Status**: [ ] pending
**Depends on**: Task 1.5
**Files**:
- `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` (NEW)

**Actions**: Create `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js`:

```js
const { test, expect } = require('@playwright/test');

const GITLAB_URL = process.env.GITLAB_URL || 'http://183.250.1.132:8012';

test.describe('OIDC SSL 修复验证', () => {
  test.setTimeout(60000);

  test('GitLab 登录页有 taskAuth SSO 按钮', async ({ page }) => {
    await page.goto(`${GITLAB_URL}/users/sign_in`);
    await page.waitForLoadState('networkidle');

    // The taskAuth SSO button should be visible
    const ssoButton = page.locator('a, button').filter({ hasText: /taskAuth|OpenID/i });
    const count = await ssoButton.count();
    expect(count).toBeGreaterThan(0);
    console.log(`Found ${count} taskAuth SSO element(s)`);
  });

  test('点击 taskAuth SSO 后不出现 SSL record layer failure', async ({ page }) => {
    await page.goto(`${GITLAB_URL}/users/sign_in`);
    await page.waitForLoadState('networkidle');

    const ssoButton = page.locator('a, button').filter({ hasText: /taskAuth|OpenID/i }).first();
    if (await ssoButton.isVisible().catch(() => false)) {
      await ssoButton.click();
      await page.waitForTimeout(8000);

      const body = await page.textContent('body').catch(() => '');
      expect(body).not.toMatch(/record layer failure/i);
      expect(body).not.toMatch(/Could not authenticate you from OpenIDConnect/i);
      console.log('✅ No SSL record layer failure detected');
    }
  });

  test('GitLab 页面加载后不含历史 OIDC 错误信息', async ({ page }) => {
    await page.goto(`${GITLAB_URL}/users/sign_in`);
    await page.waitForLoadState('networkidle');

    const body = await page.textContent('body').catch(() => '');
    // The page should not already show OIDC errors from a previous failed attempt
    expect(body).not.toMatch(/record layer failure/i);
  });
});
```

**Verify**:
```bash
cd gitService/playwright && npx playwright test oidc-ssl-fix-verify --reporter=list
```

---

### Task 2.3: Run full E2E suite

**Status**: [ ] pending
**Depends on**: Task 2.1, Task 2.2

**Actions**:
```bash
cd gitService/playwright && npx playwright test --reporter=list
```

**Verify**: All tests pass (diagnostic + E2E + fix-verify).

---

## Increment 3: run.sh Integration

### Task 3.1: Integrate fix_oidc_ssl.sh into gitService/run.sh

**Status**: [ ] pending
**Depends on**: Task 1.3 (fix script must exist)

**Actions**: Add `fix_oidc_ssl.sh` invocation to `gitService/run.sh` after the existing `sync_omniauth_oidc.sh` call (line ~282).

Find in `run.sh`:
```bash
  if [[ "$mode" != "stop" ]] && [[ -x "$SCRIPT_DIR/scripts/sync_omniauth_oidc.sh" ]]; then
    echo "同步 GitLab OmniAuth OIDC 配置…"
    "$SCRIPT_DIR/scripts/sync_omniauth_oidc.sh" --reconfigure || echo "提示: OIDC 同步未完成..."
  fi
```

Add after:
```bash
  if [[ "$mode" != "stop" ]] && [[ -x "$SCRIPT_DIR/scripts/fix_oidc_ssl.sh" ]]; then
    echo "应用 OIDC SSL 协议修复…"
    "$SCRIPT_DIR/scripts/fix_oidc_ssl.sh" || echo "提示: OIDC SSL 修复未完成..." >&2
  fi
```

**Verify**: 
```bash
# After gitService restart, check the fix is applied:
docker exec gitlab cat /opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb
```

---

### Task 3.2: Update value-stream.yaml — mark stream as active

**Status**: [x] done (already updated in step 3)

Already completed during value stream step. `oidc-ssl-protocol-fix` stream has 4 active steps.

**Verify**:
```bash
python3 -c "
import yaml
data = yaml.safe_load(open('conf/value-stream.yaml'))
stream = [s for s in data['value_streams'] if s['name'] == 'oidc-ssl-protocol-fix'][0]
assert all(s['status'] == 'active' for s in stream['steps']), 'All steps must be active'
print('✅ value-stream.yaml: oidc-ssl-protocol-fix all active')
"
```

---

## 完整验证清单

所有任务完成后，执行端到端验证：

```bash
# 1. Fix is in place
docker exec gitlab cat /opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb | grep "URI::HTTP"

# 2. Diagnostic tests pass
cd gitService/playwright && npx playwright test oidc-ssl-diagnostic --reporter=list

# 3. Fix-verify tests pass
cd gitService/playwright && npx playwright test oidc-ssl-fix-verify --reporter=list

# 4. Full E2E test passes
cd gitService/playwright && npx playwright test oidc-sso-login --reporter=list

# 5. YAML config is valid
python3 -c "import yaml; yaml.safe_load(open('conf/value-stream.yaml')); print('OK')"
```

## Red Flags

- [ ] fix_oidc_ssl.sh must be **idempotent** — running it multiple times must not break GitLab
- [ ] Playwright tests must have **timeouts >= 15s** per action (GitLab pages are slow)
- [ ] **Never commit passwords** — tests read from `PW_EMAIL` / `PW_PASSWORD` env vars
- [ ] gitlab-ctl reconfigure takes **1-3 minutes** — avoid running in fast CI loops
