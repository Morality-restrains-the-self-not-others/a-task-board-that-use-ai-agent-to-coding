import { describe, expect, it } from 'vitest'
import {
  assemblePerBindingServerStatusProps,
  isSoleActiveBinding,
} from './assemblePerBindingServerStatusProps.js'

describe('isSoleActiveBinding', () => {
  it('is true only when exactly one live binding matches the comment', () => {
    const bindings = [
      { comment_id: 'C1', status: 'starting' },
      { comment_id: 'C2', status: 'released' },
    ]
    expect(isSoleActiveBinding(bindings, 'C1')).toBe(true)
    expect(isSoleActiveBinding(bindings, 'C2')).toBe(false)
    expect(isSoleActiveBinding([
      { comment_id: 'C1', status: 'starting' },
      { comment_id: 'C2', status: 'running' },
    ], 'C1')).toBe(false)
  })
})

describe('assemblePerBindingServerStatusProps', () => {
  it('keeps released bindings visible with persisted startup logs', () => {
    const props = assemblePerBindingServerStatusProps({
      commentId: 'C1',
      running: false,
      starting: false,
      lifecycle: '已停止',
      heartbeat: {},
      bindingLogs: [
        '[06:00:14] 容器调度排队中',
        '[06:05:02] 正在调用aliyunAPI停止服务器...（触发：容器指令空闲超时回收）',
      ],
      bindings: [{ comment_id: 'C1', status: 'released' }],
      taskStatusLogs: [],
      containerName: 'task_task1_C1',
      startTraceId: 'trc_1',
    })
    expect(props.serverStatus).toBe('stopped')
    expect(props.statusLogs.some((l) => l.includes('触发：容器指令空闲超时回收'))).toBe(true)
    expect(props.startTraceId).toBe('trc_1')
  })

  it('passes last_runtime_status through as runtimeStatus and explains awaiting container', () => {
    const props = assemblePerBindingServerStatusProps({
      commentId: 'C1',
      running: false,
      starting: true,
      lifecycle: '启动中',
      heartbeat: { status: 'idle' },
      bindingLogs: ['[00:26:48] 正在调用aliyunAPI启动服务器...'],
      bindings: [{ comment_id: 'C1', status: 'starting' }],
      taskStatusLogs: [],
      containerName: 'task_task1_C1',
      startTraceId: 'dbd651948b1849c6d17177d5',
      runtimeStatus: 'Running',
      hasServerUrl: false,
    })
    expect(props.runtimeStatus).toBe('Running')
    expect(props.isServerStarting).toBe(true)
    expect(props.statusMessage).toContain('等待容器登记')
  })

  it('failed binding surfaces CSC error_reason as statusMessage for the banner', () => {
    const props = assemblePerBindingServerStatusProps({
      commentId: 'C1',
      running: false,
      starting: false,
      lifecycle: '启动失败',
      heartbeat: {},
      bindingLogs: [],
      bindings: [{ comment_id: 'C1', status: 'failed' }],
      taskStatusLogs: [],
      containerName: 'task_task1_C1',
      runtimeStatus: 'Running',
      hasServerUrl: false,
      errorReason: '云主机已运行，但容器服务未在时限内登记可达地址。',
    })
    expect(props.serverStatus).toBe('error')
    expect(props.statusMessage).toContain('可达地址')
  })
})
