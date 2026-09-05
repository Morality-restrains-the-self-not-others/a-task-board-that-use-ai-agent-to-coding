import { describe, expect, it } from 'vitest'
import { pushServerStatusLogLines, resolveStartupLogLabel } from './serverStatusLogLines.js'

describe('serverStatusLogLines', () => {
  it('resolveStartupLogLabel reads log_label', () => {
    expect(resolveStartupLogLabel({ log_label: ' [i-1、task_t1_c1] ' })).toBe('[i-1、task_t1_c1]')
    expect(resolveStartupLogLabel({})).toBe('')
  })

  it('pushServerStatusLogLines prefixes sdk_call with log_label including instance', () => {
    const logs = []
    const fixed = new Date('2026-07-23T12:00:00')
    const label = '[i-abc、task_t1_cmt-1]'
    pushServerStatusLogLines(
      logs,
      {
        status: 'sdk_call',
        message: `${label} 云厂商SDK调用信息`,
        log_label: label,
        instance_id: 'i-abc',
        container_name: 'task_t1_cmt-1',
        sdk_call: { method: 'start_vm', request_ids: ['r1'] },
      },
      `${label} 云厂商SDK调用信息`,
      fixed,
    )
    expect(logs[0]).toContain(`${label} 云厂商SDK调用信息`)
    expect(logs[1]).toContain(`${label} SDK调用: start_vm`)
    expect(logs[2]).toContain(`${label} Request IDs: r1`)
  })

  it('uses 24-hour HH:mm:ss even when Date is UTC afternoon', () => {
    const logs = []
    pushServerStatusLogLines(
      logs,
      { status: 'processing', message: '正在准备启动aliyun服务器...' },
      '正在准备启动aliyun服务器...',
      new Date('2026-08-13T15:30:59Z'),
    )
    expect(logs).toHaveLength(1)
    expect(logs[0]).toMatch(/^\[\d{2}:\d{2}:\d{2}\] 正在准备启动aliyun服务器\.\.\.$/)
  })

  it('skips empty message lines', () => {
    const logs = []
    pushServerStatusLogLines(logs, { status: 'processing', message: '   ' }, '  ', new Date())
    expect(logs).toEqual([])
  })
})
