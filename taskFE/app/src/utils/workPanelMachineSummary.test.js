import { describe, expect, it, vi } from 'vitest'
import {
  MACHINE_STATUS_FILTER,
  fetchWorkspaceMachineSummary,
  filterTodosByMachineStatus,
  formatWorkspaceMachineSummaryLabel,
  machineStatusFilterChipLabel,
  normalizeWorkspaceMachineSummary,
  toggleMachineStatusFilter,
  todoMatchesMachineStatusFilter,
} from './workPanelMachineSummary.js'

describe('normalizeWorkspaceMachineSummary', () => {
  it('normalizes API payload', () => {
    const s = normalizeWorkspaceMachineSummary({
      status: 'success',
      started_count: 2,
      idle_count: 1,
      busy_count: 1,
      idle_recycle_minutes: 30,
    })
    expect(s).toEqual({
      startedCount: 2,
      startingCount: 0,
      idleCount: 1,
      busyCount: 1,
      idleRecycleMinutes: 30,
    })
  })

  it('normalizes starting_count when present', () => {
    const s = normalizeWorkspaceMachineSummary({
      started_count: 1,
      starting_count: 3,
      idle_count: 0,
      busy_count: 1,
      idle_recycle_minutes: 30,
    })
    expect(s.startedCount).toBe(1)
    expect(s.startingCount).toBe(3)
  })

  it('returns null for invalid payload', () => {
    expect(normalizeWorkspaceMachineSummary(null)).toBeNull()
    expect(normalizeWorkspaceMachineSummary({ started_count: 'x' })).toBeNull()
  })
})

describe('formatWorkspaceMachineSummaryLabel', () => {
  it('formats started/idle/recycle', () => {
    expect(
      formatWorkspaceMachineSummaryLabel({
        startedCount: 3,
        startingCount: 0,
        idleCount: 1,
        busyCount: 2,
        idleRecycleMinutes: 45,
        }),
    ).toBe('机器节点：已启动 3 · 闲置 1 · 闲置回收：45 分钟')
  })

  it('includes starting when non-zero', () => {
    expect(
      formatWorkspaceMachineSummaryLabel({
        startedCount: 2,
        startingCount: 4,
        idleCount: 1,
        busyCount: 1,
        idleRecycleMinutes: 30,
        }),
    ).toBe('机器节点：已启动 2 · 启动中 4 · 闲置 1 · 闲置回收：30 分钟')
  })

  it('shows closed when recycle minutes is 0', () => {
    expect(
      formatWorkspaceMachineSummaryLabel({
        startedCount: 0,
        startingCount: 0,
        idleCount: 0,
        busyCount: 0,
        idleRecycleMinutes: 0,
        }),
    ).toBe('机器节点：已启动 0 · 闲置 0 · 闲置回收：已关闭')
  })

  it('falls back when summary missing', () => {
    expect(formatWorkspaceMachineSummaryLabel(null)).toBe('机器节点：—')
  })
})

describe('fetchWorkspaceMachineSummary', () => {
  it('calls summary endpoint', async () => {
    const apiFetch = vi.fn(async () => ({
      ok: true,
      json: async () => ({
        started_count: 1,
        idle_count: 0,
        busy_count: 1,
        idle_recycle_minutes: 30,
        }),
    }))
    const s = await fetchWorkspaceMachineSummary({
      apiFetch,
      tenantId: '850',
      workspaceId: '901',
    })
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/cloud/compute/workspace-machine-summary/tenant_id/850/workspace_id/901',
      expect.objectContaining({ credentials: 'include' }),
    )
    expect(s.startedCount).toBe(1)
  })

  it('skips fetch when workspace missing', async () => {
    const apiFetch = vi.fn()
    await fetchWorkspaceMachineSummary({ apiFetch, tenantId: '1', workspaceId: '' })
    expect(apiFetch).not.toHaveBeenCalled()
  })
})

describe('toggleMachineStatusFilter', () => {
  it('sets, switches, and clears', () => {
    expect(toggleMachineStatusFilter(null, MACHINE_STATUS_FILTER.STARTED)).toBe('started')
    expect(toggleMachineStatusFilter('started', MACHINE_STATUS_FILTER.STARTED)).toBeNull()
    expect(toggleMachineStatusFilter('started', MACHINE_STATUS_FILTER.IDLE)).toBe('idle')
  })
})

describe('machineStatusFilterChipLabel', () => {
  it('labels started and idle', () => {
    expect(machineStatusFilterChipLabel('started')).toBe('机器节点·已启动')
    expect(machineStatusFilterChipLabel('idle')).toBe('机器节点·闲置')
    expect(machineStatusFilterChipLabel(null)).toBe('')
  })
})

describe('todoMatchesMachineStatusFilter / filterTodosByMachineStatus', () => {
  const indicators = {
    t1: { machineRunning: true, containerRunning: false },
    t2: { machineRunning: true, containerRunning: true },
    t3: { machineRunning: false, containerRunning: false },
  }
  const todos = [{ id: 't1' }, { id: 't2' }, { id: 't3' }, { id: 't4' }]

  it('started keeps machine-running todos', () => {
    expect(todoMatchesMachineStatusFilter({ id: 't1' }, indicators, 'started')).toBe(true)
    expect(todoMatchesMachineStatusFilter({ id: 't3' }, indicators, 'started')).toBe(false)
    expect(filterTodosByMachineStatus(todos, indicators, 'started').map((t) => t.id)).toEqual([
      't1',
      't2',
    ])
  })

  it('idle keeps machine-running without container', () => {
    expect(todoMatchesMachineStatusFilter({ id: 't1' }, indicators, 'idle')).toBe(true)
    expect(todoMatchesMachineStatusFilter({ id: 't2' }, indicators, 'idle')).toBe(false)
    expect(filterTodosByMachineStatus(todos, indicators, 'idle').map((t) => t.id)).toEqual(['t1'])
  })

  it('null filter keeps all', () => {
    expect(filterTodosByMachineStatus(todos, indicators, null)).toEqual(todos)
  })
})
