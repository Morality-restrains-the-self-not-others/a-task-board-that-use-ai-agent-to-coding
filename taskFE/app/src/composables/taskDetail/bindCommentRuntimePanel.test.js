import { describe, expect, it, vi } from 'vitest'
import { bindCommentRuntimePanel, runtimeStatusFromCommentPanel } from './bindCommentRuntimePanel.js'

describe('bindCommentRuntimePanel', () => {
  it('wraps action buttons with comment id', () => {
    const openWorkbenchLink = vi.fn()
    const fetchServerRuntimeStatus = vi.fn()
    const stopServer = vi.fn()
    const bound = bindCommentRuntimePanel(
      {
        showRuntimeActionButtons: true,
        openWorkbenchLink,
        fetchServerRuntimeStatus,
        stopServer,
      },
      'cmt_99',
    )
    expect(bound.commentId).toBe('cmt_99')
    bound.openWorkbenchLink()
    bound.fetchServerRuntimeStatus()
    bound.stopServer()
    expect(openWorkbenchLink).toHaveBeenCalledWith('cmt_99')
    expect(fetchServerRuntimeStatus).toHaveBeenCalledWith('cmt_99')
    expect(stopServer).toHaveBeenCalledWith('cmt_99')
  })

  it('does not wrap when comment id empty', () => {
    const openWorkbenchLink = vi.fn()
    const panel = { openWorkbenchLink }
    expect(bindCommentRuntimePanel(panel, '')).toBe(panel)
    expect(bindCommentRuntimePanel(null, 'c1')).toBe(null)
  })

  it('overlays per-comment snapshot so two comments do not share display', () => {
    const panel = {
      serverRuntimeStatusDisplayText: '共享单例',
      showRuntimeActionButtons: false,
      snapshotForComment(commentId) {
        if (commentId === 'cmt_a') {
          return { serverRuntimeStatusDisplayText: '运行中-A', showRuntimeActionButtons: true }
        }
        if (commentId === 'cmt_b') {
          return { serverRuntimeStatusDisplayText: '已停止-B', showRuntimeActionButtons: false }
        }
        return {}
      },
    }
    const a = bindCommentRuntimePanel(panel, 'cmt_a')
    const b = bindCommentRuntimePanel(panel, 'cmt_b')
    expect(a.serverRuntimeStatusDisplayText).toBe('运行中-A')
    expect(a.showRuntimeActionButtons).toBe(true)
    expect(a.commentId).toBe('cmt_a')
    expect(b.serverRuntimeStatusDisplayText).toBe('已停止-B')
    expect(b.showRuntimeActionButtons).toBe(false)
    expect(b.commentId).toBe('cmt_b')
  })

  it('waiting_previous overlays stray Running snapshot so serial comments do not look started', () => {
    const panel = {
      serverRuntimeStatusDisplayText: '运行中',
      showRuntimeActionButtons: true,
      snapshotForComment() {
        return { serverRuntimeStatusDisplayText: '运行中', showRuntimeActionButtons: true }
      },
    }
    const bound = bindCommentRuntimePanel(panel, 'cmt_java', 'waiting_previous')
    expect(bound.serverRuntimeStatusDisplayText).toBe('等待前序')
    expect(bound.showRuntimeActionButtons).toBe(false)
    expect(bound.serverRuntimeStatusMessage).toContain('等待前序')
  })

  it('reads per-comment Released status from snapshot for summary badge', () => {
    const panel = {
      snapshotForComment(commentId) {
        if (commentId === 'cmt_rel') {
          return { serverRuntimeStatus: 'Released', serverRuntimeStatusDisplayText: '已释放' }
        }
        return { serverRuntimeStatus: 'Running', serverRuntimeStatusDisplayText: '运行中' }
      },
    }
    expect(runtimeStatusFromCommentPanel(panel, 'cmt_rel')).toBe('Released')
    expect(runtimeStatusFromCommentPanel(panel, 'cmt_run')).toBe('Running')
    expect(runtimeStatusFromCommentPanel(null, 'cmt_rel')).toBe('')
  })

  it('forwards comment created_at into snapshotForComment so CreationTime lag can render', () => {
    const snapshotForComment = vi.fn(() => ({ serverRuntimeStatusDetails: [] }))
    bindCommentRuntimePanel({ snapshotForComment }, 'cmt_lag', '', '2026-08-29T01:42:46Z')
    expect(snapshotForComment).toHaveBeenCalledWith('cmt_lag', { commentCreatedAt: '2026-08-29T01:42:46Z' })
  })
})
