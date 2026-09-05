// @ts-check
/**
 * Playwright config for billing invoice + grant-points E2E (OPT-20260823-055/056/065).
 */
import { defineConfig, devices } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { existsSync } from 'fs';
import { homedir } from 'os';
import path from 'path';

const portConfig = loadPortConfig();
const vueHost = clientReachableHost(portConfig.vue?.host, '127.0.0.1');

const chromiumCandidates = [
  path.join(homedir(), '.cache/ms-playwright/chromium_headless_shell-1228/chrome-headless-shell-linux64/chrome-headless-shell'),
  path.join(homedir(), '.cache/ms-playwright/chromium-1228/chrome-linux64/chrome'),
  path.join(homedir(), '.cache/ms-playwright/chromium-1223/chrome-linux64/chrome'),
];
const executablePath = chromiumCandidates.find((p) => existsSync(p)) || undefined;

export default defineConfig({
  testDir: '.',
  testMatch: [
    '**/BillingInvoice.admin-flow.playwright.test.js',
    '**/BillingGrantPoints.admin-flow.playwright.test.js',
  ],
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: `http://${vueHost}:${portConfig.vue.port}`,
    headless: true,
    trace: 'off',
    timeout: 60000,
    screenshot: 'off',
  },
  outputDir: './test_results',
  projects: [
    {
      name: 'chromium-bundled',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: ['--no-sandbox'],
          ...(executablePath ? { executablePath } : {}),
        },
      },
    },
  ],
});
