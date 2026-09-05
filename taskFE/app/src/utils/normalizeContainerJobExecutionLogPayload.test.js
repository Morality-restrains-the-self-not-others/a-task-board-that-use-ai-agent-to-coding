import { describe, expect, it } from 'vitest'
import { normalizeContainerJobExecutionLogPayload } from './normalizeContainerJobExecutionLogPayload.js'

describe('normalizeContainerJobExecutionLogPayload', () => {
  it('keeps already-wrapped Django/gateway contract', () => {
    const body = {
      job: { id: 'J1', output: 'hello', status: 'succeeded' },
      steps: { steps: [{ i: 1 }], note: null },
      layer_changes: { layer_id: 'L1', change_count: 0, changes: [] },
    }
    const out = normalizeContainerJobExecutionLogPayload(body)
    expect(out.job.output).toBe('hello')
    expect(out.steps.steps).toHaveLength(1)
    expect(out.layer_changes.layer_id).toBe('L1')
  })

  it('wraps flat gateway bug body so UI can read payload.job', () => {
    const body = {
      id: 'J1',
      status: 'succeeded',
      output: 'HELLO_WORLD_EXEC_LOG_MARKER_42',
      steps: [{ i: 1, kind: 'assistant' }],
    }
    const out = normalizeContainerJobExecutionLogPayload(body)
    expect(out.job).toEqual({
      id: 'J1',
      status: 'succeeded',
      output: 'HELLO_WORLD_EXEC_LOG_MARKER_42',
    })
    expect(out.steps).toEqual({ steps: [{ i: 1, kind: 'assistant' }] })
    expect(out.layer_changes).toBeNull()
  })

  it('wraps steps array when job is already nested', () => {
    const out = normalizeContainerJobExecutionLogPayload({
      job: { id: 'J2', output: 'x' },
      steps: [{ i: 9 }],
    })
    expect(out.job.id).toBe('J2')
    expect(out.steps.steps).toEqual([{ i: 9 }])
  })
})
