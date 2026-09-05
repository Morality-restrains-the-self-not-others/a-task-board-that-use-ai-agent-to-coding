// @vitest-environment node
import { describe, it, expect } from 'vitest'
import {
  agentStepCardTitle,
  agentStepTitleCommands,
} from './taskDetailExecLogFormatters.js'

describe('agentStepCardTitle / agentStepTitleCommands', () => {
  it('replaces completed text with llm_response.tool_calls arguments.command', () => {
    const step = {
      step_number: 3,
      state: 'completed',
      llm_response: {
        tool_calls: [
          {
            name: 'bash',
            arguments: { command: 'echo hello' },
          },
        ],
      },
    }
    expect(agentStepTitleCommands(step)).toEqual(['echo hello'])
    expect(agentStepCardTitle(step)).toBe('步骤 3 · ✅\necho hello')
    expect(agentStepCardTitle(step)).not.toContain('completed')
  })

  it('falls back to step-level tool_calls when llm_response has none', () => {
    const step = {
      step_number: 1,
      state: 'completed',
      tool_calls: [
        {
          name: 'bash',
          arguments: { command: 'ls -la' },
        },
      ],
    }
    expect(agentStepCardTitle(step)).toBe('步骤 1 · ✅\nls -la')
  })

  it('parses JSON-string arguments', () => {
    const step = {
      step_number: 2,
      state: 'completed',
      llm_response: {
        tool_calls: [
          {
            name: 'bash',
            arguments: JSON.stringify({ command: 'pwd' }),
          },
        ],
      },
    }
    expect(agentStepCardTitle(step)).toBe('步骤 2 · ✅\npwd')
  })

  it('shows checkmark only when completed has no command and no tool summary', () => {
    const step = { step_number: 4, state: 'completed' }
    expect(agentStepCardTitle(step)).toBe('步骤 4 · ✅')
    expect(agentStepCardTitle(step)).not.toContain('completed')
  })

  it('shows tool name + primary arg when completed has no command', () => {
    const step = {
      step_number: 5,
      state: 'completed',
      tool_calls: [
        { name: 'read_file', arguments: { path: '/tmp/a.txt' } },
      ],
    }
    expect(agentStepCardTitle(step)).toBe('步骤 5 · ✅\nread_file /tmp/a.txt')
  })

  it('keeps non-completed state labels unchanged', () => {
    expect(agentStepCardTitle({ step_number: 1, state: 'thinking' })).toBe(
      '步骤 1 · 🤔 thinking',
    )
    expect(agentStepCardTitle({ step_number: 1, state: 'error' })).toBe('步骤 1 · ❌ error')
  })

  it('OPT-20260902-022: calling_tool with command shows command under the state label', () => {
    const step = {
      step_number: 6,
      state: 'calling_tool',
      tool_calls: [
        { name: 'bash', arguments: { command: 'bash ./scripts/build.sh' } },
      ],
    }
    expect(agentStepCardTitle(step)).toBe('步骤 6 · 🔧 calling_tool\nbash ./scripts/build.sh')
    expect(agentStepCardTitle(step)).not.toMatch(/calling_tool[^\n\S]*\S/)
  })

  it('OPT-20260902-022: thinking with command keeps state label and shows command', () => {
    const step = {
      step_number: 7,
      state: 'thinking',
      llm_response: {
        tool_calls: [
          { name: 'read_file', arguments: { command: 'ls -la /tmp' } },
        ],
      },
    }
    const title = agentStepCardTitle(step)
    expect(title.startsWith('步骤 7 · 🤔 thinking')).toBe(true)
    expect(title).toContain('\nls -la /tmp')
  })

  it('joins multiple commands with newlines and does not truncate', () => {
    const long = `cd /app/${'x'.repeat(300)} && find . -type d`
    const step = {
      step_number: 3,
      state: 'completed',
      llm_response: {
        tool_calls: [
          { name: 'bash', arguments: { command: 'echo one' } },
          { name: 'bash', arguments: { command: long } },
        ],
      },
    }
    const title = agentStepCardTitle(step)
    expect(title).toBe(`步骤 3 · ✅\necho one\n${long}`)
    expect(title).toContain(long)
    expect(title).not.toContain('…')
  })

  it('puts command on its own line after the checkmark', () => {
    const step = {
      step_number: 5,
      state: 'completed',
      llm_response: {
        tool_calls: [
          {
            name: 'bash',
            arguments: { command: 'cd /app && grep -rn truncated' },
          },
        ],
      },
    }
    const title = agentStepCardTitle(step)
    expect(title).toBe('步骤 5 · ✅\ncd /app && grep -rn truncated')
    // 打勾与命令不得落在同一行（允许 ✅ 后仅换行）
    expect(title).not.toMatch(/✅[^\n\S]*\S/)
  })
})
