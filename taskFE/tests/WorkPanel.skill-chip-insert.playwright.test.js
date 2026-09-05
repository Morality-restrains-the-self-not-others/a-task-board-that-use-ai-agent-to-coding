// @ts-check
/**
 * 回归：创建任务技能 chip 点击（OPT-20260827-018）。
 * - 打开 work-panel 创建任务弹窗
 * - 描述输入 `$镜像` 选中已安装镜像后技能 chip 可见
 * - 点击默认技能 chip → `#task-description` 值为 `$镜像 /技能`（完整 mention，非只 `/技能`）
 * - chip 仍在（镜像绑定不丢失）
 *
 * 纯 mock，无真实账号依赖；参考 WorkspaceSettings.priority-field-toggle.playwright.test.js。
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
const WS_ID = 'ws-skill-chip';

const IMAGE = {
  id: 'img-agent-dev',
  name: 'agent-dev',
  image_name: 'agent-dev',
  version: 'v1',
  image_skills: {
    version: 1,
    skills: [
      { id: 'sk_dev', name: 'dev', description: '开发' },
      { id: 'sk_debug', name: 'debug', description: '调试' },
    ],
  },
};

test('work-panel 创建任务：默认技能 chip 点击写入完整 `$镜像 /技能` 且 chip 仍在', async ({ page }) => {
  test.setTimeout(120000);

  await page.context().addCookies([
    { name: 'userId', value: USER_ID, url: BASE_URL },
    { name: 'sessionid', value: 'e2e-session-skill-chip', url: BASE_URL },
    { name: 'csrftoken', value: 'e2e-csrf', url: BASE_URL },
  ]);

  await page.route(/\/api\/.*/, async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    if (url.includes(`/api/accounts/users/me/`)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: USER_ID,
          username: 'e2e',
          has_phone: true,
          current_company: { id: TENANT_ID, name: 'E2E Tenant' },
          companies: [{ id: TENANT_ID, name: 'E2E Tenant' }],
          current_workspace: { id: WS_ID, name: '技能测试空间' },
        }),
      });
      return;
    }

    // 工作空间列表
    if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}`) && method === 'GET' && !url.includes('field-settings') && !url.includes('task-kind') && !url.includes('code-lang')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([{ id: WS_ID, name: '技能测试空间', is_current: true, is_default: true, text: '技能测试空间', value: WS_ID }]),
      });
      return;
    }

    // 已安装镜像（含 image_skills → 技能 chip 数据源）
    if (url.includes('/api/cloud/installed-images/')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([IMAGE]) });
      return;
    }

    // 创建任务字段设置（默认全开）
    if (url.includes('create-task-field-settings')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ fields: { description: true, task_kind: true, code_lang: true, structured_fields: true, project_branch: true, container_image: true, feature_params: true, priority: true, due_date: true, auto_run: true, owner: true, assignees: true } }) });
      return;
    }

    // task-kind / code-lang / progress-system / deliverable / work-panel-filters / machine-summary
    if (url.includes('task-kind-options') || url.includes('code-lang-options') || url.includes('progress-system')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ options: [] }) });
      return;
    }

    // workspace-permissions
    if (url.includes('workspace-permissions')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ allowed: true }) });
      return;
    }

    // workspace-machine-summary / workspace-runtime-indicators
    if (url.includes('workspace-machine-summary') || url.includes('workspace-runtime-indicators')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'success' }) });
      return;
    }

    // todos
    if (url.includes('todos')) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    // catch-all
    await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
  });

  await page.goto(`${BASE_URL}/tenant/${TENANT_ID}/work-panel/?workspace_id=${WS_ID}`);
  await page.waitForLoadState('domcontentloaded');

  // 打开创建任务弹窗
  const createTaskBtn = page.locator('#create-task-btn');
  await createTaskBtn.waitFor({ state: 'visible', timeout: 45000 });
  await createTaskBtn.click();
  const modal = page.locator('#create-task-modal');
  await modal.waitFor({ state: 'visible', timeout: 15000 });

  // 在描述中键入 `$agent-dev`：mention 自动命中已安装镜像并绑定 → 技能 chip 出现
  const desc = page.locator('#task-description');
  await desc.waitFor({ state: 'visible', timeout: 10000 });
  await desc.click();
  await desc.pressSequentially('$agent-dev', { delay: 20 });

  // 技能 chip 应出现（镜像已绑定）
  const chips = page.locator('[data-testid="task-description-skill-chips"] button');
  await chips.first().waitFor({ state: 'visible', timeout: 10000 });
  expect(await chips.count()).toBeGreaterThanOrEqual(1);

  // 点击默认技能 chip
  await chips.first().click();

  // 断言描述写入完整 mention `$agent-dev /dev`
  const descValue = await desc.inputValue();
  expect(descValue).toContain('$agent-dev /dev');

  // chip 仍在（镜像绑定未丢失）
  await expect(chips.first()).toBeVisible();
});
