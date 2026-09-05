/**
 * api-base.js — 运行时 API 根路径覆盖（非 Django 运行时配置）
 *
 * 从 index.html 内联脚本外置而来（v106 安全加固前置工程）：
 * 启用严格 CSP（script-src 'self'）时内联脚本会被拦截，
 * 本文件为同源经典脚本，document.currentScript 语义保持不变。
 *
 * 覆盖方式（与内联版一致）：
 *   1. URL 查询参数 ?apiBaseUrl=...（优先级最高）
 *   2. <script src="/api-base.js" data-api-base="..."> 属性
 * 均缺省时保持空串（同源 API）。
 */
(function() {
  window.__TASK2APP_API_BASE_URL__ = (function() {
    var p = new URLSearchParams(window.location.search).get('apiBaseUrl');
    if (p) return p;
    var s = document.currentScript && document.currentScript.dataset && document.currentScript.dataset.apiBase;
    return s || '';
  })();
})();
