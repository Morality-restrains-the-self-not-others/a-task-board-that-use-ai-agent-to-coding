// @ts-check
/**
 * start-vm-auto 容器初始化 E2E：RunInstances UserData 占位符须被替换，实例内 Docker 容器应就绪。
 *
 * 验证策略（按优先级）：
 * 1. server-content 200 — 网关经公网 IP:8080 拉取容器 HTTP（host 网络）
 * 2. 可选 SSH（PLAYWRIGHT_INSTANCE_SSH_PRIVATE_KEY + public_ip）校验 init 脚本无占位符且 docker ps 运行中
 *
 * 运行：
 * PLAYWRIGHT_SITE_ORIGIN=http://127.0.0.1:4000 \
 * PLAYWRIGHT_GATEWAY_ORIGIN=http://127.0.0.1:18081 \
 * PLAYWRIGHT_TASK_ID=task_12590983282794675865 \
 * PLAYWRIGHT_WORKSPACE_ID=861623708318031872 \
 * PLAYWRIGHT_CONTAINER_IMAGE_ID=862588024964280320 \
 * PLAYWRIGHT_EXPECT_CONTAINER_IMAGE_REF='registry.cn-hongkong.aliyuncs.com/.../tag' \
 * npx playwright test tests/TaskDetail.start-vm-auto-container-init.playwright.test.js \
 *   --config=playwright.config.chromium.js
 */
import { test, expect } from '@playwright/test';
import { loginViaGatewayApi } from './helpers/gatewayLoginE2e.js';
import { assertInitFromTask2appLogViaSsh } from './helpers/instanceUserdataSshCheck.js';
import {
  WARM_TASK_ID,
  WARM_WORKSPACE_ID,
  postStartVmAuto,
  pollServerRuntimeStatus,
  waitForInstanceId,
  waitForServerContentReady,
  assertWorkbenchLinkForInstance,
} from './helpers/startVmAutoE2e.js';

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'http://127.0.0.1:4000').replace(/\/$/, '');
const GATEWAY = (process.env.PLAYWRIGHT_GATEWAY_ORIGIN || 'http://127.0.0.1:18081').replace(/\/$/, '');
const TASK_ID = WARM_TASK_ID;
const WORKSPACE_ID = WARM_WORKSPACE_ID;
const EXPECT_IMAGE_REF = String(process.env.PLAYWRIGHT_EXPECT_CONTAINER_IMAGE_REF || '').trim();
const SSH_PRIVATE_KEY = String(process.env.PLAYWRIGHT_INSTANCE_SSH_PRIVATE_KEY || '').trim();
const SKIP_SSH = String(process.env.E2E_SKIP_INSTANCE_SSH || '') === '1';

test.describe('TaskDetail start-vm-auto container init E2E', () => {
  test('start-vm-auto 后实例内容器应启动（UserData 占位符已替换）', async ({ page, request }) => {
    test.setTimeout(1_200_000);

    const loginData = await loginViaGatewayApi(page, {
      email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
      password: process.env.PLAYWRIGHT_TEST_PASSWORD,
      siteOrigin: SITE,
      gatewayOrigin: GATEWAY,
    });

    const precheck = await pollServerRuntimeStatus(request, loginData.token, TASK_ID, WORKSPACE_ID);
    expect(precheck.status).toBe(200);
    let instanceId = String(precheck.body?.instance_id || '').trim();

    if (!instanceId) {
      const apiStart = await postStartVmAuto(request, loginData.token, TASK_ID, WORKSPACE_ID);
      expect(apiStart.status, `start-vm-auto body=${JSON.stringify(apiStart.body).slice(0, 300)}`).toBe(200);
      expect(apiStart.body?.status).toBe('success');
      instanceId = await waitForInstanceId(request, loginData.token, TASK_ID, WORKSPACE_ID);
      expect(instanceId, 'instance_id after start-vm-auto').not.toBe('');
    }

    await assertWorkbenchLinkForInstance(instanceId, request, loginData.token, TASK_ID, WORKSPACE_ID);

    const runtimeDeadline = Date.now() + 15 * 60_000;
    let publicIp = '';
    while (Date.now() < runtimeDeadline) {
      const snap = await pollServerRuntimeStatus(request, loginData.token, TASK_ID, WORKSPACE_ID);
      expect(snap.status).toBe(200);
      publicIp = String(
        snap.body?.public_ip ||
          snap.body?.instance_attribute?.body?.PublicIpAddress?.IpAddress?.[0] ||
          '',
      ).trim();
      const runtimeStatus = String(snap.body?.runtime_status || '').trim();
      if (publicIp && runtimeStatus === 'Running') {
        break;
      }
      await page.waitForTimeout(15_000);
    }

    const content = await waitForServerContentReady(request, loginData.token, TASK_ID, WORKSPACE_ID);
    expect(content.body?.status).toBe('success');
    expect(String(content.body?.content || content.body?.html || '')).not.toBe('');

    if (!SKIP_SSH && SSH_PRIVATE_KEY && publicIp) {
      const sshResult = await assertInitFromTask2appLogViaSsh(SSH_PRIVATE_KEY, publicIp, {
        maxAttempts: 24,
        intervalMs: 20_000,
        requireContainerRunning: true,
        expectedContainerImageRef: EXPECT_IMAGE_REF || undefined,
      });
      expect(sshResult.stdout).toContain('TASK2APP_E2E_NO_IMAGE_PLACEHOLDER');
      expect(sshResult.stdout).toContain('TASK2APP_E2E_CONTAINER_RUNNING');
    } else {
      test.info().annotations.push({
        type: 'note',
        description:
          '未配置 PLAYWRIGHT_INSTANCE_SSH_PRIVATE_KEY 或 E2E_SKIP_INSTANCE_SSH=1，已通过 server-content 校验容器 HTTP 就绪',
      });
    }
  });
});
