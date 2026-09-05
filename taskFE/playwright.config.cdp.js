// @ts-check
/** Playwright 经 CDP 9222 连接外部 Chrome（见 01_core_testing_rules.md） */
import { defineConfig, devices } from '@playwright/test';
import baseConfig from './playwright.config.js';
import { loadPortConfig } from './helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const siteOrigin = (process.env.PLAYWRIGHT_SITE_ORIGIN || '').trim().replace(/\/+$/, '');
const baseURL = siteOrigin || `http://${portConfig.vue.host}:${portConfig.vue.port}`;
const wsEndpoint = process.env.PW_WS_ENDPOINT || '';

export default defineConfig({
  ...baseConfig,
  fullyParallel: false,
  workers: 1,
  timeout: 120000,
  use: {
    ...baseConfig.use,
    baseURL,
    ...(wsEndpoint ? { connectOptions: { wsEndpoint } } : {}),
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        channel: undefined,
        headless: true,
      },
    },
  ],
});
