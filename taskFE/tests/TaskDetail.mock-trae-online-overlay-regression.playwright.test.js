// @ts-check
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const EMAIL = process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com';
const PASSWORD = process.env.PLAYWRIGHT_TEST_PASSWORD;
test.skip(!PASSWORD, 'PASSWORD env required (no hardcoded fallback)');
const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = process.env.PLAYWRIGHT_TASK_ID || '829542111324585984';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  process.env.PLAYWRIGHT_TASK_DETAIL_PATH ||
    `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/829542111324585984/?accessCode=u824976301710503936`
);
const PROMPT = '用 java 撰写一个hello world程序';

async function login(page) {
  await page.goto('/auth/login/');
  await page.waitForLoadState('domcontentloaded');
  const emailTab = page.locator('text=邮箱').or(page.locator('button:has-text("邮箱")')).first();
  if (await emailTab.isVisible().catch(() => false)) {
    await emailTab.click();
    await page.waitForTimeout(300);
  }
  await page.locator('#email').fill(EMAIL);
  await page.locator('#password').fill(PASSWORD);
  await page.locator('form').first().evaluate((form) => form.requestSubmit());
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(1500);
}

async function waitContainerPageUrl(page) {
  /** @type {string | null} */
  let containerPageUrl = null;
  await expect
    .poll(
      async () => {
        const ret = await page.evaluate(
          async ([tenantId, taskId]) => {
            const path = `/api/cloud/compute/container-task-ui-context/tenant_id/${tenantId}/?task_id=${encodeURIComponent(taskId)}`;
            const headers = { Accept: 'application/json' };
            const tok = localStorage.getItem('authToken');
            if (tok) headers.Authorization = `Token ${tok}`;
            const r = await fetch(path, { credentials: 'include', headers });
            const data = await r.json().catch(() => ({}));
            return { status: r.status, u: data.container_page_url || '' };
          },
          [TENANT_ID, TASK_ID],
        );
        if (ret.status !== 200) return false;
        const u = (ret.u || '').trim();
        if (!u) return false;
        containerPageUrl = u;
        return true;
      },
      { timeout: 180000, message: '等待容器 UI 地址就绪' },
    )
    .toBe(true);
  return containerPageUrl;
}

test('模拟启动容器后发送指令，不再出现 overlay 挂载报错', async ({ page }) => {
  test.skip(process.env.PRE_COMMIT === '1', 'pre-commit 不跑 10min 级 Docker/online 全栈回归，请本地或 CI 单独执行');
  test.setTimeout(10 * 60 * 1000);
  await login(page);

  await page.goto(TASK_DETAIL_PATH);
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(1200);

  const startBtn = page.getByRole('button', { name: /模拟启动（onlineService/ });
  await expect(startBtn).toBeVisible({ timeout: 30000 });
  await startBtn.click();

  await expect(
    page.locator('div.font-mono.whitespace-pre-wrap').filter({ hasText: /服务已在 http:\/\/127\.0\.0\.1:/ }),
  ).toBeVisible({ timeout: 600000 });

  const containerUrl = await waitContainerPageUrl(page);
  expect(containerUrl && containerUrl.includes('/ui/')).toBe(true);

  const uiPage = await page.context().newPage();
  await uiPage.goto(containerUrl, { waitUntil: 'domcontentloaded', timeout: 120000 });
  await uiPage.waitForSelector('#btnRefresh', { timeout: 30000 });
  await uiPage.locator('#btnRefresh').click();

  await uiPage.waitForFunction(
    () => {
      // @ts-ignore
      if (!window.jQuery || !window.jQuery.fn || !window.jQuery.fn.zTree) return false;
      // @ts-ignore
      const z = window.jQuery.fn.zTree.getZTreeObj('ztree_layer_graph');
      if (!z) return false;
      const nodes = z.transformToArray(z.getNodes()) || [];
      return nodes.some((n) => String(n.id || '').startsWith('__layer__:'));
    },
    { timeout: 120000 },
  );

  await uiPage.evaluate(() => {
    // @ts-ignore
    const z = window.jQuery.fn.zTree.getZTreeObj('ztree_layer_graph');
    const nodes = z.transformToArray(z.getNodes()) || [];
    const layers = nodes.filter((n) => String(n.id || '').startsWith('__layer__:'));
    if (!layers.length) throw new Error('未找到层级节点');
    for (const ln of layers) {
      z.selectNode(ln, false, true);
      if (typeof ln.tId === 'string') {
        const a = document.getElementById(`${ln.tId}_a`);
        if (a) a.dispatchEvent(new MouseEvent('click', { bubbles: true }));
      }
      const bar = document.getElementById('layerRelationActions');
      const hasCreate =
        bar &&
        Array.from(bar.querySelectorAll('button')).some((b) =>
          (b.textContent || '').includes('创建并执行'),
        );
      if (hasCreate) return;
    }
    throw new Error('未找到可叠建新指令的层级节点（可能均在运行中）');
  });

  const actions = uiPage.locator('#layerRelationActions');
  await expect(actions.locator('textarea').first()).toBeVisible({ timeout: 20000 });
  await actions.locator('textarea').first().fill(PROMPT);
  await actions.locator('select').first().selectOption('trae');

  const createBtn = actions.getByRole('button', { name: '创建并执行' }).first();
  await expect(createBtn).toBeVisible({ timeout: 10000 });
  await createBtn.click();

  const jobCard = uiPage.locator('.job-card').first();
  await expect(jobCard).toBeVisible({ timeout: 120000 });
  await expect(jobCard.locator('.status.completed, .status.failed, .status.interrupted')).toBeVisible({
    timeout: 420000,
  });

  const out = (await jobCard.locator('pre.out').first().innerText().catch(() => '')) || '';
  expect(out).not.toContain('overlay prepare failed');
  expect(out).not.toContain('returned non-zero exit status 32');
  expect(out).not.toContain("no .trajectories/trajectory_*.json under layer");

  await uiPage.close();
});

