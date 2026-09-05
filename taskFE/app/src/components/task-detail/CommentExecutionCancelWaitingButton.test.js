// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CommentExecutionCancelWaitingButton.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount } = await import('@vue/test-utils')

vi.mock('../../utils/modalService.js', () => ({
  default: {
    confirm: vi.fn(),
  },
}))

const { default: CommentExecutionCancelWaitingButton } = await import('./CommentExecutionCancelWaitingButton.vue')
const { default: modalService } = await import('../../utils/modalService.js')

describe('CommentExecutionCancelWaitingButton', () => {
  beforeEach(() => {
    modalService.confirm.mockReset()
  })

  it('hides when not visible', () => {
    const wrapper = mount(CommentExecutionCancelWaitingButton, {
      props: { visible: false },
    })
    expect(wrapper.find('[data-testid="comment-execution-cancel-waiting"]').exists()).toBe(false)
  })

  it('shows terminate control when waiting', () => {
    const wrapper = mount(CommentExecutionCancelWaitingButton, {
      props: { visible: true, commentId: 'c1' },
    })
    expect(wrapper.get('[data-testid="comment-execution-cancel-waiting"]').text()).toContain('终止')
  })

  it('emits cancel-waiting after confirm', async () => {
    modalService.confirm.mockResolvedValue(true)
    const wrapper = mount(CommentExecutionCancelWaitingButton, {
      props: { visible: true, commentId: 'c1' },
    })
    await wrapper.get('[data-testid="comment-execution-cancel-waiting"]').trigger('click')
    await wrapper.vm.$nextTick()
    expect(modalService.confirm).toHaveBeenCalled()
    expect(wrapper.emitted('cancel-waiting')).toBeTruthy()
  })

  it('does not emit when confirm cancelled', async () => {
    modalService.confirm.mockRejectedValue(false)
    const wrapper = mount(CommentExecutionCancelWaitingButton, {
      props: { visible: true, commentId: 'c1' },
    })
    await wrapper.get('[data-testid="comment-execution-cancel-waiting"]').trigger('click')
    await wrapper.vm.$nextTick()
    expect(wrapper.emitted('cancel-waiting')).toBeFalsy()
  })
})

}
