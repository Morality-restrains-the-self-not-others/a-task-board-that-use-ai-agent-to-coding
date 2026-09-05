// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import {
  shouldShowMachineOwnerHint,
  normalizeOwnerTaskIds,
  buildMachineOwnerEntries,
} from './machineOwnerHint.js'

describe('machineOwnerHint', () => {
  it('容器未运行时不展示', () => {
    expect(shouldShowMachineOwnerHint({
      container_running: false,
      instance_id: 'i-1',
      machine_owner_task_ids: ['t1'],
    })).toBe(false)
  })

  it('API 标记容器运行且有所属任务时展示', () => {
    expect(shouldShowMachineOwnerHint({
      container_running: true,
      instance_id: 'i-1',
      machine_owner_task_ids: ['t1'],
    })).toBe(true)
  })

  it('前端已有容器 URL 时也可展示', () => {
    expect(shouldShowMachineOwnerHint({
      container_running: false,
      instance_id: 'i-1',
      machine_owner_task_ids: ['t1'],
    }, { serverUrl: 'http://127.0.0.1:8080/' })).toBe(true)
  })

  it('无 instance_id 不展示', () => {
    expect(shouldShowMachineOwnerHint({
      container_running: true,
      machine_owner_task_ids: ['t1'],
    })).toBe(false)
  })

  it('normalizeOwnerTaskIds 去重去空', () => {
    expect(normalizeOwnerTaskIds({
      machine_owner_task_ids: ['a', '', 'a', 'b'],
    })).toEqual(['a', 'b'])
  })

  it('buildMachineOwnerEntries 本任务无链接、他任务有详情链接', () => {
    const entries = buildMachineOwnerEntries(
      ['task-a', 'task-b'],
      'task-a',
      { tenantId: 'ten1', workspaceId: 'ws1' },
    )
    expect(entries).toEqual([
      { taskId: 'task-a', isViewer: true, href: null },
      {
        taskId: 'task-b',
        isViewer: false,
        href: '/tenant/ten1/workspace/ws1/task-detail/task-b/',
      },
    ])
  })
})
