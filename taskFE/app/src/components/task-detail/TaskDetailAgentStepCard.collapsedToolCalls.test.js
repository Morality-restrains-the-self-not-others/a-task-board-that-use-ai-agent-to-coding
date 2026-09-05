// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailAgentStepCard.collapsedToolCalls.test.js requires vitest runtime')
} else {
const { describe, it, expect } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailAgentStepCard } = await import('./TaskDetailAgentStepCard.vue')

function makeCardWithToolCall() {
  return {
    key: 'step-1',
    title: '步骤 1 · ✅',
    modelBadge: 'DeepSeek',
    usageBadge: 'Token I/O 4329/224',
    bodyMode: 'text',
    richHtml: '',
    markdownSource: '',
    plainSubtitle: 'thinking body',
    toolResultText: '',
    timestamp: '',
    error: '',
    jsonPretty: '{}',
    rawStep: {
      step_number: 1,
      state: 'completed',
      tool_calls: [
        {
          name: 'bash',
          arguments: {
            command:
              'cd /app/onlineProject_state/layers/20260718_192245_290c27/ram-work && find . -type f -name "*.tsx"',
          },
        },
      ],
    },
    stepIdx: 0,
  }
}

describe('TaskDetailAgentStepCard collapsed tool calls', () => {
  it('does not show tool call preview in summary when step is collapsed', () => {
    const wrapper = mount(TaskDetailAgentStepCard, {
      props: {
        card: makeCardWithToolCall(),
        open: false,
      },
    })
    const details = wrapper.get('[data-testid="layer-agent-step-card"]')
    expect(details.element.open).toBe(false)
    expect(wrapper.find('[data-testid="layer-agent-step-tool-calls-preview"]').exists()).toBe(
      false,
    )
    expect(wrapper.text()).not.toContain('工具调用')
  })

  it('does not show tool call preview when step is expanded', async () => {
    const wrapper = mount(TaskDetailAgentStepCard, {
      props: {
        card: makeCardWithToolCall(),
        open: true,
      },
    })
    expect(wrapper.get('[data-testid="layer-agent-step-card"]').element.open).toBe(true)
    expect(wrapper.find('[data-testid="layer-agent-step-tool-calls-preview"]').exists()).toBe(
      false,
    )
  })
})

}
