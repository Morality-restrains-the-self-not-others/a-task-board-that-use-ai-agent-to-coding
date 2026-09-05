'use strict';

const { describe, it } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const dir = __dirname;

describe('precise-restart empty registry (clone-run SOURCE_ROOT)', () => {
  it('05.js empty click names the registry file_path from the API', () => {
    const src = fs.readFileSync(path.join(dir, '05.js'), 'utf8');
    assert.match(src, /result\.file_path/);
    assert.match(src, /当前读取/);
    assert.match(src, /SOURCE_ROOT/);
  });

  it('05.js click fills empty registry from dirty-tree scan before prompting', () => {
    const src = fs.readFileSync(path.join(dir, '05.js'), 'utf8');
    assert.match(src, /fill_from_scan=1/);
    assert.match(src, /showEmptyPreciseRestartRegistry/);
  });

  it('11.js empty label title shows the absolute registry path', () => {
    const src = fs.readFileSync(path.join(dir, '11.js'), 'utf8');
    assert.match(src, /result\.file_path/);
    assert.match(src, /登记文件:/);
  });

  it('11.js treats empty POST as a no-op prompt, not 精准编译重启失败', () => {
    const src = fs.readFileSync(path.join(dir, '11.js'), 'utf8');
    assert.match(src, /status === 'empty'/);
    assert.match(src, /showEmptyPreciseRestartRegistry/);
    assert.match(src, /no registered services/);
    const failIdx = src.indexOf("精准编译重启失败");
    const emptyIdx = src.indexOf("status === 'empty'");
    assert.ok(emptyIdx !== -1 && failIdx !== -1, 'both empty-status and failure banner must exist');
    assert.ok(emptyIdx < failIdx, 'empty status must be handled before the failure banner');
  });

  it('OPT-20260902-019: 05.js helper uses an info toast, not the red request-error banner', () => {
    const src = fs.readFileSync(path.join(dir, '05.js'), 'utf8');
    const start = src.indexOf('function showEmptyPreciseRestartRegistry');
    assert.ok(start !== -1, 'helper must exist in 05.js');
    const fn = src.slice(start, src.indexOf('\n}\n', start));
    assert.ok(!fn.includes('showRequestError'), 'helper must not call showRequestError');
    assert.match(fn, /showToast\(/);
    assert.match(fn, /type:\s*'info'/);
    assert.ok(!fn.includes('data-traceId'), 'info toast is not a request failure and must not set data-traceId');
  });
});
