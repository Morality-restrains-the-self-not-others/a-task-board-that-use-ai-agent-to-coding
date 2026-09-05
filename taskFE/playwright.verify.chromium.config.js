// @ts-check
import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadPortConfig, clientReachableHost } from './helpers/loadConfYaml.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const portConfig = loadPortConfig();
const vueHost = clientReachableHost(portConfig.vue?.host);

export default defineConfig({
  testDir: './tests',
  testMatch: '**/*.playwright.test.js',
  fullyParallel: false,
  workers: 1,
  timeout: 120000,
  expect: { timeout: 15000 },
  use: {
    baseURL: `http://${vueHost}:${portConfig.vue.port}`,
    trace: 'on-first-retry',
    actionTimeout: 90000,
    headless: true,
    launchOptions: {
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--proxy-server=direct://', '--proxy-bypass-list=*'],
    },
  },
  outputDir: './tests/test_results',
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
