// @ts-check
/** 用于核验的 Playwright 配置：不启动 webServer，假定服务已运行 */
import { defineConfig, devices } from '@playwright/test';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadPortConfig, clientReachableHost } from './helpers/loadConfYaml.mjs';

// 获取当前文件路径
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
    // 忽略开发机环境 SOCKS/HTTP 代理，避免 ERR_PROXY_CONNECTION_FAILED（与 verify.chromium 一致）
    launchOptions: {
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--proxy-server=direct://', '--proxy-bypass-list=*'],
    },
  },
  outputDir: './tests/test_results',
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  // 不配置 webServer，由用户手动启动前后端
});
