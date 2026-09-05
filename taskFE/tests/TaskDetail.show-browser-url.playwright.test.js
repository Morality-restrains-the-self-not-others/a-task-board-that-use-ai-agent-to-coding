// @ts-check
/**
 * 手工演示：启动 webServer 后以有界面浏览器打开指定任务详情 URL，便于观察加载全流程。
 * 运行示例（项目 app 目录下）：
 *   （工作目录 task2app/playwright）npx playwright test -c playwright.config.js tests/TaskDetail.show-browser-url.playwright.test.js --headed --slow-mo=300
 */
import { test } from '@playwright/test';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

test.use({
  launchOptions: {
    slowMo: 300,
  },
});

const TASK_DETAIL_URL =
  `http://localhost:4000/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/839037065709281280/?accessCode=u824976301710503936`;

test.describe.configure({ timeout: 600_000 });

test('headed 展示任务详情页加载', async ({ page }) => {
  await page.goto(TASK_DETAIL_URL, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('networkidle', { timeout: 180_000 }).catch(() => {});
  // 保持窗口一段时间，便于观察页面与后续前端请求
  await page.waitForTimeout(120_000);
});
