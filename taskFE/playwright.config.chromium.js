// @ts-check
/** Playwright 经 bundled Chromium（无本机 Chrome 时） */
import baseConfig from './playwright.config.js';
import { loadPortConfig } from './helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();
const siteOrigin = (process.env.PLAYWRIGHT_SITE_ORIGIN || '').trim().replace(/\/+$/, '');
const baseURL = siteOrigin || `http://${portConfig.vue.host}:${portConfig.vue.port}`;

export default {
  ...baseConfig,
  fullyParallel: false,
  workers: 1,
  use: {
    ...baseConfig.use,
    baseURL,
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...baseConfig.projects?.[0]?.use,
        channel: undefined,
        headless: true,
        launchOptions: {
          args: ['--no-sandbox'],
        },
      },
    },
  ],
};
