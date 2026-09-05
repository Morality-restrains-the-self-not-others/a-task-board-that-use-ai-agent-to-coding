// @ts-check
/**
 * 功能参数设置页：feature-params API 应返回 200，且不出现数据库列名错误。
 *
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;

test.describe('WorkspaceSettings 功能参数', () => {
  test('打开功能参数页时 feature-params API 返回 200', async ({ page }) => {
    const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
    const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
    test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD');

    await playwrightLoginWithLegalAccept(page, { email, password });

    const featureParamsResponsePromise = page.waitForResponse(
      (response) =>
        response.url().includes(`/api/cloud/feature-params/tenant_id/${TENANT_ID}`)
        && response.request().method() === 'GET',
      { timeout: 30000 },
    );

    await page.goto(`/tenant/${TENANT_ID}/settings/feature-params/`);
    await page.waitForLoadState('domcontentloaded');

    const featureParamsResponse = await featureParamsResponsePromise;
    expect(featureParamsResponse.status(), `feature-params status: ${featureParamsResponse.status()}`).toBe(200);

    const body = await featureParamsResponse.json();
    expect(body).toHaveProperty('data');
    expect(body.data).toMatchObject({
      providers: expect.any(Array),
      agent_model: expect.any(String),
      agent_model_provider: expect.any(String),
      agent_max_steps: expect.any(String),
      summary_model: expect.any(String),
      summary_model_provider: expect.any(String),
      env_preview: expect.objectContaining({
        TASK_AGENT_MAX_STEPS: expect.any(String),
        TASK_LLM_PROVIDERS_JSON: expect.any(String),
      }),
    });

    await expect(page.getByRole('heading', { name: '环境变量预览' })).toBeVisible();
    const preview = page.locator('textarea[readonly]').last();
    await expect(preview).toHaveValue(/TASK_AGENT_MAX_STEPS=/);
    await expect(preview).toHaveValue(/TASK_LLM_PROVIDERS_JSON=/);
    await expect(page.getByText('数据库错误', { exact: false })).toHaveCount(0);
    await expect(page.getByText('no such column', { exact: false })).toHaveCount(0);
    await expect(page.getByText('加载失败', { exact: false })).toHaveCount(0);
  });
});
