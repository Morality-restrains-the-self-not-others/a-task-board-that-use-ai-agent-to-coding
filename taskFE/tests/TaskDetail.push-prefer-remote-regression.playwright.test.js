// @ts-check
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');
const TASK_DETAIL_URL =
  process.env.PLAYWRIGHT_TASK_DETAIL_URL ||
  `http://localhost:4000/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/840502615007272960/?accessCode=u824976301710503936`;

async function firstEnabledZtreeActionButton(ztreePanel, actionLabel) {
  const buttons = ztreePanel.getByRole('button', { name: actionLabel });
  await expect(buttons.first()).toBeVisible({ timeout: 120000 });
  const n = await buttons.count();
  for (let i = 0; i < n; i++) {
    const b = buttons.nth(i);
    if (await b.isEnabled().catch(() => false)) {
      return b;
    }
  }
  throw new Error(`zTree 内没有可点击的「${actionLabel}」按钮`);
}

test.describe('TaskDetail 推送 prefer_container_remote 回归', () => {
  test('多仓库任务推送应优先容器远端凭据', async ({ page }) => {
    test.setTimeout(360000);
    page.on('dialog', (d) => void d.accept());

    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: 'http://localhost:4000',
    });

    await page.goto(TASK_DETAIL_URL);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 180000 });

    const nameBtns = ztreePanel.locator('button.break-words.flex-1.min-w-0');
    await expect(nameBtns.first()).toBeVisible({ timeout: 120000 });
    const n = await nameBtns.count();
    let clickedLayer = false;
    for (let i = 0; i < n; i++) {
      const label = ((await nameBtns.nth(i).innerText()) || '').trim();
      if (label === '可写层' || /^可写层（/.test(label)) continue;
      await nameBtns.nth(i).click();
      clickedLayer = true;
      break;
    }
    expect(clickedLayer, '应存在至少一个可选中的层级节点').toBe(true);
    await page.waitForTimeout(500);

    const pushBtn = await firstEnabledZtreeActionButton(ztreePanel, '推送');
    const [pushResp] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 180000 },
      ),
      pushBtn.click(),
    ]);

    const rawPushPost = pushResp.request().postData();
    const pushPost = rawPushPost ? JSON.parse(rawPushPost) : {};
    expect(pushPost?.prefer_container_remote).toBe(true);
    expect(String(pushPost?.identity_id || '').trim()).toBe('');
  });
});
