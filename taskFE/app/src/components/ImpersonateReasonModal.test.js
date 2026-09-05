// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ImpersonateReasonModal.test.js requires vitest runtime')
} else {
const { describe, expect, it, afterEach } = await import('vitest')
const { mount } = await import('@vue/test-utils')

const { default: ImpersonateReasonModal, IMPERSONATE_REASON_PROMPT, IMPERSONATE_REASON_PLACEHOLDER, FORBIDDEN_IMPERSONATION_REASONS } = await import('./ImpersonateReasonModal.vue')

describe('ImpersonateReasonModal', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders the explanation prompt and placeholder', () => {
    const wrapper = mount(ImpersonateReasonModal, { props: { reason: '', pending: false, error: '' } })
    expect(wrapper.text()).toContain(IMPERSONATE_REASON_PROMPT)
    expect(wrapper.find('[data-testid="impersonate-reason-input"]').attributes('placeholder')).toBe(IMPERSONATE_REASON_PLACEHOLDER)
  })

  it('exposes the prompt/placeholder as forbidden reason constants', () => {
    expect(FORBIDDEN_IMPERSONATION_REASONS.has(IMPERSONATE_REASON_PROMPT)).toBe(true)
    expect(FORBIDDEN_IMPERSONATION_REASONS.has(IMPERSONATE_REASON_PLACEHOLDER)).toBe(true)
    expect(FORBIDDEN_IMPERSONATION_REASONS.size).toBe(2)
  })

  it('emits update:reason on input and confirm on confirm click', async () => {
    const wrapper = mount(ImpersonateReasonModal, { props: { reason: '', pending: false, error: '' } })
    const input = wrapper.find('[data-testid="impersonate-reason-input"]')
    await input.setValue('排查线上工单问题')
    expect(wrapper.emitted('update:reason')).toBeTruthy()
    expect(wrapper.emitted('update:reason')[0][0]).toBe('排查线上工单问题')
    await wrapper.find('[data-testid="impersonate-reason-confirm"]').trigger('click')
    expect(wrapper.emitted('confirm')).toBeTruthy()
  })
})
}
