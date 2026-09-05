import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { readUseTaskDetailBundle } from './useTaskDetailBundle.js'

describe('useTaskDetail bootstrap clone-log wiring', () => {
  it('maps clone-log failure into containerBootstrapFailureMessage', () => {
    const src = readUseTaskDetailBundle()
    assert.match(src, /const onBootstrapCloneLogFailure = \(message\) =>/)
    assert.match(src, /containerBootstrapFailureMessage\.value = msg/)
    assert.match(src, /onBootstrapCloneLogFailure,/)
  })

  it('refetches clone-log on heartbeat empty snapshot and tab visibility', () => {
    const src = readUseTaskDetailBundle()
    assert.match(src, /fetchContainerBootstrapCloneLog,/)
    assert.match(
      src,
      /void refreshZTreeExecutionLog\(\)\n  void fetchContainerBootstrapCloneLog\(\)/,
    )
  })
})
