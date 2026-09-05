// @vitest-environment node
// 回归测试：relayToTraeApiUrl 必须输出 funcName-first 云网关路径形态
// 缺陷（OPT-20260809-030）：修复前输出 `.../task_id/${taskId}relay-to-trae/${path}`
// —— funcName 紧贴 task_id 值、缺少 `/` 分隔，ParseConventionPath 会把 funcName
// 吞进 task_id 值导致 sub 为空 → HTTP 404（页面「读取文件树失败」同源路径缺陷）。
import { describe, expect, it } from 'vitest'
import { relayToTraeApiUrl, resolveRelayContextIds, resolveServerConfigTaskId } from './serverConfigRouteHelpers.js'

describe('relayToTraeApiUrl — funcName-first 路径约定（回归）', () => {
  const route = { params: { tenant: 't1', workspace: 'w1' } }
  const task = { id: 'tk1', tenant_id: 't1', workspace_id: 'w1' }

  it('输出 funcName-first 形态：compute/<funcName>/tenant_id/…', () => {
    const url = relayToTraeApiUrl({ route, task, suffix: 'token-init' })
    expect(url).toBe(
      '/api/cloud/compute/relay-to-trae/tenant_id/t1/workspace_id/w1/task_id/tk1/token-init',
    )
  })

  it('funcName 不被吞进 task_id 值（修复前缺陷形态 task_id/<id>relay-to-trae）', () => {
    const url = relayToTraeApiUrl({ route, task, suffix: 'token-init' })
    expect(url).not.toMatch(/task_id\/[^/]+relay-to-trae/)
    expect(url).toMatch(/\/relay-to-trae\/tenant_id\//)
  })

  it('task_id 值后必须紧跟路径分隔符 /', () => {
    const url = relayToTraeApiUrl({ route, task, suffix: 'token-init' })
    expect(url).toMatch(/task_id\/tk1\//)
  })

  it('suffix 前导斜杠被归一化', () => {
    expect(relayToTraeApiUrl({ route, task, suffix: '///token-init' })).toContain(
      '/task_id/tk1/token-init',
    )
  })

  it('多个 funcName 变体均保持 funcName-first', () => {
    const a = relayToTraeApiUrl({ route, task, suffix: 'token-persist' })
    const b = relayToTraeApiUrl({ route, task, suffix: 'status' })
    expect(a).toMatch(/\/relay-to-trae\/tenant_id\/t1\//)
    expect(b).toMatch(/\/relay-to-trae\/tenant_id\/t1\//)
    expect(a).not.toContain('/task_id/tk1token-persist')
    expect(b).not.toContain('/task_id/tk1status')
  })
})

describe('resolveServerConfigTaskId / resolveRelayContextIds — 路由 taskId 回退', () => {
  it('task 无 id 时回退 route.params.taskId', () => {
    const routeWithTask = {
      params: { tenant: 't1', workspaceId: 'w1', taskId: 'task_from_route' },
    }
    expect(resolveServerConfigTaskId({ route: routeWithTask, task: {} })).toBe('task_from_route')
    expect(
      resolveRelayContextIds({ route: routeWithTask, task: {}, workspaceId: 'w1' }).task_id,
    ).toBe('task_from_route')
  })

  it('props.taskId 优先于 task.id 与 route', () => {
    const ids = resolveServerConfigTaskId({
      route: { params: { taskId: 'route-task' } },
      task: { id: 'task-obj' },
      taskId: 'prop-task',
    })
    expect(ids).toBe('prop-task')
  })

  it('task.pk 在无 id 时可用', () => {
    expect(resolveServerConfigTaskId({ task: { pk: 'pk-1' }, route: { params: {} } })).toBe('pk-1')
  })
})
