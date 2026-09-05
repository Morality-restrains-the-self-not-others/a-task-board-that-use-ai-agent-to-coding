// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] cancelWaitingPreviousBinding.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

vi.mock('../../utils/commentExecutionApi.js', () => ({
  cancelCommentContainerBinding: vi.fn(),
}))

const { cancelCommentContainerBinding } = await import('../../utils/commentExecutionApi.js')
const { createCancelWaitingPreviousBinding } = await import('./cancelWaitingPreviousBinding.js')

describe('createCancelWaitingPreviousBinding', () => {
  beforeEach(() => {
    cancelCommentContainerBinding.mockReset()
  })

  it('calls cancel API and refreshes bindings', async () => {
    cancelCommentContainerBinding.mockResolvedValue({
      status: 'success',
      binding: { comment_id: 'c2', status: 'cancelled' },
    })
    const refreshBindings = vi.fn().mockResolvedValue(undefined)
    const appendBindingStatusLog = vi.fn()
    const api = createCancelWaitingPreviousBinding({
      ids: () => ({ tid: 't1', wid: 'w1', tk: 'task1' }),
      refreshBindings,
      appendBindingStatusLog,
    })
    await api.cancelWaitingPreviousBinding('c2')
    expect(cancelCommentContainerBinding).toHaveBeenCalledWith({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      commentId: 'c2',
    })
    expect(appendBindingStatusLog).toHaveBeenCalledWith('c2', 'cancelled')
    expect(refreshBindings).toHaveBeenCalled()
    expect(api.isCancelWaitingBusy('c2')).toBe(false)
  })
})
}
