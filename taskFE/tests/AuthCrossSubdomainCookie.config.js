// @ts-check
/** 专项配置：跨子域会话 cookie 回归（对线上生产运行；绝对 URL，不依赖本地 baseURL） */
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: 'AuthCrossSubdomainCookie.playwright.test.js',
  workers: 1,
  timeout: 180000,
  retries: 0,
  reporter: [['line']],
  use: {
    headless: true,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  outputDir: './test_results',
});
