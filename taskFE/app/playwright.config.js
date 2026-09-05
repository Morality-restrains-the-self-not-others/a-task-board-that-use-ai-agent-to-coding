// @ts-check
import { defineConfig, devices } from '@playwright/test';

/**
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests/playwright',
  testMatch: '**/*.playwright.test.js',
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,
  /* Reporter to use. See https://playwright.dev/docs/test-reporters */
  reporter: [
    ['html', { open: 'never', outputFolder: './tests/playwright/playwright-report' }],
    ['json', { outputFile: './tests/playwright/test_results/playwright-report.json' }]
  ],
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: 'http://localhost:3000',

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',
    
    /* Record console logs */
    console: 'on',

    /* Test timeout */
    timeout: 60000,

    /* Screenshot settings */
    screenshot: 'off',
  },
  
  /* Test output directory */
  outputDir: './tests/playwright/test_results',

  /* Configure projects for major browsers */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'], channel: 'chrome' },
    },

    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },

    /* Test against mobile viewports. */
    // {
    //   name: 'Mobile Chrome',
    //   use: { ...devices['Pixel 5'] },
    // },
    // {
    //   name: 'Mobile Safari',
    //   use: { ...devices['iPhone 12'] },
    // },

    /* Test against branded browsers. */
    // {
    //   name: 'Microsoft Edge',
    //   use: { ...devices['Desktop Edge'], channel: 'msedge' },
    // },
    // {
    //   name: 'Google Chrome',
    //   use: { ...devices['Desktop Chrome'], channel: 'chrome' },
    // },
  ],

  /* Run backend/frontend dev servers before starting tests. 本地默认复用已启动服务；CI 时启动新服务 */
  webServer: [
    {
      command: 'python3 manage.py runserver 8000',
      url: 'http://localhost:8000/api/core/test-number-serialization/',
      reuseExistingServer: !process.env.CI,
      cwd: '../../Saas_project',
    },
    {
      command: 'npm run dev',
      url: 'http://localhost:3000/',
      reuseExistingServer: !process.env.CI,
      cwd: './',
    }
  ],
});
