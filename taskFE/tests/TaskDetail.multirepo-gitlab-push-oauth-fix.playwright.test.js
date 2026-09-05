// @ts-check
/**
 * 验收：多仓 prefer_container_remote 推送不再因 GitLab HTTPS 无凭证报
 * `could not read Username ... terminal prompts disabled`。
 *
 * 页面：task_12939023414154091865（somanyad + somanyad-emailD）
 * 前置：taskCloudService prepare 已能换票；容器 onlineServiceJS 支持 oauth-access-push。
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || 'task_12939023414154091865';
const SITE_BASE = (process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:4000').replace(/\/$/, '');
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');

const TASK_DETAIL_URL =
  process.env.PLAYWRIGHT_TASK_DETAIL_URL ||
  `${SITE_BASE}/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/`;

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

test.describe('TaskDetail 多仓 GitLab push OAuth 修复验收', () => {
  test('推送不应再报 terminal prompts disabled / could not read Username', async ({ page }) => {
    test.skip(process.env.PRE_COMMIT === '1', 'pre-commit 不跑长链路联调');
    test.setTimeout(600000);

    page.on('dialog', (d) => void d.accept());

    await playwrightLoginWithLegalAccept(page, {
      email: EMAIL,
      password: PASSWORD,
      baseURL: SITE_BASE,
    });

    await page.goto(TASK_DETAIL_URL);
    await page.waitForLoadState('domcontentloaded');

    const ztreePanel = page.getByTestId('comment-layer-ztree-panel');
    await expect(ztreePanel).toBeVisible({ timeout: 180000 });

    // 选中可写层（跳过「可写层」标题行）
    const nameBtns = ztreePanel.locator('button.break-words.flex-1.min-w-0');
    await expect(nameBtns.first()).toBeVisible({ timeout: 120000 });
    const n = await nameBtns.count();
    let clicked = false;
    for (let i = 0; i < n; i++) {
      const label = ((await nameBtns.nth(i).innerText()) || '').trim();
      if (label === '可写层' || /^可写层（/.test(label)) continue;
      await nameBtns.nth(i).click();
      clicked = true;
      break;
    }
    expect(clicked, '应存在可选中的层级节点').toBe(true);

    // 若已有提交按钮可用则先提交（幂等）；否则直接推送
    const submitCandidates = ztreePanel.getByRole('button', { name: '提交' });
    if ((await submitCandidates.count()) > 0) {
      for (let i = 0; i < (await submitCandidates.count()); i++) {
        const b = submitCandidates.nth(i);
        if (await b.isEnabled().catch(() => false)) {
          await b.click().catch(() => {});
          await page.waitForTimeout(2000);
          break;
        }
      }
    }

    const pushBtn = await firstEnabledZtreeActionButton(ztreePanel, '推送');
    const [pushResp] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes('container-layer-git-push') && r.request().method() === 'POST',
        { timeout: 180000 },
      ),
      pushBtn.click(),
    ]);

    const rawPost = pushResp.request().postData() || '{}';
    const pushPost = JSON.parse(rawPost);
    expect(pushPost.prefer_container_remote).toBe(true);

    const status = pushResp.status();
    const bodyText = await pushResp.text().catch(() => '');
    let body = {};
    try {
      body = JSON.parse(bodyText);
    } catch {
      body = { detail: bodyText };
    }
    const detail = String(body.detail || bodyText || '');

    expect(
      detail,
      `推送不应再因缺少 HTTPS 凭据失败；status=${status} detail=${detail.slice(0, 400)}`,
    ).not.toMatch(/could not read Username|terminal prompts disabled/i);

    // 成功 2xx，或业务可读错误（如权限/分支冲突），但不得再是凭据提示禁用
    if (status >= 400) {
      console.warn('[push-verify] non-2xx after OAuth fix:', status, detail.slice(0, 500));
    }
    expect(status, `unexpected empty/gateway error: ${detail.slice(0, 300)}`).toBeLessThan(500);
  });
});
