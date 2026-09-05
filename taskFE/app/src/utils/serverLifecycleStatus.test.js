// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  resolveServerLifecycleLabel,
  serverLifecycleDotClass,
  serverLifecycleTextClass,
  SERVER_LIFECYCLE_STYLES,
  SERVER_STATUS_CODE_TO_LABEL,
} from './serverLifecycleStatus.js'

describe('resolveServerLifecycleLabel', () => {
  it('running 优先于 starting / status', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: true,
        isServerStarting: true,
        serverStatus: 'error',
      }),
    ).toBe('已启动')
  })

  it('starting 优先于 status', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: false,
        isServerStarting: true,
        serverStatus: 'stopped',
      }),
    ).toBe('启动中')
  })

  it('maps every SERVER_STATUS_CODE_TO_LABEL entry', () => {
    for (const [code, label] of Object.entries(SERVER_STATUS_CODE_TO_LABEL)) {
      expect(resolveServerLifecycleLabel({ serverStatus: code })).toBe(label)
    }
  })

  it('未知或空 status → 未启动', () => {
    expect(resolveServerLifecycleLabel({})).toBe('未启动')
    expect(resolveServerLifecycleLabel({ serverStatus: '' })).toBe('未启动')
    expect(resolveServerLifecycleLabel({ serverStatus: '  ' })).toBe('未启动')
    expect(resolveServerLifecycleLabel({ serverStatus: 'weird' })).toBe('未启动')
  })

  it('trims serverStatus', () => {
    expect(resolveServerLifecycleLabel({ serverStatus: '  error  ' })).toBe('启动失败')
  })

  it('冷打开：runtimeStatus=Running 且标志位未回填 → 已启动', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: false,
        isServerStarting: false,
        serverStatus: '',
        runtimeStatus: 'Running',
      }),
    ).toBe('已启动')
  })

  it('VM Running 但 binding 仍 starting（等容器登记）→ 等待容器，不得显示启动中', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: false,
        isServerStarting: true,
        serverStatus: 'processing',
        runtimeStatus: 'Running',
      }),
    ).toBe('等待容器')
  })

  it('serverStatus=error 优先于 runtime Running（超时收口后 VM 仍 Running）', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: false,
        isServerStarting: false,
        serverStatus: 'error',
        runtimeStatus: 'Running',
      }),
    ).toBe('启动失败')
  })

  it('SSE 已报 error 时即使 flags 仍 starting 也显示启动失败', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: false,
        isServerStarting: true,
        serverStatus: 'error',
        runtimeStatus: 'Running',
      }),
    ).toBe('启动失败')
  })

  it('冷打开：runtimeStatus 过渡态 → 启动中', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: false,
        runtimeStatus: 'Starting',
      }),
    ).toBe('启动中')
  })

  it('冷打开：runtimeStatus 非服务态 → 已停止', () => {
    expect(
      resolveServerLifecycleLabel({
        isServerRunning: false,
        runtimeStatus: 'Stopped',
      }),
    ).toBe('已停止')
  })
})

describe('serverLifecycle visual classes', () => {
  it('every SERVER_LIFECYCLE_STYLES label has matching helpers', () => {
    for (const [label, style] of Object.entries(SERVER_LIFECYCLE_STYLES)) {
      expect(serverLifecycleDotClass(label)).toBe(style.dot)
      expect(serverLifecycleTextClass(label)).toBe(style.text)
    }
  })

  it('unknown label falls back to 未启动 styles', () => {
    const fallback = SERVER_LIFECYCLE_STYLES['未启动']
    expect(serverLifecycleDotClass('不存在的态')).toBe(fallback.dot)
    expect(serverLifecycleTextClass('不存在的态')).toBe(fallback.text)
  })
})

describe('extension contract: add status without touching the panel', () => {
  it('码表与样式表是唯一扩展点（导出表可迭代）', () => {
    expect(Object.keys(SERVER_STATUS_CODE_TO_LABEL).length).toBeGreaterThan(0)
    expect(Object.keys(SERVER_LIFECYCLE_STYLES)).toEqual([
      '已启动',
      '启动中',
      '等待容器',
      '启动失败',
      '已停止',
      '未启动',
    ])
  })

  it('码表中的每个 label 必须在样式表有定义（否则面板圆点会 silent fallback）', () => {
    for (const label of Object.values(SERVER_STATUS_CODE_TO_LABEL)) {
      expect(SERVER_LIFECYCLE_STYLES[label], `missing style for ${label}`).toBeTruthy()
    }
    expect(SERVER_LIFECYCLE_STYLES['已启动']).toBeTruthy()
  })

  it('模拟新增 serverStatus 码：只靠码表即可解析（面板无状态枚举）', () => {
    // 契约说明：真实新增时在 SERVER_STATUS_CODE_TO_LABEL 加一行；
    // 此处用「已知码 + 未知码」证明解析完全由表驱动，不依赖组件。
    expect(resolveServerLifecycleLabel({ serverStatus: 'initializing' })).toBe('启动中')
    expect(resolveServerLifecycleLabel({ serverStatus: 'future_status_xyz' })).toBe('未启动')
  })
})
