// @ts-check
/**
 * Playwright config for CreateProject GitLab OAuth full-flow E2E tests.
 * Uses Playwright bundled Chromium headless shell (no system Chrome needed).
 *
 * Usage:
 *   npx playwright test -c taskFE/tests/CreateProject.gitlab-oauth.config.js \
 *     taskFE/tests/CreateProject.gitlab-oauth-full-flow.playwright.test.js \
 *     --project=chromium-bundled
 */
import { defineConfig, devices } from '@playwright/test';
import { existsSync } from 'fs';
import { homedir } from 'os';
import path from 'path';

// Prefer headless shell (no X server needed) over full Chromium
const chromiumCandidates = [
  path.join(homedir(), '.cache/ms-playwright/chromium_headless_shell-1228/chrome-headless-shell-linux64/chrome-headless-shell'),
  path.join(homedir(), '.cache/ms-playwright/chromium-1228/chrome-linux64/chrome'),
];
const executablePath = chromiumCandidates.find((p) => existsSync(p)) || undefined;

const SITE_BASE = process.env.SITE_BASE || 'http://183.250.1.132:4000';
const siteUrl = new URL(SITE_BASE);

export default defineConfig({
  testDir: '.',
  testMatch: '**/CreateProject.gitlab-oauth-full-flow.playwright.test.js',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list'], ['html', { open: 'never', outputFolder: './playwright-report-local' }]],
  use: {
    baseURL: `${siteUrl.protocol}//${siteUrl.host}`,
    headless: true,
    trace: 'on',
    timeout: 180000, // 3 minutes per test (OAuth flow can be slow)
    screenshot: 'on',
    video: 'off',
  },

  outputDir: './test_results',

  projects: [
    {
      name: 'chromium-bundled',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: ['--no-sandbox', '--disable-setuid-sandbox'],
          ...(executablePath ? { executablePath } : {}),
        },
      },
    },
  ],
});
