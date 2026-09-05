// @ts-check
/**
 * OPT-20260809-020: create-task-modal 环境变量选择器禁用态（无任何可用来源）
 *
 * 覆盖目标：测试租户无公司/工作空间/个人环境变量时，打开创建任务弹窗 →
 *   1. #feature-params-source-selector 处于 disabled（featureParamsSourcesAvailable=false）
 *   2. [data-testid="feature-params-source-unavailable-hint"] 提示文案可见
 *
 * 前置条件（mock 模拟）：/api/cloud/feature-params/ 返回
 * env_var_sources_available={company:false, workspace:false}，且个人配置列表为空。
 * 基于 mock（page.route + cookies），无需真实后端或凭据。
 *
 * 运行：
 *   npx playwright test --config=playwright.config.cdp.js WorkPanel.create-task-feature-params-unavailable.playwright.test.js
 */
import { test, expect } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_SITE_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID =
  process.env.PW_WORKSPACE_ID ||
  process.env.PLAYWRIGHT_WORKSPACE_ID ||
  PW_WORKSPACE_ID;
const USER_ID = process.env.PLAYWRIGHT_TEST_USER_ID || '827923618451263488';

// ====== Mock 数据 ======

const PROGRESS_COLUMNS = [
  { id: '0', name: '待处理', order: 0 },
  { id: '1', name: '进行中', order: 1 },
  { id: '2', name: '已完成', order: 2 },
];
const DELIVERABLE_TYPES = {
  current_deliverable_objs: [{ id: '2001', name: 'Feature', order: 0 }],
};

// ====== Helpers ======

let cachedIndexHtml = '';

/** 预置认证态并 mock WorkPanel 初始化所需业务 API（结构参照 optimization-regression）。 */
async function setupCommonMocks(page) {
  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-fp-unavailable', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  if (!cachedIndexHtml) {
    const resp = await page.request.get(`${BASE_URL}/`);
    cachedIndexHtml = await resp.text();
  }

  // 绕过服务端 forward-auth 302：/tenant/ 文档导航直接返回 SPA 入口（与 companySwitcher 一致）
  await page.route('**/tenant/**', async (route) => {
    if (route.request().resourceType() === 'document') {
      await route.fulfill({ status: 200, contentType: 'text/html', body: cachedIndexHtml });
    } else {
      await route.continue();
    }
  });

  // 兜底 mock 全部 /api/：未显式覆盖的业务 API 以空对象快速响应，
  // 避免真实网关 401 触发全局跳转登录（以下具体路由注册在后，按 LIFO 优先匹配）
  await page.route('**/api/**', async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.route(`**/api/accounts/users/me/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify({
        id: USER_ID, username: 'e2e',
        current_company: { id: TENANT_ID, name: 'E2E' },
        companies: [{ id: TENANT_ID, name: 'E2E' }],
        current_workspace: { id: WORKSPACE_ID, name: 'E2E WS' },
      }),
    });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify([{ id: WORKSPACE_ID, name: 'E2E WS', is_current: true, is_default: true }]),
    });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ columns: PROGRESS_COLUMNS }) });
  });

  await page.route(`**/manage-deliverable-system/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(DELIVERABLE_TYPES) });
  });

  await page.route(`**/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/work-panel-filters/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ bars: [] }) });
  });

  await page.route('**/cloud/compute/workspace-runtime-indicators/**', async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', indicators: [] }) });
  });

  await page.route('**/cloud/compute/workspace-machine-summary/**', async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success', started_count: 0, idle_count: 0, busy_count: 0 }) });
  });

  await page.route(`**/api/projects/tenant_id/${TENANT_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });

  await page.route(`**/api/cloud/installed-images/tenant_id/${TENANT_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });

  await page.route(`**/projects/workspace-access/workspace-collaborators/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });

  await page.route(`**/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (r) => {
    if (r.request().method() !== 'GET') { await r.continue(); return; }
    await r.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });
}

/**
 * 无任何可用环境变量来源。
 * 同时兼容两种消费结构：
 *  - 旧 dist：body.data.extra_env_vars / company_config / workspace_config（数组 keyed vars）
 *  - 新源码（OPT-20260809-019）：data.env_var_sources_available（view=summary 响应；兼容顶层 flag）
 * 两者任一命中都会使 featureParamsSourcesAvailable=false → 选择器 disabled。
 */
async function setupFeatureParamsUnavailableMock(page) {
  await page.route(`**/api/cloud/feature-params/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify({
        data: {
          extra_env_vars: [],
          company_config: { extra_env_vars: [] },
          workspace_config: { extra_env_vars: [] },
        },
        env_var_sources_available: { company: false, workspace: false },
      }),
    });
  });

  await page.route(`**/api/personal/feature-params-configs/**`, async (r) => {
    await r.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ configs: [] }) });
  });
}

// ====== Tests ======

test.describe('OPT-20260809-020: create-task-modal 环境变量选择器禁用态', () => {
  test('无任何环境变量来源时，选择器 disabled 且 unavailable hint 可见', async ({ page }) => {
    await setupCommonMocks(page);
    await setupFeatureParamsUnavailableMock(page);

    await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WORKSPACE_ID}`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    // 打开创建任务弹窗
    const createBtn = page.locator('#create-task-btn');
    await expect(createBtn).toBeVisible({ timeout: 15000 });
    await createBtn.click();
    await expect(page.locator('#create-task-modal')).toBeVisible({ timeout: 15000 });

    // 断言环境变量选择器处于 disabled（可用性拉取完成 → sourcesAvailable=false）
    const selector = page.locator('#feature-params-source-selector');
    await expect(selector).toBeDisabled({ timeout: 15000 });

    // 断言 unavailable 提示文案可见
    const hint = page.locator('[data-testid="feature-params-source-unavailable-hint"]');
    await expect(hint).toBeVisible({ timeout: 5000 });
    await expect(hint).toContainText('暂无可用智能体资源配置');

    // 反向确认选择器仍渲染（未因不可用被移除），仅是禁用
    await expect(selector).toBeVisible();
  });
});
