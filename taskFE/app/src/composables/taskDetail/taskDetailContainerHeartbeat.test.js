// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import {
  CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS,
  createContainerHeartbeatState,
  createLayerGraphFetchBackoffState,
  createLayerGraphHeartbeatRefresh,
  createServerRuntimeLayerGraphGateState,
  layerGraphSnapshotHasActiveJob,
  layerGraphSnapshotHasContent,
} from './taskDetailContainerHeartbeat.js'

function makeHeartbeatDeps(overrides = {}) {
  return {
    effectiveTenantId: ref('t1'),
    effectiveWorkspaceId: ref('w1'),
    effectiveTaskId: ref('task-1'),
    containerEndpointRegistered: ref(true),
    containerPageUrl: ref(''),
    containerVscodeUrl: ref(''),
    containerPageLinkPendingReveal: ref(false),
    layerGraphSnapshot: ref(null),
    serverUrl: ref(''),
    isServerRunning: ref(true),
    isServerStarting: ref(false),
    resetLayerGraphFetchBackoff: vi.fn(),
    ensureServerRuntimeAllowsContainerLayerGraph: vi.fn(async () => true),
    refreshLayerGraphFromServer: vi.fn(),
    applyLayerGraphFromPayload: vi.fn(),
    establishSSEConnection: vi.fn(),
    ...overrides,
  }
}

describe('createContainerHeartbeatState', () => {
  it('initializes idle heartbeat and transport flags', () => {
    const hb = createContainerHeartbeatState(makeHeartbeatDeps())
    expect(hb.containerHeartbeatStatus.value).toBe('idle')
    expect(hb.containerHttpUnreachable.value).toBe(false)
    expect(hb.containerLayerGraphAuthInvalid.value).toBe(false)
    expect(hb.containerHeartbeatLogLines.value).toEqual([])
  })

  it('markContainerTransportOk clears unreachable state', () => {
    const hb = createContainerHeartbeatState(makeHeartbeatDeps())
    hb.containerHttpUnreachable.value = true
    hb.markContainerTransportOk()
    expect(hb.containerHttpUnreachable.value).toBe(false)
    expect(hb.containerUnreachableProbeBackoffMs).toBe(CONTAINER_UNREACHABLE_PROBE_BACKOFF_INITIAL_MS)
  })

  it('marks container unreachable on layer-graph 503 (online service not ready)', () => {
    const hb = createContainerHeartbeatState(makeHeartbeatDeps())
    hb.markContainerTransportUnreachableIfForwardingFailed(
      503,
      '{"detail":"容器内在线服务未启动，请先启动容器"}',
    )
    expect(hb.containerHttpUnreachable.value).toBe(true)
  })

  it('startContainerHeartbeat calls establishSSEConnection when ids present', () => {
    const establishSSEConnection = vi.fn()
    const hb = createContainerHeartbeatState(makeHeartbeatDeps({ establishSSEConnection }))
    hb.startContainerHeartbeat()
    expect(establishSSEConnection).toHaveBeenCalledWith('task-1')
  })

  it('applyContainerHeartbeatSeqFromSse parses numeric seq fields', () => {
    const hb = createContainerHeartbeatState(makeHeartbeatDeps())
    hb.applyContainerHeartbeatSeqFromSse({
      container_seq: '3',
      saas_ack: 7,
      uplink_ok: true,
      downlink_ok: false,
    })
    expect(hb.containerHeartbeatSeqInfo.value.containerSeq).toBe(3)
    expect(hb.containerHeartbeatSeqInfo.value.saasAck).toBe(7)
    expect(hb.containerHeartbeatSeqInfo.value.uplinkOk).toBe(true)
    expect(hb.containerHeartbeatSeqInfo.value.downlinkOk).toBe(false)
  })

  it('markServerRuntimeServing clears serverRuntimeNotServing without wiping URLs', () => {
    const deps = makeHeartbeatDeps({
      containerPageUrl: ref('http://page'),
      containerVscodeUrl: ref('http://vscode'),
      serverUrl: ref('http://server'),
      containerEndpointRegistered: ref(true),
    })
    const hb = createContainerHeartbeatState(deps)
    hb.serverRuntimeNotServing.value = true
    hb.markServerRuntimeServing()
    expect(hb.serverRuntimeNotServing.value).toBe(false)
    expect(deps.containerPageUrl.value).toBe('http://page')
    expect(deps.containerVscodeUrl.value).toBe('http://vscode')
    expect(deps.serverUrl.value).toBe('http://server')
    expect(deps.containerEndpointRegistered.value).toBe(true)
  })
})

describe('layerGraphSnapshot helpers', () => {
  it('layerGraphSnapshotHasContent requires a real writable layer, not empty pending anchors', () => {
    expect(layerGraphSnapshotHasContent(null)).toBe(false)
    expect(layerGraphSnapshotHasContent({ layers: [] })).toBe(false)
    expect(layerGraphSnapshotHasContent({ layers: [{ layer_id: 'l1' }] })).toBe(true)
    expect(
      layerGraphSnapshotHasContent({
        layers: [{ layer_id: 'boot', meta_kind: 'empty', bootstrap_pending: true }],
      }),
    ).toBe(false)
  })

  it('layerGraphSnapshotHasActiveJob detects running jobs', () => {
    expect(layerGraphSnapshotHasActiveJob({ jobs: [{ status: 'done' }] })).toBe(false)
    expect(layerGraphSnapshotHasActiveJob({ jobs: [{ status: 'running' }] })).toBe(true)
    expect(layerGraphSnapshotHasActiveJob({ jobs: [{ status: 'PENDING' }] })).toBe(true)
  })
})

describe('createLayerGraphHeartbeatRefresh', () => {
  it('refreshes when snapshot is empty', () => {
    const refreshLayerGraphFromServer = vi.fn()
    const refresh = createLayerGraphHeartbeatRefresh({
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      containerLayerGraphAuthInvalid: ref(false),
      containerPageLinkPendingReveal: ref(false),
      layerGraphSnapshot: ref(null),
      refreshLayerGraphFromServer,
    })
    refresh.maybeRefreshLayerGraphOnContainerHeartbeatOk()
    expect(refreshLayerGraphFromServer).toHaveBeenCalled()
  })

  it('also fetches bootstrap clone-log when snapshot is empty', () => {
    const refreshLayerGraphFromServer = vi.fn()
    const fetchContainerBootstrapCloneLog = vi.fn()
    const refresh = createLayerGraphHeartbeatRefresh({
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      containerLayerGraphAuthInvalid: ref(false),
      containerPageLinkPendingReveal: ref(false),
      layerGraphSnapshot: ref({ layers: [], jobs: [] }),
      refreshLayerGraphFromServer,
      fetchContainerBootstrapCloneLog,
    })
    refresh.maybeRefreshLayerGraphOnContainerHeartbeatOk()
    expect(refreshLayerGraphFromServer).toHaveBeenCalled()
    expect(fetchContainerBootstrapCloneLog).toHaveBeenCalled()
  })

  it('still refreshes when snapshot only has empty bootstrap_pending layer', () => {
    const refreshLayerGraphFromServer = vi.fn()
    const fetchContainerBootstrapCloneLog = vi.fn()
    const refresh = createLayerGraphHeartbeatRefresh({
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      containerLayerGraphAuthInvalid: ref(false),
      containerPageLinkPendingReveal: ref(false),
      layerGraphSnapshot: ref({
        layers: [{ layer_id: 'boot', meta_kind: 'empty', bootstrap_pending: true }],
        jobs: [],
      }),
      refreshLayerGraphFromServer,
      fetchContainerBootstrapCloneLog,
    })
    refresh.maybeRefreshLayerGraphOnContainerHeartbeatOk()
    expect(refreshLayerGraphFromServer).toHaveBeenCalled()
    expect(fetchContainerBootstrapCloneLog).toHaveBeenCalled()
  })
})

describe('createServerRuntimeLayerGraphGateState', () => {
  it('resetServerRuntimeLayerGraphGateCache clears gate', () => {
    const gate = createServerRuntimeLayerGraphGateState({
      containerEndpointRegistered: ref(true),
      containerHeartbeatStatus: ref('connected'),
      containerHttpUnreachable: ref(false),
    })
    gate.serverRuntimeLayerGraphGateAllowed = false
    gate.resetServerRuntimeLayerGraphGateCache()
    expect(gate.serverRuntimeLayerGraphGateAllowed).toBe(true)
  })
})

describe('formatLayerGraphCommandErrorForUser 403', () => {
  it('does not leave users with a bare HTTP 403', () => {
    const hb = createContainerHeartbeatState(makeHeartbeatDeps())
    const fmt = hb.formatLayerGraphCommandErrorForUser
    expect(fmt(403, 'HTTP 403')).not.toBe('HTTP 403')
    expect(fmt(403, 'HTTP 403')).toMatch(/权限|授权|登录|刷新/)
    expect(fmt(403, '403 Forbidden')).not.toBe('HTTP 403')
    expect(fmt(403, 'forbidden scope')).toMatch(/容器配置/)
    expect(fmt(403, '容器配置不存在或尚未就绪')).toMatch(/容器配置/)
    expect(fmt(403, 'budget exceeded')).toBe('budget exceeded')
  })
})

describe('createLayerGraphFetchBackoffState', () => {
  it('registerLayerGraphFetchFailure increases backoff window', () => {
    const backoff = createLayerGraphFetchBackoffState()
    const before = backoff.layerGraphFetchBackoffUntil
    backoff.registerLayerGraphFetchFailure()
    expect(backoff.layerGraphFetchBackoffUntil).toBeGreaterThan(before)
    backoff.resetLayerGraphFetchBackoff()
    expect(backoff.layerGraphFetchBackoffUntil).toBe(0)
  })
})
