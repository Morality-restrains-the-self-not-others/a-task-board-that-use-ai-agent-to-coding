/**
 * Playwright API 核验：relay 本地模式启动后，bootstrap 日志含工作分支切换，
 * 且层内各仓当前分支 = 任务工作分支。
 *
 * 说明：selected_image（Docker 镜像）模式需先重建/推送含本修复的镜像；
 * 本用例走无 installed_image_id 的本地 run.sh，直接验证源码修复。
 *
 * 运行：
 *   PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 *   PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 *   bash tests/TaskDetail.relay-bootstrap-work-branch.playwright.test.js.sh
 *   # 或 RUN_PW_TEST=1 跑本文件
 */
import { test, expect } from '@playwright/test';
import { spawnSync } from 'child_process';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || '850256677331562496';
const WORKSPACE_ID = process.env.PLAYWRIGHT_WORKSPACE_ID || '861623708318031872';
const TASK_ID = process.env.PLAYWRIGHT_RELAY_TASK_ID || 'task_12953905731855947865';
const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');
const WORK_BRANCH =
  process.env.PLAYWRIGHT_EXPECT_WORK_BRANCH ||
  'feature/2026-07-12_______daydaymoneytask_12953905731855947865_runIt';

const RELAY_BASE = `/api/cloud/compute/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/task_id/${TASK_ID}relay-to-trae`;

test.describe('TaskDetail relay bootstrap work branch checkout', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑 relay 全栈 E2E');

  test('本地模式启动后两仓切到工作分支', async ({ page }) => {
    test.setTimeout(420_000);

    const loginData = await loginViaGatewayApi(page, {
      email: EMAIL,
      password: PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });
    const token = String(loginData?.token || '').trim();
    expect(token).toBeTruthy();

    const hdr = {
      Authorization: `Token ${token}`,
      Accept: 'application/json',
      'Content-Type': 'application/json',
      Origin: SITE,
    };

    await page.request.post(`${GATEWAY}${RELAY_BASE}/stop/`, {
      headers: hdr,
      data: { task_id: TASK_ID },
    });
    await page.waitForTimeout(2000);

    // 不传 installed_image_id → 本地 onlineServiceJS/run.sh（含本仓库修复）
    const start = await page.request.post(`${GATEWAY}${RELAY_BASE}/start/`, {
      headers: hdr,
      data: {
        tenant_id: TENANT_ID,
        workspace_id: WORKSPACE_ID,
        task_id: TASK_ID,
        env: {
          ACCESS_TOKEN: '__TASK2APP_ACCESS_TOKEN__',
          TASK_API_ENDPOINT_ORIGIN: SITE,
          BUSINESS_API_ENDPOINT_ORIGIN: SITE,
        },
      },
    });
    expect(start.status(), await start.text()).toBeLessThan(300);

    let logs = '';
    await expect
      .poll(
        async () => {
          const st = await page.request.get(
            `${GATEWAY}${RELAY_BASE}/status/?task_id=${TASK_ID}&cursor=0`,
            { headers: hdr },
          );
          const body = await st.json();
          logs = Array.isArray(body.logs) ? body.logs.join('\n') : '';
          if (/BOOTSTRAP_FAILED/i.test(logs)) return 'failed';
          if (
            /BOOTSTRAP_PHASE=work_branch_checkout_done/i.test(logs) &&
            /BOOTSTRAP_COMPLETE/i.test(logs)
          ) {
            return 'ok';
          }
          return 'pending';
        },
        { timeout: 300_000, message: '应出现 work_branch_checkout_done + BOOTSTRAP_COMPLETE' },
      )
      .toBe('ok');

    expect(logs).toMatch(/\[work-branch-checkout\] ok /);
    expect(logs).toMatch(/work_branch_checkout_done/);

    const check = spawnSync(
      'bash',
      [
        '-lc',
        `
root=/tmp/ram-work/trae-agent/onlineProject_state
find "$root/layers" -name .git -type d 2>/dev/null | while read g; do
  d=$(dirname "$g")
  git -C "$d" rev-parse --abbrev-ref HEAD
done
`,
      ],
      { encoding: 'utf8' },
    );
    const heads = String(check.stdout || '')
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);
    expect(heads.length, `heads=${JSON.stringify(heads)}`).toBeGreaterThanOrEqual(2);
    for (const h of heads) {
      expect(h).toBe(WORK_BRANCH);
    }
  });
});
