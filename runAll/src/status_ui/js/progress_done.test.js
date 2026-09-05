'use strict';

const { describe, it } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const {
  progressDoneKey,
  isProgressDoneReplay,
  shouldReconnectBulkProgressSSE,
  shouldAutoHideProgressPanel
} = require('./progress_done.js');

describe('progress done replay', () => {
  const emptyStartDone = {
    operation: 'start',
    phase: 'done',
    error: '',
    failed: 0,
    done: true
  };
  const lifecycleError = {
    operation: 'start',
    phase: 'error',
    error: 'lifecycle plan must include at least one service',
    failed: 0,
    done: true
  };

  it('treats the same terminal error as a replay (toast storm guard)', () => {
    const key = progressDoneKey(lifecycleError);
    assert.equal(isProgressDoneReplay('', lifecycleError), false);
    assert.equal(isProgressDoneReplay(key, lifecycleError), true);
  });

  it('does not reconnect SSE after a done bulk snapshot', () => {
    assert.equal(shouldReconnectBulkProgressSSE({
      run_id: 'start-all-1',
      event: lifecycleError
    }), false);
    assert.equal(shouldReconnectBulkProgressSSE({
      run_id: 'start-all-1',
      event: { done: false, operation: 'start' }
    }), true);
  });

  it('keeps empty start-all no-op distinct from the lifecycle error', () => {
    assert.notEqual(progressDoneKey(emptyStartDone), progressDoneKey(lifecycleError));
  });
});

function idleHandlerHidesPanel(src) {
  const parts = src.split(/ev\.phase === 'idle'/);
  for (let i = 1; i < parts.length; i++) {
    const window = parts[i].slice(0, 500);
    if (window.includes("classList.remove('is-active')") || window.includes('classList.remove("is-active")')) {
      return true;
    }
  }
  return false;
}

describe('progress panel stays until manual clear', () => {
  const doneSuccess = { done: true, failed: 0, phase: 'done', operation: 'restart' };
  const doneFailed = { done: true, failed: 2, phase: 'error', operation: 'restart' };
  const doneCancelled = { done: true, failed: 0, phase: 'cancelled', operation: 'restart' };

  it('does not auto-hide after restart success, failure, or cancel', () => {
    assert.equal(shouldAutoHideProgressPanel(doneSuccess), false);
    assert.equal(shouldAutoHideProgressPanel(doneFailed), false);
    assert.equal(shouldAutoHideProgressPanel(doneCancelled), false);
  });

  it('does not treat SSE idle as a user dismiss', () => {
    assert.equal(shouldAutoHideProgressPanel({ phase: 'idle', done: true }), false);
  });

  it('does not schedule fade-out timers in progress SSE fragments', () => {
    const files = ['02.js', '03.js', '08.js', '11.js', '14.js'];
    for (const name of files) {
      const src = fs.readFileSync(path.join(__dirname, name), 'utf8');
      assert.doesNotMatch(src, /_progressAutoHideTimer\s*=\s*setTimeout/, name + ' still auto-hides');
      assert.equal(idleHandlerHidesPanel(src), false, name + ' idle hides the progress panel');
    }
    const src02 = fs.readFileSync(path.join(__dirname, '02.js'), 'utf8');
    assert.doesNotMatch(src02, /\? 10000 : 5000/);
    assert.doesNotMatch(src02, /Auto-hide faster for single service/);
  });

  it('does not keep the dead _progressAutoHideTimer clear logic (OPT-20260901-003)', () => {
    // The panel stays until manual 清空; any remaining reference to the dead
    // auto-hide timer misleads future edits into thinking auto-hide still exists.
    const files = ['01.js', '02.js', '03.js', '08.js', '11.js', '14.js', 'progress_done.js'];
    for (const name of files) {
      const src = fs.readFileSync(path.join(__dirname, name), 'utf8');
      assert.doesNotMatch(src, /_progressAutoHideTimer/, name + ' still references the dead auto-hide timer');
    }
  });

  it('index.html exposes a manual clear button', () => {
    const html = fs.readFileSync(path.join(__dirname, '..', 'index.html'), 'utf8');
    assert.match(html, /id="prog-clear-logs-btn"/);
    assert.match(html, /onclick="dismissProgressPanel\(\)"/);
    assert.match(html, /清空/);
  });
});
