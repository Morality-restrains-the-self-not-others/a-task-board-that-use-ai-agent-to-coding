// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailAgentStepCard.commandOutsideFold.test.js requires vitest runtime')
} else {
const { describe, it, expect } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailAgentStepCard } = await import('./TaskDetailAgentStepCard.vue')

const BASH_CMD =
  "bash sed -n '1,200p' /app/onlineProject_state/runtime/layer_artifacts/20260902_085739_3bc2e4/.trajectories/trajectory_56"

function makePlainCommandCard(overrides = {}) {
  return {
    key: 'step-46',
    title: '步骤 46 · ✅',
    modelBadge: 'DeepSeek',
    usageBadge: 'Token I/O 1/2',
    bodyMode: 'text',
    richHtml: '',
    markdownSource: '',
    plainSubtitle: BASH_CMD,
    toolResultText: '',
    timestamp: '',
    error: '',
    jsonPretty: '{}',
    rawStep: { step_number: 46, state: 'completed' },
    stepIdx: 45,
    ...overrides,
  }
}

describe('TaskDetailAgentStepCard command outside fold', () => {
  it('shows plain command in summary when the step is collapsed', () => {
    const wrapper = mount(TaskDetailAgentStepCard, {
      props: { card: makePlainCommandCard(), open: false },
    })
    const details = wrapper.get('[data-testid="layer-agent-step-card"]')
    expect(details.element.open).toBe(false)
    const summary = wrapper.get('[data-testid="layer-agent-step-card-summary"]')
    const preview = summary.get('[data-testid="layer-agent-step-command-preview"]')
    expect(preview.text()).toContain(BASH_CMD)
  })

  it('keeps the command in summary when the step is expanded', () => {
    const wrapper = mount(TaskDetailAgentStepCard, {
      props: { card: makePlainCommandCard(), open: true },
    })
    expect(wrapper.get('[data-testid="layer-agent-step-card"]').element.open).toBe(true)
    const preview = wrapper
      .get('[data-testid="layer-agent-step-card-summary"]')
      .get('[data-testid="layer-agent-step-command-preview"]')
    expect(preview.text()).toContain(BASH_CMD)
    expect(wrapper.findAll('[data-testid="layer-agent-step-command-preview"]')).toHaveLength(1)
  })

  it('does not duplicate the command pre inside the collapsed body', () => {
    const wrapper = mount(TaskDetailAgentStepCard, {
      props: { card: makePlainCommandCard(), open: false },
    })
    const previews = wrapper.findAll('[data-testid="layer-agent-step-command-preview"]')
    expect(previews).toHaveLength(1)
    const summary = wrapper.get('[data-testid="layer-agent-step-card-summary"]')
    expect(summary.find('[data-testid="layer-agent-step-command-preview"]').exists()).toBe(true)
    const body = wrapper.find('[data-testid="layer-agent-step-card-body"]')
    if (body.exists()) {
      expect(body.find('[data-testid="layer-agent-step-command-preview"]').exists()).toBe(false)
    }
  })

  it('keeps html body inside the fold when collapsed and has no plain command', () => {
    const wrapper = mount(TaskDetailAgentStepCard, {
      props: {
        card: makePlainCommandCard({
          bodyMode: 'html',
          richHtml: '<p>选项 Mock</p>',
          plainSubtitle: '',
        }),
        open: false,
      },
    })
    expect(wrapper.find('[data-testid="layer-agent-step-command-preview"]').exists()).toBe(false)
    const rich = wrapper.find('[data-testid="agent-step-rich-frame"]')
    expect(rich.exists()).toBe(true)
    expect(rich.isVisible()).toBe(false)
  })

  it('hides tool_results while collapsed', () => {
    const wrapper = mount(TaskDetailAgentStepCard, {
      props: {
        card: makePlainCommandCard({
          toolResultText: 'file content from mock',
        }),
        open: false,
      },
    })
    const summary = wrapper.get('[data-testid="layer-agent-step-card-summary"]')
    expect(summary.text()).not.toContain('tool_results.result')
    expect(summary.text()).not.toContain('file content from mock')
    expect(wrapper.get('[data-testid="layer-agent-step-command-preview"]').text()).toContain(
      BASH_CMD,
    )
  })
})

}
