// @ts-check
/** Playwright config: live customer login-history IP (OPT-20260826-002). */
import { defineConfig, devices } from '@playwright/test';
import { existsSync } from 'fs';
import { homedir } from 'os';
import path from 'path';

const siteOrigin = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com')
  .trim()
  .replace(/\/+$/, '');

const chromiumCandidates = [
  path.join(homedir(), '.cache/ms-playwright/chromium_headless_shell-1228/chrome-headless-shell-linux64/chrome-headless-shell'),
  path.join(homedir(), '.cache/ms-playwright/chromium-1228/chrome-linux64/chrome'),
  path.join(homedir(), '.cache/ms-playwright/chromium-1223/chrome-linux64/chrome'),
];
const executablePath = chromiumCandidates.find((p) => existsSync(p)) || undefined;

export default defineConfig({
  testDir: '.',
  testMatch: '**/LoginHistory.customer-ip.playwright.test.js',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: siteOrigin,
    headless: true,
    trace: 'off',
    timeout: 120000,
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
