// @ts-check
/** 使用 Playwright 内置 Chromium（无系统 Chrome 依赖）运行 SSH 分支预览 E2E */
import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { clientReachableHost, loadPortConfig } from './helpers/loadConfYaml.mjs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const portConfig = loadPortConfig();

export default defineConfig({
  testDir: './tests',
  testMatch: '**/ProjectDetail.ssh-git-url-branch-preview.playwright.test.js',
  workers: 1,
  use: {
    baseURL: `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`,
    headless: true,
    trace: 'on-first-retry',
    timeout: 60000,
  },
  projects: [
    {
      name: 'bundled-chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
