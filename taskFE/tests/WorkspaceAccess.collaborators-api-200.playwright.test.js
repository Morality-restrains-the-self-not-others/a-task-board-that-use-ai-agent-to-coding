// @ts-check
/**
 * E2E: 验证 workspace_collaborators API 返回 200（不出现 500 FieldError）
 *
 * 触发路径：登录 → 工作面板 → 点击「创建任务」打开 CreateTaskModal
 * 该模态框会调用 GET /api/projects/workspace-access/workspace-collaborators/tenant_id/{tid}?workspace_id={wid}
 *
 * 前提：测试账号须有非 default 的工作空间（种子数据默认提供）
 * 依赖：PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { PW_TENANT_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;

test.describe('workspace_collaborators API（workspace-access）', () => {
  test('打开创建任务模态框时 workspace_collaborators 应返回 200 而非 500', async ({ page }) => {
    const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
    const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
    test.skip(!email || !password, 'Set PLAYWRIGHT_TEST_EMAIL and PLAYWRIGHT_TEST_PASSWORD');

    // 收集 workspace_collaborators 的 GET 响应
    /** @type {Array<{url: string, status: number, body: string}>} */
    const collaboratorResponses = [];
    page.on('response', async (response) => {
      const url = response.url();
      if (url.includes('workspace-collaborators') && response.request().method() === 'GET') {
        let body = null;
        try {
          body = await response.text();
        } catch {
          // response body 可能已被消费
        }
        collaboratorResponses.push({ url, status: response.status(), body: body || '' });
      }
    });

    // 1. 登录
    await playwrightLoginWithLegalAccept(page, { email, password });
    expect(page.url().includes('/auth/login/'), '登录后应离开登录页').toBe(false);

    // 2. 导航到工作面板
    await page.goto(`/tenant/${TENANT_ID}/work-panel/`);
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // 3. 点击「创建任务」按钮 → CreateTaskModal 打开 → fetchCollaborators 触发
    const createTaskBtn = page.locator('#create-task-btn');
    await expect(createTaskBtn, '创建任务按钮应可见').toBeVisible({ timeout: 10000 });
    await createTaskBtn.click();
    await page.waitForTimeout(3000);

    // 4. 验证 API 响应
    //    如果当前工作空间是 default，API 不会被调用（前端有 guard）
    //    此时测试标记为 skip 而非 fail
    if (collaboratorResponses.length === 0) {
      console.warn(
        '[workspace_collaborators] 未捕获到 API 调用 — 可能是当前工作空间为 default，前端 guard 跳过了请求。' +
        '请在测试账号下确保至少有一个非 default 工作空间。',
      );
      // 不 fail — 未触发并非代码 bug，而是环境前提未满足
      test.skip(true, '当前工作空间为 default，跳过 API 验证');
      return;
    }

    // 验证所有 workspace_collaborators 响应均为 200（不应出现 500 FieldError）
    for (const r of collaboratorResponses) {
      const bodyPreview = (r.body || '').slice(0, 500);
      expect(
        r.status,
        `workspace_collaborators 应返回 200，实际 ${r.status}\nURL: ${r.url}\nbody: ${bodyPreview}`,
      ).toBe(200);
      expect(bodyPreview).not.toContain('FieldError');
      expect(bodyPreview).not.toContain('Cannot resolve keyword');
    }
  });
});
