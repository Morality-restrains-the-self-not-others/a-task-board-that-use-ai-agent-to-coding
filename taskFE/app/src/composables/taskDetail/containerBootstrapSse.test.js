// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import {
  applyContainerBootstrapFailure,
  clearContainerBootstrapFailure,
  handleContainerBootstrapSse,
  traceIdFromSseStatusData,
} from './containerBootstrapSse.js'

describe('traceIdFromSseStatusData', () => {
  it('reads trace_id and trims', () => {
    expect(traceIdFromSseStatusData({ trace_id: '  abc123  ' })).toBe('abc123')
  })

  it('reads camelCase traceId', () => {
    expect(traceIdFromSseStatusData({ traceId: 'camel-1' })).toBe('camel-1')
  })

  it('returns empty when missing or blank', () => {
    expect(traceIdFromSseStatusData({})).toBe('')
    expect(traceIdFromSseStatusData({ trace_id: '   ' })).toBe('')
    expect(traceIdFromSseStatusData(null)).toBe('')
  })
})

describe('applyContainerBootstrapFailure', () => {
  it('records message and SSE trace_id', () => {
    const deps = {
      containerBootstrapFailureMessage: ref(''),
      containerBootstrapFailureTraceId: ref(''),
    }
    applyContainerBootstrapFailure(
      {
        status: 'container_bootstrap_failed',
        message: 'repo-clone-credentials 未返回完整',
        trace_id: 'boot-trace-9',
      },
      deps,
    )
    expect(deps.containerBootstrapFailureMessage.value).toContain('repo-clone-credentials')
    expect(deps.containerBootstrapFailureTraceId.value).toBe('boot-trace-9')
  })

  it('does not invent unknown when SSE has no trace_id', () => {
    const deps = {
      containerBootstrapFailureMessage: ref(''),
      containerBootstrapFailureTraceId: ref('stale'),
    }
    applyContainerBootstrapFailure({ status: 'container_bootstrap_failed', message: 'fail' }, deps)
    expect(deps.containerBootstrapFailureTraceId.value).toBe('')
  })
})

describe('handleContainerBootstrapSse', () => {
  it('clears message and trace_id on complete', () => {
    const refreshLayerGraphFromServer = vi.fn()
    const bumpProjectFileTreeRefresh = vi.fn()
    const deps = {
      containerBootstrapFailureMessage: ref('old'),
      containerBootstrapFailureTraceId: ref('old-trace'),
      refreshLayerGraphFromServer,
      bumpProjectFileTreeRefresh,
    }
    expect(handleContainerBootstrapSse({ status: 'container_bootstrap_complete' }, deps)).toBe(true)
    expect(deps.containerBootstrapFailureMessage.value).toBe('')
    expect(deps.containerBootstrapFailureTraceId.value).toBe('')
    expect(refreshLayerGraphFromServer).toHaveBeenCalledWith(true)
    expect(bumpProjectFileTreeRefresh).toHaveBeenCalled()
  })

  it('ignores unrelated status', () => {
    expect(handleContainerBootstrapSse({ status: 'runtime_hydrate' }, {})).toBe(false)
  })

  it('fills clone-log text on container_bootstrap_progress when log is empty', () => {
    const onBootstrapCloneLogUpdate = vi.fn()
    const deps = {
      containerBootstrapFailureMessage: ref('stale'),
      containerBootstrapFailureTraceId: ref('stale-trace'),
      containerBootstrapCloneLogFull: ref(''),
      onBootstrapCloneLogUpdate,
    }
    expect(
      handleContainerBootstrapSse(
        {
          status: 'container_bootstrap_progress',
          phase: 'task_detail_begin',
          message: '开始拉取任务详情…',
        },
        deps,
      ),
    ).toBe(true)
    expect(deps.containerBootstrapFailureMessage.value).toBe('stale')
    expect(onBootstrapCloneLogUpdate).toHaveBeenCalledWith({ text: '开始拉取任务详情…' })
  })

  it('clears bootstrap failure on clone_begin progress', () => {
    const onBootstrapCloneLogUpdate = vi.fn()
    const deps = {
      containerBootstrapFailureMessage: ref('old fail'),
      containerBootstrapFailureTraceId: ref('old-trace'),
      containerBootstrapCloneLogFull: ref('开始拉取任务详情'),
      onBootstrapCloneLogUpdate,
    }
    expect(
      handleContainerBootstrapSse(
        {
          status: 'container_bootstrap_progress',
          phase: 'clone_begin',
          message: '任务详情已就绪，开始项目克隆…',
        },
        deps,
      ),
    ).toBe(true)
    expect(deps.containerBootstrapFailureMessage.value).toBe('')
    expect(deps.containerBootstrapFailureTraceId.value).toBe('')
    expect(onBootstrapCloneLogUpdate).toHaveBeenCalledWith({
      text: '任务详情已就绪，开始项目克隆…',
    })
  })
})

describe('clearContainerBootstrapFailure', () => {
  it('clears both refs', () => {
    const deps = {
      containerBootstrapFailureMessage: ref('x'),
      containerBootstrapFailureTraceId: ref('y'),
    }
    clearContainerBootstrapFailure(deps)
    expect(deps.containerBootstrapFailureMessage.value).toBe('')
    expect(deps.containerBootstrapFailureTraceId.value).toBe('')
  })
})
