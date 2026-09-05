// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  appendCommentIdPath,
  appendCommentIdQuery,
  jsonPostWithCommentId,
  setCommentIdSearchParam,
  trimCommentId,
  withCommentIdBody,
} from './containerForwardCommentId.js'

describe('containerForwardCommentId', () => {
  it('trimCommentId 去掉空白', () => {
    expect(trimCommentId('  cmt_a  ')).toBe('cmt_a')
    expect(trimCommentId('')).toBe('')
  })

  it('appendCommentIdPath 插在 query 前', () => {
    expect(appendCommentIdPath('/api/x/?layer_id=1', 'cmt_a')).toBe(
      '/api/x/comment_id/cmt_a/?layer_id=1',
    )
    expect(appendCommentIdPath('/api/x/?layer_id=1', 'cmt_a')).not.toMatch(/[?&]comment_id=/)
  })

  it('appendCommentIdPath 无 id 时保持原路径', () => {
    expect(appendCommentIdPath('/api/x/?layer_id=1', '')).toBe('/api/x/?layer_id=1')
  })

  it('appendCommentIdPath 已有 path 段时不重复追加', () => {
    expect(appendCommentIdPath('/api/x/comment_id/cmt_a/?job_id=J1', 'cmt_a')).toBe(
      '/api/x/comment_id/cmt_a/?job_id=J1',
    )
  })

  it('appendCommentIdQuery 与 path helper 同形态', () => {
    expect(appendCommentIdQuery('/api/x/?layer_id=1', 'cmt_a')).toBe(
      '/api/x/comment_id/cmt_a/?layer_id=1',
    )
  })

  it('withCommentIdBody 写入 comment_id', () => {
    expect(withCommentIdBody({ layer_id: 'L1' }, 'cmt_a')).toEqual({
      layer_id: 'L1',
      comment_id: 'cmt_a',
    })
  })

  it('withCommentIdBody 两评论 id 不同则 body 不同', () => {
    const a = withCommentIdBody({ layer_id: 'L1' }, 'cmt_a')
    const b = withCommentIdBody({ layer_id: 'L1' }, 'cmt_b')
    expect(a.comment_id).toBe('cmt_a')
    expect(b.comment_id).toBe('cmt_b')
  })

  it('jsonPostWithCommentId 序列化 body.comment_id', () => {
    const init = jsonPostWithCommentId({ layer_id: 'L1' }, 'cmt_a')
    expect(JSON.parse(init.body).comment_id).toBe('cmt_a')
    expect(init.method).toBe('POST')
  })

  it('setCommentIdSearchParam 写入 URLSearchParams（兼容旧 query）', () => {
    const qs = new URLSearchParams({ job_id: 'J1' })
    setCommentIdSearchParam(qs, 'cmt_a')
    expect(qs.get('comment_id')).toBe('cmt_a')
    expect(qs.get('job_id')).toBe('J1')
  })
})
