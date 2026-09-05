/**
 * COS 预签名直传（与 taskAiProvider frontend directUpload 同契约）。
 * OPT-20260901-008：与 taskAiProvider/frontend/src/utils/directUpload.js 为契约孪生，
 * 两仓独立无法共享文件——修改直传契约（hasPostFormFields / no-cors POST / Authorization 规则）
 * 时必须同步两文件；两侧单测同构断言同契约点。
 */

import { attachTraceIdToError } from './apiFetchTrace.js'

export function isSameOriginUploadURL(uploadURL) {
  return typeof uploadURL === 'string' && uploadURL.startsWith('/') && !uploadURL.startsWith('//')
}

export function hasPostFormFields(issued) {
  const fields = issued && issued.form_fields
  return !!fields && typeof fields === 'object' && !Array.isArray(fields) && Object.keys(fields).length > 0
}

export function buildDirectUploadHeaders({ issuedHeaders, uploadURL, bearerToken } = {}) {
  const headers = { ...(issuedHeaders || {}) }
  if (isSameOriginUploadURL(uploadURL) && bearerToken) {
    headers.Authorization = `Bearer ${bearerToken}`
  }
  return headers
}

function isOpaqueCorsFetchError(err) {
  return err instanceof TypeError && /failed to fetch/i.test(String(err.message || ''))
}

export async function uploadIssuedFile(file, issued, { fetchImpl, bearerToken } = {}) {
  const doFetch = fetchImpl || fetch
  const uploadURL = issued && issued.upload_url
  if (!uploadURL) {
    throw new Error('缺少上传地址')
  }
  if (hasPostFormFields(issued)) {
    const fd = new FormData()
    for (const [k, v] of Object.entries(issued.form_fields)) {
      fd.append(k, String(v))
    }
    fd.append('file', file)
    try {
      await doFetch(uploadURL, {
        method: issued.method || 'POST',
        body: fd,
        credentials: 'omit',
        mode: 'no-cors',
      })
    } catch (networkErr) {
      const err = networkErr instanceof Error ? networkErr : new Error(String(networkErr))
      attachTraceIdToError(err, issued.traceId)
      if (isOpaqueCorsFetchError(err)) return
      throw err
    }
    return
  }
  const headers = buildDirectUploadHeaders({
    issuedHeaders: (issued && issued.headers) || {},
    uploadURL,
    bearerToken,
  })
  const sameOrigin = isSameOriginUploadURL(uploadURL)
  let putRes
  try {
    putRes = await doFetch(uploadURL, {
      method: (issued && issued.method) || 'PUT',
      headers,
      body: file,
      credentials: sameOrigin ? 'include' : 'omit',
    })
  } catch (networkErr) {
    const err = networkErr instanceof Error ? networkErr : new Error(String(networkErr))
    attachTraceIdToError(err, issued && issued.traceId)
    throw err
  }
  if (!putRes.ok) {
    const e = new Error('直传失败')
    e.traceId = putRes.headers?.get?.('X-Trace-Id') || ''
    throw e
  }
}
