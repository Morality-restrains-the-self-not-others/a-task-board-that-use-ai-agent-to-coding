// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import {
  extractTraceId,
  looksLikeTraceId,
  resolveRequestTraceId,
  setDataTraceId,
  traceIdFromBody,
  traceIdFromHeaders,
} from './traceId.js'

describe('traceId helpers', () => {
  it('traceIdFromHeaders reads Headers and plain objects', () => {
    const h = new Headers({ 'X-Trace-Id': 'resp-1' })
    expect(traceIdFromHeaders(h)).toBe('resp-1')
    expect(traceIdFromHeaders({ 'x-trace-id': 'req-2' })).toBe('req-2')
    expect(traceIdFromHeaders({})).toBe('')
  })

  it('traceIdFromBody reads snake and camel keys', () => {
    expect(traceIdFromBody({ trace_id: 'a' })).toBe('a')
    expect(traceIdFromBody({ traceId: 'b' })).toBe('b')
    expect(traceIdFromBody({})).toBe('')
  })

  it('resolveRequestTraceId prefers response header then request then body', () => {
    expect(
      resolveRequestTraceId({
        responseHeaders: { 'X-Trace-Id': 'r' },
        requestHeaders: { 'X-Trace-Id': 'q' },
        body: { trace_id: 'b' },
      }),
    ).toBe('r')
    expect(
      resolveRequestTraceId({
        requestHeaders: { 'X-Trace-Id': 'q' },
        body: { trace_id: 'b' },
      }),
    ).toBe('q')
    expect(resolveRequestTraceId({ body: { traceId: 'b' } })).toBe('b')
  })

  it('extractTraceId reads Error.traceId and Response-like objects', () => {
    const err = new Error('x')
    err.traceId = 'e1'
    expect(extractTraceId(err)).toBe('e1')
    expect(extractTraceId({ headers: { 'X-Trace-Id': 'h1' } })).toBe('h1')
  })

  it('looksLikeTraceId accepts uuid/web-* and rejects error copy', () => {
    expect(looksLikeTraceId('web-abc123')).toBe(true)
    expect(looksLikeTraceId('9ae00c1a-e02d-45fb-a46d-9bc9991c6200')).toBe(true)
    expect(looksLikeTraceId('APBYdiiLaEWyETBNoBm2CZJkSR07YzVR')).toBe(true)
    expect(looksLikeTraceId('创建分组失败')).toBe(false)
    expect(looksLikeTraceId('网络错误')).toBe(false)
  })

  it('extractTraceId accepts shaped strings but not error copy or raw JSON bodies', () => {
    expect(extractTraceId('web-abc123')).toBe('web-abc123')
    expect(extractTraceId('创建分组失败')).toBe('')
    expect(extractTraceId('{"error":"x","detail":"y"}')).toBe('')
    expect(extractTraceId('{"trace_id":"from-body"}')).toBe('from-body')
  })

  it('setDataTraceId sets or removes data-traceId', () => {
    const el = document.createElement('div')
    setDataTraceId(el, 'tid-9')
    expect(el.getAttribute('data-traceId')).toBe('tid-9')
    setDataTraceId(el, '')
    expect(el.hasAttribute('data-traceId')).toBe(false)
  })
})

// 真实网关格式回归（2026-08-07 真机复验）：APISIX 网关与上游服务各自注入
// 一行 X-Trace-Id，浏览器把多行同名字头合并为 "id1, id2"（逗号+空格）。
// traceIdFromHeaders 必须归一化取首段，否则 looksLikeTraceId 拒绝含空格的合并串，
// 导致错误元素的 data-traceId 在真实错误路径下取不到值。
describe('gateway multi-value X-Trace-Id (comma-joined)', () => {
  it('traceIdFromHeaders splits comma-joined header and returns first segment', () => {
    const joined = 'ef4fc3d811c51cc9351f8275f927afc1, ef4fc3d811c51cc9351f8275f927afc1'
    expect(traceIdFromHeaders(new Headers({ 'X-Trace-Id': joined }))).toBe(
      'ef4fc3d811c51cc9351f8275f927afc1',
    )
    expect(traceIdFromHeaders({ 'x-trace-id': joined })).toBe(
      'ef4fc3d811c51cc9351f8275f927afc1',
    )
  })

  it('resolveRequestTraceId returns usable single id for comma-joined response header', () => {
    const joined = 'ef4fc3d811c51cc9351f8275f927afc1, ef4fc3d811c51cc9351f8275f927afc1'
    const tid = resolveRequestTraceId({
      responseHeaders: new Headers({ 'X-Trace-Id': joined }),
    })
    expect(looksLikeTraceId(tid)).toBe(true)
    expect(tid).toBe('ef4fc3d811c51cc9351f8275f927afc1')
  })

  it('extractTraceId reads Response-like object with comma-joined headers', () => {
    const joined = 'ef4fc3d811c51cc9351f8275f927afc1, ef4fc3d811c51cc9351f927afc2'
    expect(extractTraceId({ headers: { 'X-Trace-Id': joined } })).toBe(
      'ef4fc3d811c51cc9351f8275f927afc1',
    )
  })

  it('setDataTraceId sets first segment for comma-joined value', () => {
    const el = document.createElement('div')
    setDataTraceId(el, { headers: { 'X-Trace-Id': 'aaaa1111bbbb2222cccc3333dddd4444, aaaa1111bbbb2222cccc3333dddd4444' } })
    expect(el.getAttribute('data-traceId')).toBe('aaaa1111bbbb2222cccc3333dddd4444')
  })
})
