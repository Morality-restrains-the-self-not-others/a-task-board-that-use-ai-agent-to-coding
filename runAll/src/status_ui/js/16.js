// 9999「初始化全部数据库」未 migrate 标注（不进入 2s refresh()）。
var _finishInitAllDatabases;
var _finishClearAllDatabases;

function migratePendingConfirmSuffix() {
  var n = window._migratePendingCount;
  if (!n || n < 1) return '';
  return '\n\n当前有 ' + n + ' 个未应用迁移文件，初始化将按 registry 执行 migrate。';
}

function applyMigratePendingReport(data) {
  var btn = document.getElementById('dev-init-databases');
  var label = document.getElementById('dev-init-db-pending-label');
  if (!data || typeof data !== 'object') return;
  var pending = Number(data.pending_count) || 0;
  var unreachable = Number(data.unreachable_count) || 0;
  window._migratePendingCount = pending;
  var parts = [];
  var dbs = Array.isArray(data.databases) ? data.databases : [];
  for (var i = 0; i < dbs.length; i++) {
    var row = dbs[i] || {};
    var name = row.key || row.database || '?';
    if (row.status === 'pending' && row.missing && row.missing.length) {
      parts.push(name + ': ' + row.missing.join(', '));
    } else if (row.status === 'unreachable') {
      parts.push(name + ': 不可达' + (row.error ? ' (' + row.error + ')' : ''));
    }
    if (row.stale && row.stale.length) {
      parts.push(name + ': 库内多余 ' + row.stale.join(', '));
    }
  }
  var title = parts.join('\n');
  if (btn) {
    btn.classList.toggle('has-pending-migrate', pending > 0);
    btn.classList.toggle('has-migrate-unknown', pending === 0 && unreachable > 0);
    if (pending > 0) {
      btn.title = '未 migrate\n' + title;
    } else if (unreachable > 0) {
      btn.title = '部分库无法探测\n' + title;
    } else if (title) {
      btn.title = '库内多余 step（不计未 migrate）\n' + title;
    } else {
      btn.title = '';
    }
  }
  if (label) {
    if (pending > 0) {
      var txt = pending + ' 未 migrate';
      if (unreachable > 0) txt += ' · ' + unreachable + ' 库不可达';
      label.textContent = txt;
      label.title = title;
    } else if (unreachable > 0) {
      label.textContent = unreachable + ' 库不可达';
      label.title = title;
    } else {
      label.textContent = '';
      label.title = '';
    }
  }
}

async function refreshMigratePendingStatus(force) {
  try {
    var url = '/api/dev/migrate-status';
    if (force) url += '?refresh=1';
    var resp = await apiFetch(url);
    var data = await parseJsonSafe(resp);
    if (!resp.ok) return;
    applyMigratePendingReport(data);
  } catch (e) {
    // 巡检失败不打断 Status 页
  }
}

(function hookDevDBFinishForMigratePending() {
  if (typeof _finishInitAllDatabases === 'function') {
    var origInit = _finishInitAllDatabases;
    _finishInitAllDatabases = function (ev) {
      origInit(ev);
      refreshMigratePendingStatus(true);
    };
  }
  if (typeof _finishClearAllDatabases === 'function') {
    var origClear = _finishClearAllDatabases;
    _finishClearAllDatabases = function (ev) {
      origClear(ev);
      refreshMigratePendingStatus(true);
    };
  }
})();

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', function () {
    refreshMigratePendingStatus(false);
  });
} else {
  refreshMigratePendingStatus(false);
}
