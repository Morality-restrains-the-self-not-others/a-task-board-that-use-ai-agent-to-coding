import { describe, expect, it } from 'vitest'
import {
  extractStartTraceIdFromBindingLogs,
  independentStartTraceId,
  resolveBindingPersistedStartTraceId,
} from './bindingStartTraceIdFromLogs.js'

describe('extractStartTraceIdFromBindingLogs', () => {
  it('returns empty for empty/non-array', () => {
    expect(extractStartTraceIdFromBindingLogs(null)).toBe('')
    expect(extractStartTraceIdFromBindingLogs([])).toBe('')
  })

  it('extracts trailing trace_id from latest log object', () => {
    const tid = extractStartTraceIdFromBindingLogs([
      { message: '容器调度排队中' },
      { message: '未找到匹配地域的运行环境 trace_id=task_15652393783064603866' },
    ])
    expect(tid).toBe('task_15652393783064603866')
  })

  it('prefers the most recent trace_id when multiple present', () => {
    const tid = extractStartTraceIdFromBindingLogs([
      { message: 'old trace_id=trace-old' },
      { message: 'new error trace_id=trace-new' },
    ])
    expect(tid).toBe('trace-new')
  })

  it('supports plain string log lines', () => {
    expect(
      extractStartTraceIdFromBindingLogs(['[13:03:51] fail trace_id=web-start-abc']),
    ).toBe('web-start-abc')
  })
})

describe('independentStartTraceId', () => {
  it('drops values equal to the current taskId', () => {
    expect(independentStartTraceId('task_1574', 'task_1574')).toBe('')
    expect(independentStartTraceId('task_1574', 'cmt-start-aaa')).toBe('cmt-start-aaa')
  })
})

describe('resolveBindingPersistedStartTraceId', () => {
  it('prefers start_trace_id column over log suffix', () => {
    expect(
      resolveBindingPersistedStartTraceId(
        {
          start_trace_id: 'cmt-start-aaa',
          logs: [{ message: 'fail trace_id=task_cold_open_1' }],
        },
        'task1',
      ),
    ).toBe('cmt-start-aaa')
  })

  it('ignores column equal to taskId and falls back to independent log suffix', () => {
    expect(
      resolveBindingPersistedStartTraceId(
        {
          start_trace_id: 'task1',
          logs: [{ message: 'fail trace_id=web-start-abc' }],
        },
        'task1',
      ),
    ).toBe('web-start-abc')
  })
})

