import test from 'node:test';
import assert from 'node:assert/strict';
import {
  normalizeInboundMessage,
  formatSseData,
  workspaceHubKey,
} from '../src/messageNormalize.mjs';
import { SseHub } from '../src/sseHub.mjs';
import { loadTaskSseConfig } from '../src/config.mjs';

test('normalizeInboundMessage: redis payload', () => {
  const got = normalizeInboundMessage({
    task_id: 't1',
    status_data: { status: 'ok', message: 'hi' },
  });
  assert.equal(got.taskId, 't1');
  assert.equal(got.statusData.message, 'hi');
  assert.equal(got.statusData.event_name, 'server_status_update');
});

test('normalizeInboundMessage: billing user envelope without task_id', () => {
  const got = normalizeInboundMessage({
    user_id: '99',
    status_data: { status: 'completed', out_trade_no: 'WX1' },
  });
  assert.equal(got.taskId, 'billing:user:99');
  assert.equal(got.statusData.event_name, 'recharge_completed');
  assert.equal(got.statusData.out_trade_no, 'WX1');
});

test('workspaceHubKey formats workspace id', () => {
  assert.equal(workspaceHubKey('ws-1'), 'workspace:ws-1');
  assert.equal(workspaceHubKey(''), '');
});

test('normalizeInboundMessage: work panel workspace envelope', () => {
  const got = normalizeInboundMessage({
    task_id: 'workspace:ws-9',
    status_data: {
      task_id: 'task-1',
      workspace_id: 'ws-9',
      progress_column_id: 'col-done',
    },
  });
  assert.equal(got.taskId, 'workspace:ws-9');
  assert.equal(got.statusData.event_name, 'task_status_changed');
  assert.equal(got.statusData.task_id, 'task-1');
});

test('normalizeInboundMessage: work panel derives hub from workspace_id', () => {
  const got = normalizeInboundMessage({
    workspace_id: 'ws-2',
    status_data: { task_id: 't9', progress_column_id: 'c1' },
  });
  assert.equal(got.taskId, 'workspace:ws-2');
  assert.equal(got.statusData.event_name, 'task_status_changed');
});

test('normalizeInboundMessage: kafka envelope', () => {
  const got = normalizeInboundMessage({
    event_type: 'SSE_MESSAGE',
    data: { task_id: 't2', status_data: { progress: 50 } },
  });
  assert.equal(got.taskId, 't2');
  assert.equal(got.statusData.progress, 50);
});

test('SseHub publish delivers to subscriber', () => {
  const hub = new SseHub();
  const chunks = [];
  const res = {
    writableEnded: false,
    destroyed: false,
    write(s) {
      chunks.push(s);
    },
  };
  hub.subscribe('task-1', res);
  hub.publish('task-1', { status: 'running' });
  assert.match(chunks.join(''), /"status":"running"/);
});

test('formatSseData prefix', () => {
  assert.match(formatSseData({ a: 1 }), /^data: /);
});

test('loadTaskSseConfig defaults', () => {
  const cfg = loadTaskSseConfig({ TASK_SSE_TRANSPORT: 'redis', TASK_SSE_PORT: '8798' });
  assert.equal(cfg.transport, 'redis');
  assert.equal(cfg.port, 8798);
});

test('loadTaskSseConfig loads gatewayInternalSecret from env override', () => {
  const cfg = loadTaskSseConfig({
    TASK_SSE_TRANSPORT: 'redis',
    TASK_GATEWAY_INTERNAL_SECRET: 'from-env-gw',
  });
  assert.equal(cfg.gatewayInternalSecret, 'from-env-gw');
});

test('loadTaskSseConfig merges docker-infra redis host', () => {
  // conf/gateway/task-sse/docker-infra.yaml uses ${INFRA_HOST:-10.2.150.68};
  // pin INFRA_HOST so this assertion is deployment-independent.
  const prevInfraHost = process.env.INFRA_HOST;
  process.env.INFRA_HOST = '127.0.0.1';
  try {
    const cfg = loadTaskSseConfig({});
    assert.equal(cfg.redis.host, '127.0.0.1');
    assert.equal(cfg.redis.port, 6379);
  } finally {
    if (prevInfraHost === undefined) {
      delete process.env.INFRA_HOST;
    } else {
      process.env.INFRA_HOST = prevInfraHost;
    }
  }
});

test('loadTaskSseConfig maxConnections default', () => {
  const cfg = loadTaskSseConfig({ TASK_SSE_MAX_CONNECTIONS: '12' });
  assert.equal(cfg.maxConnections, 12);
});

test('SseHub totalConnections counts subscribers', () => {
  const hub = new SseHub();
  const res = { writableEnded: false, destroyed: false, write() {} };
  hub.subscribe('t1', res);
  hub.subscribe('t2', res);
  assert.equal(hub.totalConnections(), 2);
});
