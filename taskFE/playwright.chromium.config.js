// @ts-check
/** 本地无系统 Chrome / 无 DISPLAY 时使用 bundled Chromium 跑 E2E */
import baseConfig from './playwright.config.js';

/** @type {import('@playwright/test').PlaywrightTestConfig} */
const config = {
  ...baseConfig,
  use: {
    ...baseConfig.use,
    headless: true,
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...baseConfig.projects?.[0]?.use,
        channel: undefined,
      },
    },
  ],
};

export default config;
