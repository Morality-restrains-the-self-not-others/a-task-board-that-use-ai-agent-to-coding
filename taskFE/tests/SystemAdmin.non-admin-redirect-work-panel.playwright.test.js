// @ts-check
/**
 * E2E：非系统管理员访问 /system-admin/ 应跳转到其租户 work-panel。
 *
 * 账号：默认 e2e.nonadmin.sysadmin.redirect@ljytest.com（自行注册的非超管）
 * 推荐运行（公网 + CDP，与 Login.daydaymoney 同模式）：
 *   bash taskFE/tests/SystemAdmin.non-admin-redirect-work-panel.playwright.test.sh
 *
 * 本文件保留为 Playwright 用例入口；`.sh` 实际执行同目录 `-cdp.mjs`。
 */
import { test, expect } from '@playwright/test';
import { spawnSync } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

test.describe('非系统管理员访问 system-admin 跳转 work-panel', () => {
  test('登录后打开 /system-admin/ 落到 /tenant/{id}/work-panel/', async () => {
    const script = path.join(__dirname, 'SystemAdmin.non-admin-redirect-work-panel-cdp.mjs');
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
        E2E_NONADMIN_EMAIL:
          process.env.E2E_NONADMIN_EMAIL || 'e2e.nonadmin.sysadmin.redirect@ljytest.com',
        E2E_NONADMIN_PASSWORD: process.env.E2E_NONADMIN_PASSWORD || 'E2eNonAdmin!8677',
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
