// @ts-check
import { defineConfig, devices, chromium } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { clientReachableHost, loadPortConfig } from './helpers/loadConfYaml.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const vueAppRoot = path.resolve(__dirname, './app');

const portConfig = loadPortConfig();
const isCI = !!process.env.CI;

const CDP_PORT = 9222;
const CDP_URL = `http://127.0.0.1:${CDP_PORT}`;

async function connectToChrome() {
  return await chromium.connectOverCDP(CDP_URL);
}

export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.playwright.test.js',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { open: 'never', outputFolder: './tests/playwright-report' }],
    ['json', { outputFile: './tests/test_results/playwright-report.json' }]
  ],
  use: {
    baseURL: `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`,
    headless: !!process.env.CI,
    trace: 'on-first-retry',
    console: 'on',
    timeout: 60000,
    screenshot: 'off',
  },
  
  outputDir: './tests/test_results',

  projects: [
    {
      name: 'chromium',
      use: { 
        ...devices['Desktop Chrome'],
        channel: 'chrome',
      },
    },
  ],

  // Django saas-backend 已退役（2026-07-30）：移除 manage.py runserver webServer 项
  // （OPT-20260806-053：django runserver/test-number-serialization 死引用迁移），
  // CI 下仅需前端 dev server。
  webServer: isCI
    ? [
        {
          command: 'npm run dev',
          url: `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}/`,
          reuseExistingServer: false,
          cwd: vueAppRoot,
        }
      ]
    : undefined,

});