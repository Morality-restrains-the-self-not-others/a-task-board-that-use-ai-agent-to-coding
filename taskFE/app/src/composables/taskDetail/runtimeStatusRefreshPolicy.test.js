import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  commentIdForRuntimeRefresh,
  commentIdFromStatusEvent,
  runtimeStatusFromStatusEvent,
  applyPushedRuntimeSnapshot,
  shouldRefreshRuntimeOnStatusMessage,
} from './runtimeStatusRefreshPolicy.js'
import { createCommentRuntimeSnapshotStore } from './commentRuntimeSnapshotStore.js'

const here = dirname(fileURLToPath(import.meta.url))

describe('shouldRefreshRuntimeOnStatusMessage', () => {
  it('true for start success and stop success copy', () => {
    expect(shouldRefreshRuntimeOnStatusMessage('aliyun服务器启动成功！')).toBe(true)
    expect(shouldRefreshRuntimeOnStatusMessage('Mock 实例已停止')).toBe(true)
    expect(shouldRefreshRuntimeOnStatusMessage('等待启动...')).toBe(false)
    expect(shouldRefreshRuntimeOnStatusMessage('')).toBe(false)
  })
})

describe('commentIdForRuntimeRefresh', () => {
  it('uses event comment_id on start success, never a poll-slot fallback', () => {
    expect(commentIdForRuntimeRefresh({
      message: 'aliyun服务器启动成功！',
      eventCommentId: 'C1',
    })).toBe('C1')
  })

  it('skips Describe when success/stop copy has no event comment_id', () => {
    expect(commentIdForRuntimeRefresh({
      message: 'aliyun服务器启动成功！',
      eventCommentId: '',
    })).toBe('')
    expect(commentIdForRuntimeRefresh({
      message: '停止虚拟机成功',
      eventCommentId: null,
    })).toBe('')
  })

  it('does not Describe on unrelated status copy even if an id is present', () => {
    expect(commentIdForRuntimeRefresh({
      message: '正在创建实例',
      eventCommentId: 'C1',
    })).toBe('')
  })
})

describe('commentIdFromStatusEvent', () => {
  it('reads comment_id / commentId and ignores missing payload', () => {
    expect(commentIdFromStatusEvent({ comment_id: 'C1' })).toBe('C1')
    expect(commentIdFromStatusEvent({ commentId: 'C2' })).toBe('C2')
    expect(commentIdFromStatusEvent({})).toBe('')
    expect(commentIdFromStatusEvent(null)).toBe('')
  })
})

describe('runtimeStatusFromStatusEvent', () => {
  it('reads runtime_status from the push payload', () => {
    expect(runtimeStatusFromStatusEvent({ runtime_status: 'Running' })).toBe('Running')
    expect(runtimeStatusFromStatusEvent({ runtimeStatus: 'Stopped' })).toBe('Stopped')
    expect(runtimeStatusFromStatusEvent({})).toBe('')
  })
})

describe('applyPushedRuntimeSnapshot', () => {
  it('writes the comment slot without implying an HTTP call', () => {
    const store = createCommentRuntimeSnapshotStore()
    expect(applyPushedRuntimeSnapshot(store, {
      commentId: 'C1',
      runtimeStatus: 'Running',
      message: 'aliyun服务器启动成功！',
      traceId: 'tr-1',
    })).toBe(true)
    expect(store.get('C1')).toMatchObject({
      status: 'Running',
      message: 'aliyun服务器启动成功！',
      traceId: 'tr-1',
      loading: false,
    })
  })

  it('does not write when comment_id or runtime_status is missing', () => {
    const store = createCommentRuntimeSnapshotStore()
    expect(applyPushedRuntimeSnapshot(store, {
      commentId: '',
      runtimeStatus: 'Running',
    })).toBe(false)
    expect(applyPushedRuntimeSnapshot(store, {
      commentId: 'C1',
      runtimeStatus: '',
    })).toBe(false)
    expect(store.state.value).toEqual({})
  })
})

describe('useServerConfigRuntime has no background Describe timers', () => {
  it('does not start runtimeStatusPoll or startingFallbackPollId', () => {
    const src = readFileSync(join(here, 'useServerConfigRuntime.js'), 'utf8')
    expect(src).not.toMatch(/createRuntimeStatusPollController/)
    expect(src).not.toMatch(/startingFallbackPollId/)
    expect(src).not.toMatch(/fetchServerRuntimeStatus\(cid\)\s*\n\s*fetchServerContent/)
  })
})

describe('no UI poll loops for startup-status or binding advance', () => {
  it('binding advance does not start a 30s timer', () => {
    const src = readFileSync(join(here, 'useBindingAdvancePolling.js'), 'utf8')
    expect(src).not.toMatch(/setInterval/)
    expect(src).not.toMatch(/POLL_INTERVAL_MS/)
  })

  it('startup status controller does not setInterval', () => {
    const src = readFileSync(join(here, 'serverStartupStatusPoll.js'), 'utf8')
    expect(src).not.toMatch(/setInterval/)
  })
})
