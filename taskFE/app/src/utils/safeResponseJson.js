/**
 * Safe response.json() wrapper that preserves traceId even when the response
 * body is not valid JSON (e.g. gateway 502 HTML, Django crash page).
 *
 * Usage:
 *   const { data, error, traceId } = await safeResponseJson(response)
 *   if (error) { el.setAttribute('data-traceId', traceId); ... }
 *
 * OPT-20260726-032: Created to systematically eliminate bare .json() calls
 * that lose traceId when parsing fails.
 */

import { extractTraceId, looksLikeTraceId } from './traceId.js'

/**
 * Safely parse a fetch Response as JSON, preserving traceId even on parse failure.
 *
 * @param {Response} response - fetch Response object (may have apiFetch-injected .traceId)
 * @param {{ fallback?: any }} [opts]
 * @returns {Promise<{ data: any, error: string, traceId: string }>}
 */
export async function safeResponseJson(response, opts = {}) {
  const traceId = extractTraceId(response) || extractTraceId(response?.headers) || ''

  if (!response || typeof response.json !== 'function') {
    return { data: opts.fallback ?? null, error: 'response is not a fetch Response', traceId }
  }

  try {
    const data = await response.json()
    // Extract traceId from parsed body as fallback
    const bodyTraceId = extractTraceId(data)
    return {
      data,
      error: '',
      traceId: (looksLikeTraceId(traceId) ? traceId : '') ||
               (looksLikeTraceId(bodyTraceId) ? bodyTraceId : ''),
    }
  } catch (e) {
    // .json() failed — likely HTML response (502, crash page, etc.)
    return {
      data: opts.fallback ?? null,
      error: e.message || 'Invalid JSON response',
      traceId,
    }
  }
}

/**
 * Safely parse a fetch Response, returning parsed JSON or a fallback value.
 * Simpler API for cases where traceId tracking isn't needed.
 *
 * @param {Response} response
 * @param {any} fallback
 * @returns {Promise<any>}
 */
export async function safeJson(response, fallback = null) {
  const { data } = await safeResponseJson(response, { fallback })
  return data
}
