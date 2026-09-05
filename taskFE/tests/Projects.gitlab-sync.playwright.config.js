// @ts-check
/**
 * Playwright config for Projects GitLab sync E2E tests.
 */
import { defineConfig, devices } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import { existsSync } from 'fs';
import { homedir } from 'os';
import path from 'path';

const portConfig = loadPortConfig();

const chromiumCandidates = [
  path.join(homedir(), '.cache/ms-playwright/chromium_headless_shell-1228/chrome-headless-shell-linux64/chrome-headless-shell'),
  path.join(homedir(), '.cache/ms-playwright/chromium-1228/chrome-linux64/chrome'),
];
const executablePath = chromiumCandidates.find((p) => existsSync(p)) || undefined;

const defaultBase =
  process.env.SITE_BASE ||
  process.env.BASE_URL ||
  portConfig.vue?.publicBaseUrl ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`;

export default defineConfig({
  testDir: '.',
  testMatch: '**/Projects.gitlab-sync.playwright.test.js',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  timeout: 120000,
  reporter: [['list']],
  use: {
    baseURL: defaultBase.replace(/\/$/, ''),
    headless: !process.env.PW_HEADED,
    trace: 'on-first-retry',
    actionTimeout: 60000,
    navigationTimeout: 60000,
    screenshot: 'only-on-failure',
  },
  outputDir: './test_results',
  projects: [
    {
      name: 'chromium-bundled',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: [
            '--no-sandbox',
            '--proxy-bypass-list=<-loopback>,127.0.0.1,localhost,183.250.1.132',
          ],
          ...(executablePath ? { executablePath } : {}),
        },
      },
    },
  ],
});
