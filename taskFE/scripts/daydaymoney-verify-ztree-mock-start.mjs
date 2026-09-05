/**
 * 联调核验：登录 daydaymoney → 打开带 mockStart 的任务详情 → 等待评论区 zTree 面板出现。
 *
 * 环境变量（勿写入仓库）：
 *   DAYDAYMONEY_EMAIL / DAYDAYMONEY_PASSWORD — 登录凭据
 *   DAYDAYMONEY_BASE — 站点根，默认 http://daydaymoney.com
 *   DAYDAYMONEY_TASK_URL — 完整任务详情 URL（须含 mockContainerStart=true），默认使用仓库内联调示例路径
 *
 * 运行（在 task2app/playwright 目录）：
 *   DAYDAYMONEY_EMAIL=... DAYDAYMONEY_PASSWORD=... node ./scripts/daydaymoney-verify-ztree-mock-start.mjs
 */
// @ts-check
import { chromium } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { playwrightLoginWithLegalAccept } from '../tests/playwrightLogin.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from '../tests/playwrightTenantEnv.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const BASE = (process.env.DAYDAYMONEY_BASE || 'http://daydaymoney.com').replace(/\/$/, '');
const TASK_URL =
  process.env.DAYDAYMONEY_TASK_URL ||
  `http://daydaymoney.com/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/843161273287778304/?mockContainerStart=true&github=ok`;

const email = process.env.DAYDAYMONEY_EMAIL || '';
const password = process.env.DAYDAYMONEY_PASSWORD || '';

async function main() {
  if (!email || !password) {
    console.error('请设置环境变量 DAYDAYMONEY_EMAIL 与 DAYDAYMONEY_PASSWORD');
    process.exit(1);
  }

  const browser = await chromium.launch({ channel: 'chrome', headless: !!process.env.CI });
  const page = await browser.newPage();

  try {
    await playwrightLoginWithLegalAccept(page, { email, password, baseURL: BASE });
    await page.goto(TASK_URL, { waitUntil: 'domcontentloaded', timeout: 120000 });
    await page.waitForLoadState('networkidle', { timeout: 60000 }).catch(() => {});

    const extractBtn = page.getByRole('button', { name: '提取变量' });
    if (await extractBtn.isVisible({ timeout: 8000 }).catch(() => false)) {
      await extractBtn.click();
      await page.getByText(/已从 UserData 提取|提取中|环境变量来源/).first().waitFor({ state: 'visible', timeout: 120000 }).catch(() => {});
    }

    const startContainer = page.getByRole('button', { name: '启动容器' });
    if (await startContainer.isVisible({ timeout: 5000 }).catch(() => false)) {
      const disabled = await startContainer.isDisabled().catch(() => true);
      if (!disabled) {
        await startContainer.click();
      }
    }

    const panel = page.getByTestId('comment-layer-ztree-panel');
    await panel.waitFor({ state: 'visible', timeout: 900000 });
    console.log('OK: comment-layer-ztree-panel 已可见');
  } finally {
    await browser.close();
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
