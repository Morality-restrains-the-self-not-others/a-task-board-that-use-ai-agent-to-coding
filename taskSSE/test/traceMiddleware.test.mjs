import test from 'node:test';
import assert from 'node:assert/strict';
import { withTraceMiddleware, otelTraceIdHex, TRACE_HEADER } from '../src/traceMiddleware.mjs';

test('otelTraceIdHex strips UUID hyphens', () => {
  assert.equal(
    otelTraceIdHex('b7906027-3792-4215-9fab-6b044565a272'),
    'b7906027379242159fab6b044565a272',
  );
});

test('withTraceMiddleware rejects trace-id-only requests', async () => {
  const handler = withTraceMiddleware('task-sse-test', (_req, res) => {
    res.writeHead(200);
    res.end('ok');
  });
  const captured = { statusCode: 0, body: '' };
  await new Promise((resolve) => {
    const req = {
      method: 'GET',
      url: '/health',
      headers: { [TRACE_HEADER]: 'legacy-trace-only12345' },
    };
    const res = {
      statusCode: 200,
      headers: {},
      setHeader(name, value) {
        this.headers[name.toLowerCase()] = value;
      },
      writeHead(code) {
        this.statusCode = code;
        captured.statusCode = code;
      },
      end(chunk) {
        captured.body = String(chunk || '');
        resolve();
      },
      on() {},
    };
    handler(req, res);
  });
  assert.equal(captured.statusCode, 400);
  assert.match(captured.body, /trace propagation incomplete/);
});

test('withTraceMiddleware propagates trace/span headers and logs http_request', async () => {
  const logs = [];
  const origWrite = process.stdout.write.bind(process.stdout);
  process.stdout.write = (chunk) => {
    logs.push(String(chunk));
    return true;
  };
  try {
    const handler = withTraceMiddleware('task-sse-test', (_req, res) => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end('{"ok":true}');
    });
    await new Promise((resolve) => {
      const req = {
        method: 'GET',
        url: '/health',
        headers: {
          [TRACE_HEADER]: 'web-trace-test12345678',
          'x-parent-span-id': 'a1b2c3d4e5f67890',
        },
      };
      const res = {
        statusCode: 200,
        headers: {},
        setHeader(name, value) {
          this.headers[name.toLowerCase()] = value;
        },
        writeHead(code, _headers) {
          this.statusCode = code;
        },
        end() {
          this.emit?.('finish');
          resolve();
        },
        on(event, fn) {
          if (event === 'finish') this.emit = fn;
        },
      };
      handler(req, res);
    });
    assert.equal(logs.length >= 1, true);
    const payload = JSON.parse(logs[0].trim());
    assert.equal(payload.msg, 'http_request');
    assert.equal(payload.trace_id, 'web-trace-test12345678');
    assert.equal(payload.parent_span_id, 'a1b2c3d4e5f67890');
    assert.match(payload.span_id, /^[0-9a-f]{16}$/);
    assert.equal(payload.service, 'task-sse-test');
  } finally {
    process.stdout.write = origWrite;
  }
});
