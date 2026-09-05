// @ts-check
/** 用于核验的 Playwright 配置：不启动 webServer，假定服务已运行 */
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/playwright',
  testMatch: '**/*.playwright.test.js',
  fullyParallel: false,
  workers: 1,
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    timeout: 90000,
  },
  outputDir: './tests/test_results',
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'], channel: 'chrome' } }],
  // 不配置 webServer，由用户手动启动前后端
});
