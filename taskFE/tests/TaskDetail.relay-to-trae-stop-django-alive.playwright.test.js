// @ts-check
/**
 * 回归：relay /v1/stop 清理 onlineServiceJS 端口时，不得误杀连入 8765 的本地服务客户端。
 *
 * 根因：`lsof -ti :8765` 会匹配 LISTEN 与 ESTABLISHED 客户端；stop 时 reset 请求进行中
 * 的客户端进程会被 SIGTERM，表现为任务挂起。
 *
 * OPT-20260806-053 迁移：Django saas-backend 已退役（2026-07-30），原以 Django runserver
 * (8001) 作探测目标；现改用 taskAuth (8003) —— 同为本地常驻服务，验证 relay stop
 * 不会误杀连入 8765 的其他本地服务客户端。
 *
 * 集成：PLAYWRIGHT_RELAY_STOP_DJANGO=1，本机 go_relay(8797)、onlineServiceJS(8765)、taskAuth(8003)
 */
import { test, expect } from '@playwright/test';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

const portConfig = loadPortConfig();

// OPT-20260806-053: django.port → taskAuth 端口（Django 退役，探测目标迁移）
const PROBE_ORIGIN = `http://127.0.0.1:${portConfig.taskAuth?.port || 8003}`;
const RELAY_ORIGIN = `http://127.0.0.1:${portConfig.relayToTrae.port}`;
const RELAY_SECRET = portConfig.relayToTrae.secret || 'dev-secret';
const ONLINE_PORT = 8765;

const INTEGRATION = process.env.PLAYWRIGHT_RELAY_STOP_DJANGO === '1';

async function pingLocalService(request, timeoutMs = 5000) {
  const resp = await request.get(`${PROBE_ORIGIN}/api/health/`, {
    timeout: timeoutMs,
  });
  return resp.ok();
}

async function isOnlineServiceListening(request) {
  try {
    const resp = await request.get(`http://127.0.0.1:${ONLINE_PORT}/api/jobs`, { timeout: 2000 });
    return resp.status() === 401 || resp.ok();
  } catch {
    return false;
  }
}

test.describe('relay stop 不得挂起本地服务', () => {
  test('go_relay /v1/stop 后本地服务（原 Django 探测位）仍应在 5s 内响应', async ({ request }) => {
    test.skip(!INTEGRATION, 'Set PLAYWRIGHT_RELAY_STOP_DJANGO=1 to run against local stack');

    expect(await pingLocalService(request)).toBe(true);

    const relayHealth = await request.get(`${RELAY_ORIGIN}/health`, { timeout: 3000 });
    expect(relayHealth.ok()).toBe(true);

    test.skip(!(await isOnlineServiceListening(request)), 'onlineServiceJS 未在 8765 监听，请先直接启动');

    const stopResp = await request.post(`${RELAY_ORIGIN}/v1/stop`, {
      headers: {
        'X-Relay-To-Trae-Secret': RELAY_SECRET,
        'Content-Type': 'application/json',
      },
      timeout: 120_000,
    });
    expect(stopResp.ok(), `stop failed: ${stopResp.status()} ${await stopResp.text()}`).toBe(true);

    await expect(async () => {
      expect(await pingLocalService(request)).toBe(true);
    }).toPass({ timeout: 10_000, intervals: [500, 1000, 2000] });
  });
});
