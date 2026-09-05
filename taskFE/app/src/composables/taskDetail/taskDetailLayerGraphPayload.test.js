// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import {
  createApplyLayerGraphFromPayload,
  reconcileLayerGraphJobs,
} from './taskDetailLayerGraphPayload.js'

describe('reconcileLayerGraphJobs', () => {
  it('prefers layer.job_status terminal over stale job running', () => {
    const out = reconcileLayerGraphJobs(
      [{ id: 'J1', layer_id: 'L1', status: 'running' }],
      [],
      [{ layer_id: 'L1', job_status: 'completed' }],
    )
    expect(out[0].status).toBe('completed')
  })

  it('does not regress local terminal status to active from lagging /api/jobs', () => {
    const out = reconcileLayerGraphJobs(
      [{ id: 'J1', layer_id: 'L1', status: 'running' }],
      [{ id: 'J1', layer_id: 'L1', status: 'completed' }],
      [{ layer_id: 'L1', job_status: 'running' }],
    )
    expect(out[0].status).toBe('completed')
  })
})

describe('createApplyLayerGraphFromPayload', () => {
  it('ignores empty layers when snapshot already has layers', () => {
    const layerGraphSnapshot = ref({
      layers: [{ layer_id: 'l1' }],
      jobs: [],
      layers_root: '',
      bootstrap_layer_id: '',
    })
    const containerLayerGraphAuthInvalid = ref(true)
    const apply = createApplyLayerGraphFromPayload({ layerGraphSnapshot, containerLayerGraphAuthInvalid })
    apply({ layers: [], jobs: [] })
    expect(layerGraphSnapshot.value.layers).toHaveLength(1)
    expect(containerLayerGraphAuthInvalid.value).toBe(true)
  })

  it('writes merged snapshot and clears auth invalid flag', () => {
    const layerGraphSnapshot = ref(null)
    const containerLayerGraphAuthInvalid = ref(true)
    const apply = createApplyLayerGraphFromPayload({ layerGraphSnapshot, containerLayerGraphAuthInvalid })
    apply({
      layers: [{ layer_id: 'l2', git_remote: { ahead: 0 } }],
      jobs: [{ id: 'j1', status: 'done' }],
      layers_root: '/layers',
      bootstrap_layer_id: 'boot',
    })
    expect(layerGraphSnapshot.value.layers).toHaveLength(1)
    expect(layerGraphSnapshot.value.jobs).toHaveLength(1)
    expect(layerGraphSnapshot.value.layers_root).toBe('/layers')
    expect(containerLayerGraphAuthInvalid.value).toBe(false)
  })

  it('reconciles stale running job against layer completed on apply', () => {
    const layerGraphSnapshot = ref({
      layers: [{ layer_id: 'L1', job_status: 'completed' }],
      jobs: [{ id: 'J1', layer_id: 'L1', status: 'completed' }],
      layers_root: '',
      bootstrap_layer_id: '',
    })
    const containerLayerGraphAuthInvalid = ref(false)
    const apply = createApplyLayerGraphFromPayload({ layerGraphSnapshot, containerLayerGraphAuthInvalid })
    apply({
      layers: [{ layer_id: 'L1', job_status: 'completed' }],
      jobs: [{ id: 'J1', layer_id: 'L1', status: 'running' }],
    })
    expect(layerGraphSnapshot.value.jobs[0].status).toBe('completed')
  })
})
