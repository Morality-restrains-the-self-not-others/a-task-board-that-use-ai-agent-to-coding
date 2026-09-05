// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  applyLegacyStartupStatusDom,
  buildLegacyStartupStatusPollUrl,
  getWorkspaceIdFromLegacyContext,
  startLegacyStartupStatusPoll,
} from './modal-task-detail-startup.js'

describe('modal-task-detail-startup', () => {
  it('buildLegacyStartupStatusPollUrl includes event_id', () => {
    const url = buildLegacyStartupStatusPollUrl({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      eventId: '862532588237045760',
    })
    expect(url).toContain('server-startup-status')
    expect(url).toContain('event_id=862532588237045760')
  })

  it('buildLegacyStartupStatusPollUrl puts comment_id in path not query', () => {
    const url = buildLegacyStartupStatusPollUrl({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      eventId: '862532588237045760',
      commentId: 'cmt_a',
    })
    expect(url).toContain('/comment_id/cmt_a/')
    expect(url).not.toMatch(/[?&]comment_id=/)
  })

  it('buildLegacyStartupStatusPollUrl without commentId keeps task-level', () => {
    const url = buildLegacyStartupStatusPollUrl({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
    })
    expect(url).toContain('server-startup-status')
    expect(url).not.toMatch(/[?&]comment_id=/)
    expect(url).not.toContain('/comment_id/')
  })

  it('getWorkspaceIdFromLegacyContext reads current workspace', () => {
    global.window = {
      currentUser: { current_workspace: { id: 'ws-1' } },
      location: { pathname: '/tenant/t1/work_panel/' },
    }
    expect(getWorkspaceIdFromLegacyContext()).toBe('ws-1')
  })

  it('applyLegacyStartupStatusDom updates status message element', () => {
    const el = { textContent: '', style: {}, classList: { remove: () => {}, add: () => {} } }
    global.document = {
      getElementById: (id) => {
        if (id === 'status-message') return el
        return null
      },
    }
    applyLegacyStartupStatusDom({
      status: 'processing',
      message: '正在启动...',
      progress: 10,
    })
    expect(el.textContent).toBe('正在启动...')
  })

  it('startLegacyStartupStatusPoll does not start a timer', () => {
    const calls = []
    global.window = {
      setInterval: (...args) => {
        calls.push(args)
        return 1
      },
      apiFetch: async () => ({ ok: true, json: async () => ({ status: 'processing' }) }),
    }
    startLegacyStartupStatusPoll({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      eventId: '1',
    })
    expect(calls).toEqual([])
  })
})
