// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  commentBindingStatusLabel,
  commentExecutionBindingBadgeText,
  commentExecutionInactiveHint,
  commentHasEffectivePredecessors,
  isCommentExecutionReleased,
  listCommentPredecessors,
} from './useCommentExecutionContext.js'

describe('listCommentPredecessors', () => {
  it('returns [] for independent comments', () => {
    const comments = [
      { id: 'C1', content: 'first', commentKind: 'user' },
      { id: 'C2', content: 'second', execution_mode: 'independent' },
    ]
    expect(listCommentPredecessors(comments[1], comments)).toEqual([])
  })

  it('lists implicit predecessors in task order with binding status', () => {
    const comments = [
      { id: 'C1', content: '先跑测试套件', commentKind: 'user' },
      { id: 'C2', content: '再部署', commentKind: 'ai' },
      { id: 'C3', content: '等前序', commentKind: 'user' },
    ]
    const statusById = { C1: 'running', C2: 'completed' }
    const rows = listCommentPredecessors(comments[2], comments, {
      bindingStatusFor: (id) => statusById[id] || '',
    })
    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({
      id: 'C1',
      summary: '先跑测试套件',
      status: 'running',
      statusLabel: '运行中',
      finished: false,
      missing: false,
    })
    expect(rows[1]).toMatchObject({
      id: 'C2',
      status: 'completed',
      statusLabel: '已完成',
      finished: true,
    })
  })

  it('skips container_agent comments as implicit predecessors', () => {
    const comments = [
      { id: 'C1', content: 'user', commentKind: 'user' },
      { id: 'A1', content: 'agent step', commentKind: 'container_agent' },
      { id: 'C2', content: 'next', commentKind: 'user' },
    ]
    const rows = listCommentPredecessors(comments[2], comments, {
      bindingStatusFor: () => 'pending',
    })
    expect(rows.map((r) => r.id)).toEqual(['C1'])
  })

  it('uses explicit depends_on_comment_ids and keeps missing ids', () => {
    const comments = [
      { id: 'C1', content: 'alpha', commentKind: 'user' },
      { id: 'C9', content: 'waiter', depends_on_comment_ids: ['C1', 'missing_99'] },
    ]
    const rows = listCommentPredecessors(comments[1], comments, {
      bindingStatusFor: (id) => (id === 'C1' ? 'failed' : ''),
    })
    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({
      id: 'C1',
      statusLabel: '失败',
      finished: true,
      missing: false,
    })
    expect(rows[1]).toMatchObject({
      id: 'missing_99',
      summary: '前序评论 missing_99（不在当前页）',
      statusLabel: '未加载',
      finished: false,
      missing: true,
    })
  })

  it('labels unloaded explicit predecessors as 未加载 instead of 未知', () => {
    const comments = [
      { id: 'C1', content: 'waiter', depends_on_comment_ids: ['ghost_7'] },
    ]
    const rows = listCommentPredecessors(comments[0], comments, {
      bindingStatusFor: () => '',
    })
    expect(rows[0]).toMatchObject({
      id: 'ghost_7',
      statusLabel: '未加载',
      missing: true,
      finished: false,
    })
    expect(rows[0].summary).toContain('不在当前页')
  })

  it('truncates long summaries at 72 chars', () => {
    const long = '前序正文'.repeat(40)
    const comments = [
      { id: 'C1', content: long, commentKind: 'user' },
      { id: 'C2', content: 'now', commentKind: 'user' },
    ]
    const [row] = listCommentPredecessors(comments[1], comments)
    expect(row.summary.endsWith('…')).toBe(true)
    expect(row.summary.length).toBe(73)
  })
})

describe('commentHasEffectivePredecessors', () => {
  it('is true when any listed predecessor is unfinished', () => {
    const comments = [
      { id: 'C1', content: 'a', commentKind: 'user' },
      { id: 'C2', content: 'b', commentKind: 'user' },
    ]
    expect(
      commentHasEffectivePredecessors(comments[1], comments, {
        bindingStatusFor: (id) => (id === 'C1' ? 'running' : ''),
      }),
    ).toBe(true)
  })

  it('is false when all predecessors are completed/failed/released', () => {
    const comments = [
      { id: 'C1', content: 'a', commentKind: 'user' },
      { id: 'C2', content: 'b', commentKind: 'user' },
    ]
    expect(
      commentHasEffectivePredecessors(comments[1], comments, {
        bindingStatusFor: () => 'completed',
      }),
    ).toBe(false)
  })
})

describe('commentBindingStatusLabel', () => {
  it('maps waiting_previous to 等待前序', () => {
    expect(commentBindingStatusLabel('waiting_previous')).toBe('等待前序')
  })

  it('maps cancelled to 已终止', () => {
    expect(commentBindingStatusLabel('cancelled')).toBe('已终止')
  })

  it('returns 未知 for empty status', () => {
    expect(commentBindingStatusLabel('')).toBe('未知')
  })
})

describe('commentExecutionBindingBadgeText', () => {
  it('overlays Released runtime onto stale running binding', () => {
    expect(commentExecutionBindingBadgeText('running', 'Released')).toBe('服务器已释放')
    expect(commentExecutionBindingBadgeText('running', '已释放')).toBe('服务器已释放')
    expect(commentExecutionBindingBadgeText('running', 'terminated')).toBe('服务器已释放')
  })

  it('uses binding released even without runtime snapshot', () => {
    expect(commentExecutionBindingBadgeText('released', '')).toBe('服务器已释放')
  })

  it('keeps 容器 运行中 when runtime is still Running', () => {
    expect(commentExecutionBindingBadgeText('running', 'Running')).toBe('容器 运行中')
  })

  it('does not treat Stopped as 服务器已释放', () => {
    expect(commentExecutionBindingBadgeText('running', 'Stopped')).toBe('容器 运行中')
  })
})

describe('isCommentExecutionReleased', () => {
  it('matches badge overlay for Released runtime', () => {
    expect(isCommentExecutionReleased('running', 'Released')).toBe(true)
    expect(isCommentExecutionReleased('released', '')).toBe(true)
    expect(isCommentExecutionReleased('running', 'Running')).toBe(false)
  })
})

describe('commentExecutionInactiveHint', () => {
  it('announces release when runtime snapshot is Released despite stale running binding', () => {
    expect(
      commentExecutionInactiveHint({ bindingStatus: 'running', serverRuntimeStatus: 'Released' }),
    ).toBe('服务器已释放，本评论容器不再运行。')
    expect(
      commentExecutionInactiveHint({ bindingStatus: 'running', serverRuntimeStatus: '已释放' }),
    ).toBe('服务器已释放，本评论容器不再运行。')
    expect(
      commentExecutionInactiveHint({ bindingStatus: 'running', serverRuntimeStatus: 'Terminated' }),
    ).toBe('服务器已释放，本评论容器不再运行。')
  })

  it('announces release when binding itself is released (no runtime snapshot)', () => {
    expect(
      commentExecutionInactiveHint({ bindingStatus: 'released', ownsSharedContainer: true }),
    ).toBe('服务器已释放，本评论容器不再运行。')
  })

  it('keeps original hint while runtime is still Running', () => {
    expect(
      commentExecutionInactiveHint({ bindingStatus: 'running', ownsSharedContainer: true, serverRuntimeStatus: 'Running' }),
    ).toBe('本评论已挂接独立实例；连接与运行状态见本面板。')
  })

  it('does not treat Stopped as released', () => {
    expect(
      commentExecutionInactiveHint({ bindingStatus: 'running', serverRuntimeStatus: 'Stopped' }),
    ).toBe('本评论已独立调度，独立实例分配中。')
  })
})
