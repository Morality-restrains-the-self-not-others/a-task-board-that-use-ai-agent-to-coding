// ── Custom Modal System (replaces browser alert/confirm/prompt) ──
(function() {
  'use strict';

  var activeModal = null;

  function createModalHTML(title, bodyHTML, footerHTML) {
    return '' +
      '<div class="modal-overlay" role="dialog" aria-modal="true" aria-labelledby="modal-title">' +
        '<div class="modal-dialog">' +
          '<div class="modal-header">' +
            '<span class="modal-title" id="modal-title">' + (title || 'runAll') + '</span>' +
            '<button class="modal-close-btn" aria-label="关闭" data-modal-action="close">&times;</button>' +
          '</div>' +
          '<div class="modal-body">' + bodyHTML + '</div>' +
          (footerHTML ? '<div class="modal-footer">' + footerHTML + '</div>' : '') +
        '</div>' +
      '</div>';
  }

  function escapeHTML(str) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  function escapeAttr(str) {
    return String(str).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function formatMessage(msg) {
    return escapeHTML(String(msg || '')).replace(/\n/g, '<br>');
  }

  // ── Diagnostic payload builder (AI-agent-friendly) ──

  function buildDiagnosticPayload(message, meta) {
    meta = meta || {};
    var lines = [];

    lines.push('# 诊断报告');
    lines.push('');
    lines.push('## 基本信息');
    lines.push('- **时间**: ' + new Date().toISOString());
    lines.push('- **页面**: ' + (window.location.href || ''));
    lines.push('- **操作类型**: ' + (meta.opLabel || meta.title || '未知'));
    if (meta.opDetail) {
      lines.push('- **操作详情**: ' + meta.opDetail);
    }

    // Try to extract trace_id / request ID from the error banner
    try {
      var banner = document.getElementById('requestErrorBanner');
      if (banner) {
        var bannerText = (banner.textContent || '').trim();
        if (bannerText) {
          var traceMatch = bannerText.match(/trace[_-]?id[=:]\s*([a-fA-F0-9-]{8,})/);
          if (traceMatch) {
            lines.push('- **Trace ID**: `' + traceMatch[1] + '`');
          }
          if (!traceMatch) {
            var reqIdMatch = bannerText.match(/request[_-]?id[=:]\s*(\S+)/);
            if (reqIdMatch) {
              lines.push('- **Request ID**: `' + reqIdMatch[1] + '`');
            }
          }
        }
      }
    } catch (_) { /* ignore */ }

    lines.push('');
    lines.push('## 详细信息');
    lines.push('');
    // message text — preserve as plain text (strip HTML tags from formatted message)
    var rawMsg = String(message || '');
    lines.push(rawMsg);

    return lines.join('\n');
  }

  // ── Clipboard helpers ──

  function copyToClipboard(text) {
    // Modern async clipboard API
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text).then(function() { return true; }).catch(function() {
        return fallbackCopy(text);
      });
    }
    return fallbackCopy(text);
  }

  function fallbackCopy(text) {
    return new Promise(function(resolve) {
      var textarea = document.createElement('textarea');
      textarea.value = text;
      textarea.style.position = 'fixed';
      textarea.style.left = '-9999px';
      textarea.style.top = '-9999px';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.focus();
      textarea.select();
      try {
        var ok = document.execCommand('copy');
        resolve(ok);
      } catch (_) {
        resolve(false);
      } finally {
        document.body.removeChild(textarea);
      }
    });
  }

  function showCopyToast(success) {
    // Remove any existing toast
    var existing = document.querySelector('.modal-copy-toast');
    if (existing) existing.parentNode.removeChild(existing);

    var toast = document.createElement('div');
    toast.className = 'modal-copy-toast';
    toast.textContent = success ? '✓ 已复制诊断信息，可直接投喂给智能体' : '✗ 复制失败，请手动选择文本后 Ctrl+C';
    toast.style.background = success ? '#065f46' : '#7f1d1d';
    toast.style.borderColor = success ? '#34d399' : '#ef4444';
    toast.style.color = success ? '#bbf7d0' : '#fecaca';
    document.body.appendChild(toast);
    setTimeout(function() {
      if (toast.parentNode) toast.parentNode.removeChild(toast);
    }, 2200);
  }

  // Build footer HTML with copy button + action buttons
  function buildFooter(copyMeta, actionButtonsHTML) {
    var copyHint = escapeAttr('复制「' + ((copyMeta && copyMeta.opLabel) || (copyMeta && copyMeta.title) || '诊断信息') + '」的完整诊断报告，可直接投喂给 AI 智能体分析');
    return '' +
      '<div class="modal-footer-left">' +
        '<button class="modal-btn modal-btn-copy" data-modal-action="copy" title="' + copyHint + '">📋 复制诊断信息</button>' +
      '</div>' +
      '<div class="modal-footer-right">' +
        actionButtonsHTML +
      '</div>';
  }

  // ── Core modal ──

  function showModal(title, bodyHTML, footerHTML, copyMeta) {
    return new Promise(function(resolve) {
      // Remove any existing modal
      if (activeModal) {
        activeModal.remove();
        activeModal = null;
      }

      var container = document.createElement('div');
      container.innerHTML = createModalHTML(title, bodyHTML, footerHTML);
      var overlay = container.firstElementChild;

      // Store diagnostic metadata for copy action
      overlay._modalCopyMeta = copyMeta || { title: title };

      document.body.appendChild(overlay);
      activeModal = overlay;

      // Focus trap: find first focusable element
      var focusable = overlay.querySelectorAll('button, input, [tabindex]:not([tabindex="-1"])');
      if (focusable.length > 0) {
        setTimeout(function() { focusable[0].focus(); }, 80);
      }

      // Close handler
      function close(result) {
        if (!activeModal || activeModal !== overlay) return;
        overlay.classList.add('closing');
        activeModal = null;
        setTimeout(function() {
          if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
        }, 140);
        resolve(result);
      }

      // Click on overlay backdrop
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
          close(undefined);
        }
      });

      // Delegate clicks on data-modal-action
      overlay.addEventListener('click', function(e) {
        var target = e.target;
        var action = target.getAttribute('data-modal-action');
        if (!action) {
          var parent = target.closest('[data-modal-action]');
          if (parent) action = parent.getAttribute('data-modal-action');
        }
        if (action === 'close') {
          close(undefined);
        } else if (action === 'confirm') {
          close(true);
        } else if (action === 'cancel') {
          close(false);
        } else if (action === 'submit-prompt') {
          var input = overlay.querySelector('.modal-input');
          close(input ? input.value : '');
        } else if (action === 'copy') {
          // Copy diagnostic payload
          var meta = overlay._modalCopyMeta || {};
          var payload = buildDiagnosticPayload(meta.message || '', meta);
          var btn = target.closest('.modal-btn-copy');
          copyToClipboard(payload).then(function(ok) {
            showCopyToast(ok);
            if (btn && ok) {
              btn.classList.add('copied');
              setTimeout(function() { btn.classList.remove('copied'); }, 2000);
            }
          });
        }
      });

      // Keyboard shortcuts
      function onKeyDown(e) {
        if (e.key === 'Escape') {
          e.preventDefault();
          close(undefined);
          return;
        }
        if (e.key === 'Enter') {
          // 文本框内回车应换行，不触发弹窗操作
          var tag = (e.target && e.target.tagName || '').toLowerCase();
          if (tag === 'textarea') return;

          // prompt 模态框：当前聚焦在 .modal-input 时提交输入值
          var promptInput = overlay.querySelector('.modal-input');
          if (promptInput && document.activeElement === promptInput) {
            e.preventDefault();
            close(promptInput.value);
            return;
          }

          // alert / confirm 模态框：回车触发"确定"按钮
          var confirmBtn = overlay.querySelector('[data-modal-action="confirm"]');
          if (confirmBtn) {
            e.preventDefault();
            confirmBtn.click();
            return;
          }

          // 兜底：无确认按钮时关闭（alert 模式）
          close(undefined);
        }
      }
      overlay.addEventListener('keydown', onKeyDown);
    });
  }

  // ── Public API ──

  /**
   * Show an alert-style modal (replaces alert()).
   * @param {string} message — message text (supports \n for line breaks)
   * @param {object} [opts]
   * @param {string} [opts.title]   — modal title
   * @param {string} [opts.type]    — 'info' | 'success' | 'warning' | 'error'
   * @param {string} [opts.opLabel] — human-readable operation label for copy payload
   */
  window.showModalAlert = function(message, opts) {
    opts = opts || {};
    var type = opts.type || 'info';
    var title = opts.title || '';
    var icons = { info: 'ℹ️', success: '✅', warning: '⚠️', error: '❌' };
    var labels = { info: '提示', success: '完成', warning: '警告', error: '错误' };
    var btnClass = type === 'error' ? 'modal-btn-danger' : (type === 'warning' ? 'modal-btn-warn' : (type === 'success' ? 'modal-btn-success' : 'modal-btn-primary'));
    if (!title) title = (icons[type] || icons.info) + ' ' + (labels[type] || labels.info);

    var copyMeta = {
      title: title,
      message: message,
      opLabel: opts.opLabel || labels[type] || '提示',
      opDetail: opts.opDetail || ''
    };

    return showModal(
      title,
      '<div class="modal-message">' + formatMessage(message) + '</div>',
      buildFooter(copyMeta, '<button class="modal-btn ' + btnClass + '" data-modal-action="confirm">确定</button>'),
      copyMeta
    );
  };

  /**
   * Show a confirm-style modal (replaces window.confirm()).
   */
  window.showModalConfirm = function(message, opts) {
    opts = opts || {};
    var title = opts.title || '⚠️ 确认操作';

    var copyMeta = {
      title: title,
      message: message,
      opLabel: opts.opLabel || '确认操作',
      opDetail: opts.opDetail || ''
    };

    return showModal(
      title,
      '<div class="modal-message">' + formatMessage(message) + '</div>',
      buildFooter(copyMeta,
        '<button class="modal-btn modal-btn-cancel" data-modal-action="cancel">取消</button>' +
        '<button class="modal-btn modal-btn-primary" data-modal-action="confirm">确定</button>'
      ),
      copyMeta
    ).then(function(result) { return result === true; });
  };

  /**
   * Show a prompt-style modal (replaces window.prompt()).
   */
  window.showModalPrompt = function(message, defaultValue, opts) {
    opts = opts || {};
    var title = opts.title || '📝 请输入';
    var val = defaultValue != null ? escapeHTML(String(defaultValue)) : '';

    var copyMeta = {
      title: title,
      message: message + (defaultValue ? '\n\n默认值: ' + defaultValue : ''),
      opLabel: opts.opLabel || '用户输入',
      opDetail: opts.opDetail || ''
    };

    return showModal(
      title,
      '<div class="modal-message">' + formatMessage(message) + '</div>' +
      '<input class="modal-input" type="text" value="' + val + '" placeholder="请输入 trace_id ...">',
      buildFooter(copyMeta,
        '<button class="modal-btn modal-btn-cancel" data-modal-action="cancel">取消</button>' +
        '<button class="modal-btn modal-btn-primary" data-modal-action="submit-prompt">确定</button>'
      ),
      copyMeta
    ).then(function(result) {
      return (result === undefined || result === false) ? null : String(result);
    });
  };

})();
