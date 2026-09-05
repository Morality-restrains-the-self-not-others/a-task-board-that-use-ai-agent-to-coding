// Progress-done replay helpers for Status UI bulk operations.
// Empty/error terminal events must not reconnect SSE or re-toast on /api/status poll.

function progressDoneKey(ev) {
  ev = ev || {};
  return String(ev.operation || '') + '\0' + String(ev.phase || '') + '\0' + String(ev.error || '') + '\0' + String(ev.failed || 0);
}

function isProgressDoneReplay(prevKey, ev) {
  return prevKey === progressDoneKey(ev);
}

function shouldReconnectBulkProgressSSE(snap) {
  return !(snap && snap.event && snap.event.done);
}

function shouldAutoHideProgressPanel(_ev) {
  return false;
}

function updateProgressClearButton(visible) {
  var btn = document.getElementById('prog-clear-logs-btn');
  if (!btn) return;
  btn.style.display = visible ? '' : 'none';
}

function keepProgressPanelAfterStreamEnd() {
  updateProgressClearButton(true);
}

function dismissProgressPanel() {
  // Anti-Replay-OK: local UI dismiss, no write API
  var panel = document.getElementById('start-all-progress-panel');
  if (!panel) return;
  panel.classList.remove('is-active', 'is-fading', 'is-done', 'has-errors', 'is-cancelled');
  if (typeof updateProgressCancelButton === 'function') updateProgressCancelButton(false, false);
  if (typeof updateProgressCopyButton === 'function') updateProgressCopyButton(false);
  updateProgressClearButton(false);
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    progressDoneKey: progressDoneKey,
    isProgressDoneReplay: isProgressDoneReplay,
    shouldReconnectBulkProgressSSE: shouldReconnectBulkProgressSSE,
    shouldAutoHideProgressPanel: shouldAutoHideProgressPanel
  };
}
