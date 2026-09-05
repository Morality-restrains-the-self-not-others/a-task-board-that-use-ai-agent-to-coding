import { describe, expect, it } from 'vitest'
import { createBindingStatusLogs } from './createBindingStatusLogs.js'

describe('createBindingStatusLogs timezone/trace_id dedupe', () => {
  it('replaces a live line with the backend line that carries trace_id instead of duplicating', () => {
    const api = createBindingStatusLogs()
    api.appendBindingLogLine(
      'C1',
      '[i-abc、task_t1_c1] 容器运行时服务已就绪',
    )
    expect(api.perBindingStatusLogs.value.C1).toHaveLength(1)

    api.mergeBackendBindingLogs('C1', [
      {
        stage: 'server_scheduling',
        message: '[i-abc、task_t1_c1] 容器运行时服务已就绪 trace_id=bb1157e1b936e3d8eb7e7b05',
        created_at: '2026-08-13T15:30:59Z',
      },
    ])
    const lines = api.perBindingStatusLogs.value.C1
    expect(lines).toHaveLength(1)
    expect(lines[0]).toContain('容器运行时服务已就绪')
    expect(lines[0]).toContain('trace_id=bb1157e1b936e3d8eb7e7b05')
    expect(lines[0]).toMatch(/^\[\d{2}:\d{2}:\d{2}\]/)
  })

  it('does not append a second copy on repeated hydrate', () => {
    const api = createBindingStatusLogs()
    const row = {
      stage: 'pending',
      message: '容器调度排队中',
      created_at: '2026-08-13T15:27:27Z',
    }
    api.mergeBackendBindingLogs('C1', [row])
    api.mergeBackendBindingLogs('C1', [row])
    expect(api.perBindingStatusLogs.value.C1).toHaveLength(1)
  })

  it('merges COS-hydrated list API logs into 启动日志 lines', () => {
    const api = createBindingStatusLogs()
    api.mergeBackendBindingLogs('C1', [
      {
        stage: 'starting',
        message: '正在启动容器实例',
        created_at: '2026-08-27T02:00:00Z',
      },
    ])
    const lines = api.perBindingStatusLogs.value.C1
    expect(lines).toHaveLength(1)
    expect(lines[0]).toContain('正在启动容器实例')
  })
})
