// @ts-check
/** Playwright config: mock admin login-history href + privacy gate (local Vue). */
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
  testMatch: '**/LoginHistory.admin-href.playwright.test.js',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
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
