// API 工具函数，包装 fetch 以使用全局配置的 API_BASE_URL

import { getActiveToken } from '../domain/auth/services/saved_accounts_store.js'
import {
  handleForwardAuthSessionExpired,
  isForwardAuthSessionExpired,
  isTaskDetailAccessCodeSharePage,
} from './requestErrorDisplay.js'
import {
  TRACE_ID_HEADER,
  attachTraceIdToError,
  attachTraceIdToResponse,
  attachTracePropagationHeaders,
} from './apiFetchTrace.js'
import { gatewayUnavailableMessage } from './httpError.js'

// 模块级 token 缓存：避免每次请求都走 postMessage 异步获取
// 由登录/切换账号流程通过 setCachedAuthToken() 写入
let _cachedAuthToken = null
let _tokenInitPromise = null

/**
 * 获取缓存的 authToken。首次调用时自动从本机账号槽（localStorage）加载。
 * @returns {Promise<string|null>}
 */
async function resolveAuthToken() {
  if (_cachedAuthToken !== null) return _cachedAuthToken || null
  if (!_tokenInitPromise) {
    // 账号槽为同步读取，Promise.resolve 统一为 Promise 形态
    _tokenInitPromise = Promise.resolve(getActiveToken()).then((t) => {
      _cachedAuthToken = t || ''
      return _cachedAuthToken
    }).catch(() => {
      _cachedAuthToken = ''
      return ''
    })
  }
  return _tokenInitPromise
}

/**
 * 重新从本机账号槽加载 token。
 */
export async function reloadAuthToken() {
  _cachedAuthToken = null
  _tokenInitPromise = null
  return resolveAuthToken()
}

/**
 * 更新缓存的 authToken（登录/切换账号成功后调用）。
 * @param {string|null} token
 */
export function setCachedAuthToken(token) {
  _cachedAuthToken = token || ''
  _tokenInitPromise = null
}

/**
 * 清除缓存的 authToken（登出时调用）。
 */
export function clearCachedAuthToken() {
  _cachedAuthToken = ''
  _tokenInitPromise = null
}

// In-flight GET request deduplication: concurrent identical GET requests share
// one underlying fetch promise, eliminating redundant /me/ and other API calls
// fired by multiple independent components on the same page.
const _inflightGetRequests = new Map();

/**
 * Build a dedup key for in-flight GET requests.
 * Only deduplicates same-URL+same-headers; caller-supplied AbortSignal or
 * credentials differences are treated as distinct requests.
 */
function _inflightKey(url, options) {
  const cred = options.credentials === 'omit' ? 'omit' : (options.credentials || 'include');
  const h = options.headers || {};
  const auth = h['Authorization'] || h['authorization'] || '';
  return `${cred}|${auth}|${url}`;
}

/**
 * 简易 deferred：同步注册的占位 promise，由实际 fetch 结果填充。
 * OPT-20260808-014: 消除「去重检查在 await 之前、注册在 await 之后」的竞态窗口
 * （trace 头/token 解析挂起期间并发相同 GET 全部穿透检查，Navbar/Sidebar/路由
 * 守卫的并发 /me/ 在每个页面重复发出，弱网下成倍放大连接挂起与 CPU 压力）。
 * isSettled() 供 finally 兜底判断：任何提前退出路径都不允许悬挂等待者。
 */
function _deferredPromise() {
  let resolve;
  let reject;
  let settled = false;
  const promise = new Promise((res, rej) => {
    resolve = (v) => { settled = true; res(v); };
    reject = (e) => { settled = true; rej(e); };
  });
  return { promise, resolve, reject, isSettled: () => settled };
}

/**
 * 包装的 fetch 函数，自动使用全局配置的 API_BASE_URL
 * @param {string} url - API 端点或完整 URL
 * @param {object} options - fetch 选项
 * @returns {Promise<Response>} - fetch 响应
 */
export async function apiFetch(url, options = {}) {
  if (!window.config) {
    console.error('全局配置未加载');
  }

  // 获取 API 基础 URL
  const baseUrl = window.config?.API_BASE_URL ?? '';

  // 构建完整 URL
  let fullUrl;
  if (url.startsWith('http://') || url.startsWith('https://')) {
    fullUrl = url;
  } else {
    // 确保 URL 路径正确拼接
    const normalizedUrl = url.startsWith('/') ? url : `/${url}`;
    fullUrl = `${baseUrl}${normalizedUrl}`;
  }

  // 确保选项对象存在
  options = options || {};

  // 确保 headers 对象存在
  options.headers = options.headers || {};

  // 请求超时：默认 30s，调用方可传入 options.timeout 覆盖；传 0 表示不限时
  const timeoutMs = typeof options.timeout === 'number' && options.timeout >= 0
    ? options.timeout
    : 30000;
  delete options.timeout;

  // GET/HEAD 不应带 application/json，少数网关/中间件会异常处理
  const methodUpper = String(options.method || 'GET').toUpperCase();

  // In-flight GET dedup: share the same fetch promise for concurrent identical GETs.
  // Multiple components on the same page often fire the same /me/ request
  // independently; this coalesces them into one HTTP call.
  let _dedupKey = null;
  let _dedupPlaceholder = null;
  if (methodUpper === 'GET' && !options.signal && !options._skipDedup) {
    _dedupKey = _inflightKey(fullUrl, options);
    const inflight = _inflightGetRequests.get(_dedupKey);
    if (inflight) {
      console.log('[apiFetch] 复用进行中的 GET 请求:', fullUrl);
      const res = await inflight;
      return res.clone();
    }
    // OPT-20260808-014: 同步注册占位 promise（而非在 fetch 真正发出后才注册），
    // 随后的 await（trace 头/token 解析）期间并发相同 GET 命中占位复用。
    _dedupPlaceholder = _deferredPromise();
    // 无并发等待者时占位被 reject（如请求超时）会产生 unhandled rejection，空消费者兜底
    _dedupPlaceholder.promise.catch(() => {});
    _inflightGetRequests.set(_dedupKey, _dedupPlaceholder.promise);
  }

  // AbortController 超时控制：仅当不存在 caller 传入的 signal 时才创建
  let abortController = null;
  let timeoutId = null;
  if (!options.signal && timeoutMs > 0) {
    abortController = new AbortController();
    options.signal = abortController.signal;
    timeoutId = setTimeout(() => abortController.abort(), timeoutMs);
  }

  await attachTracePropagationHeaders(options.headers);
  const requestTraceId =
    options.headers[TRACE_ID_HEADER] || options.headers['x-trace-id'] || '';

  const omitBodyContentType =
    methodUpper === 'GET' || methodUpper === 'HEAD' || options.body == null;
  if (!options.headers['Content-Type'] && !options.headers['content-type']) {
    if (options.body instanceof FormData || options.body instanceof URLSearchParams) {
      // 对于 FormData 和 URLSearchParams，让浏览器自动设置 Content-Type
    } else if (!omitBodyContentType) {
      options.headers['Content-Type'] = 'application/json';
    }
  }
  
  // 默认包含凭据（cookies）
  if (options.credentials === undefined) {
    options.credentials = 'include';
  }

  // 内部标记：勿传给 fetch
  const retriedWithoutAuth = Boolean(options._retriedWithoutAuth);
  delete options._retriedWithoutAuth;
  // 只读探测（如 Git OAuth 绑定状态）在未登录时视为未绑定，不得整页踢登录。
  const skipSessionExpiredRedirect = Boolean(options.skipSessionExpiredRedirect);
  delete options.skipSessionExpiredRedirect;

  // 添加认证token到请求头（调用方显式传入 Authorization 时不覆盖）
  const callerHasAuth = Object.keys(options.headers).some(
    (k) => k.toLowerCase() === 'authorization',
  );
  // accessCode 分享页（task-detail + accessCode）：访客靠分享码授权，本机残留失效 Token
  // 会让网关 forward-auth 失败盖过分享码，故不带 Authorization（仍 credentials: include）。
  const onAccessCodeSharePage =
    typeof window !== 'undefined' &&
    isTaskDetailAccessCodeSharePage(
      window.location.pathname || '',
      window.location.search || '',
    );
  const authToken = !callerHasAuth && !onAccessCodeSharePage ? (await resolveAuthToken()) : null;
  if (authToken) {
    options.headers['Authorization'] = `Token ${authToken}`;
    console.log('API 请求添加了认证token');
  }
  
  console.log('API 请求:', fullUrl, options);

  // Wrap the actual fetch to register & cleanup in-flight dedup map
  const _doFetch = async () => {
    const resp = await fetch(fullUrl, options);
    return resp;
  };

  let _fetchPromise;
  if (_dedupPlaceholder) {
    // 占位由实际请求结果填充；失败同样传递 reject，等待者不会悬挂
    _fetchPromise = _doFetch();
    _fetchPromise.then(
      (resp) => _dedupPlaceholder.resolve(resp),
      (err) => _dedupPlaceholder.reject(err),
    );
  } else {
    _fetchPromise = _doFetch();
    if (_dedupKey) {
      _inflightGetRequests.set(_dedupKey, _fetchPromise);
    }
  }

  try {
    const response = await _fetchPromise;
    console.log('API 响应:', fullUrl, response.status);

    // 检查是否是未授权或重定向响应
    if (!response.ok) {
      try {
        const errorData = await response.clone().json();
        response._errorData = errorData;
        console.log('API 错误响应数据:', errorData);

        // 网关 forward-auth 会话失效（无法解析登录凭据）全站统一收口：
        // 任何页面/组件的请求命中即引导重新登录（带回跳地址），页面不再各自接线。
        // 探测类接口（如 Git OAuth 绑定状态）传 skipSessionExpiredRedirect：
        // 未登录视为未绑定，不得把 accessCode 分享页整页踢到登录。
        if (isForwardAuthSessionExpired(errorData) && !skipSessionExpiredRedirect) {
          handleForwardAuthSessionExpired();
        }

        const onAccessCodeSharePage =
          typeof window !== 'undefined' &&
          isTaskDetailAccessCodeSharePage(
            window.location.pathname || '',
            window.location.search || '',
          );
        if (errorData.redirect_url && !skipSessionExpiredRedirect && !onAccessCodeSharePage) {
          console.log('收到重定向指令，跳转到:', errorData.redirect_url);
          window.location.href = errorData.redirect_url;
          const redirectErr = new Error('Redirecting to login page');
          attachTraceIdToError(
            redirectErr,
            attachTraceIdToResponse(response, options.headers, errorData) || requestTraceId,
          );
          throw redirectErr;
        }

        // 残留无效 Token 会使 AllowAny 公开接口也 403；清掉后无 Authorization 再试一次
        const detail = errorData?.detail;
        const invalidToken =
          response.status === 403 &&
          (detail === 'Invalid token.' ||
            detail === 'Invalid token' ||
            (typeof detail === 'string' && /invalid token/i.test(detail)));
        if (invalidToken && authToken && !retriedWithoutAuth) {
          clearCachedAuthToken();
          const retryHeaders = { ...options.headers };
          delete retryHeaders.Authorization;
          delete retryHeaders.authorization;
          return apiFetch(url, {
            ...options,
            headers: retryHeaders,
            _retriedWithoutAuth: true,
            skipSessionExpiredRedirect,
          });
        }
      } catch (jsonError) {
        if (jsonError.message !== 'Redirecting to login page') {
          try {
            const errorText = await response.clone().text();
            response._errorData = { _rawErrorText: errorText };
            console.log('API 错误响应（非JSON）:', errorText.substring(0, 200));
          } catch {
            response._errorData = {};
            console.log('响应不是有效的JSON且无法读取文本:', jsonError);
          }
        } else {
          throw jsonError;
        }
      }
    }

    attachTraceIdToResponse(response, options.headers, response._errorData);

    // 覆盖 response.json() 将 traceId 注入解析后的响应体，
    // 消除全站 showRequestError(data) 调用中 data-traceId 缺口
    const _originalJson = response.json.bind(response);
    response.json = async function () {
      const body = await _originalJson();
      if (response.traceId && body && typeof body === 'object' && !Array.isArray(body)) {
        body._traceId = response.traceId;
      }
      return body;
    };

    return response;
  } catch (error) {
    // 请求超时 → 友好错误
    if (error.name === 'AbortError' && abortController && timeoutId) {
      console.error('API 请求超时:', fullUrl, `(${timeoutMs}ms)`);
      const timeoutError = new Error(`请求超时（${timeoutMs / 1000} 秒），请检查网络后重试`);
      timeoutError.name = 'TimeoutError';
      attachTraceIdToError(timeoutError, requestTraceId);
      throw timeoutError;
    }
    attachTraceIdToError(error, error.traceId || requestTraceId);
    console.error('API 请求错误:', fullUrl, error);
    throw error;
  } finally {
    if (_dedupKey) {
      _inflightGetRequests.delete(_dedupKey);
      // 兜底：异常/重试等提前退出路径上占位仍未 settle → reject，等待者不悬挂
      if (_dedupPlaceholder && !_dedupPlaceholder.isSettled()) {
        _dedupPlaceholder.reject(new Error('apiFetch 提前终止，共享请求未完成'));
      }
    }
    if (timeoutId) clearTimeout(timeoutId);
  }
}

/**
 * 发送 GET 请求
 * @param {string} url - API 端点
 * @param {object} options - 额外选项
 * @returns {Promise<Response>}
 */
export function get(url, options = {}) {
  return apiFetch(url, {
    ...options,
    method: 'GET'
  });
}

/**
 * 发送 POST 请求
 * @param {string} url - API 端点
 * @param {object} data - 请求数据
 * @param {object} options - 额外选项
 * @returns {Promise<Response>}
 */
export function post(url, data, options = {}) {
  return apiFetch(url, {
    ...options,
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/**
 * 发送 PUT 请求
 * @param {string} url - API 端点
 * @param {object} data - 请求数据
 * @param {object} options - 额外选项
 * @returns {Promise<Response>}
 */
export function put(url, data, options = {}) {
  return apiFetch(url, {
    ...options,
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/**
 * 发送 DELETE 请求
 * @param {string} url - API 端点
 * @param {object} options - 额外选项
 * @returns {Promise<Response>}
 */
export function del(url, options = {}) {
  return apiFetch(url, {
    ...options,
    method: 'DELETE'
  });
}

/**
 * 安全解析 JSON 响应：非 ok 时提取 _errorData 错误信息，自动继承 response.traceId。
 * 消除各调用方重复的 `response.ok` 检查 + JSON 解析防御代码。
 *
 * @param {Response} response - apiFetch 返回的 Response 对象
 * @returns {Promise<any>} 解析后的 JSON 对象
 */
export async function safeParseResponse(response) {
  if (!response.ok) {
    const ed = response._errorData
    let errorMsg = extractErrorMessage(ed, response) || `HTTP ${response.status}`
    const err = new Error(errorMsg)
    if (response.traceId) err.traceId = response.traceId
    err.response = response
    throw err
  }
  try {
    return await response.json()
  } catch (jsonError) {
    const err = new Error(jsonError.message || '响应不是有效的 JSON')
    if (response.traceId) err.traceId = response.traceId
    err.response = response
    err.cause = jsonError
    throw err
  }
}

/**
 * 从 API 错误响应中提取用户可读的错误消息。
 * 兼容 DRF (detail/non_field_errors)、Django (message)、taskBill Go (error) 多种后端格式，
 * 以及网关转发失败时返回的非 JSON 响应体。
 *
 * @param {object|null} body - 已解析的 JSON 响应体（或 {}）
 * @param {Response} [response] - fetch Response 对象（可能带有 _errorData 扩展），可选
 * @param {string} [fallback] - 未提取到任何消息时的回退值，默认 null
 * @returns {string|null} 提取到的错误消息，否则返回 fallback
 */
export function extractErrorMessage(body, response, fallback = null) {
  if (body && typeof body === 'object') {
    if (typeof body.message === 'string' && body.message.trim()) return body.message.trim()
    if (typeof body.detail === 'string' && body.detail.trim()) return body.detail.trim()
    if (typeof body.error === 'string' && body.error.trim()) return body.error.trim()
    if (Array.isArray(body.non_field_errors) && body.non_field_errors.length) {
      const msg = body.non_field_errors[0]
      if (typeof msg === 'string' && msg.trim()) return msg.trim()
    }
  }
  const gw = gatewayUnavailableMessage(typeof response === 'object' ? response?.status : undefined)
  if (gw) return gw
  // 非 JSON 响应（网关 HTML 错误页等）：尝试从 apiFetch 注入的 _errorData 提取
  const raw = response?._errorData?._rawErrorText
  if (typeof raw === 'string' && raw.trim()) {
    const stripped = raw.replace(/<[^>]*>/g, '').trim()
    if (stripped) return stripped.substring(0, 200)
  }
  return fallback
}

/** company_members API：兼容数组旧格式与 { members, meta } 新格式 */
export function parseCompanyMembersResponse(data) {
  if (Array.isArray(data)) {
    return { members: data, meta: {} }
  }
  if (data && typeof data === 'object') {
    return {
      members: Array.isArray(data.members) ? data.members : [],
      meta: data.meta && typeof data.meta === 'object' ? data.meta : {},
    }
  }
  return { members: [], meta: {} }
}
