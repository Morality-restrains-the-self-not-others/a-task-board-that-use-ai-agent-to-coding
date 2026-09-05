// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { ref, nextTick } from 'vue'
import { installTaskDetailZTreeExecLogWatchers } from './taskDetailZTreeExecLogWatchers.js'

describe('installTaskDetailZTreeExecLogWatchers', () => {
  it('hydrates execution log from SaaS when job is selected even if endpoint is not registered', async () => {
    const refreshZTreeExecutionLog = vi.fn().mockResolvedValue(undefined)
    const zlog = {
      selectedLayerGraphNodeLogKey: ref(''),
      refreshZTreeExecutionLog,
      activeJobExecLogPoller: { sync: vi.fn(), stop: vi.fn() },
      zTreeLogTargets: ref({ jobId: '', layerId: '' }),
      prefetchLayerChangeSummariesForDirtyLayers: vi.fn(),
    }
    const deps = {
      layerGraphSnapshot: ref({ jobs: [] }),
      containerEndpointRegistered: ref(false),
      containerHttpUnreachable: ref(false),
    }
    installTaskDetailZTreeExecLogWatchers(zlog, deps)
    zlog.zTreeLogTargets.value = { jobId: 'J1', layerId: 'L1' }
    await nextTick()
    expect(refreshZTreeExecutionLog).toHaveBeenCalled()
    expect(zlog.activeJobExecLogPoller.sync).toHaveBeenCalled()
  })

  it('still refreshes SaaS execution log when containerReleased but skips layer-change prefetch', async () => {
    const refreshZTreeExecutionLog = vi.fn().mockResolvedValue(undefined)
    const prefetchLayerChangeSummariesForDirtyLayers = vi.fn()
    const zlog = {
      selectedLayerGraphNodeLogKey: ref('node-1'),
      refreshZTreeExecutionLog,
      activeJobExecLogPoller: { sync: vi.fn(), stop: vi.fn() },
      zTreeLogTargets: ref({ jobId: 'J1', layerId: 'L1' }),
      prefetchLayerChangeSummariesForDirtyLayers,
      containerReleased: ref(true),
    }
    const deps = {
      layerGraphSnapshot: ref({ jobs: [] }),
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      containerReleased: ref(true),
    }
    installTaskDetailZTreeExecLogWatchers(zlog, deps)
    zlog.zTreeLogTargets.value = { jobId: 'J2', layerId: 'L1' }
    deps.layerGraphSnapshot.value = { jobs: [{ id: 'J2' }] }
    await nextTick()
    expect(refreshZTreeExecutionLog).toHaveBeenCalled()
    expect(prefetchLayerChangeSummariesForDirtyLayers).not.toHaveBeenCalled()
  })
  it('refreshes archived execution log when selected node changes while released', async () => {
    const refreshZTreeExecutionLog = vi.fn().mockResolvedValue(undefined)
    const zlog = {
      selectedLayerGraphNodeLogKey: ref('node-1'),
      refreshZTreeExecutionLog,
      activeJobExecLogPoller: { sync: vi.fn(), stop: vi.fn() },
      zTreeLogTargets: ref({ jobId: 'J1', layerId: 'L1' }),
      prefetchLayerChangeSummariesForDirtyLayers: vi.fn(),
      containerReleased: ref(true),
    }
    const deps = {
      layerGraphSnapshot: ref({ jobs: [] }),
      containerEndpointRegistered: ref(false),
      containerHttpUnreachable: ref(false),
      containerReleased: ref(true),
    }
    installTaskDetailZTreeExecLogWatchers(zlog, deps)
    zlog.selectedLayerGraphNodeLogKey.value = 'node-2'
    await nextTick()
    expect(refreshZTreeExecutionLog).toHaveBeenCalledWith({ reset: true })
  })
})
