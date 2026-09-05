// @vitest-environment node
import { describe, it, expect } from 'vitest'
import { agentStepToolCallsPreview } from './taskDetailExecLogFormatters.js'

describe('agentStepToolCallsPreview', () => {
  it('returns empty for missing tool_calls', () => {
    expect(agentStepToolCallsPreview(null)).toEqual([])
    expect(agentStepToolCallsPreview({})).toEqual([])
  })

  it('maps bash command and path hints', () => {
    const rows = agentStepToolCallsPreview({
      tool_calls: [
        { name: 'bash', arguments: { command: 'cd /app && ls' } },
        { name: 'read', arguments: { path: '/tmp/a.ts' } },
        { name: 'write', arguments: { file_path: '/tmp/b.ts' } },
      ],
    })
    expect(rows).toEqual([
      { name: 'bash', hint: 'cd /app && ls' },
      { name: 'read', hint: '/tmp/a.ts' },
      { name: 'write', hint: '/tmp/b.ts' },
    ])
  })

  it('truncates long hints', () => {
    const long = 'x'.repeat(300)
    const rows = agentStepToolCallsPreview(
      { tool_calls: [{ name: 'bash', arguments: { command: long } }] },
      { hintMax: 20 },
    )
    expect(rows[0].name).toBe('bash')
    expect(rows[0].hint.length).toBeLessThanOrEqual(20)
    expect(rows[0].hint.endsWith('…')).toBe(true)
  })
})
