// @ts-check
import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { clientReachableHost, loadPortConfig } from './helpers/loadConfYaml.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const portConfig = loadPortConfig();

export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.playwright.test.js',
  fullyParallel: false,
  forbidOnly: false,
  retries: 0,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`,
    headless: true,
    trace: 'off',
    screenshot: 'off',
  },
  outputDir: './tests/test_results',
  projects: [
    {
      name: 'chromium-headless',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
