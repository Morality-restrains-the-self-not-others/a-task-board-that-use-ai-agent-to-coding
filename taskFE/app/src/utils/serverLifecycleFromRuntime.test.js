// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { resolveLifecycleFlagsFromRuntimeStatus } from './serverLifecycleFromRuntime.js'

describe('resolveLifecycleFlagsFromRuntimeStatus', () => {
  it('Running → running（大小写不敏感）', () => {
    expect(resolveLifecycleFlagsFromRuntimeStatus('Running')).toEqual({ kind: 'running' })
    expect(resolveLifecycleFlagsFromRuntimeStatus('running')).toEqual({ kind: 'running' })
    expect(resolveLifecycleFlagsFromRuntimeStatus('  RUNNING  ')).toEqual({ kind: 'running' })
  })

  it('过渡态 → starting，并给出可映射的 serverStatusCode', () => {
    expect(resolveLifecycleFlagsFromRuntimeStatus('Initializing')).toEqual({
      kind: 'starting',
      serverStatusCode: 'initializing',
    })
    expect(resolveLifecycleFlagsFromRuntimeStatus('Starting')).toEqual({
      kind: 'starting',
      serverStatusCode: 'starting',
    })
    expect(resolveLifecycleFlagsFromRuntimeStatus('Pending')).toEqual({
      kind: 'starting',
      serverStatusCode: 'processing',
    })
  })

  it('非服务态 → not_serving', () => {
    expect(resolveLifecycleFlagsFromRuntimeStatus('Stopped')).toEqual({
      kind: 'not_serving',
      serverStatusCode: 'stopped',
    })
    expect(resolveLifecycleFlagsFromRuntimeStatus('Released')).toEqual({
      kind: 'not_serving',
      serverStatusCode: 'stopped',
    })
  })

  it('空或未知 → null（不覆盖 SSE）', () => {
    expect(resolveLifecycleFlagsFromRuntimeStatus('')).toBeNull()
    expect(resolveLifecycleFlagsFromRuntimeStatus(null)).toBeNull()
    expect(resolveLifecycleFlagsFromRuntimeStatus(undefined)).toBeNull()
    expect(resolveLifecycleFlagsFromRuntimeStatus('WeirdCloudState')).toBeNull()
  })
})
