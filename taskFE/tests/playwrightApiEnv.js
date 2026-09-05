// @ts-check
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

/** 前端站点 origin（登录页） */
export function playwrightSiteOrigin() {
  const fromEnv = (process.env.LOGIN_URL || process.env.PLAYWRIGHT_BASE_URL || '').trim();
  if (fromEnv) return fromEnv.replace(/\/$/, '');
  const cfg = loadPortConfig();
  const vue = cfg.vue || {};
  if (vue.publicBaseUrl) return String(vue.publicBaseUrl).replace(/\/$/, '');
  const host = vue.host || '127.0.0.1';
  const port = Number(vue.port) || 4000;
  return `http://${host}:${port}`;
}

/** API 网关 origin（/api/* 请求） */
export function playwrightApiOrigin() {
  const fromEnv = (process.env.PLAYWRIGHT_API_BASE_URL || process.env.VITE_API_BASE_URL || '').trim();
  if (fromEnv) return fromEnv.replace(/\/$/, '');
  const cfg = loadPortConfig();
  const vue = cfg.vue || {};
  if (vue.apiBaseUrl) return String(vue.apiBaseUrl).replace(/\/$/, '');
  return 'http://127.0.0.1:18081';
}

/** 测试账号（只读环境变量；口令禁止硬编码回退 — OPT-20260827-015） */
export function playwrightTenantCredentials() {
  const email =
    process.env.PLAYWRIGHT_TEST_EMAIL ||
    process.env.LOGIN_EMAIL ||
    process.env.E2E_EMAIL ||
    'contact@daydaymoney.com';
  const password =
    process.env.PLAYWRIGHT_TEST_PASSWORD ||
    process.env.LOGIN_PASSWORD ||
    process.env.E2E_PASSWORD;
  if (!password) {
    throw new Error(
      'playwrightTenantCredentials: PLAYWRIGHT_TEST_PASSWORD / LOGIN_PASSWORD / E2E_PASSWORD env required (no hardcoded fallback)'
    );
  }
  return { email, password };
}
