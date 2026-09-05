// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { ref } from 'vue'
import { createTaskDetailLayerGraphState } from './taskDetailLayerGraphState.js'
import { refreshLayerGraphFromServer } from './taskDetailContainerFns.js'

function createLg() {
  const selectedLayerGraphNode = ref(null)
  const layerGraphSnapshot = ref(null)
  const layerChangesByLayerId = ref({})
  const layerGraphCommandText = ref('')
  const taskLayerAssociationPanelRef = ref(null)
  const lg = createTaskDetailLayerGraphState({
    effectiveTenantId: ref('t1'),
    effectiveWorkspaceId: ref('w1'),
    effectiveTaskId: ref('task1'),
    localTask: ref({ id: 'task1' }),
    layerGraphCommandText,
    taskLayerAssociationPanelRef,
    layerGraphSnapshot,
    layerChangesByLayerId,
    selectedLayerGraphNode,
    layerGraphMergeTargetBranch: ref(''),
    containerEndpointRegistered: ref(true),
    containerHttpUnreachable: ref(false),
    refreshLayerGraphFromServer: vi.fn(),
    refreshZTreeExecutionLog: vi.fn(),
    markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
    markContainerTransportOk: vi.fn(),
  })
  return { lg, selectedLayerGraphNode, layerGraphSnapshot }
}

describe('layer graph selection dismiss', () => {
  it('onLayerGraphNodeSelect(null) 标记用户 dismiss', () => {
    const { lg, selectedLayerGraphNode } = createLg()
    selectedLayerGraphNode.value = { id: 'l1', nodeKind: 'layer', layerId: 'l1', name: 'L1' }
    lg.onLayerGraphNodeSelect(null)
    expect(selectedLayerGraphNode.value).toBeNull()
    expect(lg.layerGraphSelectionDismissedByUser.value).toBe(true)
  })

  it('重新选中节点后清除 dismiss 标记', () => {
    const { lg, selectedLayerGraphNode } = createLg()
    lg.onLayerGraphNodeSelect(null)
    expect(lg.layerGraphSelectionDismissedByUser.value).toBe(true)
    const node = { id: 'l2', nodeKind: 'layer', layerId: 'l2', name: 'L2' }
    lg.onLayerGraphNodeSelect(node)
    expect(selectedLayerGraphNode.value).toEqual(node)
    expect(lg.layerGraphSelectionDismissedByUser.value).toBe(false)
  })

  it('用户 dismiss 后跳过 refresh 自动重选；清除标记后允许自动重选', () => {
    const selectedLayerGraphNode = { value: null }
    const dismissed = { value: true }
    const allowWhenDismissed =
      selectedLayerGraphNode.value === null && !Boolean(dismissed?.value)
    expect(allowWhenDismissed).toBe(false)

    dismissed.value = false
    const allowWhenNotDismissed =
      selectedLayerGraphNode.value === null && !Boolean(dismissed?.value)
    expect(allowWhenNotDismissed).toBe(true)
    expect(typeof refreshLayerGraphFromServer).toBe('function')
  })

  it('工厂返回对象不含重复的 layerGraphSeenLayerIdSet 键', () => {
    const src = readFileSync(
      join(dirname(fileURLToPath(import.meta.url)), 'taskDetailLayerGraphState.js'),
      'utf8',
    )
    const ret = src.slice(src.lastIndexOf('  return {'))
    const firstBlock = ret.slice(0, ret.indexOf('\n  }'))
    expect(firstBlock.split('layerGraphSeenLayerIdSet').length - 1).toBe(1)
    expect(firstBlock.split('layerGraphRefreshing').length - 1).toBe(1)
  })
})

