'use strict';

const { describe, it } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const path = require('path');

const dir = __dirname;
const css = fs.readFileSync(path.join(dir, '..', 'css', '02.css'), 'utf8');
const js = fs.readFileSync(path.join(dir, '02.js'), 'utf8');

describe('progress error banner (#prog-error) remains fully readable', () => {
  it('02.css wraps long latest-error text instead of clipping it', () => {
    const block = css.slice(css.indexOf('.progress-error {'), css.indexOf('.progress-error.is-visible'));
    assert.match(block, /overflow-wrap:\s*anywhere/);
    assert.match(block, /white-space:\s*pre-wrap/);
  });

  it('active progress panel scrolls instead of overflow:hidden clipping errors', () => {
    assert.match(css, /\.start-all-progress\.is-active\s*\{[^}]*overflow-y:\s*auto/s);
  });

  it('updateProgress copies the full error onto title for hover', () => {
    const idx = js.indexOf("textContent = '最新错误: ' + ev.error");
    assert.ok(idx !== -1, 'latest-error assignment must exist');
    const window = js.slice(idx, idx + 280);
    assert.match(window, /\.title\s*=/);
  });

  it('updateProgress stamps data-traceId onto #prog-error when run_id present', () => {
    const idx = js.indexOf("textContent = '最新错误: ' + ev.error");
    assert.ok(idx !== -1, 'latest-error assignment must exist');
    const window = js.slice(idx, idx + 500);
    // Must gate on a real run_id, never fall back to a literal "unknown".
    assert.match(window, /if\s*\(ev\.run_id\)/);
    assert.match(window, /setAttribute\('data-traceId',\s*ev\.run_id\)/);
    assert.match(window, /removeAttribute\('data-traceId'\)/);
  });

  it('showProgress clears a stale data-traceId on a fresh run', () => {
    const reset = js.slice(js.indexOf('function showProgress(op)'), js.indexOf('function updateProgressCancelButton'));
    assert.match(reset, /removeAttribute\('data-traceId'\)/);
  });
});
