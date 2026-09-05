// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailZTreeExecLogLiveOutput.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const {
  deriveLiveOutputDisplay,
  deriveJobOutputDisplay,
  deriveAgentStepCards,
  deriveJobCommandHead,
  mergeLayerChangesIntoPayload,
} = await import('./taskDetailZTreeExecLogLiveOutput.js')

describe('derive live output from slot fields', () => {
  it('shows only the selected job chunk', () => {
    const map = { 'job-a': 'output from A', 'job-b': 'output from B' }
    expect(deriveLiveOutputDisplay(map, 'job-a')).toBe('output from A')
    expect(deriveLiveOutputDisplay(map, 'job-b')).toBe('output from B')
    expect(deriveLiveOutputDisplay(map, 'job-missing')).toBe('')
    expect(deriveLiveOutputDisplay(map, '')).toBe('')
  })

  it('agent step cards follow payload steps not a shared list', () => {
    const cardsA = deriveAgentStepCards({
      steps: { steps: [{ type: 'think', content: 'A step' }] },
    })
    const cardsB = deriveAgentStepCards({
      steps: { steps: [{ type: 'think', content: 'B step' }] },
    })
    expect(cardsA.some((c) => String(c.jsonPretty || c.subtitle || '').includes('A step') || c.rawStep?.content === 'A step')).toBe(true)
    expect(cardsA.some((c) => c.rawStep?.content === 'B step')).toBe(false)
    expect(cardsB.some((c) => c.rawStep?.content === 'B step')).toBe(true)
    expect(cardsB.some((c) => c.rawStep?.content === 'A step')).toBe(false)
  })

  it('command head comes from payload job.command', () => {
    expect(deriveJobCommandHead({ job: { command: 'run A' } })).toBe('run A')
    expect(deriveJobCommandHead({ job: { command: 'run B' } })).toBe('run B')
    expect(deriveJobCommandHead(null)).toBe('')
  })

  it('job output hides when it duplicates live display', () => {
    const payload = { job: { output: 'same-text' } }
    expect(deriveJobOutputDisplay(payload, 'same-text')).toBe('')
    expect(deriveJobOutputDisplay(payload, '')).toBe('same-text')
  })

  it('mergeLayerChangesIntoPayload only when selected layer matches', () => {
    const payload = { job: { id: 'job-a' } }
    const changes = { layer_id: 'layer-a', changes: [{ path: 'a.txt' }] }
    expect(mergeLayerChangesIntoPayload(payload, changes, 'layer-b')).toBe(payload)
    expect(mergeLayerChangesIntoPayload(payload, changes, 'layer-a').layer_changes).toEqual(changes)
  })
})
}
