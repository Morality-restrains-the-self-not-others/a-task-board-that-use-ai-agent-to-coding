// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  bootstrapAnchorDisplayCommand,
  bootstrapAnchorZtStyle,
  isBootstrapAnchorLayer,
  layerZtreeBootstrapAnchorFields,
  resolveLayerGraphHasRealWritableLayer,
  snapshotHasRealWritableLayer,
  ztreeHasRealWritableLayer,
  bootstrapFailureMessageFromLayers,
  renameBootstrapAnchorPendingNameOnFailure,
} from './layerZtreeBootstrapAnchor.js'

describe('layerZtreeBootstrapAnchor', () => {
  const emptyPending = {
    layer_id: '20260827_074114_cdea43',
    meta_kind: 'empty',
    bootstrap_pending: true,
  }
  const emptyFailed = {
    ...emptyPending,
    bootstrap_pending: false,
    bootstrap_failed: true,
    bootstrap_error: 'repo-clone-credentials 未返回完整',
  }
  const realLayer = {
    layer_id: 'L-clone',
    meta_kind: 'clone',
    command: 'git clone https://example.com/r.git',
  }

  it('treats empty / pending / failed as bootstrap anchors, not real writable layers', () => {
    expect(isBootstrapAnchorLayer(emptyPending)).toBe(true)
    expect(isBootstrapAnchorLayer({ meta_kind: 'empty' })).toBe(true)
    expect(isBootstrapAnchorLayer(emptyFailed)).toBe(true)
    expect(isBootstrapAnchorLayer(realLayer)).toBe(false)
    expect(
      isBootstrapAnchorLayer({
        layer_id: '20260827_074114_cdea43',
        mind_state: 'pending',
        git_worktree_dirty: false,
        command: null,
      }),
    ).toBe(true)
    expect(snapshotHasRealWritableLayer([emptyPending])).toBe(false)
    expect(snapshotHasRealWritableLayer([emptyFailed])).toBe(false)
    expect(snapshotHasRealWritableLayer([emptyPending, realLayer])).toBe(true)
    expect(snapshotHasRealWritableLayer([])).toBe(false)
  })

  it('ztreeHasRealWritableLayer ignores virtual root and bootstrapAnchor layer rows', () => {
    expect(
      ztreeHasRealWritableLayer([
        { id: '__layer_graph_root__', nodeKind: 'virtual' },
        { id: '__layer__:boot', nodeKind: 'layer', bootstrapAnchor: true },
      ]),
    ).toBe(false)
    expect(
      ztreeHasRealWritableLayer([
        { id: '__layer_graph_root__', nodeKind: 'virtual' },
        { id: '__layer__:L1', nodeKind: 'layer' },
      ]),
    ).toBe(true)
    expect(ztreeHasRealWritableLayer([{ id: 'L1' }])).toBe(true)
  })

  it('resolveLayerGraphHasRealWritableLayer prefers explicit flag', () => {
    expect(
      resolveLayerGraphHasRealWritableLayer({
        layerGraphHasRealWritableLayer: false,
        layerGraphZNodes: [{ id: 'L1', nodeKind: 'layer' }],
      }),
    ).toBe(false)
    expect(
      resolveLayerGraphHasRealWritableLayer({
        layerGraphZNodes: [{ id: 'L1', nodeKind: 'layer', bootstrapAnchor: true }],
      }),
    ).toBe(false)
  })

  it('failed empty layer displays failure command instead of 正在准备可写层', () => {
    expect(bootstrapAnchorDisplayCommand(emptyPending)).toBe('正在准备可写层')
    expect(bootstrapAnchorDisplayCommand(emptyFailed)).toContain('repo-clone-credentials')
    expect(bootstrapAnchorDisplayCommand(realLayer)).toBe('')
    expect(bootstrapAnchorZtStyle(emptyPending)).toBe('active')
    expect(layerZtreeBootstrapAnchorFields(emptyPending)).toEqual({ bootstrapAnchor: true })
    expect(layerZtreeBootstrapAnchorFields(realLayer)).toEqual({})
  })

  it('bootstrapFailureMessageFromLayers reads failed empty layer error', () => {
    expect(bootstrapFailureMessageFromLayers([emptyPending])).toBe('')
    expect(bootstrapFailureMessageFromLayers([emptyFailed])).toContain('repo-clone-credentials')
  })

  it('renames 正在准备可写层 on credentials failure', () => {
    const nodes = [
      { id: '__layer__:boot', nodeKind: 'layer', bootstrapAnchor: true, name: '正在准备可写层' },
    ]
    const renamed = renameBootstrapAnchorPendingNameOnFailure(
      nodes,
      'error_code=REPO_CLONE_CREDENTIALS_INCOMPLETE',
    )
    expect(renamed[0].name).toBe('引导克隆失败')
    expect(renameBootstrapAnchorPendingNameOnFailure(nodes, '')[0].name).toBe('正在准备可写层')
  })
})
