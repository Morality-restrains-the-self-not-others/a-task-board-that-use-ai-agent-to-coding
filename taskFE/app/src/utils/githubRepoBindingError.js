/** GitHub 仓库绑定错误文案 + data-traceId 赋值 */
import { extractTraceId } from './traceId.js'
import { resolveApiErrorMessage } from './workPanelFormat.js'

export function assignGithubRepoBindingError(errorRef, traceIdRef, payload, { fallback, response } = {}) {
  const data = payload && typeof payload === 'object' ? payload : {}
  errorRef.value = resolveApiErrorMessage(data, { fallback: fallback || '操作失败' })
  traceIdRef.value = extractTraceId(response) || extractTraceId(data) || extractTraceId(payload) || ''
}

export function assignGithubRepoBindingCatchError(errorRef, traceIdRef, err, fallback) {
  errorRef.value = err?.message || fallback || '操作失败'
  traceIdRef.value = extractTraceId(err) || ''
}

export function makeGithubRepoBindingHttpError(data, response, fallback) {
  const err = new Error(resolveApiErrorMessage(data, { fallback }))
  err.traceId = extractTraceId(response) || extractTraceId(data) || ''
  return err
}
