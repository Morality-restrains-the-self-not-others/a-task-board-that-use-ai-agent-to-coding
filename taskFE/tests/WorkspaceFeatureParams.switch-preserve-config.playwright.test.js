// @ts-check
/**
 * 工作空间功能参数 — 公司默认↔自定义切换时保留已保存的自定义配置
 *
 * 场景:
 *   1. 登录 → 进入工作空间功能参数页
 *   2. 切换到「自定义工作空间配置」，填入自定义 agent_model
 *   3. 保存 → 验证保存成功
 *   4. 切换到「使用公司默认配置」→ 保存
 *   5. 刷新页面
 *   6. 切换回「自定义工作空间配置」
 *   7. 断言 agent_model 仍是步骤 2 填入的值（未被清空）
 *
 * 运行:
 *   cd taskFE
 *   LOGIN_URL='http://127.0.0.1:4000' \
 *   LOGIN_EMAIL='contact@daydaymoney.com' \
 *   LOGIN_PASSWORD='<env:PLAYWRIGHT_TEST_PASSWORD>' \
 *   PW_TENANT_ID='850256677331562496' \
 *   PW_WORKSPACE_ID='857903329669984256' \
 *   npx playwright test -c playwright.verify.config.js \
 *     tests/WorkspaceFeatureParams.switch-preserve-config.playwright.test.js --project=chromium
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import {
  installApisixCorsWorkaround,
  trackConsoleErrorsAfterLogin,
  waitForLoginLegalPolicies,
} from './helpers/remoteLoginE2e.js';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const LOGIN_EMAIL = process.env.LOGIN_EMAIL || '';
const LOGIN_PASSWORD = process.env.LOGIN_PASSWORD || '';
const LOGIN_URL = (process.env.LOGIN_URL || '').replace(/\/$/, '');
const TENANT_ID = process.env.PW_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || '';

const CUSTOM_AGENT_MODEL = 'e2e-switch-preserve-test';

test.describe('工作空间功能参数切换保留配置', () => {
  test.beforeEach(async ({ page }) => {
    await installApisixCorsWorkaround(page);
  });

  test('自定义→公司默认→刷新→切回自定义，自定义配置不丢失', async ({ page }) => {
    test.setTimeout(180000);
    test.skip(!LOGIN_EMAIL || !LOGIN_PASSWORD, '请设置 LOGIN_EMAIL / LOGIN_PASSWORD');
    test.skip(!WORKSPACE_ID, '请设置 PW_WORKSPACE_ID');

    const root = LOGIN_URL || '';
    const loginPath = `${root}/auth/login/`;

    // --- Console error tracking ---
    const { errors: consoleErrors, markLoginDone } = trackConsoleErrorsAfterLogin(page);
    await waitForLoginLegalPolicies(page, loginPath);

    // ================================================================
    // Step 1: Login
    // ================================================================
    await playwrightLoginWithLegalAccept(page, {
      email: LOGIN_EMAIL,
      password: LOGIN_PASSWORD,
      baseURL: root,  // 远程服务器必须传 baseURL，否则 cookie domain 不匹配
    });
    markLoginDone();

    // ================================================================
    // Step 2: Navigate to workspace feature params page
    // ================================================================
    const workspaceUrl = `${root}/tenant/${TENANT_ID}/settings/workspace/${WORKSPACE_ID}/feature-params/`;
    await page.goto(workspaceUrl, { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(500);

    // Verify we didn't get redirected to login
    if (page.url().includes('/auth/login')) {
      throw new Error(`登录失败：被重定向到登录页。URL=${page.url()}`);
    }

    await expect(page.locator('h2').first()).toContainText('工作空间环境变量', { timeout: 10000 });
    console.log('✓ 工作空间功能参数页加载成功');

    // ================================================================
    // Step 3: Switch to custom workspace config
    // ================================================================
    const customRadio = page.locator('input[type="radio"][value="false"]');
    await customRadio.check();
    await page.waitForTimeout(300);
    console.log('✓ 已切换到自定义工作空间配置');

    // ================================================================
    // Step 4: Expand LLM config panel & fill in custom agent model
    // ================================================================
    const llmToggle = page.getByRole('button', { name: /LLM 供应商.*模型配置/ });
    await llmToggle.click();
    await page.waitForTimeout(300);

    // Find the agent model input inside the expanded panel
    const agentModelInput = page.locator('input[placeholder*="gpt-4.1"]').first();
    await expect(agentModelInput).toBeVisible({ timeout: 5000 });

    // Fill in a recognizable test value
    await agentModelInput.fill(CUSTOM_AGENT_MODEL);
    console.log(`✓ 已填入 agent_model: "${CUSTOM_AGENT_MODEL}"`);

    // ================================================================
    // Step 5: Save custom config
    // ================================================================
    const saveBtn = page.getByRole('button', { name: '保存' });
    await saveBtn.click();

    // Wait for success message
    await expect(page.getByText('保存成功')).toBeVisible({ timeout: 15000 });
    console.log('✓ 自定义配置保存成功');

    // ================================================================
    // Step 6: Switch to company default & save
    // ================================================================
    const companyDefaultRadio = page.locator('input[type="radio"][value="true"]');
    await companyDefaultRadio.check();
    await page.waitForTimeout(300);
    console.log('✓ 已切换到使用公司默认配置');

    await saveBtn.click();
    await expect(page.getByText('保存成功')).toBeVisible({ timeout: 15000 });
    console.log('✓ 公司默认配置保存成功');

    // ================================================================
    // Step 7: Refresh the page
    // ================================================================
    await page.reload({ waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await expect(page.locator('h2').first()).toContainText('工作空间环境变量', { timeout: 10000 });
    console.log('✓ 页面刷新完成');

    // Verify we're still in company default mode after reload
    const companyDefaultRadioAfterReload = page.locator('input[type="radio"][value="true"]');
    await expect(companyDefaultRadioAfterReload).toBeChecked({ timeout: 5000 });
    console.log('✓ 刷新后仍为使用公司默认配置模式');

    // ================================================================
    // Step 8: Switch back to custom workspace config
    // ================================================================
    await customRadio.check();
    await page.waitForTimeout(300);
    console.log('✓ 已切换回自定义工作空间配置');

    // Expand the LLM panel again
    await llmToggle.click();
    await page.waitForTimeout(300);

    // ================================================================
    // Step 9: Assert the previously saved agent_model still exists
    // ================================================================
    const restoredAgentInput = page.locator('input[placeholder*="gpt-4.1"]').first();
    await expect(restoredAgentInput).toBeVisible({ timeout: 5000 });
    const restoredValue = await restoredAgentInput.inputValue();
    console.log(`  恢复后 agent_model 值: "${restoredValue}"`);

    // The core assertion: the custom value must be preserved
    expect(restoredValue, `agent_model should be "${CUSTOM_AGENT_MODEL}" after switching back to custom`)
      .toBe(CUSTOM_AGENT_MODEL);
    console.log('✓ 自定义 agent_model 在切换后完整保留');

    // ================================================================
    // Step 10: Verify no critical console errors
    // ================================================================
    const criticalErrors = consoleErrors.filter(
      (e) => !e.includes('favicon') && !e.includes('third-party'),
    );
    if (criticalErrors.length > 0) {
      console.warn('⚠ 控制台错误:', criticalErrors);
    }
    expect(criticalErrors.length).toBe(0);
    console.log('✓ 无控制台错误');
  });
});
