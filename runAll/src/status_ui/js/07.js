// ── Execution Queue for Bulk Operations ──
// Hooks into SSE progress via _onBulkProgressDone callback (set in 02.js updateProgress)
var _onBulkProgressDone = null;

var bulkQueue = {
  items: [],
  processing: false,
  currentOp: null,
  // Live progress snapshot for queue-status-bar (started/total/current/failed).
  progress: null,
  // Track which type has confirm dialogs to avoid double-confirm on queue replay
  confirmFlags: {},

  applyProgress: function (ev, op) {
    if (!ev || typeof ev !== 'object') return;
    var started = typeof ev.started === 'number' ? ev.started : 0;
    var total = typeof ev.total === 'number' ? ev.total : 0;
    var failed = typeof ev.failed === 'number' ? ev.failed : 0;
    var remaining = typeof ev.remaining === 'number' ? ev.remaining : Math.max(0, total - started - failed);
    this.progress = {
      op: op || ev.operation || '',
      started: started,
      total: total,
      failed: failed,
      remaining: remaining,
      current: ev.current || '',
      error: ev.error || '',
      done: !!ev.done,
      phase: ev.phase || ''
    };
    if (ev.done) {
      // Keep last snapshot briefly for render; clear on next processNext/idle.
    }
    this.renderQueueBar();
  },

  waitingProgressHint: function (op) {
    var key = String(op || '');
    if (key === 'start' || key === 'start-all') return '启动/健康检查中…';
    if (key === 'stop' || key === 'stop-all') return '关闭中…';
    if (key === 'build' || key === 'build-all') return '编译中…';
    if (key === 'restart' || key === 'restart-all' || key === 'precise-restart') return '重启/健康检查中…';
    if (key === 'init-db') return '初始化中…';
    if (key === 'clear-db') return '清空中…';
    return '执行中…';
  },

  formatProgressDetail: function () {
    var p = this.progress;
    if (!p || !p.total) return '';
    var parts = [];
    parts.push((p.started + p.failed) + '/' + p.total);
    if (p.current) parts.push(p.current);
    else if (!p.done) parts.push(this.waitingProgressHint(p.op));
    if (p.failed > 0) parts.push('失败 ' + p.failed);
    if (p.error) parts.push(p.error.length > 80 ? (p.error.slice(0, 80) + '…') : p.error);
    return parts.join(' · ');
  },

  adoptExternalOp: function (type, label, ev) {
    if (!type) return;
    if (this.currentOp && this.currentOp.status === 'running') {
      if (ev) this.applyProgress(ev, type === 'precise-restart' ? 'precise-restart' : (ev.operation || type));
      return;
    }
    var self = this;
    this.processing = true;
    this.currentOp = {
      id: 'external-' + type,
      type: type,
      label: label || type,
      status: 'running',
      external: true,
      fn: null
    };
    if (!_onBulkProgressDone) {
      _onBulkProgressDone = function (err) {
        if (err) {
          showToast('「' + (self.currentOp ? self.currentOp.label : type) + '」执行出错: ' + err, { type: 'error' });
        }
        self.currentOp = null;
        self.processing = false;
        self.progress = null;
        _onBulkProgressDone = null;
        self.render();
      };
    }
    if (ev) this.applyProgress(ev, type === 'precise-restart' ? 'precise-restart' : (ev.operation || type));
    else this.render();
  },

  clearExternalOpIfIdle: function () {
    if (!this.currentOp || !this.currentOp.external || this.currentOp.status !== 'running') {
      return;
    }
    // Backend finished (no active_bulk_progress) while UI still shows external op.
    this.currentOp = null;
    this.processing = this.items.length > 0;
    this.progress = null;
    if (_onBulkProgressDone) _onBulkProgressDone = null;
    this.render();
    if (this.processing) {
      var self = this;
      setTimeout(function () { self.processNext(); }, 200);
    }
  },

  enqueue: function (type, label, fn, needsConfirm) {
    // If the exact same type is running, block
    if (this.currentOp && this.currentOp.type === type && this.currentOp.status === 'running') {
      showToast('「' + label + '」正在执行中，请稍候', { type: 'warning', duration: 3000 });
      return false;
    }
    // If same type is pending, block
    for (var i = 0; i < this.items.length; i++) {
      if (this.items[i].type === type) {
        showToast('「' + label + '」已在队列中，请等待执行', { type: 'warning', duration: 3000 });
        return false;
      }
    }

    var item = {
      id: 'q-' + Date.now(),
      type: type,
      label: label,
      status: 'pending',
      fn: fn,
      needsConfirm: needsConfirm
    };
    this.items.push(item);
    this.render();
    this.syncPendingToServer();
    if (!this.processing) {
      this.processNext();
    }
    return true;
  },

  processNext: async function () {
    if (this.items.length === 0) {
      this.processing = false;
      this.currentOp = null;
      this.render();
      return;
    }
    this.processing = true;
    var item = this.items.shift();
    item.status = 'running';
    this.currentOp = item;
    this.progress = null;
    this.render();
    await this.syncPendingToServer();

    var self = this;
    // Set completion callback before starting
    _onBulkProgressDone = function (err) {
      if (err) {
        item.status = 'failed';
        item.error = err;
        showToast('「' + item.label + '」执行出错: ' + err, { type: 'error' });
      } else {
        item.status = 'completed';
      }
      self.currentOp = null;
      self.progress = null;
      _onBulkProgressDone = null;
      self.render();
      // Process next after brief delay so UI can show completion
      setTimeout(function () { self.processNext(); }, 600);
    };

    // Also hook SSE error cases
    var origOnError = window._bulkSSEOnError;
    window._bulkSSEOnError = function (errMsg) {
      if (_onBulkProgressDone) {
        var cb = _onBulkProgressDone;
        _onBulkProgressDone = null;
        cb(errMsg || 'SSE 连接中断');
      }
      if (typeof origOnError === 'function') origOnError(errMsg);
    };

    try {
      var ret = item.fn();
      if (ret && typeof ret.then === 'function') {
        await ret;
      }
    } catch (e) {
      item.status = 'failed';
      item.error = e.message;
      showToast('「' + item.label + '」执行出错: ' + e.message, { type: 'error' });
      self.currentOp = null;
      _onBulkProgressDone = null;
      self.render();
      setTimeout(function () { self.processNext(); }, 600);
    }
  },

  interruptCurrent: async function () {
    if (!(this.currentOp && this.currentOp.status === 'running')) {
      return false;
    }
    var label = this.currentOp.label;
    // Disconnect ALL SSE sources first to prevent stale events from
    // triggering the next operation's _onBulkProgressDone callback.
    disconnectStartAllSSE();
    disconnectStopAllSSE();
    disconnectBuildAllSSE();
    disconnectRestartAllSSE();
    disconnectPreciseRestartSSE();
    // Must await: precise-restart previously fell through to start-all/cancel
    // and UI cleared while the backend kept running.
    try {
      var cancelOp = (typeof cancelOpFromQueueType === 'function')
        ? cancelOpFromQueueType(this.currentOp.type)
        : '';
      await cancelProgressOperation(cancelOp || undefined);
    } catch (_) {}
    this.currentOp.status = 'cancelled';
    showToast('「' + label + '」已中断', { type: 'warning', duration: 4000 });
    var self = this;
    _onBulkProgressDone = null;
    this.progress = null;
    setTimeout(function () {
      self.currentOp = null;
      self.render();
      self.processNext();
    }, 800);
    return true;
  },

  cancelItem: function (id) {
    for (var i = 0; i < this.items.length; i++) {
      if (this.items[i].id === id) {
        var label = this.items[i].label;
        this.items.splice(i, 1);
        this.render();
        this.syncPendingToServer();
        showToast('已取消排队: ' + label, { type: 'info', duration: 3000 });
        return true;
      }
    }
    return false;
  },

  cancelAllPending: function () {
    if (this.items.length === 0) return;
    var count = this.items.length;
    this.items = [];
    this.render();
    this.syncPendingToServer();
    showToast('已取消 ' + count + ' 个排队中的操作', { type: 'info', duration: 3000 });
  },

  render: function () {
    this.renderQueueBar();
    this.updateButtonStates();
  },

  renderQueueBar: function () {
    var bar = document.getElementById('queue-status-bar');
    if (!bar) return;

    var html = '';
    if (this.currentOp) {
      var detail = this.formatProgressDetail();
      html += '<span class="queue-current">' +
        '<span class="spinner"></span> 正在执行: ' + esc(this.currentOp.label);
      if (detail) {
        html += '<span class="queue-progress-detail" title="' + escAttr(detail) + '"> — ' +
          esc(detail) + '</span>';
      }
      html += '</span>';
    }
    if (this.items.length > 0) {
      html += '<span class="queue-pending">排队中 (' + this.items.length + '):</span>';
      for (var i = 0; i < this.items.length; i++) {
        html += '<span class="queue-item">' + esc(this.items[i].label) +
          ' <button class="queue-item-cancel-btn" type="button" data-queue-id="' + this.items[i].id +
          '" title="取消此操作">×</button></span>';
      }
      html += ' <button class="queue-cancel-all-btn" type="button" id="queue-cancel-all-btn">取消全部</button>';
    }
    if (this.currentOp && this.currentOp.status === 'running') {
      html += ' <button class="queue-interrupt-btn" type="button" id="queue-interrupt-btn">中断当前</button>';
    }

    if (html) {
      bar.innerHTML = html;
      bar.style.display = '';
      // Bind dynamic buttons
      var cancelAllBtn = document.getElementById('queue-cancel-all-btn');
      if (cancelAllBtn) {
        cancelAllBtn.addEventListener('click', function () { bulkQueue.cancelAllPending(); });
      }
      var interruptBtn = document.getElementById('queue-interrupt-btn');
      if (interruptBtn) {
        interruptBtn.addEventListener('click', function () { bulkQueue.interruptCurrent(); });
      }
      // Bind individual item cancel buttons
      var itemCancelBtns = bar.querySelectorAll('.queue-item-cancel-btn');
      for (var j = 0; j < itemCancelBtns.length; j++) {
        itemCancelBtns[j].addEventListener('click', function () {
          var id = this.getAttribute('data-queue-id');
          if (id) bulkQueue.cancelItem(id);
        });
      }
    } else {
      bar.style.display = 'none';
    }
  },

  updateButtonStates: function () {
    var types = ['start-all', 'stop-all', 'build-all', 'restart-all', 'precise-restart', 'clear-db', 'init-db'];
    var btnMap = {
      'start-all': 'start-all-btn',
      'stop-all': 'stop-all-btn',
      'build-all': 'build-all-btn',
      'restart-all': 'restart-all-btn',
      'precise-restart': 'precise-restart-btn',
      'clear-db': 'dev-clear-databases',
      'init-db': 'dev-init-databases'
    };

    for (var i = 0; i < types.length; i++) {
      var type = types[i];
      var btn = document.getElementById(btnMap[type]);
      if (!btn) continue;

      var blocked = false;
      if (this.currentOp && this.currentOp.type === type && this.currentOp.status === 'running') {
        blocked = true;
      }
      for (var j = 0; j < this.items.length; j++) {
        if (this.items[j].type === type) {
          blocked = true;
          break;
        }
      }

      if (blocked) {
        btn.disabled = true;
        btn.classList.add('is-disabled');
        btn.title = '此操作已在队列中';
      } else if (!lastDAGBootInProgress) {
        btn.disabled = false;
        btn.classList.remove('is-disabled');
        btn.title = '';
      }
    }
  },

  bulkOpExecutor: function (type) {
    if (type === 'start-all') return window._execStartAll;
    if (type === 'stop-all') return window._execStopAll;
    if (type === 'restart-all') return window._execRestartAll;
    if (type === 'build-all') return window._execBuildAll;
    if (type === 'precise-restart') return window._execPreciseRestart;
    if (type === 'clear-db') return window._execClearAllDatabases;
    if (type === 'init-db') return window._execInitAllDatabases;
    return null;
  },

  syncPendingToServer: function () {
    var pending = [];
    for (var i = 0; i < this.items.length; i++) {
      pending.push({
        id: this.items[i].id,
        type: this.items[i].type,
        label: this.items[i].label
      });
    }
    return fetch('/api/exec-queue', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pending: pending })
    }).catch(function () {});
  },

  restorePending: function (pending) {
    // Fresh page: local items empty. Do not wipe a queue the user just enqueued
    // while /api/status still has a stale empty pending list.
    if (this.items.length > 0) return;
    if (!pending || pending.length === 0) return;
    var existing = {};
    if (this.currentOp && this.currentOp.type) {
      existing[this.currentOp.type] = true;
    }
    for (var j = 0; j < pending.length; j++) {
      var p = pending[j];
      if (!p || !p.type || existing[p.type]) continue;
      this.items.push({
        id: p.id || ('q-' + p.type),
        type: p.type,
        label: p.label || p.type,
        status: 'pending',
        fn: (function (opType) {
          return function () {
            var exec = bulkQueue.bulkOpExecutor(opType);
            if (typeof exec === 'function') {
              exec();
              return;
            }
            if (typeof _onBulkProgressDone === 'function' && _onBulkProgressDone) {
              var cb = _onBulkProgressDone;
              _onBulkProgressDone = null;
              cb('无法恢复操作: ' + opType);
            }
          };
        })(p.type),
        needsConfirm: false
      });
      existing[p.type] = true;
    }
    if (this.items.length > 0) {
      this.render();
      // Hot-replace orphans land in pending with no current worker — start replay.
      if (!this.processing && !this.currentOp) {
        var self = this;
        setTimeout(function () { self.processNext(); }, 300);
      }
    }
  }
};
