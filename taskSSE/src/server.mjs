import http from 'node:http';
import { URL } from 'node:url';
import { loadTaskSseConfig } from './config.mjs';
import { SseHub } from './sseHub.mjs';
import { startMessageBus } from './bus/index.mjs';
import {
  normalizeInboundMessage,
  formatSseData,
  billingUserHubKey,
  workspaceHubKey,
} from './messageNormalize.mjs';
import { withTraceMiddleware } from './traceMiddleware.mjs';

// Convention-compliant paths: /api/sse/{eventType}/tenant_id/{tid}/workspace_id/{wid}/task_id/{taskId}
const SSE_PATH_RE =
  /^\/api\/sse\/server-startup-status\/tenant_id\/([^/]+)\/workspace_id\/([^/]+)\/task_id\/([^/]+)\/?$/;
const BILLING_SSE_PATH_RE =
  /^\/api\/sse\/recharge-events\/tenant_id\/([^/]+)\/?$/;
const WORK_PANEL_SSE_PATH_RE =
  /^\/api\/sse\/work-panel-events\/tenant_id\/([^/]+)\/workspace_id\/([^/]+)\/?$/;

function json(res, status, body) {
  const text = JSON.stringify(body);
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
  res.end(text);
}

function readJsonBody(req) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    req.on('data', (c) => chunks.push(c));
    req.on('end', () => {
      try {
        const raw = Buffer.concat(chunks).toString('utf8').trim();
        resolve(raw ? JSON.parse(raw) : {});
      } catch (e) {
        reject(e);
      }
    });
    req.on('error', reject);
  });
}

function secretOk(req, expected) {
  if (!expected) return true;
  const got = String(req.headers['x-task-sse-secret'] || '').trim();
  return got === expected;
}

/** 浏览器 SSE：须带网关 proxy-rewrite 的 X-TaskGateway-Internal-Secret（fail-closed）。 */
export function gatewayInternalSecretOk(req, expected) {
  const want = String(expected || '').trim();
  if (!want) return false;
  const got = String(req.headers['x-taskgateway-internal-secret'] || '').trim();
  return got !== '' && got === want;
}

function rejectUnlessGateway(req, res, cfg) {
  if (gatewayInternalSecretOk(req, cfg.gatewayInternalSecret)) return false;
  const want = String(cfg.gatewayInternalSecret || '').trim();
  const pathName = String(req.url || '/').split('?')[0];
  process.stdout.write(
    `${JSON.stringify({
      ts: new Date().toISOString(),
      level: 'warn',
      service: 'task-sse',
      msg: 'gateway_internal_secret_rejected',
      reason: want ? 'mismatch_or_missing_header' : 'expected_secret_empty_fail_closed',
      trace_id: req.traceId || '',
      path: pathName,
    })}\n`,
  );
  json(res, 403, { detail: 'forbidden: gateway internal secret required' });
  return true;
}

export async function createTaskSseServer(cfg = loadTaskSseConfig()) {
  const hub = new SseHub();
  let busCloser = null;

  const onMessage = (taskId, statusData) => {
    hub.publish(taskId, statusData);
  };

  try {
    busCloser = await startMessageBus(cfg, onMessage, console.log);
  } catch (err) {
    console.warn('[taskSSE] message bus start failed (HTTP /internal/publish still works):', err?.message || err);
  }

  const server = http.createServer(withTraceMiddleware('task-sse', async (req, res) => {
    const url = new URL(req.url || '/', `http://${req.headers.host || 'localhost'}`);

    if (req.method === 'GET' && url.pathname === '/health') {
      return json(res, 200, {
        ok: true,
        service: 'task-sse',
        transport: cfg.transport,
        connections: hub.totalConnections(),
        maxConnections: cfg.maxConnections ?? 0,
      });
    }

    if (req.method === 'POST' && url.pathname === '/internal/publish') {
      if (!secretOk(req, cfg.secret)) {
        return json(res, 401, { detail: 'unauthorized' });
      }
      try {
        const body = await readJsonBody(req);
        const normalized = normalizeInboundMessage(body);
        if (!normalized) {
          return json(res, 400, { detail: 'invalid payload' });
        }
        const delivered = hub.publish(normalized.taskId, normalized.statusData);
        return json(res, 200, { status: 'ok', delivered, task_id: normalized.taskId });
      } catch {
        return json(res, 400, { detail: 'invalid json' });
      }
    }

    if (req.method === 'GET') {
      const workPanelMatch = url.pathname.match(WORK_PANEL_SSE_PATH_RE);
      if (workPanelMatch) {
        if (rejectUnlessGateway(req, res, cfg)) return;
        const tenantId = workPanelMatch[1];
        const workspaceId = workPanelMatch[2];
        const hubKey = workspaceHubKey(workspaceId);
        if (!hubKey) {
          return json(res, 400, { detail: 'workspace_id required' });
        }
        const userId = String(req.headers['x-user-id'] || '').trim();
        const maxConn = Number(cfg.maxConnections) || 0;
        if (maxConn > 0 && hub.totalConnections() >= maxConn) {
          return json(res, 503, {
            detail: 'sse connection limit reached',
            connections: hub.totalConnections(),
            maxConnections: maxConn,
          });
        }
        res.writeHead(200, {
          'Content-Type': 'text/event-stream; charset=utf-8',
          'Cache-Control': 'no-cache',
          Connection: 'keep-alive',
          'X-Accel-Buffering': 'no',
        });
        res.write(
          formatSseData({
            event_name: 'work_panel_sse_connected',
            status: 'connected',
            tenant_id: tenantId,
            workspace_id: workspaceId,
            user_id: userId,
          }),
        );
        const unsubscribe = hub.subscribe(hubKey, res);
        const heartbeat = setInterval(() => {
          if (res.writableEnded || res.destroyed) return;
          try {
            res.write(formatSseData({ type: 'heartbeat', event_name: 'work_panel_sse_heartbeat' }));
          } catch {
            clearInterval(heartbeat);
          }
        }, cfg.heartbeatSec * 1000);
        req.on('close', () => {
          clearInterval(heartbeat);
          unsubscribe();
          if (!res.writableEnded) res.end();
        });
        return;
      }

      const billingMatch = url.pathname.match(BILLING_SSE_PATH_RE);
      if (billingMatch) {
        if (rejectUnlessGateway(req, res, cfg)) return;
        const tenantId = billingMatch[1];
        // 仅信任网关 forward-auth 注入的 X-User-Id；忽略 query user_id（防伪造）
        const userId = String(req.headers['x-user-id'] || '').trim();
        const hubKey = billingUserHubKey(userId);
        if (!hubKey) {
          return json(res, 401, { detail: 'X-User-Id required (gateway auth)' });
        }
        const maxConn = Number(cfg.maxConnections) || 0;
        if (maxConn > 0 && hub.totalConnections() >= maxConn) {
          return json(res, 503, {
            detail: 'sse connection limit reached',
            connections: hub.totalConnections(),
            maxConnections: maxConn,
          });
        }
        res.writeHead(200, {
          'Content-Type': 'text/event-stream; charset=utf-8',
          'Cache-Control': 'no-cache',
          Connection: 'keep-alive',
          'X-Accel-Buffering': 'no',
        });
        res.write(
          formatSseData({
            event_name: 'recharge_sse_connected',
            status: 'connected',
            tenant_id: tenantId,
            user_id: userId,
          }),
        );
        const unsubscribe = hub.subscribe(hubKey, res);
        const heartbeat = setInterval(() => {
          if (res.writableEnded || res.destroyed) return;
          try {
            res.write(formatSseData({ type: 'heartbeat', event_name: 'recharge_sse_heartbeat' }));
          } catch {
            clearInterval(heartbeat);
          }
        }, cfg.heartbeatSec * 1000);
        req.on('close', () => {
          clearInterval(heartbeat);
          unsubscribe();
          if (!res.writableEnded) res.end();
        });
        return;
      }

      const m = url.pathname.match(SSE_PATH_RE);
      if (m) {
        if (rejectUnlessGateway(req, res, cfg)) return;
        const taskId = m[3];
        const maxConn = Number(cfg.maxConnections) || 0;
        if (maxConn > 0 && hub.totalConnections() >= maxConn) {
          console.warn(
            `[taskSSE] event=sse_connection_limit_reached connections=${hub.totalConnections()} max=${maxConn}`,
          );
          return json(res, 503, {
            detail: 'sse connection limit reached',
            connections: hub.totalConnections(),
            maxConnections: maxConn,
          });
        }
        res.writeHead(200, {
          'Content-Type': 'text/event-stream; charset=utf-8',
          'Cache-Control': 'no-cache',
          Connection: 'keep-alive',
          'X-Accel-Buffering': 'no',
        });
        // 握手 ack：前端用 EventSource open / sseLive 展示连接态，不放用户可见启动文案
        const connected = {
          status: 'connected',
          message: '',
          progress: 0,
          event_name: 'server_status_update',
        };
        res.write(formatSseData(connected));

        const unsubscribe = hub.subscribe(taskId, res);
        const heartbeat = setInterval(() => {
          if (res.writableEnded || res.destroyed) return;
          try {
            res.write(formatSseData({ type: 'heartbeat', event_name: 'server_status_update' }));
          } catch {
            clearInterval(heartbeat);
          }
        }, cfg.heartbeatSec * 1000);

        req.on('close', () => {
          clearInterval(heartbeat);
          unsubscribe();
          if (!res.writableEnded) res.end();
        });
        return;
      }
    }

    json(res, 404, { detail: 'not found', path: url.pathname });
  }));

  return {
    server,
    hub,
    cfg,
    async close() {
      await new Promise((resolve) => server.close(resolve));
      if (busCloser) await busCloser.close();
    },
  };
}

export async function main() {
  const cfg = loadTaskSseConfig();
  const { server } = await createTaskSseServer(cfg);
  server.listen(cfg.port, cfg.host, () => {
    const gw = cfg.gatewayInternalSecret ? 'required' : 'MISSING(fail-closed)';
    console.log(
      `[taskSSE] listening on http://${cfg.host}:${cfg.port} transport=${cfg.transport} gatewaySecret=${gw}`,
    );
  });
}

if (process.env.TASK_SSE_SKIP_MAIN !== '1') {
  main().catch((err) => {
    console.error('[taskSSE] fatal', err);
    process.exit(1);
  });
}
