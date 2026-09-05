// @vitest-environment node
import { describe, it, expect } from 'vitest'
import {
  agentStepCardSubtitle,
  agentStepPlainBodyForPre,
} from './taskDetailExecLogFormatters.js'

describe('agentStepCardSubtitle / agentStepPlainBodyForPre', () => {
  it('does not surface tool_calls as plain body (bash command line)', () => {
    const step = {
      step_number: 6,
      state: 'completed',
      tool_calls: [
        {
          name: 'bash',
          arguments: {
            command:
              'cd /app/onlineProject_state/layers/20260719_073014_f2af66/ram-work && grep -rn "project" --include="*.py" task2app/',
          },
        },
      ],
    }
    expect(agentStepCardSubtitle(step)).toBe('')
    expect(agentStepPlainBodyForPre(step)).toBe('')
  })

  it('hides indigo plain body when title already shows tool commands', () => {
    const step = {
      step_number: 3,
      state: 'completed',
      llm_response: {
        content: 'bash cd /app/onlineProject_state/layers/x/ram-work && find . -type d',
        tool_calls: [
          {
            name: 'bash',
            arguments: {
              command: 'cd /app/onlineProject_state/layers/x/ram-work && find . -type d',
            },
          },
        ],
      },
    }
    expect(agentStepCardSubtitle(step)).toContain('bash cd')
    expect(agentStepPlainBodyForPre(step)).toBe('')
  })

  it('still returns llm_response content for text body mode', () => {
    const step = {
      step_number: 1,
      state: 'thinking',
      llm_response: { content: 'thinking about the next edit' },
    }
    expect(agentStepCardSubtitle(step)).toContain('thinking about the next edit')
    expect(agentStepPlainBodyForPre(step)).toContain('thinking about the next edit')
  })
})
