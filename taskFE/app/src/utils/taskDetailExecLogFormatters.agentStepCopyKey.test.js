import { describe, expect, it } from 'vitest'
import { agentStepCopyKey } from './taskDetailExecLogFormatters.js'

describe('agentStepCopyKey', () => {
  it('优先使用稳定 id，不随 step_number 变化', () => {
    expect(agentStepCopyKey({ id: 's1', step_number: 1 }, 0)).toBe('id-s1')
    expect(agentStepCopyKey({ id: 's1', step_number: 2 }, 0)).toBe('id-s1')
  })

  it('无 id 时用下标，step_number 从缺失到有值时 key 不变', () => {
    expect(agentStepCopyKey({ step_number: undefined, state: 'thinking' }, 0)).toBe('idx-0')
    expect(agentStepCopyKey({ step_number: 1, state: 'thinking' }, 0)).toBe('idx-0')
    expect(agentStepCopyKey({ step_number: 1, state: 'completed' }, 1)).toBe('idx-1')
  })

  it('支持 step_id', () => {
    expect(agentStepCopyKey({ step_id: 'abc' }, 3)).toBe('sid-abc')
  })
})
