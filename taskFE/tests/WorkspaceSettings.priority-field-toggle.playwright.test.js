// @ts-check
/**
 * 回归：settings/task-panel 创建任务可选字段中优先级开关对工作面板创建任务的影响。
 * - 关闭优先级 → 创建任务中 #task-priority 不存在
 * - 重新开启 → #task-priority 出现
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
const WS_ID = 'ws-priority-toggle';

// 共享字段设置状态
let currentFieldSettings = null;

test.describe('WorkspaceSettings 优先级字段开关', () => {
  async function setupApiMocks(page, initialFieldSettings) {
    // 初始化字段设置（所有字段默认开启）
    currentFieldSettings = initialFieldSettings || { priority: true };

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
            current_workspace: { id: WS_ID, name: '优先级测试空间' },
          }),
        });
        return;
      }

      // 工作空间列表
      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}`) && method === 'GET' && !url.includes('/create-task-field-settings') && !url.includes('/task-kind-options') && !url.includes('/code-lang-options')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ id: WS_ID, name: '优先级测试空间', is_current: true, is_default: true, text: '优先级测试空间', value: WS_ID }]),
        });
        return;
      }

      // 创建任务字段设置
      if (url.includes('/create-task-field-settings/')) {
        if (method === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ fields: currentFieldSettings }),
          });
          return;
        }
        if (method === 'PUT') {
          const body = route.request().postDataJSON() || {};
          currentFieldSettings = body.fields || {};
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ fields: currentFieldSettings }),
          });
          return;
        }
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

      // workspace-machine-summary / workspace-runtime-indicators
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

      // catch-all
      await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
    });
  }

  test('禁用优先级字段后工作面板创建任务无优先级字段，重新启用后出现', async ({ page }) => {
    test.setTimeout(180000);

    // 初始状态：所有字段开启（含 priority）
    await page.context().addCookies([
      { name: 'userId', value: USER_ID, url: BASE_URL },
      { name: 'sessionid', value: 'e2e-session-priority', url: BASE_URL },
      { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
    ]);

    const allFieldsEnabled = {
      description: true, task_kind: true, code_lang: true, structured_fields: true,
      project_branch: true, container_image: true, feature_params: true,
      priority: true, due_date: true, auto_run: true, owner: true, assignees: true,
    };

    await setupApiMocks(page, allFieldsEnabled);

    // === 第 1 步：验证当前 work-panel 创建任务中优先级字段可见 ===
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WS_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    const createTaskBtn = page.locator('#create-task-btn');
    await expect(createTaskBtn).toBeVisible({ timeout: 45000 });
    await createTaskBtn.click();

    const createTaskModal = page.locator('#create-task-modal');
    await expect(createTaskModal).toBeVisible({ timeout: 15000 });
    await page.waitForTimeout(2000);

    // 优先级字段应存在
    const prioritySelect = createTaskModal.locator('#task-priority');
    await expect(prioritySelect).toBeVisible({ timeout: 5000 });

    // 关闭弹窗
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);

    // === 第 2 步：打开 settings/task-panel，禁用 priority ===
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/settings/task-panel/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2500);

    const openFieldSettingsBtn = page.getByTestId('open-create-task-field-settings').first();
    await expect(openFieldSettingsBtn).toBeVisible({ timeout: 45000 });
    await openFieldSettingsBtn.click();

    const modal = page.getByTestId('create-task-field-settings-modal');
    await expect(modal).toBeVisible({ timeout: 15000 });

    // 找到 priority 勾选框并取消勾选
    const priorityCheckbox = modal.getByTestId('create-task-field-setting-priority').locator('input[type="checkbox"]');
    await expect(priorityCheckbox).toBeVisible();
    await expect(priorityCheckbox).toBeChecked();

    await priorityCheckbox.uncheck();
    await expect(priorityCheckbox).not.toBeChecked();

    // 保存
    const saveBtn = modal.getByTestId('create-task-field-settings-save');
    await expect(saveBtn).toBeVisible();
    await saveBtn.click();
    await expect(modal).toBeHidden({ timeout: 15000 });

    // === 第 3 步：再次去 work-panel，确认 #task-priority 不存在 ===
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WS_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    await page.locator('#create-task-btn').click();
    await expect(page.locator('#create-task-modal')).toBeVisible({ timeout: 15000 });
    await page.waitForTimeout(2000);

    // 优先级字段不应存在
    await expect(page.locator('#task-priority')).toHaveCount(0, { timeout: 5000 });

    // 关闭弹窗
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);

    // === 第 4 步：返回 settings/task-panel 重新启用 priority ===
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/settings/task-panel/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2500);

    await page.getByTestId('open-create-task-field-settings').first().click();
    await expect(page.getByTestId('create-task-field-settings-modal')).toBeVisible({ timeout: 15000 });

    // 重新勾选 priority
    const priorityCheckboxAgain = page.getByTestId('create-task-field-setting-priority').locator('input[type="checkbox"]');
    await expect(priorityCheckboxAgain).not.toBeChecked();
    await priorityCheckboxAgain.check();
    await expect(priorityCheckboxAgain).toBeChecked();

    await page.getByTestId('create-task-field-settings-save').click();
    await expect(page.getByTestId('create-task-field-settings-modal')).toBeHidden({ timeout: 15000 });

    // === 第 5 步：再去 work-panel 验证 #task-priority 重新出现 ===
    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WS_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    await page.locator('#create-task-btn').click();
    await expect(page.locator('#create-task-modal')).toBeVisible({ timeout: 15000 });
    await page.waitForTimeout(2000);

    await expect(page.locator('#task-priority')).toBeVisible({ timeout: 5000 });
  });
});
