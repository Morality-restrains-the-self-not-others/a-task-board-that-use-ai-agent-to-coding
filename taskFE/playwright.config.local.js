// @ts-check
import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadPortConfig } from './helpers/loadConfYaml.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const portConfig = loadPortConfig();

export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.playwright.test.js',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: [
    ['html', { open: 'never', outputFolder: './tests/playwright-report-local' }],
    ['json', { outputFile: './tests/test_results/playwright-report-local.json' }]
  ],
  use: {
    baseURL: `http://${portConfig.vue.host}:${portConfig.vue.port}`,
    headless: false,
    trace: 'on-first-retry',
    console: 'on',
    timeout: 120000,
    screenshot: 'off',
  },
  
  outputDir: './tests/test_results',

  projects: [
    {
      name: 'chromium-local',
      use: { 
        ...devices['Desktop Chrome'],
        channel: 'chrome',
      },
    },
  ],

  launchOptions: {
    args: ['--remote-debugging-port=9222'],
  },
});