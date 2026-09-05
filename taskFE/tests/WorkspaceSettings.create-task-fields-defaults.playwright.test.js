// @ts-check
/**
 * 回归：创建工作空间面板创建任务可选字段默认值与启用后效果。
 * - code_lang 默认关闭（checkbox 未勾选）
 * - structured_fields 默认关闭
 * - 启用后在工作面板创建任务中显示对应控件
 *
 * 纯 mock，无真实账号依赖。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';
const WS_ID = 'ws-fields-default';

// 保存字段设置的状态，供跨页面 mock 共享
let savedFieldSettings = null;

test.describe('WorkspaceSettings 创建任务字段默认值', () => {
  test('code_lang 和 structured_fields 默认关闭，开启后在创建任务中显示', async ({ page }) => {
    test.setTimeout(180000);

    // 字段设置 API mock 工厂
    function setupApiMocks(page) {
      // /api/user/{id}/accounts/users/me/
      return page.route(`**/api/accounts/users/me/**`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: USER_ID,
            username: 'e2e',
            current_company: { id: TENANT_ID, name: 'E2E Tenant' },
            companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
            current_workspace: { id: WS_ID, name: '字段测试空间' },
          }),
        });
      });
    }

    await page.context().addCookies([
      { name: 'userId', value: USER_ID, url: BASE_URL },
      { name: 'sessionid', value: 'e2e-session-create-fields', url: BASE_URL },
      { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
    ]);

    // 设置 work-panel, progress-system, machine-policy 等 API mock
    // 先设置通用的 mock 拦截
    await page.route(/\/api\/.*/, async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      // 用户信息
      if (url.includes(`/api/accounts/users/me/`)) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: USER_ID,
            username: 'e2e',
            current_company: { id: TENANT_ID, name: 'E2E Tenant' },
            companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
            current_workspace: { id: WS_ID, name: '字段测试空间' },
          }),
        });
        return;
      }

      // 工作空间列表
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}`) && method === 'GET' && !url.includes('/create-task-field-settings') && !url.includes('/task-kind-options') && !url.includes('/code-lang-options')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: WS_ID, name: '字段测试空间', is_current: true, is_default: true }]),
        });
        return;
      }

      // 创建任务字段设置 — GET 返回默认（无持久化配置）
      if (url.includes('/create-task-field-settings/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: savedFieldSettings
            ? JSON.stringify({ fields: savedFieldSettings })
            : JSON.stringify({ fields: null }),
        });
        return;
      }

      // 创建任务字段设置 — PUT 保存
      if (url.includes('/create-task-field-settings/') && method === 'PUT') {
        const body = route.request().postDataJSON() || {};
        savedFieldSettings = body.fields || {};
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ fields: savedFieldSettings }),
        });
        return;
      }

      // task-kind-options / code-lang-options
      if (url.includes('/task-kind-options/') || url.includes('/code-lang-options/')) {
        if (method === 'GET') {
          await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ options: [] }) });
          return;
        }
        if (method === 'PUT') {
          await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ options: [] }) });
          return;
        }
      }

      // machine-policy
      if (url.includes('workspace-machine-policy')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', idle_recycle_minutes: 30, prefer_idle_reuse: true, enabled_authorization_ids: [] }) });
        return;
      }

      // progress-system
      if (url.includes('progress-system')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: [] }) });
        return;
      }

      // work-panel-filters
      if (url.includes('work-panel-filters')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', version: 1 }) });
        return;
      }

      // deliverable-system
      if (url.includes('deliverable-system') || url.includes('manage-deliverable')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ current_deliverable_objs: [] }) });
        return;
      }

      // workspace-machine-summary
      if (url.includes('workspace-machine-summary') || url.includes('workspace-runtime-indicators')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success' }) });
        return;
      }

      // workspace-collaborators / workspace-permissions
      if (url.includes('workspace-collaborators') || url.includes('workspace-permissions')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      // work-panel todos
      if (url.includes('todos')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      // tasks/task-panel (create-task)
      if (url.includes('tasks') || url.includes('create-task')) {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
        return;
      }

      // catch-all
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });

    // === 第一步：打开 settings/task-panel 检查默认值 ===
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/settings/task-panel/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2500);

    // 点击"创建字段"按钮打开模态
    const openFieldSettingsBtn = page.getByTestId('open-create-task-field-settings').first();
    await expect(openFieldSettingsBtn).toBeVisible({ timeout: 45000 });
    await openFieldSettingsBtn.click();

    const modal = page.getByTestId('create-task-field-settings-modal');
    await expect(modal).toBeVisible({ timeout: 15000 });

    // 检查 code_lang 默认为未勾选
    const codeLangCheckbox = modal.getByTestId('create-task-field-setting-code_lang').locator('input[type="checkbox"]');
    await expect(codeLangCheckbox).toBeVisible();
    await expect(codeLangCheckbox).not.toBeChecked();

    // 检查 structured_fields 默认为未勾选
    const structuredCheckbox = modal.getByTestId('create-task-field-setting-structured_fields').locator('input[type="checkbox"]');
    await expect(structuredCheckbox).toBeVisible();
    await expect(structuredCheckbox).not.toBeChecked();

    // === 第二步：启用 code_lang 和 structured_fields 并保存 ===
    await codeLangCheckbox.check();
    await structuredCheckbox.check();
    await expect(codeLangCheckbox).toBeChecked();
    await expect(structuredCheckbox).toBeChecked();

    const saveBtn = modal.getByTestId('create-task-field-settings-save');
    await expect(saveBtn).toBeVisible();
    await saveBtn.click();

    // 等待模态关闭
    await expect(modal).toBeHidden({ timeout: 15000 });

    // === 第三步：打开 work-panel 创建任务，验证控件出现 ===
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WS_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    const createTaskBtn = page.locator('#create-task-btn');
    await expect(createTaskBtn).toBeVisible({ timeout: 45000 });
    await createTaskBtn.click();

    const createTaskModal = page.locator('#create-task-modal');
    await expect(createTaskModal).toBeVisible({ timeout: 15000 });
    await page.waitForTimeout(2000);

    // code_lang 控件应可见
    const codeLangField = createTaskModal.locator('[data-testid="create-task-field-codeLang"]');
    await expect(codeLangField).toBeVisible({ timeout: 5000 });

    // structured_fields 控件应可见
    const structuredFieldsContainer = createTaskModal.locator('[data-testid="create-task-structured-fields"]');
    await expect(structuredFieldsContainer).toBeVisible({ timeout: 5000 });
  });
});
