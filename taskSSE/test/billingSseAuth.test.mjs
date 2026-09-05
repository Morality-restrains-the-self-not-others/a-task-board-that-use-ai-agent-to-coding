import test from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';

process.env.TASK_SSE_SKIP_MAIN = '1';
const { createTaskSseServer, gatewayInternalSecretOk } = await import('../src/server.mjs');

const GW_SECRET = 'test-gateway-secret';

async function withServer(fn) {
  const { server, close } = await createTaskSseServer({
    host: '127.0.0.1',
    port: 0,
    transport: 'redis',
    secret: 'test',
    gatewayInternalSecret: GW_SECRET,
    heartbeatSec: 30,
    maxConnections: 50,
    redis: { host: '127.0.0.1', port: 6379, channelPrefix: 'sse:' },
    kafka: { bootstrapServers: 'localhost:9093', topic: 'sse-message', groupId: 't' },
    enabled: true,
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  const { port } = server.address();
  try {
    await fn(port);
  } finally {
    await close();
  }
}

function request(port, path, headers = {}) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      { hostname: '127.0.0.1', port, path, method: 'GET', headers },
      (res) => {
        const chunks = [];
        res.on('data', (c) => chunks.push(c));
        res.on('end', () => {
          resolve({
            status: res.statusCode,
            body: Buffer.concat(chunks).toString('utf8'),
          });
        });
      },
    );
    req.on('error', reject);
    req.end();
  });
}

test('gatewayInternalSecretOk fail-closed when expected empty', () => {
  assert.equal(
    gatewayInternalSecretOk({ headers: { 'x-taskgateway-internal-secret': 'x' } }, ''),
    false,
  );
});

test('billing SSE rejects forged X-User-Id without gateway secret', async () => {
  await withServer(async (port) => {
    const res = await request(port, '/api/sse/recharge-events/tenant_id/1/', {
      'X-User-Id': 'spoof',
    });
    assert.equal(res.status, 403);
    assert.match(res.body, /gateway internal secret/);
  });
});

test('billing SSE rejects query user_id without gateway secret', async () => {
  await withServer(async (port) => {
    const res = await request(
      port,
      '/api/sse/recharge-events/tenant_id/1/?user_id=spoof',
    );
    assert.equal(res.status, 403);
  });
});

test('billing SSE accepts gateway secret + X-User-Id', async () => {
  await withServer(async (port) => {
    const res = await new Promise((resolve, reject) => {
      const req = http.request(
        {
          hostname: '127.0.0.1',
          port,
          path: '/api/sse/recharge-events/tenant_id/1/',
          method: 'GET',
          headers: {
            'X-User-Id': '42',
            'X-TaskGateway-Internal-Secret': GW_SECRET,
          },
        },
        (r) => {
          let buf = '';
          r.on('data', (c) => {
            buf += c.toString('utf8');
            if (buf.includes('recharge_sse_connected')) {
              r.destroy();
              resolve({ status: r.statusCode, body: buf });
            }
          });
          r.on('error', () => resolve({ status: r.statusCode, body: buf }));
          r.on('end', () => resolve({ status: r.statusCode, body: buf }));
        },
      );
      req.on('error', reject);
      req.end();
    });
    assert.equal(res.status, 200);
    assert.match(res.body, /"user_id":"42"/);
  });
});

test('task startup SSE requires gateway secret', async () => {
  await withServer(async (port) => {
    const denied = await request(
      port,
      '/api/sse/server-startup-status/tenant_id/1/workspace_id/2/task_id/3/',
    );
    assert.equal(denied.status, 403);

    const res = await new Promise((resolve, reject) => {
      const req = http.request(
        {
          hostname: '127.0.0.1',
          port,
          path: '/api/sse/server-startup-status/tenant_id/1/workspace_id/2/task_id/3/',
          method: 'GET',
          headers: { 'X-TaskGateway-Internal-Secret': GW_SECRET },
        },
        (r) => {
          let buf = '';
          r.on('data', (c) => {
            buf += c.toString('utf8');
            if (buf.includes('server_status_update')) {
              r.destroy();
              resolve({ status: r.statusCode, body: buf });
            }
          });
          r.on('error', () => resolve({ status: r.statusCode, body: buf }));
          r.on('end', () => resolve({ status: r.statusCode, body: buf }));
        },
      );
      req.on('error', reject);
      req.end();
    });
    assert.equal(res.status, 200);
  });
});

test('work-panel SSE rejects without gateway secret', async () => {
  await withServer(async (port) => {
    const res = await request(port, '/api/sse/work-panel-events/tenant_id/1/workspace_id/ws-a/');
    assert.equal(res.status, 403);
  });
});

test('work-panel SSE accepts gateway secret and emits connected', async () => {
  await withServer(async (port) => {
    const res = await new Promise((resolve, reject) => {
      const req = http.request(
        {
          hostname: '127.0.0.1',
          port,
          path: '/api/sse/work-panel-events/tenant_id/1/workspace_id/ws-a/',
          method: 'GET',
          headers: {
            'X-User-Id': '7',
            'X-TaskGateway-Internal-Secret': GW_SECRET,
          },
        },
        (r) => {
          let buf = '';
          r.on('data', (c) => {
            buf += c.toString('utf8');
            if (buf.includes('work_panel_sse_connected')) {
              r.destroy();
              resolve({ status: r.statusCode, body: buf });
            }
          });
          r.on('error', () => resolve({ status: r.statusCode, body: buf }));
          r.on('end', () => resolve({ status: r.statusCode, body: buf }));
        },
      );
      req.on('error', reject);
      req.end();
    });
    assert.equal(res.status, 200);
    assert.match(res.body, /"workspace_id":"ws-a"/);
  });
});
