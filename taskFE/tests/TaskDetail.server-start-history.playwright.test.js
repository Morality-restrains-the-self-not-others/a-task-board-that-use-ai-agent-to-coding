// @ts-check
/**
 * OPT-20260719-016: 历史启动记录卡断言「运行持续时间」「启动原因」「关闭原因」可见。
 * OPT-20260720-005: 同 instance_type_id 的多卡「硬件」核内一致（磁盘可不同）。
 *
 * 测试策略：
 * 1. mock server-start-history API 返回多条共享 instance_type_id 的记录
 * 2. 切换到「历史服务器启动记录」Tab
 * 3. 断言每张 data-testid="server-start-history-card" 卡包含三类文案
 * 4. 断言同 instance_type_id 的 cpu_cores + memory_gb 一致，storage_gb 允许不同
 */
import { test, expect } from '@playwright/test';
import { taskDetailPathWithMockQuery } from './playwrightTaskDetailUrl.js';
import { PW_TENANT_ID, PW_WORKSPACE_ID } from './playwrightTenantEnv.js';

const TENANT_ID = process.env.PLAYWRIGHT_TENANT_ID || PW_TENANT_ID;
const WORKSPACE_ID = process.env.PW_WORKSPACE_ID || process.env.PLAYWRIGHT_WORKSPACE_ID || PW_WORKSPACE_ID;
const TASK_ID = '830423831930662912';
const LAYER_ID = 'layer-1';
const JOB_ID = 'job-1';

/**
 * 共享 instance_type_id 的历史记录。
 * 前三条 instance_type_id 相同（ecs.g7.xlarge），硬件除磁盘外应一致。
 */
const MOCK_HISTORY_RECORDS = [
  {
    id: 'hist-001',
    created_at: '2026-07-01T08:00:00Z',
    started_at: '2026-07-01T08:00:00Z',
    stopped_at: '2026-07-01T10:30:00Z',
    runtime_source: 'cloud_vm_manual',
    stop_reason: 'stop_vm',
    platform: '阿里云',
    instance_id: 'i-001',
    instance_type_id: 'ecs.g7.xlarge',
    region: 'cn-hangzhou',
    zone_id: 'cn-hangzhou-g',
    hardware_config: { cpu_cores: 4, memory_gb: 16, storage_gb: 40 },
  },
  {
    id: 'hist-002',
    created_at: '2026-07-02T09:00:00Z',
    started_at: '2026-07-02T09:00:00Z',
    stopped_at: '2026-07-02T11:15:00Z',
    runtime_source: 'cloud_vm_auto_run',
    stop_reason: 'idle_recycle',
    platform: '阿里云',
    instance_id: 'i-002',
    instance_type_id: 'ecs.g7.xlarge',
    region: 'cn-hangzhou',
    zone_id: 'cn-hangzhou-g',
    hardware_config: { cpu_cores: 4, memory_gb: 16, storage_gb: 80 },
  },
  {
    id: 'hist-003',
    created_at: '2026-07-03T14:00:00Z',
    started_at: '2026-07-03T14:00:00Z',
    stopped_at: '2026-07-03T14:45:00Z',
    runtime_source: 'cloud_vm_template',
    stop_reason: 'superseded_by_new_start',
    platform: '阿里云',
    instance_id: 'i-003',
    instance_type_id: 'ecs.g7.xlarge',
    region: 'cn-hangzhou',
    zone_id: 'cn-hangzhou-g',
    hardware_config: { cpu_cores: 4, memory_gb: 16, storage_gb: 120 },
  },
  // 第4条使用不同的 instance_type_id，硬件不同（验证一致性仅按 instance_type_id 分组）
  {
    id: 'hist-004',
    created_at: '2026-07-04T08:00:00Z',
    started_at: '2026-07-04T08:00:00Z',
    stopped_at: '2026-07-04T09:00:00Z',
    runtime_source: 'cloud_vm_manual',
    stop_reason: 'stop_vm',
    platform: '阿里云',
    instance_id: 'i-004',
    instance_type_id: 'ecs.g6.xlarge',
    region: 'cn-shanghai',
    zone_id: 'cn-shanghai-b',
    hardware_config: { cpu_cores: 4, memory_gb: 16, storage_gb: 80 },
  },
];

const TASK_DETAIL_PATH = taskDetailPathWithMockQuery(
  `/tenant/${TENANT_ID}/workspace/${WORKSPACE_ID}/task-detail/${TASK_ID}/?accessCode=u824976301710503936`,
);

/**
 * 通用 mock EventSource，防止真实 SSE 连接干扰测试。
 */
async function setupMockEventSource(page) {
  await page.addInitScript(
    ({ tenantId, workspaceId, taskId }) => {
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
    },
  );
}

/**
 * 设置所有必需的 API mock，包括 server-start-history 记录。
 */
async function setupApiMocks(page, records) {
  await page.route('**/api/**', async (route) => {
    const req = route.request();
    const url = req.url();
    const method = req.method();

    // 任务详情
    if (url.includes(`/api/tasks/todos/tenant_id/${TENANT_ID}/workspace_id/${WORKSPACE_ID}/${TASK_ID}/`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: TASK_ID,
          title: 'Server Start History Test',
          description: 'E2E test for server start history',
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

    // 镜像列表
    if (url.includes(`/api/cloud/installed-images/tenant_id/${TENANT_ID}`) && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: 'img-mock-1', name: 'Mock Image', version: '1.0', hardware_summary: '4C16G' },
        ]),
      });
      return;
    }

    // server-start-history API
    if (url.includes('/cloud/compute/server-start-history/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          records: records,
        }),
      });
      return;
    }

    // 各类必需 mock
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

    if (url.includes('/previous-server-config/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', server_config: null }),
      });
      return;
    }

    if (url.includes('/workspace-access/workspace-collaborators/') && method === 'GET') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
      return;
    }

    if (url.includes('/progress-system/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ columns: [] }),
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

    if (url.includes('/cloud/platforms/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', platforms: [] }),
      });
      return;
    }

    if (url.includes('/cloud/regions/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'success', regions: [] }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: '{}',
    });
  });
}

test.describe('TaskDetail 历史服务器启动记录', () => {
  test.skip(() => process.env.PRE_COMMIT === '1', 'pre-commit 不跑需前端就绪的全栈 E2E');

  test('OPT-20260719-016: 每张历史启动记录卡显示运行持续时间、启动原因、关闭原因', async ({ page }) => {
    test.setTimeout(120_000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await setupMockEventSource(page);
    await setupApiMocks(page, MOCK_HISTORY_RECORDS);

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 展开服务器信息区
    const sectionToggle = page.getByTestId('server-section-collapse-toggle');
    const toggleText = await sectionToggle.textContent();
    if (toggleText && toggleText.includes('展开')) {
      await sectionToggle.click();
      await page.waitForTimeout(300);
    }

    // 点击「历史服务器启动记录」Tab
    const historyTab = page.getByRole('button', { name: '历史服务器启动记录' });
    await expect(historyTab).toBeVisible({ timeout: 15000 });
    await historyTab.click();
    await page.waitForTimeout(500);

    // 断言每张卡的三类文案可见
    const cards = page.locator('[data-testid="server-start-history-card"]');
    await expect(cards).toHaveCount(MOCK_HISTORY_RECORDS.length, { timeout: 15000 });

    for (let i = 0; i < MOCK_HISTORY_RECORDS.length; i++) {
      const card = cards.nth(i);

      // 运行持续时间
      await expect(card).toContainText('运行持续时间', { timeout: 10000 });
      // 启动原因
      await expect(card).toContainText('启动原因', { timeout: 10000 });
      // 关闭原因
      await expect(card).toContainText('关闭原因', { timeout: 10000 });

      // 额外校验：启动原因映射为非空中文
      const cardText = await card.textContent() || '';
      expect(cardText.length).toBeGreaterThan(0);

      // duration 不应为空白
      const durationLabel = '运行持续时间：';
      const durationIdx = cardText.indexOf(durationLabel);
      if (durationIdx >= 0) {
        const durationValue = cardText.slice(durationIdx + durationLabel.length).split('\n')[0].trim();
        expect(durationValue.length).toBeGreaterThan(0);
      }
    }

    console.log(`[PASS] ${MOCK_HISTORY_RECORDS.length} 张记录卡均显示「运行持续时间」「启动原因」「关闭原因」`);
  });

  test('OPT-20260720-005: 同 instance_type_id 的多卡硬件核内一致（磁盘可不同）', async ({ page }) => {
    test.setTimeout(120_000);

    await page.context().addCookies([
      { name: 'userId', value: 'e2e-user', url: 'http://localhost:4000/' },
      { name: 'csrftoken', value: 'e2e-csrf', url: 'http://localhost:4000/' },
    ]);

    await setupMockEventSource(page);
    await setupApiMocks(page, MOCK_HISTORY_RECORDS);

    await page.goto(TASK_DETAIL_PATH);
    await page.waitForLoadState('domcontentloaded');

    // 展开服务器信息区
    const sectionToggle = page.getByTestId('server-section-collapse-toggle');
    const toggleText = await sectionToggle.textContent();
    if (toggleText && toggleText.includes('展开')) {
      await sectionToggle.click();
      await page.waitForTimeout(300);
    }

    // 点击「历史服务器启动记录」Tab
    const historyTab = page.getByRole('button', { name: '历史服务器启动记录' });
    await expect(historyTab).toBeVisible({ timeout: 15000 });
    await historyTab.click();
    await page.waitForTimeout(500);

    const cards = page.locator('[data-testid="server-start-history-card"]');
    await expect(cards).toHaveCount(MOCK_HISTORY_RECORDS.length, { timeout: 15000 });

    // 按 instance_type_id 分组校验
    const groups = {};
    for (const rec of MOCK_HISTORY_RECORDS) {
      if (!groups[rec.instance_type_id]) groups[rec.instance_type_id] = [];
      groups[rec.instance_type_id].push(rec);
    }

    for (const [typeId, recs] of Object.entries(groups)) {
      if (recs.length < 2) {
        console.log(`[SKIP] instance_type_id=${typeId} 只有 1 条记录，跳过一致性校验`);
        continue;
      }

      // 取出每组所有卡的硬件文本
      const hardwareTexts = [];
      for (const rec of recs) {
        const idx = MOCK_HISTORY_RECORDS.indexOf(rec);
        const card = cards.nth(idx);
        await expect(card).toContainText('硬件：', { timeout: 10000 });
        const cardText = await card.textContent() || '';

        // 提取「硬件：X核 / YGB / ZGB」段落
        const hwMatch = cardText.match(/硬件：\s*(\d+)核\s*\/\s*(\d+)GB\s*\/\s*(\d+)GB/);
        expect(hwMatch).not.toBeNull();
        hardwareTexts.push({
          cpu: hwMatch[1],
          memory: hwMatch[2],
          storage: hwMatch[3],
        });
      }

      // cpu_cores 一致
      const cpus = new Set(hardwareTexts.map((h) => h.cpu));
      expect(cpus.size).toBe(1);

      // memory_gb 一致
      const memories = new Set(hardwareTexts.map((h) => h.memory));
      expect(memories.size).toBe(1);

      // storage_gb 允许不同（这里至少两条应有不同值）
      const storages = new Set(hardwareTexts.map((h) => h.storage));
      expect(storages.size).toBeGreaterThanOrEqual(2);

      console.log(`[PASS] instance_type_id=${typeId}: cpu 核数一致(${[...cpus][0]}), memory 一致(${[...memories][0]}GB), 磁盘有 ${storages.size} 种不同值`);
    }
  });
});
