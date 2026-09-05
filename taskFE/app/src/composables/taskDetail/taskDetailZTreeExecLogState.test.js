// @vitest-environment node
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import {
  resolveZTreeLogTargets,
  normalizeLayerChangesPayload,
  mergeLayerChangesPage,
  layerChangesContentFingerprint,
  layerChangePathIsGitInternal,
  createTaskDetailZTreeExecLogState,
} from './taskDetailZTreeExecLogState.js'

describe('resolveZTreeLogTargets', () => {
  const jobs = [
    { id: 'job-clone', layer_id: 'layer-1', command_kind: 'clone', created_at: '2026-01-01T00:00:00Z' },
    { id: 'job-run-1', layer_id: 'layer-1', command_kind: 'trae', created_at: '2026-01-01T01:00:00Z' },
    { id: 'job-run-2', layer_id: 'layer-1', command_kind: 'trae', created_at: '2026-01-01T02:00:00Z' },
    { id: 'job-layer-2', layer_id: 'layer-2', command_kind: 'shell', created_at: '2026-01-02T00:00:00Z' },
  ]

  it('resolves layer node id with __layer__: prefix to newest non-clone job', () => {
    const node = {
      nodeKind: 'layer',
      id: '__layer__:layer-1',
      layerId: 'layer-1',
    }
    expect(resolveZTreeLogTargets(node, jobs)).toEqual({
      layerId: 'layer-1',
      jobId: 'job-run-2',
    })
  })

  it('resolves job node directly', () => {
    const node = { nodeKind: 'job', id: 'job-layer-2' }
    expect(resolveZTreeLogTargets(node, jobs)).toEqual({
      layerId: 'layer-2',
      jobId: 'job-layer-2',
    })
  })

  it('returns empty targets for virtual nodes', () => {
    expect(resolveZTreeLogTargets({ nodeKind: 'virtual', id: 'root' }, jobs)).toEqual({
      layerId: '',
      jobId: '',
    })
  })
})

describe('normalizeLayerChangesPayload', () => {
  it('filters .git internal paths from changes', () => {
    const result = normalizeLayerChangesPayload({
      layer_id: 'layer-1',
      changes: [
        { path: 'src/main.js', kind: 'modified' },
        { path: '.git/config', kind: 'modified' },
        { path: 'pkg/.git/HEAD', kind: 'modified' },
      ],
      truncated: false,
    })

    expect(result).not.toBeNull()
    expect(result.layer_id).toBe('layer-1')
    expect(result.changes).toHaveLength(1)
    expect(result.changes[0].path).toBe('src/main.js')
    expect(result.change_count).toBe(1)
  })

  it('returns null when layer_id is missing', () => {
    expect(normalizeLayerChangesPayload({ changes: [] })).toBeNull()
  })

  it('preserves server change_count / has_more / next_offset for paged payloads', () => {
    const result = normalizeLayerChangesPayload({
      layer_id: 'layer-1',
      changes: [{ path: 'a.js', kind: 'added' }],
      change_count: 250,
      has_more: true,
      next_offset: 100,
      truncated: false,
    })
    expect(result.change_count).toBe(250)
    expect(result.has_more).toBe(true)
    expect(result.next_offset).toBe(100)
    expect(result.changes).toHaveLength(1)
  })
})

describe('mergeLayerChangesPage', () => {
  it('appends unique paths and keeps pagination metadata', () => {
    const prev = normalizeLayerChangesPayload({
      layer_id: 'L1',
      changes: [{ path: 'a.js', kind: 'added' }],
      change_count: 3,
      has_more: true,
      next_offset: 1,
    })
    const page = normalizeLayerChangesPayload({
      layer_id: 'L1',
      changes: [
        { path: 'a.js', kind: 'added' },
        { path: 'b.js', kind: 'modified' },
      ],
      change_count: 3,
      has_more: true,
      next_offset: 2,
    })
    const merged = mergeLayerChangesPage(prev, page)
    expect(merged.changes.map((c) => c.path)).toEqual(['a.js', 'b.js'])
    expect(merged.change_count).toBe(3)
    expect(merged.has_more).toBe(true)
    expect(merged.next_offset).toBe(2)
  })
})

describe('layerChangePathIsGitInternal', () => {
  it('detects git internal paths', () => {
    expect(layerChangePathIsGitInternal('.git/config')).toBe(true)
    expect(layerChangePathIsGitInternal('foo/.git/HEAD')).toBe(true)
    expect(layerChangePathIsGitInternal('src/app.js')).toBe(false)
  })
})

describe('ingestLayerChangesFromExecutionPayload file-tree bump gate', () => {
  function makeState(bumpSpy) {
    const layerChangesByLayerId = ref({})
    const state = createTaskDetailZTreeExecLogState({
      selectedLayerGraphNode: ref(null),
      layerGraphSnapshot: ref(null),
      layerChangesByLayerId,
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('42'),
      taskRepoRows: ref([]),
      repoCloneIdentityIdForUrl: () => '',
      bumpProjectFileTreeRefresh: bumpSpy,
    })
    return { state, layerChangesByLayerId }
  }

  it('bumps file tree only when layer_changes fingerprint changes', () => {
    const bumpSpy = vi.fn()
    const { state } = makeState(bumpSpy)
    const payload = {
      layer_changes: {
        layer_id: 'layer-1',
        changes: [{ path: 'a.js', kind: 'modified' }],
        truncated: false,
      },
    }
    state.ingestLayerChangesFromExecutionPayload(payload)
    expect(bumpSpy).toHaveBeenCalledTimes(1)
    state.ingestLayerChangesFromExecutionPayload(payload)
    expect(bumpSpy).toHaveBeenCalledTimes(1)
    state.ingestLayerChangesFromExecutionPayload({
      layer_changes: {
        layer_id: 'layer-1',
        changes: [
          { path: 'a.js', kind: 'modified' },
          { path: 'b.js', kind: 'added' },
        ],
        truncated: false,
      },
    })
    expect(bumpSpy).toHaveBeenCalledTimes(2)
  })

  it('fingerprint is order-independent for same change set', () => {
    const a = normalizeLayerChangesPayload({
      layer_id: 'L',
      changes: [
        { path: 'b.js', kind: 'added' },
        { path: 'a.js', kind: 'modified' },
      ],
    })
    const b = normalizeLayerChangesPayload({
      layer_id: 'L',
      changes: [
        { path: 'a.js', kind: 'modified' },
        { path: 'b.js', kind: 'added' },
      ],
    })
    expect(layerChangesContentFingerprint(a)).toBe(layerChangesContentFingerprint(b))
  })
})

describe('createTaskDetailZTreeExecLogState comment_id gate', () => {
  function makeState(overrides = {}) {
    return createTaskDetailZTreeExecLogState({
      selectedLayerGraphNode: ref({ nodeKind: 'job', id: 'job-run-1' }),
      layerGraphSnapshot: ref({
        jobs: [{ id: 'job-run-1', layer_id: 'layer-1', status: 'completed', command_kind: 'trae' }],
        layers: [],
      }),
      layerChangesByLayerId: ref({}),
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('42'),
      taskRepoRows: ref([]),
      repoCloneIdentityIdForUrl: () => '',
      ...overrides,
    })
  }

  it('disables layer-changes refresh without comment_id', () => {
    const state = makeState({ commentId: ref(''), displayComments: ref([]) })
    expect(state.layerChangesRefreshEnabled.value).toBe(false)
  })

  it('enables layer-changes refresh when comment_id is present', () => {
    const state = makeState({ commentId: ref('cmt-exec') })
    expect(state.layerChangesRefreshEnabled.value).toBe(true)
  })
})

describe('createTaskDetailZTreeExecLogState git action gates', () => {
  it('blocks staged actions while selected job is running', () => {
    const selectedLayerGraphNode = ref({ nodeKind: 'job', id: 'job-run-1' })
    const layerGraphSnapshot = ref({
      jobs: [{ id: 'job-run-1', layer_id: 'layer-1', status: 'running', command_kind: 'trae' }],
      layers: [],
    })
    const state = createTaskDetailZTreeExecLogState({
      selectedLayerGraphNode,
      layerGraphSnapshot,
      layerChangesByLayerId: ref({}),
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('42'),
      taskRepoRows: ref([]),
      repoCloneIdentityIdForUrl: () => '',
    })
    expect(state.layerChangesGitStagedActionsBlocked.value).toBe(true)
    layerGraphSnapshot.value.jobs[0].status = 'completed'
    expect(state.layerChangesGitStagedActionsBlocked.value).toBe(false)
  })

  it('blocks commit when linked repos lack clone identity', () => {
    const state = createTaskDetailZTreeExecLogState({
      selectedLayerGraphNode: ref(null),
      layerGraphSnapshot: ref(null),
      layerChangesByLayerId: ref({}),
      containerEndpointRegistered: ref(true),
      containerHttpUnreachable: ref(false),
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('42'),
      taskRepoRows: ref([{ url: 'https://github.com/a/b.git' }]),
      repoCloneIdentityIdForUrl: () => '',
    })
    expect(state.layerChangesGitCommitIdentityBlocked.value).toBe(true)
  })
})
