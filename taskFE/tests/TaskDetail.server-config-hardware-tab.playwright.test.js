// @ts-check
/**
 * 回归：任务详情评论区 @镜像 后「服务器硬件配置」云平台下拉应可初始化并可交互。
 * 环境与硬件由 composer 直接渲染，不再 Teleport。
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';
import { mentionInstalledImageInComment } from './helpers/commentImageMentionE2e.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;

const TASK_ID = '830423831930662912';
const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

test.describe('TaskDetail 服务器硬件配置（评论区镜像下）', () => {
  test('镜像下硬件配置云平台下拉可用（有选项且可 selectOption）', async ({ page }) => {
    await page.addInitScript(
      ({ tenantId, workspaceId, taskId, jobId }) => {
        class MockEventSource {
          static CONNECTING = 0;
          static OPEN = 1;
          static CLOSED = 2;

          constructor(url) {
            this.url = String(url || '');
            this.readyState = MockEventSource.OPEN;
            this.withCredentials = true;
            this.onmessage = null;
            this.onerror = null;
            this.onclose = null;
            this._listeners = { open: [] };

            setTimeout(() => {
              this._emitOpen();
              const expected = `/api/sse/server-startup-status/tenant_id/${tenantId}/workspace_id/${workspaceId}/task_id/${taskId}/`;
              if (!this.url.includes(expected)) return;
            }, 30);
          }

          addEventListener(type, cb) {
            if (!this._listeners[type]) this._listeners[type] = [];
            this._listeners[type].push(cb);
          }

          close() {
            this.readyState = MockEventSource.CLOSED;
            if (typeof this.onclose === 'function') {
              this.onclose();
            }
          }

          _emitOpen() {
            const ev = { type: 'open' };
            for (const cb of this._listeners.open || []) {
              try {
                cb(ev);
              } catch (_) {}
            }
          }
        }

        window.EventSource = MockEventSource;
      },
      {
        tenantId: TENANT_ID,
        workspaceId: WORKSPACE_ID,
        taskId: TASK_ID,
        jobId: JOB_ID,
      },
    );

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await page.route('**/api/**', async (route) => {
      const req = route.request();
      const url = req.url();
      const method = req.method();

      if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}${TASK_ID}/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: TASK_ID,
            title: 'Mock Task',
            description: 'Mock',
            created_at: '2026-01-01T00:00:00Z',
            created_by: { username: 'mock-user' },
            comments: [],
            ai_comments: [],
            assignees: [],
            workspace_id: WORKSPACE_ID,
            container_image_id: 'img-mock-1',
          }),
        });
        return;
      }

      if (url.includes('/mock-trae-online-log') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ log: 'mock-snapshot', in_progress: false }),
        });
        return;
      }

      if (
        url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) &&
        url.includes('/regions/') &&
        method === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([{ region_id: 'cn-hangzhou', region_name: '华东1（杭州）' }]),
        });
        return;
      }

      if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            { id: 'img-mock-1', name: 'Mock Image', version: '1.0', hardware_summary: '2C4G' },
          ]),
        });
        return;
      }

      if (
        url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/`) &&
        method === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            platforms: [
              {
                id: 'platform-mock-1',
                platform_type: 'aliyun',
                remark: 'mock',
                platform_name: '阿里云 Mock',
              },
            ],
          }),
        });
        return;
      }

      if (
        url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/cloud/platforms/default-config/`) &&
        method === 'GET'
      ) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            default_configs: [
              {
                platform_type: 'aliyun',
                remark: 'mock',
                authorization_id: 'auth-mock-1',
                config: { bandwidth_charging_mode: 'PayByTraffic' },
              },
            ],
          }),
        });
        return;
      }

      if (url.includes('/previous-server-config/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', server_config: null }),
        });
        return;
      }

      if (url.includes('/server-runtime-status/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', runtime_status: 'Stopped' }),
        });
        return;
      }

      if (url.includes('/server-content/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', content: '', http_status: 200 }),
        });
        return;
      }

      if (url.includes('/cloud/compute/server-start-history/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'success', records: [] }),
        });
        return;
      }

      if (url.includes('/cloud-platform/') && url.includes('/cloud/regions/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            regions: [{ region_id: 'cn-hangzhou', region_name: '华东1（杭州）' }],
          }),
        });
        return;
      }

      if (url.includes('/projects/workspace-access/workspace-collaborators/') && method === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
        return;
      }

      if (url.includes(`/api/projects/workspaces/tenant_id/${TENANT_ID}${WORKSPACE_ID}/progress-system/`) && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ columns: [] }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-task-ui-context/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            container_endpoint_registered: false,
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-layer-graph/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            layers_root: '/workspace/layers',
            bootstrap_layer_id: LAYER_ID,
            layers: [],
            jobs: [],
          }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-clone-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ layer_id: LAYER_ID, text: '' }),
        });
        return;
      }

      if (url.includes('/cloud/compute/container-job-execution-log/') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job: { id: JOB_ID, status: 'completed', command: 'echo', output: '' },
            steps: { steps: [] },
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '{}',
      });
    });

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    await mentionInstalledImageInComment(page, { imageText: 'Mock Image' });

    const hardwarePanel = page.getByTestId('server-hardware-config-panel');
    await expect(hardwarePanel).toBeVisible({ timeout: 30000 });

    const imageSelect = page.getByTestId('comment-composer-image-select');
    await expect(imageSelect).toBeVisible();
    const envSlot = page.getByTestId('comment-composer-env-hardware-slot');
    await expect(envSlot.getByTestId('server-image-config-card')).toBeVisible();
    await expect(envSlot.getByTestId('server-hardware-config-panel')).toBeVisible();
    await expect(envSlot.getByTestId('hardware-config-comment-bar')).toHaveCount(0);
    await expect(imageSelect.getByTestId('comment-composer-env-hardware-slot')).toHaveCount(0);
    const fpSlot = page.getByTestId('comment-composer-feature-params-slot');
    await expect(fpSlot).toBeVisible();
    const layoutOk = await page.evaluate(() => {
      const img = document.querySelector('[data-testid="comment-composer-image-select"]');
      const fp = document.querySelector('[data-testid="comment-composer-feature-params-slot"]');
      const hw = document.querySelector('[data-testid="comment-composer-env-hardware-slot"]');
      const row = document.querySelector('[data-testid="comment-composer-run-config-row"]');
      if (!img || !fp || !hw || !row) return false;
      const following = Node.DOCUMENT_POSITION_FOLLOWING;
      return img.parentElement === fp.parentElement
        && img.parentElement === row
        && (img.compareDocumentPosition(fp) & following) !== 0
        && (fp.compareDocumentPosition(hw) & following) !== 0;
    });
    expect(layoutOk).toBe(true);
    await expect(page.getByTestId('server-config-hardware-tab')).toHaveCount(0);

    const openTemp = page.getByTestId('open-temporary-hardware-config-btn');
    if (await openTemp.count()) {
      await openTemp.click();
    }

    const platformSelect = hardwarePanel.locator('#cloud-platform-select');
    await expect(platformSelect).toBeVisible({ timeout: 15000 });
    await expect(platformSelect).toBeEnabled();

    await expect.poll(async () => platformSelect.locator('option').count(), { timeout: 15000 }).toBeGreaterThan(0);

    await platformSelect.selectOption({ index: 0 });
    const value = await platformSelect.inputValue();
    expect(value).toBeTruthy();
  });
});
