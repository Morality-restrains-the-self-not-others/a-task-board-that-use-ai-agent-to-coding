import { describe, expect, it } from 'vitest'
import {
  commentIdFromButtonArg,
  createScopedCommentIdMemory,
  stopVmBodyWithCommentId,
  trimCommentId,
  withCommentIdQuery,
} from './cloudComputeCommentQuery.js'

describe('withCommentIdQuery', () => {
  const base =
    '/api/cloud/compute/workbench-link/tenant_id/t1/workspace_id/ws1?task_id=task-1'

  it('appends comment_id for comment card actions', () => {
    expect(withCommentIdQuery(base, 'cmt_abc')).toBe(
      '/api/cloud/compute/workbench-link/tenant_id/t1/workspace_id/ws1/comment_id/cmt_abc/?task_id=task-1',
    )
    expect(withCommentIdQuery(base, 'cmt_abc')).not.toMatch(/[?&]comment_id=/)
  })

  it('encodes special characters', () => {
    expect(withCommentIdQuery(base, 'c mt')).toContain('/comment_id/c%20mt/')
  })

  it('skips empty comment id (caller must not send the request)', () => {
    expect(withCommentIdQuery(base, '')).toBe(base)
    expect(withCommentIdQuery(base, '  ')).toBe(base)
    expect(withCommentIdQuery(base, null)).toBe(base)
  })
})

describe('stopVmBodyWithCommentId', () => {
  it('includes comment_id when stopping from a comment card', () => {
    expect(stopVmBodyWithCommentId('task-1', 'cmt_1')).toEqual({
      task_id: 'task-1',
      comment_id: 'cmt_1',
      stop_reason: 'user_stop',
    })
  })

  it('omits comment_id when empty', () => {
    expect(stopVmBodyWithCommentId('task-1', '')).toEqual({
      task_id: 'task-1',
      stop_reason: 'user_stop',
    })
  })
})

describe('trimCommentId', () => {
  it('trims', () => {
    expect(trimCommentId('  x  ')).toBe('x')
  })
})

describe('createScopedCommentIdMemory', () => {
  it('remembers string comment id for poll and keeps it on click-event', () => {
    const mem = createScopedCommentIdMemory()
    expect(mem.fromAction('cmt_1')).toBe('cmt_1')
    expect(mem.forPoll()).toBe('cmt_1')
    expect(mem.fromAction()).toBe('cmt_1')
    expect(mem.fromAction({ type: 'click' })).toBe('cmt_1')
    expect(mem.forPoll()).toBe('cmt_1')
    expect(commentIdFromButtonArg('c2')).toBe('c2')
    expect(commentIdFromButtonArg({ type: 'click' })).toBe('')
  })
})
