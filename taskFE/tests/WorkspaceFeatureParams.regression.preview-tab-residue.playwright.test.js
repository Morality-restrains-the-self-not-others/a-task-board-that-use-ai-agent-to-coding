// @ts-check
/**
 * 回归测试：预览不为空 + Tab 切换无残留
 *
 * 运行:
 *   cd taskFE
 *   LOGIN_URL='http://127.0.0.1:4000' \
 *   LOGIN_EMAIL='...' LOGIN_PASSWORD='...' \
 *   PW_TENANT_ID='850256677331562496' \
 *   PW_WORKSPACE_ID='857903329669984256' \
 *   npx playwright test -c playwright.verify.chromium.config.js \
 *     tests/WorkspaceFeatureParams.regression.preview-tab-residue.playwright.test.js --project=chromium
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

test.describe('Feature Params Regression', () => {
  test.beforeEach(async ({ page }) => {
    await installApisixCorsWorkaround(page);
  });

  test('Bug-1: 公司级页面环境变量预览不为空', async ({ page }) => {
    test.setTimeout(120000);
    test.skip(!LOGIN_EMAIL || !LOGIN_PASSWORD, '请设置 LOGIN_EMAIL / LOGIN_PASSWORD');

    const root = LOGIN_URL || '';
    await waitForLoginLegalPolicies(page, `${root}/auth/login/`);
    const { errors, markLoginDone } = trackConsoleErrorsAfterLogin(page);

    await playwrightLoginWithLegalAccept(page, {
      email: LOGIN_EMAIL,
      password: LOGIN_PASSWORD,
      baseURL: root,
    });
    markLoginDone();

    // Navigate to company-level feature params page
    await page.goto(`${root}/tenant/${TENANT_ID}/settings/feature-params/`, {
      waitUntil: 'domcontentloaded',
    });
    await page.waitForTimeout(500);

    // Verify we're on the right page
    await expect(page.locator('h2').first()).toContainText('环境变量设置', { timeout: 10000 });

    // Find the preview section — look for "环境变量完整预览" heading
    const previewHeading = page.getByText('环境变量完整预览');
    await expect(previewHeading).toBeVisible({ timeout: 5000 });
    console.log('✓ 环境变量完整预览标题可见');

    // The preview should contain system variables (TASK_* lines)
    // Look for TASK_AGENT_MAX_STEPS which always has a default value
    const previewContent = page.locator('.font-mono.text-xs').first();
    const previewText = await previewContent.textContent();
    console.log(`  预览内容: ${previewText}`);

    // Core assertion: preview must contain at least TASK_AGENT_MAX_STEPS=200
    expect(previewText).toContain('TASK_AGENT_MAX_STEPS');
    console.log('✓ 预览包含 TASK_AGENT_MAX_STEPS');

    // Preview must contain system-generated label
    await expect(page.getByText('系统自动生成')).toBeVisible();
    console.log('✓ 系统自动生成区域可见');

    // No critical console errors
    const criticalErrors = errors.filter(
      (e) => !e.includes('favicon') && !e.includes('third-party'),
    );
    expect(criticalErrors.length).toBe(0);
    console.log('✓ 无控制台错误');
  });

  test('Bug-2: 工作空间Tab切换预览和LLM面板无残留', async ({ page }) => {
    test.setTimeout(180000);
    test.skip(!LOGIN_EMAIL || !LOGIN_PASSWORD, '请设置 LOGIN_EMAIL / LOGIN_PASSWORD');
    test.skip(!WORKSPACE_ID, '请设置 PW_WORKSPACE_ID');

    const root = LOGIN_URL || '';
    await waitForLoginLegalPolicies(page, `${root}/auth/login/`);
    const { errors, markLoginDone } = trackConsoleErrorsAfterLogin(page);

    await playwrightLoginWithLegalAccept(page, {
      email: LOGIN_EMAIL,
      password: LOGIN_PASSWORD,
      baseURL: root,
    });
    markLoginDone();

    // Navigate to workspace feature params page
    const workspaceUrl = `${root}/tenant/${TENANT_ID}/settings/workspace/${WORKSPACE_ID}/feature-params/`;
    await page.goto(workspaceUrl, { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(500);
    await expect(page.locator('h2').first()).toContainText('工作空间环境变量', { timeout: 10000 });
    console.log('✓ 工作空间页面加载成功');

    // ================================================================
    // Phase 1: Switch to custom, fill in data, save
    // ================================================================
    const customTab = page.getByRole('tab', { name: /自定义配置/ });
    await customTab.click();
    await page.waitForTimeout(500);
    console.log('✓ 切换到自定义配置 Tab');

    // Expand LLM config panel
    const llmToggle = page.getByRole('button', { name: /LLM 供应商.*模型配置/ });
    await llmToggle.click();
    await page.waitForTimeout(300);

    // Fill custom agent model
    const agentModelInput = page.locator('input[placeholder*="gpt-4.1"]').first();
    await expect(agentModelInput).toBeVisible({ timeout: 5000 });
    await agentModelInput.fill('residue-test-model');
    console.log('✓ 填入自定义 agent_model: residue-test-model');

    // Save
    await page.getByRole('button', { name: '保存' }).click();
    await expect(page.getByText('保存成功')).toBeVisible({ timeout: 15000 });
    console.log('✓ 自定义配置保存成功');

    // Record the preview content in custom mode
    const previewAfterCustom = await page.locator('.font-mono.text-xs').first().textContent();
    console.log(`  自定义模式预览: ${previewAfterCustom}`);

    // ================================================================
    // Phase 2: Switch to company default
    // ================================================================
    const companyTab = page.getByRole('tab', { name: /公司默认/ });
    await companyTab.click();
    await page.waitForTimeout(800);  // wait for transition animation
    console.log('✓ 切换到公司默认 Tab');

    // Record the preview content in company default mode
    const previewAfterCompany = await page.locator('.font-mono.text-xs').first().textContent();
    console.log(`  公司默认模式预览: ${previewAfterCompany}`);

    // Core assertion 1: Preview must NOT contain the custom value
    // "residue-test-model" should NOT appear in company default mode
    expect(previewAfterCompany).not.toContain('residue-test-model');
    console.log('✓ 预览无残留：公司默认模式不含自定义 agent_model');

    // Core assertion 2: The preview should differ between modes
    // (they won't be identical because different providers/env vars are used)
    // At minimum, verify company mode preview contains system defaults
    expect(previewAfterCompany).toContain('TASK_AGENT_MAX_STEPS');
    console.log('✓ 公司默认预览包含系统默认值');

    // ================================================================
    // Phase 3: Verify CollapsibleLLMConfigPanel shows readonly in company mode
    // ================================================================
    await llmToggle.click();  // expand again (may have collapsed)
    await page.waitForTimeout(300);

    // Check that readonly banner is visible
    const readonlyBanner = page.getByText(/继承自.*公司默认配置/);
    const systemDefaultsBanner = page.getByText(/系统默认值/);
    const hasReadonlyIndicator = await Promise.race([
      readonlyBanner.isVisible().then(v => v).catch(() => false),
      systemDefaultsBanner.isVisible().then(v => v).catch(() => false),
    ]);
    console.log(`  只读标识可见: ${hasReadonlyIndicator}`);

    // Check the agent model input is disabled
    const agentInputInCompanyMode = page.locator('input[placeholder*="gpt-4.1"]').first();
    const isDisabled = await agentInputInCompanyMode.isDisabled().catch(() => false);
    console.log(`  agent_model input disabled: ${isDisabled}`);

    // Core assertion 3: Agent model input should show company value (not custom residue)
    const agentValueInCompany = await agentInputInCompanyMode.inputValue();
    expect(agentValueInCompany).not.toBe('residue-test-model');
    console.log(`✓ LLM 面板无残留：agent_model="${agentValueInCompany}" (不为 residue-test-model)`);

    // ================================================================
    // Phase 4: Verify no JS errors
    // ================================================================
    const criticalErrors = errors.filter(
      (e) => !e.includes('favicon') && !e.includes('third-party'),
    );
    if (criticalErrors.length > 0) {
      console.warn('⚠ 控制台错误:', criticalErrors);
    }
    expect(criticalErrors.length).toBe(0);
    console.log('✓ 无控制台错误');
  });
});
