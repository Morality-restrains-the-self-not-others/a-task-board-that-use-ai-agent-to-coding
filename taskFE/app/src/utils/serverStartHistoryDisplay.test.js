import { describe, expect, it } from 'vitest'
import {
  formatServerStartHistoryDuration,
  formatServerStartHistoryStartReason,
  formatServerStartHistoryStopReason,
} from './serverStartHistoryDisplay.js'

describe('formatServerStartHistoryDuration', () => {
  it('UT-033-01 formats closed session from started_at to stopped_at', () => {
    const started = '2026-07-19T00:00:00.000Z'
    const stopped = '2026-07-19T01:30:00.000Z'
    const text = formatServerStartHistoryDuration({
      started_at: started,
      stopped_at: stopped,
    })
    expect(text).toBe('1 小时 30 分钟')
  })

  it('UT-033-02 formats open session using nowMs', () => {
    const startedMs = Date.parse('2026-07-19T00:00:00.000Z')
    const text = formatServerStartHistoryDuration(
      { started_at: '2026-07-19T00:00:00.000Z' },
      startedMs + 45_000,
    )
    expect(text).toBe('45 秒')
  })

  it('falls back to created_at when started_at missing', () => {
    const text = formatServerStartHistoryDuration(
      {
        created_at: '2026-07-19T00:00:00.000Z',
        stopped_at: '2026-07-19T00:00:10.000Z',
      },
    )
    expect(text).toBe('10 秒')
  })

  it('returns - when no start timestamp', () => {
    expect(formatServerStartHistoryDuration({})).toBe('-')
  })
})

describe('formatServerStartHistoryStartReason', () => {
  it('UT-033-03 maps runtime_source labels', () => {
    expect(formatServerStartHistoryStartReason('cloud_vm')).toBe('云虚拟机启动')
    expect(formatServerStartHistoryStartReason('cloud_vm_manual')).toBe('任务详情手动选配启动')
    expect(formatServerStartHistoryStartReason('cloud_vm_template')).toBe('任务详情项目模版启动')
    expect(formatServerStartHistoryStartReason('cloud_vm_auto_run')).toBe('创建任务/工作台自动运行启动')
    expect(formatServerStartHistoryStartReason('cloud_vm_comment_mention')).toBe('评论区$镜像启动')
    expect(formatServerStartHistoryStartReason('cloud_vm_terminal_migrate')).toBe('终态迁移再供给启动')
    expect(formatServerStartHistoryStartReason('relay_local')).toBe('本地中继启动')
    expect(formatServerStartHistoryStartReason('')).toBe('-')
    expect(formatServerStartHistoryStartReason('unknown_src')).toBe('unknown_src')
  })
})

describe('formatServerStartHistoryStopReason', () => {
  it('UT-033-04 maps stop_reason labels', () => {
    expect(formatServerStartHistoryStopReason('stop_vm')).toBe('用户停止虚拟机')
    expect(formatServerStartHistoryStopReason('idle_recycle')).toBe('空闲回收')
    expect(formatServerStartHistoryStopReason('orphan_reconcile')).toBe('孤儿实例对账清理')
    expect(formatServerStartHistoryStopReason('')).toBe('-')
    expect(formatServerStartHistoryStopReason('relay_stop')).toBe('本地中继停止')
    expect(formatServerStartHistoryStopReason('superseded_by_new_start')).toBe('被新启动会话取代')
    expect(formatServerStartHistoryStopReason('user_stop')).toBe('用户点击停止服务器')
    expect(formatServerStartHistoryStopReason('instruction_idle')).toBe('容器指令空闲超时回收')
  })
})
