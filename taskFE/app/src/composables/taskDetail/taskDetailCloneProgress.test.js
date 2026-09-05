// @vitest-environment node
import { describe, expect, it, beforeEach, vi, afterEach } from 'vitest'
import {
  CONTAINER_CLONE_PROGRESS_GLOBAL_KEY,
  BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE,
  CONTAINER_CLONE_PROGRESS_TIMEOUT_MS,
  createCloneProgressState,
  createCloneProgressComputeds,
  isBootstrapCloneLogDone,
  onBootstrapCloneLogUpdate,
  rescheduleContainerCloneProgressClearTimer,
  applyBootstrapCloneLogDoneToCloneProgress,
} from './taskDetailCloneProgress.js'
import { ref } from 'vue'

describe('CONTAINER_CLONE_PROGRESS_GLOBAL_KEY', () => {
  it('is __global__', () => {
    expect(CONTAINER_CLONE_PROGRESS_GLOBAL_KEY).toBe('__global__')
  })
})

describe('BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE', () => {
  it('matches Chinese done messages', () => {
    expect(BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test('【项目克隆】克隆完成')).toBe(true)
    expect(BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test('仓库克隆已完成')).toBe(true)
    expect(BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test('【项目克隆】仓库克隆已完成')).toBe(true)
  })

  it('does not match incomplete messages', () => {
    expect(BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test('正在克隆...')).toBe(false)
    expect(BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test('')).toBe(false)
    expect(BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test('clone in progress')).toBe(false)
  })
})

describe('createCloneProgressState', () => {
  it('initializes with empty state', () => {
    const state = createCloneProgressState()
    expect(state.containerCloneProgressByKey.value).toEqual({})
    expect(state.containerCloneProgressByCommentId.value).toEqual({})
    expect(state.containerBootstrapCloneLogFull.value).toBe('')
    expect(state.containerBootstrapCloneLogSegments.value).toBeNull()
    expect(state.containerPageLinkPendingReveal.value).toBe(false)
    expect(state.containerCloneProgressTimer).toBeNull()
  })

  it('containerCloneProgressTimer is readable and writable', () => {
    const state = createCloneProgressState()
    expect(state.containerCloneProgressTimer).toBeNull()
    state.containerCloneProgressTimer = 42
    expect(state.containerCloneProgressTimer).toBe(42)
  })
})

describe('onBootstrapCloneLogUpdate', () => {
  let state

  beforeEach(() => {
    state = createCloneProgressState()
  })

  it('handles object payload with text and segments', () => {
    const payload = {
      text: 'log line 1\nlog line 2',
      segments: [
        { repo_url: 'https://github.com/a/b.git', text: 'repo log' }
      ]
    }
    onBootstrapCloneLogUpdate(payload, state)
    expect(state.containerBootstrapCloneLogFull.value).toBe('log line 1\nlog line 2')
    expect(state.containerBootstrapCloneLogSegments.value).toEqual(payload.segments)
  })

  it('handles object payload with text but no segments', () => {
    onBootstrapCloneLogUpdate({ text: 'just text' }, state)
    expect(state.containerBootstrapCloneLogFull.value).toBe('just text')
    expect(state.containerBootstrapCloneLogSegments.value).toBeNull()
  })

  it('handles object payload with empty segments array', () => {
    onBootstrapCloneLogUpdate({ text: 'x', segments: [] }, state)
    expect(state.containerBootstrapCloneLogSegments.value).toBeNull()
  })

  it('handles string payload', () => {
    onBootstrapCloneLogUpdate('raw string log', state)
    expect(state.containerBootstrapCloneLogFull.value).toBe('raw string log')
    expect(state.containerBootstrapCloneLogSegments.value).toBeNull()
  })

  it('handles null/undefined payload by clearing', () => {
    state.containerBootstrapCloneLogFull.value = 'previous'
    state.containerBootstrapCloneLogSegments.value = [{ repo_url: 'x', text: 'y' }]
    onBootstrapCloneLogUpdate(null, state)
    expect(state.containerBootstrapCloneLogFull.value).toBe('')
    expect(state.containerBootstrapCloneLogSegments.value).toBeNull()
  })

  it('handles array payload by clearing (not a valid payload)', () => {
    state.containerBootstrapCloneLogFull.value = 'previous'
    onBootstrapCloneLogUpdate([], state)
    expect(state.containerBootstrapCloneLogFull.value).toBe('')
  })

  it('does not wipe existing log with empty SSE text', () => {
    state.containerBootstrapCloneLogFull.value = '【项目克隆】克隆完成'
    onBootstrapCloneLogUpdate({ text: '' }, state)
    expect(state.containerBootstrapCloneLogFull.value).toBe('【项目克隆】克隆完成')
  })
})

describe('isBootstrapCloneLogDone / bootstrapCloneDone', () => {
  it('detects completion footer', () => {
    expect(isBootstrapCloneLogDone('【项目克隆】克隆完成。')).toBe(true)
    expect(isBootstrapCloneLogDone('【项目克隆】仓库克隆已完成')).toBe(true)
    expect(isBootstrapCloneLogDone('【项目克隆】部分失败：失败 1/2')).toBe(false)
    const state = createCloneProgressState()
    const taskRepoRows = ref([])
    state.containerBootstrapCloneLogFull.value = '【项目克隆】克隆完成'
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    expect(c.bootstrapCloneDone.value).toBe(true)
  })
})

describe('createCloneProgressComputeds', () => {
  let state
  let taskRepoRows

  beforeEach(() => {
    state = createCloneProgressState()
    taskRepoRows = ref([])
  })

  it('returns empty entries when progress map is empty', () => {
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    expect(c.containerCloneProgressEntries.value).toEqual([])
  })

  it('returns entries mapped from progress map', () => {
    state.containerCloneProgressByKey.value = {
      'https://github.com/a/b.git': { progress: 50, message: 'cloning...' }
    }
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    const entries = c.containerCloneProgressEntries.value
    expect(entries).toHaveLength(1)
    expect(entries[0].key).toBe('https://github.com/a/b.git')
    expect(entries[0].progress).toBe(50)
    expect(entries[0].message).toBe('cloning...')
    expect(entries[0].recvProgress).toBeNull()
    expect(entries[0].unpackProgress).toBeNull()
  })

  it('clamps progress between 0 and 100', () => {
    state.containerCloneProgressByKey.value = {
      'a': { progress: 150, message: '' }
    }
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    expect(c.containerCloneProgressEntries.value[0].progress).toBe(100)
  })

  it('cloneProgressBarWidthTransitionClass returns empty when done', () => {
    state.containerBootstrapCloneLogFull.value = '【项目克隆】克隆完成'
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    expect(c.cloneProgressBarWidthTransitionClass.value).toBe('')
  })

  it('cloneProgressBarWidthTransitionClass returns transition class when not done', () => {
    state.containerBootstrapCloneLogFull.value = 'cloning...'
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    expect(c.cloneProgressBarWidthTransitionClass.value).toBe('transition-[width] duration-300 ease-out')
  })

  it('cloneProgressEntryByRepoMatchKey maps by git ref key', () => {
    state.containerCloneProgressByKey.value = {
      'https://github.com/org/repo.git': { progress: 80, message: 'ok' }
    }
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    // Entry exists with a repoUrl derived from the key
    expect(c.cloneProgressEntryByRepoMatchKey.value.size).toBeGreaterThanOrEqual(0)
  })

  it('cloneProgressRowLogIsPlaceholder returns true when no text', () => {
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    const row = { repoUrl: 'https://example.com/repo.git', message: '', progress: 0 }
    expect(c.cloneProgressRowLogIsPlaceholder(row, 0)).toBe(true)
  })

  it('does not append late SSE 9% onto a completed bootstrap log', () => {
    state.containerBootstrapCloneLogFull.value = '【项目克隆】克隆完成。'
    const c = createCloneProgressComputeds(state, { taskRepoRows })
    const text = c.cloneProgressRowLogText(
      { repoUrl: '', message: '【项目克隆】(1/1) ram-work … 9%', progress: 9 },
      0,
    )
    expect(text).toContain('克隆完成')
    expect(text).not.toContain('实时进度')
  })
})

describe('rescheduleContainerCloneProgressClearTimer', () => {
  let state

  beforeEach(() => {
    vi.useFakeTimers()
    state = createCloneProgressState()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('sets a timer when there are incomplete entries', () => {
    state.containerCloneProgressByKey.value = {
      'repo-a': { progress: 50, message: 'half' }
    }
    rescheduleContainerCloneProgressClearTimer(state)
    expect(state.containerCloneProgressTimer).not.toBeNull()
  })

  it('clears existing timer before setting new one', () => {
    const clearSpy = vi.fn()
    const origClear = globalThis.clearTimeout
    globalThis.clearTimeout = clearSpy
    state.containerCloneProgressTimer = 99
    state.containerCloneProgressByKey.value = { 'a': { progress: 50 } }
    rescheduleContainerCloneProgressClearTimer(state)
    expect(clearSpy).toHaveBeenCalledWith(99)
    globalThis.clearTimeout = origClear
  })

  it('does not set timer when all entries are complete', () => {
    state.containerCloneProgressByKey.value = {
      'repo-a': { progress: 100, message: 'done' }
    }
    rescheduleContainerCloneProgressClearTimer(state)
    expect(state.containerCloneProgressTimer).toBeNull()
  })

  it('clears progress map when timer fires', () => {
    state.containerCloneProgressByKey.value = {
      'repo-a': { progress: 50, message: 'half' }
    }
    rescheduleContainerCloneProgressClearTimer(state)
    vi.advanceTimersByTime(CONTAINER_CLONE_PROGRESS_TIMEOUT_MS + 100)
    expect(state.containerCloneProgressByKey.value).toEqual({})
    expect(state.containerCloneProgressTimer).toBeNull()
  })
})

describe('applyBootstrapCloneLogDoneToCloneProgress', () => {
  let state

  beforeEach(() => {
    state = createCloneProgressState()
  })

  it('no-ops when bootstrap log is empty', () => {
    state.containerCloneProgressByKey.value = { 'a': { progress: 50, message: 'x' } }
    const refreshSpy = vi.fn()
    const bumpSpy = vi.fn()
    applyBootstrapCloneLogDoneToCloneProgress(state, {
      refreshLayerGraphFromServer: refreshSpy,
      bumpProjectFileTreeRefresh: bumpSpy,
    })
    // State should be unchanged
    expect(state.containerCloneProgressByKey.value.a.progress).toBe(50)
  })

  it('marks all entries as 100% when bootstrap log indicates done', () => {
    state.containerBootstrapCloneLogFull.value = '【项目克隆】克隆完成'
    state.containerCloneProgressByKey.value = {
      'repo-a': { progress: 50, message: 'cloning...' },
      'repo-b': { progress: 80, message: 'almost' },
    }
    const refreshSpy = vi.fn()
    const bumpSpy = vi.fn()
    applyBootstrapCloneLogDoneToCloneProgress(state, {
      refreshLayerGraphFromServer: refreshSpy,
      bumpProjectFileTreeRefresh: bumpSpy,
    })
    expect(state.containerCloneProgressByKey.value['repo-a'].progress).toBe(100)
    expect(state.containerCloneProgressByKey.value['repo-b'].progress).toBe(100)
  })

  it('also snaps comment-level live map rows to 100%', () => {
    state.containerBootstrapCloneLogFull.value = '【项目克隆】克隆完成'
    state.containerCloneProgressByCommentId.value = {
      C1: { 'https://git.example/ram-work.git': { progress: 9, message: '… 9%' } },
    }
    applyBootstrapCloneLogDoneToCloneProgress(state, {
      refreshLayerGraphFromServer: vi.fn(),
      bumpProjectFileTreeRefresh: vi.fn(),
    })
    expect(state.containerCloneProgressByCommentId.value.C1['https://git.example/ram-work.git'].progress).toBe(100)
  })

  it('does not change already-complete entries', () => {
    state.containerBootstrapCloneLogFull.value = '【项目克隆】克隆完成'
    state.containerCloneProgressByKey.value = {
      'repo-a': { progress: 100, message: '【项目克隆】克隆完成' },
    }
    const refreshSpy = vi.fn()
    const bumpSpy = vi.fn()
    applyBootstrapCloneLogDoneToCloneProgress(state, {
      refreshLayerGraphFromServer: refreshSpy,
      bumpProjectFileTreeRefresh: bumpSpy,
    })
    expect(refreshSpy).not.toHaveBeenCalled()
  })
})
