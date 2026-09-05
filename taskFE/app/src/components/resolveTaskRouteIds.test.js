import { describe, expect, it } from 'vitest'
import { resolveTaskRouteIds, resolveTaskRouteIdsComplete } from './resolveTaskRouteIds.js'

describe('resolveTaskRouteIds', () => {
  it('work-panel 形态：route 仅有 tenant 时用 props 补齐 workspace/taskId', () => {
    const ids = resolveTaskRouteIds({
      tenantId: '850256677331562496',
      workspaceId: '861623708318031872',
      taskId: 'task_1',
      routeParams: { tenant: '850256677331562496' },
    })
    expect(ids).toEqual({
      tenantId: '850256677331562496',
      workspaceId: '861623708318031872',
      taskId: 'task_1',
    })
    expect(resolveTaskRouteIdsComplete({
      tenantId: '850256677331562496',
      workspaceId: '861623708318031872',
      taskId: 'task_1',
      routeParams: { tenant: '850256677331562496' },
    }).complete).toBe(true)
  })

  it('props 优先于 route.params（即使 route 也有完整参数）', () => {
    const ids = resolveTaskRouteIds({
      tenantId: 'prop-t',
      workspaceId: 'prop-w',
      taskId: 'prop-task',
      routeParams: { tenant: 'route-t', workspace: 'route-w', taskId: 'route-task' },
    })
    expect(ids).toEqual({
      tenantId: 'prop-t',
      workspaceId: 'prop-w',
      taskId: 'prop-task',
    })
  })

  it('无 props 时回退 route.params（含 workspace 与 workspaceId）', () => {
    expect(
      resolveTaskRouteIds({
        routeParams: { tenant: 't1', workspace: 'w1', taskId: 'task_a' },
      }),
    ).toEqual({ tenantId: 't1', workspaceId: 'w1', taskId: 'task_a' })

    expect(
      resolveTaskRouteIds({
        routeParams: { tenant: 't1', workspaceId: 'w2', taskId: 'task_b' },
      }),
    ).toEqual({ tenantId: 't1', workspaceId: 'w2', taskId: 'task_b' })
  })

  it('可从 task 对象回退 tenant/workspace/taskId，并展开 workspace 对象', () => {
    const ids = resolveTaskRouteIds({
      routeParams: {},
      task: {
        id: 42,
        tenant_id: 'tt',
        workspace: { id: 'ww' },
      },
    })
    expect(ids).toEqual({ tenantId: 'tt', workspaceId: 'ww', taskId: '42' })
  })

  it('缺任一 ID 时 complete 为 false', () => {
    expect(
      resolveTaskRouteIdsComplete({
        routeParams: { tenant: 't1' },
      }).complete,
    ).toBe(false)
  })
})
