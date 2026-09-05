// @ts-check
/**
 * 真实探针回归（会创建阿里云 ECS，可能产生费用）：
 * 1) 调用系统管理员 probe-start-test 启动测试实例
 * 2) 通过 SSH 校验 /root/init_from_task2app.sh.log 非空
 * 3) 可选校验 /root/init_from_task2app.sh 中 CONTAINER_IMAGE 是否包含期望镜像引用（含 tag）
 *
 * 运行示例：
 * E2E_EMAIL=... E2E_PASSWORD='...' E2E_VERIFY_CLOUD_PROBE=1 \
 * E2E_PROBE_INSTALLED_IMAGE_ID=1234567890 E2E_SYSTEM_ADMIN_UID=1 \
 * E2E_EXPECT_CONTAINER_IMAGE_REF='registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64_2026-05-08_09-46' \
 * npx playwright test -c playwright.config.js \
 * tests/CloudAuthorizations.probe-start-ssh-modal.playwright.test.js --project=chromium
 */
import { test, expect } from '@playwright/test';
import { submitLoginWithEmailPassword } from './helpers/e2eLogin.js';
import { assertInitFromTask2appLogViaSsh } from './helpers/instanceUserdataSshCheck.js';

test.describe('Cloud Authorizations — probe start + ssh userdata check', () => {
  test('probe-start-test returns ssh key and remote init script contains expected image tag', async ({ page }) => {
    const email = process.env.E2E_EMAIL || '';
    const password = process.env.E2E_PASSWORD || '';
    const enabled = String(process.env.E2E_VERIFY_CLOUD_PROBE || '') === '1';
    const installedImageId = String(process.env.E2E_PROBE_INSTALLED_IMAGE_ID || '').trim();
    const systemAdminUid = String(process.env.E2E_SYSTEM_ADMIN_UID || '').trim() || '1';
    const expectedContainerImageRef = String(process.env.E2E_EXPECT_CONTAINER_IMAGE_REF || '').trim();
    const skipSsh = String(process.env.E2E_SKIP_INSTANCE_SSH || '') === '1';
    const requireSshdDropin = String(process.env.E2E_REQUIRE_SSHD_DROPIN || '') === '1';

    test.skip(!enabled, '未设置 E2E_VERIFY_CLOUD_PROBE=1，跳过真实云探针用例');
    test.skip(!email || !password, '未设置 E2E_EMAIL / E2E_PASSWORD');
    test.skip(!installedImageId, '未设置 E2E_PROBE_INSTALLED_IMAGE_ID');

    await page.goto('/auth/login/');
    await page.waitForLoadState('networkidle');
    await submitLoginWithEmailPassword(page, email, password);
    await page.waitForURL((u) => !u.pathname.includes('/auth/login'), { timeout: 60000 });

    const startResp = await page.request.post(
      `/api/system-admin/${encodeURIComponent(systemAdminUid)}/cloud/server-images/${encodeURIComponent(installedImageId)}/probe-start-test/`,
      {
        data: {
          instance_charge_type: 'PostPaid',
          cpu_cores: 1,
          memory_gb: 1,
          storage_gb: 40,
          internet_max_bandwidth_out: 5,
          auto_release_enabled: true,
          auto_release_minutes: 35,
        },
      }
    );
    const data = await startResp.json().catch(() => ({}));
    expect(startResp.ok(), `probe-start-test failed: ${JSON.stringify(data)}`).toBeTruthy();

    const publicIp = String(data?.public_ip || data?.start_result?.public_ip || '').trim();
    const sshPrivateKeyPem = String(data?.ssh_private_key_pem || '').trim();
    expect(publicIp, `missing public_ip in response: ${JSON.stringify(data)}`).toBeTruthy();
    expect(sshPrivateKeyPem, 'missing ssh_private_key_pem in probe response').toBeTruthy();

    if (skipSsh) {
      test.info().annotations.push({
        type: 'note',
        description: 'E2E_SKIP_INSTANCE_SSH=1，已跳过 SSH 远端 init_from_task2app.sh 校验',
      });
      return;
    }

    const sshResult = await assertInitFromTask2appLogViaSsh(sshPrivateKeyPem, publicIp, {
      maxAttempts: 36,
      intervalMs: 15000,
      requireSshdDropin,
      expectedContainerImageRef: expectedContainerImageRef || undefined,
    });
    expect(sshResult.stdout).toContain('TASK2APP_E2E_LOG_OK');
  });
});
