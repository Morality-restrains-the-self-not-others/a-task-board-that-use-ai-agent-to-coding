process.env.TASK_SSE_SKIP_MAIN = '1';
import test from 'node:test';
import assert from 'node:assert/strict';
import { SseHub } from '../src/sseHub.mjs';
import {
  normalizeInboundMessage,
  formatSseData,
  billingUserHubKey,
  workspaceHubKey,
} from '../src/messageNormalize.mjs';
import { otelTraceIdHex, resolveInboundCorrelation, outboundTraceHeaders } from '../src/traceMiddleware.mjs';

// 静态 import 会被 ESM hoisting 提前求值，此时 TASK_SSE_SKIP_MAIN 尚未生效，
// server.mjs 的 main() 会启动（监听端口 + 订阅 Redis）导致测试结束后进程不退出。
// 与 billingSseAuth.test.mjs 一致改用动态导入。
const { gatewayInternalSecretOk } = await import('../src/server.mjs');

// --- SseHub extended tests ---

test('SseHub: empty taskId subscribe returns noop', () => {
  const hub = new SseHub();
  const unsub = hub.subscribe('', {});
  assert.equal(typeof unsub, 'function');
  assert.equal(hub.totalConnections(), 0);
});

test('SseHub: publish to no subscribers returns 0', () => {
  const hub = new SseHub();
  assert.equal(hub.publish('nonexist', {}), 0);
});

test('SseHub: unsubscribed client is cleaned up', () => {
  const hub = new SseHub();
  const res = { writableEnded: false, destroyed: false, write() {} };
  const unsub = hub.subscribe('t-clean', res);
  assert.equal(hub.totalConnections(), 1);
  unsub();
  assert.equal(hub.totalConnections(), 0);
  assert.equal(hub.connectionCount('t-clean'), 0);
});

test('SseHub: destroyed client skipped on publish', () => {
  const hub = new SseHub();
  const res = { writableEnded: false, destroyed: true, write() {} };
  hub.subscribe('t-destroyed', res);
  assert.equal(hub.publish('t-destroyed', { status: 'ok' }), 0);
  assert.equal(hub.totalConnections(), 0);
});

test('SseHub: publish to multiple subscribers delivers to all', () => {
  const hub = new SseHub();
  const c1 = [], c2 = [];
  hub.subscribe('t-multi', { writableEnded: false, destroyed: false, write(s) { c1.push(s); } });
  hub.subscribe('t-multi', { writableEnded: false, destroyed: false, write(s) { c2.push(s); } });
  const sent = hub.publish('t-multi', { status: 'broadcast' });
  assert.equal(sent, 2);
  assert.match(c1.join(''), /broadcast/);
  assert.match(c2.join(''), /broadcast/);
});

test('SseHub: connectionCount returns correct count', () => {
  const hub = new SseHub();
  assert.equal(hub.connectionCount('none'), 0);
  const res = { writableEnded: false, destroyed: false, write() {} };
  hub.subscribe('t-count', res);
  assert.equal(hub.connectionCount('t-count'), 1);
});

test('SseHub: double unsubscribe is safe', () => {
  const hub = new SseHub();
  const res = { writableEnded: false, destroyed: false, write() {} };
  const unsub = hub.subscribe('t-double', res);
  unsub();
  unsub(); // should not throw
  assert.equal(hub.totalConnections(), 0);
});

test('SseHub: subscribe same taskId multiple times', () => {
  const hub = new SseHub();
  const r1 = { writableEnded: false, destroyed: false, write() {} };
  const r2 = { writableEnded: false, destroyed: false, write() {} };
  hub.subscribe('t-same', r1);
  hub.subscribe('t-same', r2);
  assert.equal(hub.connectionCount('t-same'), 2);
  assert.equal(hub.totalConnections(), 2);
});

test('SseHub: whitespace-only key treated as empty', () => {
  const hub = new SseHub();
  const res = { writableEnded: false, destroyed: false, write() {} };
  const unsub = hub.subscribe('   ', res);
  assert.equal(typeof unsub, 'function');
  assert.equal(hub.totalConnections(), 0);
});

// --- gatewayInternalSecretOk tests ---

test('gatewayInternalSecretOk rejects empty expected', () => {
  const req = { headers: { 'x-taskgateway-internal-secret': 'secret' } };
  assert.equal(gatewayInternalSecretOk(req, ''), false);
});

test('gatewayInternalSecretOk rejects empty header', () => {
  const req = { headers: {} };
  assert.equal(gatewayInternalSecretOk(req, 'secret'), false);
});

test('gatewayInternalSecretOk accepts matching secret', () => {
  const req = { headers: { 'x-taskgateway-internal-secret': 'my-secret' } };
  assert.equal(gatewayInternalSecretOk(req, 'my-secret'), true);
});

test('gatewayInternalSecretOk rejects mismatched secret', () => {
  const req = { headers: { 'x-taskgateway-internal-secret': 'wrong' } };
  assert.equal(gatewayInternalSecretOk(req, 'correct'), false);
});

// --- messageNormalize extended edge cases ---

test('normalizeInboundMessage: null input', () => {
  assert.equal(normalizeInboundMessage(null), null);
});

test('normalizeInboundMessage: undefined input', () => {
  assert.equal(normalizeInboundMessage(undefined), null);
});

test('normalizeInboundMessage: non-object input', () => {
  assert.equal(normalizeInboundMessage('string'), null);
});

test('normalizeInboundMessage: empty object', () => {
  assert.equal(normalizeInboundMessage({}), null);
});

test('normalizeInboundMessage: no task_id or status_data', () => {
  assert.equal(normalizeInboundMessage({ foo: 'bar' }), null);
});

test('normalizeInboundMessage: with message wrapper', () => {
  const got = normalizeInboundMessage({
    task_id: 't99',
    message: { status: 'running', progress: 50 },
  });
  assert.equal(got.taskId, 't99');
  assert.equal(got.statusData.status, 'running');
  assert.equal(got.statusData.progress, 50);
});

test('normalizeInboundMessage: nested data envelope', () => {
  const got = normalizeInboundMessage({
    data: { task_id: 'tn', status_data: { progress: 100 } },
  });
  assert.equal(got.taskId, 'tn');
  assert.equal(got.statusData.progress, 100);
});

test('normalizeInboundMessage: preserves event_name if present', () => {
  const got = normalizeInboundMessage({
    task_id: 't-custom',
    status_data: { event_name: 'custom_event', detail: 'x' },
  });
  assert.equal(got.statusData.event_name, 'custom_event');
});

test('billingUserHubKey: empty input', () => {
  assert.equal(billingUserHubKey(''), '');
  assert.equal(billingUserHubKey(null), '');
  assert.equal(billingUserHubKey(undefined), '');
});

test('workspaceHubKey: empty input', () => {
  assert.equal(workspaceHubKey(''), '');
  assert.equal(workspaceHubKey('  '), '');
});

test('formatSseData: complex objects', () => {
  const result = formatSseData({ status: 'ok', nested: { a: 1, b: [1, 2, 3] } });
  assert.match(result, /^data: /);
  assert.match(result, /"nested"/);
  assert.match(result, /\n\n$/);
});

// --- traceMiddleware extended tests ---

test('resolveInboundCorrelation generates new trace when no headers', () => {
  const corr = resolveInboundCorrelation({ headers: {} });
  assert.ok(corr.traceId);
  assert.ok(corr.spanId);
  assert.match(corr.traceId, /^tsse-/);
  assert.match(corr.spanId, /^[0-9a-f]{16}$/);
});

test('resolveInboundCorrelation uses X-Trace-Id header', () => {
  const corr = resolveInboundCorrelation({
    headers: { 'x-trace-id': 'custom-trace-abc12345' },
  });
  assert.equal(corr.traceId, 'custom-trace-abc12345');
  assert.match(corr.spanId, /^[0-9a-f]{16}$/);
});

test('resolveInboundCorrelation uses traceparent header', () => {
  const corr = resolveInboundCorrelation({
    headers: {
      traceparent: '00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01',
    },
  });
  assert.equal(corr.traceId, 'tp-4bf92f3577b34da6a3ce929d0e0e4736');
  assert.equal(corr.parentSpanId, '00f067aa0ba902b7');
});

test('outboundTraceHeaders generates proper headers', () => {
  const headers = outboundTraceHeaders({
    traceId: 'trace-123',
    spanId: 'abcdef1234567890',
  });
  assert.equal(headers['X-Trace-Id'], 'trace-123');
  assert.equal(headers['X-Parent-Span-Id'], 'abcdef1234567890');
  assert.ok(headers.traceparent);
  assert.match(headers.traceparent, /^00-/);
});

test('outboundTraceHeaders empty correlation', () => {
  const headers = outboundTraceHeaders({});
  assert.equal(Object.keys(headers).length, 0);
});

test('otelTraceIdHex fallback for non-uuid', () => {
  const hex = otelTraceIdHex('arbitrary-string');
  assert.equal(hex.length, 32);
  assert.match(hex, /^[0-9a-f]{32}$/);
});
