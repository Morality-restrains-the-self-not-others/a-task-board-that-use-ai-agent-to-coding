import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { readUseTaskDetailBundle } from './useTaskDetailBundle.js'

describe('useTaskDetail bootstrap failure trace wiring', () => {
  it('exports containerBootstrapFailureTraceId next to the failure message', () => {
    const src = readUseTaskDetailBundle()
    assert.match(src, /createContainerBootstrapFailureState/)
    assert.match(src, /containerBootstrapFailureTraceId/)
    assert.match(src, /containerBootstrapFailureMessage,\s*containerBootstrapFailureTraceId/)
  })
})
