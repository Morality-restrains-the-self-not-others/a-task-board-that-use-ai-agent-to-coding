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
  testMatch: '**/CreateProject.oauth-button.playwright.test.js',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: `http://${portConfig.vue.host}:${portConfig.vue.port}`,
    headless: true,
    trace: 'off',
    timeout: 60000,
    screenshot: 'off',
  },

  outputDir: './tests/test_results',

  projects: [
    {
      name: 'chromium-bundled',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: ['--no-sandbox'],
          executablePath: '/home/ljy/.cache/ms-playwright/chromium-1228/chrome-linux64/chrome',
        },
      },
    },
  ],
});
