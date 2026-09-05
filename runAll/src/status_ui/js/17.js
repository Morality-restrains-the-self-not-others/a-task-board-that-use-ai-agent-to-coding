// Toast Notification System（自 04.js 抽出，OPT-20260823-025）。
// 本文件在全部脚本之后加载；showToast/dismissToast 仅在运行时（点击/SSE）调用，
// 因此依赖的 esc（04.js）与 toastCounter 均为调用时已就绪。
var toastCounter = 0;
var toastDefaultDuration = 7000;
var TOAST_MAX_VISIBLE = 5;

function getToastContainer() {
  var el = document.getElementById('toast-container');
  if (!el) {
    el = document.createElement('div');
    el.id = 'toast-container';
    el.className = 'toast-container';
    el.setAttribute('aria-live', 'polite');
    el.setAttribute('aria-label', 'Notifications');
    document.body.appendChild(el);
  }
  return el;
}

function showToast(message, opts) {
  opts = opts || {};
  var type = opts.type || 'error';
  var duration = typeof opts.duration === 'number' ? opts.duration : toastDefaultDuration;

  var container = getToastContainer();
  var text = String(message || '');
  var iconMap = { error: '✕', success: '✓', warning: '⚠', info: 'ℹ' };
  var icon = iconMap[type] || 'ℹ';

  // 去重：相同 message+type 的可见 toast 不重复创建，只刷新自动关闭计时。
  var existing = container.querySelectorAll('.toast.toast-' + type);
  for (var i = 0; i < existing.length; i++) {
    var bodyEl = existing[i].querySelector('.toast-body');
    if (bodyEl && bodyEl.textContent === text) {
      var dupId = existing[i].id;
      if (duration > 0) {
        clearTimeout(existing[i]._timer);
        existing[i]._timer = setTimeout(function () { dismissToast(dupId); }, duration);
      }
      return dupId;
    }
  }

  var id = 'toast-' + (++toastCounter);

  var toast = document.createElement('div');
  toast.id = id;
  toast.className = 'toast toast-' + type;
  toast.setAttribute('role', 'alert');
  toast.innerHTML =
    '<span class="toast-icon">' + icon + '</span>' +
    '<span class="toast-body">' + esc(text) + '</span>' +
    '<button class="toast-close" type="button" aria-label="关闭" onclick="dismissToast(\'' + id + '\')">×</button>';

  container.appendChild(toast);

  // 硬上限：超过 TOAST_MAX_VISIBLE 时立即移除最旧，不等待 280ms 动画。
  // 防止任何重放路径（如 start-all 空计划 bug 的 toastCounter 35000）再次打爆 DOM。
  var toasts = container.querySelectorAll('.toast');
  if (toasts.length > TOAST_MAX_VISIBLE) {
    var oldest = toasts[0];
    if (oldest && oldest.parentNode) oldest.parentNode.removeChild(oldest);
  }

  // Auto-dismiss
  if (duration > 0) {
    toast._timer = setTimeout(function () { dismissToast(id); }, duration);
  }

  return id;
}

function dismissToast(id) {
  var toast = document.getElementById(id);
  if (!toast) return;
  toast.classList.add('toast-removing');
  setTimeout(function () {
    if (toast && toast.parentNode) toast.parentNode.removeChild(toast);
  }, 280);
}
