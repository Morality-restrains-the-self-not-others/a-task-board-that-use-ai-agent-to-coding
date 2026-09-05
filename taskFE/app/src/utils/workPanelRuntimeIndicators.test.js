import { describe, expect, it, vi } from 'vitest'
import {
  fetchWorkspaceRuntimeIndicators,
  normalizeWorkspaceRuntimeIndicators,
  resolveTaskRuntimeIndicatorFlags,
} from './workPanelRuntimeIndicators.js'

describe('workPanelRuntimeIndicators', () => {
  it('normalizes API indicators by task_id (T1)', () => {
    const map = normalizeWorkspaceRuntimeIndicators({
      status: 'success',
      indicators: [
        { task_id: 't1', machine_running: true, container_running: false },
        { task_id: 't2', machine_running: true, container_running: true },
        { task_id: '', machine_running: true },
        null,
      ],
    })
    expect(map.t1).toEqual({ machineRunning: true, containerRunning: false })
    expect(map.t2).toEqual({ machineRunning: true, containerRunning: true })
    expect(map['']).toBeUndefined()
  })

  it('resolves machine-only flags (T2)', () => {
    expect(resolveTaskRuntimeIndicatorFlags({ machineRunning: true, containerRunning: false })).toEqual({
      machineRunning: true,
      containerRunning: false,
      showMachineRing: true,
      showContainerDot: false,
      showRuntimeIndicators: true,
    })
  })

  it('resolves machine+container flags (T3)', () => {
    expect(resolveTaskRuntimeIndicatorFlags({ machineRunning: true, containerRunning: true })).toEqual({
      machineRunning: true,
      containerRunning: true,
      showMachineRing: true,
      showContainerDot: true,
      showRuntimeIndicators: true,
    })
  })

  it('hides indicators when neither running (T4)', () => {
    expect(resolveTaskRuntimeIndicatorFlags(undefined)).toEqual({
      machineRunning: false,
      containerRunning: false,
      showMachineRing: false,
      showContainerDot: false,
      showRuntimeIndicators: false,
    })
  })

  it('fetchWorkspaceRuntimeIndicators maps response', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        status: 'success',
        indicators: [{ task_id: '42', machine_running: true, container_running: true }],
      }),
    }))
    const map = await fetchWorkspaceRuntimeIndicators({
      apiFetch,
      tenantId: '850',
      workspaceId: '901',
    })
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/cloud/compute/workspace-runtime-indicators/tenant_id/850/workspace_id/901',
      expect.objectContaining({ credentials: 'include' }),
    )
    expect(map['42']).toEqual({ machineRunning: true, containerRunning: true })
  })

  it('fetchWorkspaceRuntimeIndicators skips default workspace', async () => {
    const apiFetch = vi.fn()
    const map = await fetchWorkspaceRuntimeIndicators({
      apiFetch,
      tenantId: '850',
      workspaceId: 'default',
    })
    expect(apiFetch).not.toHaveBeenCalled()
    expect(map).toEqual({})
  })
})
