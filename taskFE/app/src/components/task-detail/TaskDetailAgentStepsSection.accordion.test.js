// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailAgentStepsSection.accordion.test.js requires vitest runtime')
} else {
const { describe, it, expect } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: TaskDetailAgentStepsSection } = await import('./TaskDetailAgentStepsSection.vue')

function makeCard(stepNumber, state = 'completed', extraRaw = {}) {
  const stateLabel = state === 'completed' ? '✅' : `✅ ${state}`
  return {
    key: `step-${stepNumber}`,
    title: `步骤 ${stepNumber} · ${stateLabel}`,
    modelBadge: 'DeepSeek',
    usageBadge: 'Token I/O 1/2',
    bodyMode: 'text',
    richHtml: '',
    markdownSource: '',
    plainSubtitle: `body-${stepNumber}`,
    toolResultText: '',
    timestamp: '',
    error: '',
    jsonPretty: '{}',
    rawStep: { step_number: stepNumber, state, ...extraRaw },
    stepIdx: stepNumber - 1,
  }
}

describe('TaskDetailAgentStepsSection accordion', () => {
  it('defaults to all steps collapsed', async () => {
    const wrapper = mount(TaskDetailAgentStepsSection, {
      props: {
        cards: [makeCard(1), makeCard(2), makeCard(3)],
        jobStatus: 'completed',
      },
    })
    await wrapper.vm.$nextTick()
    const cards = wrapper.findAll('[data-testid="layer-agent-step-card"]')
    expect(cards).toHaveLength(3)
    expect(cards[0].element.open).toBe(false)
    expect(cards[1].element.open).toBe(false)
    expect(cards[2].element.open).toBe(false)
  })

  it('keeps exclusive accordion: opening one closes another', async () => {
    const wrapper = mount(TaskDetailAgentStepsSection, {
      props: {
        cards: [makeCard(1), makeCard(2)],
        jobStatus: 'completed',
      },
    })
    await wrapper.vm.$nextTick()
    const cards = wrapper.findAll('[data-testid="layer-agent-step-card"]')
    expect(cards[0].element.open).toBe(false)
    expect(cards[1].element.open).toBe(false)

    cards[0].element.open = true
    await cards[0].trigger('toggle')
    await wrapper.vm.$nextTick()
    expect(cards[0].element.open).toBe(true)
    expect(cards[1].element.open).toBe(false)

    cards[1].element.open = true
    await cards[1].trigger('toggle')
    await wrapper.vm.$nextTick()
    expect(cards[0].element.open).toBe(false)
    expect(cards[1].element.open).toBe(true)
  })

  it('does not auto-open running steps', async () => {
    const wrapper = mount(TaskDetailAgentStepsSection, {
      props: {
        cards: [makeCard(1, 'completed'), makeCard(2, 'running')],
        jobStatus: 'running',
      },
    })
    await wrapper.vm.$nextTick()
    const cards = wrapper.findAll('[data-testid="layer-agent-step-card"]')
    expect(cards[0].element.open).toBe(false)
    expect(cards[1].element.open).toBe(false)
  })

  it('does not show tool call preview on collapsed steps', async () => {
    const wrapper = mount(TaskDetailAgentStepsSection, {
      props: {
        cards: [
          makeCard(1, 'completed', {
            tool_calls: [{ name: 'bash', arguments: { command: 'echo hi' } }],
          }),
          makeCard(2, 'completed'),
        ],
        jobStatus: 'completed',
      },
    })
    await wrapper.vm.$nextTick()
    const cards = wrapper.findAll('[data-testid="layer-agent-step-card"]')
    expect(cards[0].element.open).toBe(false)
    const preview = cards[0].find('[data-testid="layer-agent-step-tool-calls-preview"]')
    expect(preview.exists()).toBe(false)
    expect(wrapper.text()).not.toContain('工具调用')
  })

  it('isolates step chevron group-open from ended-collapsed parent details', async () => {
    // 外层「代理步骤」details 结束后默认折叠；若子卡片用匿名 group-open，箭头会永远旋转像展开态
    const wrapper = mount(TaskDetailAgentStepsSection, {
      props: {
        cards: [makeCard(1), makeCard(2)],
        jobStatus: 'completed',
      },
    })
    await wrapper.vm.$nextTick()
    const section = wrapper.get('[data-testid="layer-agent-steps-cards"]')
    expect(section.element.open).toBe(false)
    expect(section.classes()).toContain('group/agent-steps')
    expect(section.classes()).not.toContain('group')

    const cards = wrapper.findAll('[data-testid="layer-agent-step-card"]')
    expect(cards[0].element.open).toBe(false)
    expect(cards[0].classes()).toContain('group/agent-step')
    expect(cards[0].classes()).not.toContain('group')

    const chevron = cards[0].get('summary span[aria-hidden="true"]')
    expect(chevron.text()).toBe('▶')
    expect(chevron.classes()).toContain('group-open/agent-step:rotate-90')
    expect(chevron.classes()).not.toContain('group-open:rotate-90')
  })

  it('keeps parent details open while steps are running', async () => {
    const wrapper = mount(TaskDetailAgentStepsSection, {
      props: {
        cards: [makeCard(1, 'running'), makeCard(2, 'completed')],
        jobStatus: 'running',
      },
    })
    await wrapper.vm.$nextTick()
    const section = wrapper.get('[data-testid="layer-agent-steps-cards"]')
    expect(section.element.open).toBe(true)
  })

  it('collapses parent details once running ends', async () => {
    const wrapper = mount(TaskDetailAgentStepsSection, {
      props: {
        cards: [makeCard(1, 'running')],
        jobStatus: 'running',
      },
    })
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[data-testid="layer-agent-steps-cards"]').element.open).toBe(true)
    await wrapper.setProps({ jobStatus: 'completed', cards: [makeCard(1, 'completed')] })
    expect(wrapper.get('[data-testid="layer-agent-steps-cards"]').element.open).toBe(false)
  })
})

}
