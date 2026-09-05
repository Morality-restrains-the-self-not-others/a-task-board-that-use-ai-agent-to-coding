import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { readUseTaskDetailBundle } from './useTaskDetailBundle.js'

describe('useTaskDetail zlog released wiring', () => {
  it('passes serverRuntimeNotServing into createTaskDetailZTreeExecLogState', () => {
    const src = readUseTaskDetailBundle()
    const i = src.indexOf('const zlog = createTaskDetailZTreeExecLogState')
    assert.ok(i >= 0, 'missing zlog factory call')
    const chunk = src.slice(i, i + 1200)
    assert.match(chunk, /serverRuntimeNotServing/)
  })
})
