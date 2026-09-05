// @ts-check
/**
 * E2E：系统超管访问 /system-admin/ 应停留，不跳转 work-panel。
 *
 * 账号：默认 author@example.com
 * 推荐运行：
 *   bash taskFE/tests/SystemAdmin.superadmin-stays-on-system-admin.playwright.test.sh
 */
import { test, expect } from '@playwright/test';
import { spawnSync } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

test.describe('系统超管访问 system-admin 应停留', () => {
  test('登录后打开 /system-admin/ 仍在 system-admin', async () => {
    const script = path.join(__dirname, 'SystemAdmin.superadmin-stays-on-system-admin-cdp.mjs');
    const r = spawnSync(process.execPath, [script], {
      encoding: 'utf8',
      env: {
        ...process.env,
        NO_PROXY: process.env.NO_PROXY || '*',
        http_proxy: '',
        https_proxy: '',
        all_proxy: '',
        HTTP_PROXY: '',
        HTTPS_PROXY: '',
        ALL_PROXY: '',
        PLAYWRIGHT_SITE_ORIGIN:
          process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com',
        E2E_SUPERADMIN_EMAIL: process.env.E2E_SUPERADMIN_EMAIL || 'author@example.com',
        E2E_SUPERADMIN_PASSWORD: process.env.E2E_SUPERADMIN_PASSWORD,
        CDP_URL: process.env.CDP_URL || process.env.PW_CDP_URL || 'http://127.0.0.1:9223',
      },
      timeout: 180000,
    });
    if (r.stdout) console.log(r.stdout);
    if (r.stderr) console.error(r.stderr);
    expect(r.status, `CDP 脚本退出码 ${r.status}`).toBe(0);
    expect(String(r.stdout || '')).toMatch(/\bPASS\b/);
  });
});
