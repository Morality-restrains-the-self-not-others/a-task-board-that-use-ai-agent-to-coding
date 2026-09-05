import { describe, expect, it } from 'vitest'
import {
  filterTodosByMachineRuntime,
  machineRuntimeFilterChipLabel,
  machineRuntimeFilterMismatch,
  normalizeMachineRuntimeFilter,
  toggleMachineRuntimeFilter,
} from './workPanelMachineRuntimeFilter.js'

const todos = [{ id: 'a' }, { id: 'b' }, { id: 'c' }, { id: 'd' }]
const indicators = {
  a: { machineRunning: true, containerRunning: true },
  b: { machineRunning: true, containerRunning: false },
  c: { machineRunning: false, containerRunning: false },
}

describe('workPanelMachineRuntimeFilter', () => {
  it('T1: null filter returns original list', () => {
    expect(filterTodosByMachineRuntime(todos, indicators, null)).toEqual(todos)
    expect(filterTodosByMachineRuntime(todos, indicators, undefined)).toEqual(todos)
  })

  it('T2: started keeps machineRunning tasks', () => {
    expect(filterTodosByMachineRuntime(todos, indicators, 'started').map((t) => t.id)).toEqual([
      'a',
      'b',
    ])
  })

  it('T3: idle keeps machineRunning && !containerRunning', () => {
    expect(filterTodosByMachineRuntime(todos, indicators, 'idle').map((t) => t.id)).toEqual(['b'])
  })

  it('T4: invalid filter treated as none; toggle clears/switches', () => {
    expect(normalizeMachineRuntimeFilter('busy')).toBeNull()
    expect(filterTodosByMachineRuntime(todos, indicators, 'busy')).toEqual(todos)
    expect(toggleMachineRuntimeFilter(null, 'started')).toBe('started')
    expect(toggleMachineRuntimeFilter('started', 'started')).toBeNull()
    expect(toggleMachineRuntimeFilter('started', 'idle')).toBe('idle')
    expect(machineRuntimeFilterChipLabel('idle')).toBe('过滤：闲置')
    expect(machineRuntimeFilterChipLabel(null)).toBe('')
  })

  it('T5: indicators not ready keeps full list under started filter', () => {
    expect(
      filterTodosByMachineRuntime(todos, {}, 'started', { indicatorsReady: false }).map((t) => t.id),
    ).toEqual(['a', 'b', 'c', 'd'])
  })

  it('T6: mismatch when summary started>0 but filtered empty after ready', () => {
    expect(
      machineRuntimeFilterMismatch({
        filter: 'started',
        filteredCount: 0,
        todosCount: 5,
        indicatorsReady: true,
        startedCount: 1,
        idleCount: 0,
      }),
    ).toBe(true)
    expect(
      machineRuntimeFilterMismatch({
        filter: 'started',
        filteredCount: 0,
        todosCount: 5,
        indicatorsReady: false,
        startedCount: 1,
        idleCount: 0,
      }),
    ).toBe(false)
    expect(
      machineRuntimeFilterMismatch({
        filter: 'started',
        filteredCount: 1,
        todosCount: 5,
        indicatorsReady: true,
        startedCount: 1,
        idleCount: 0,
      }),
    ).toBe(false)
  })
})
