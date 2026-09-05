// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentExecutionApi.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

vi.mock('./apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('./apiUtils.js')
const {
  advanceCommentContainerBindings,
  cancelCommentContainerBinding,
  ensureCommentContainerBinding,
  normalizeExecutionMode,
  patchAICommentExecutionMode,
  patchHumanCommentExecutionMode,
} = await import('./commentExecutionApi.js')

describe('commentExecutionApi', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('normalizeExecutionMode', () => {
    expect(normalizeExecutionMode('')).toBe('wait_previous')
    expect(normalizeExecutionMode('independent')).toBe('independent')
    expect(normalizeExecutionMode('other')).toBe('wait_previous')
  })

  it('PATCH human comment execution_mode', async () => {
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({ execution_mode: 'independent' }) })
    await patchHumanCommentExecutionMode({
      tenantId: 't1',
      taskId: 'task1',
      commentId: 'c1',
      executionMode: 'independent',
    })
    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringContaining('/tasks/task1/comments/c1/'),
      expect.objectContaining({ method: 'PATCH' }),
    )
  })

  it('PATCH ai comment execution_mode', async () => {
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
    await patchAICommentExecutionMode({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      commentId: 'a1',
      executionMode: 'wait_previous',
    })
    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringContaining('/ai-comments/a1/'),
      expect.objectContaining({ method: 'PATCH' }),
    )
  })

  it('ensure binding treats 409 as exists (legacy gateway)', async () => {
    apiFetch.mockResolvedValue({ status: 409, ok: false })
    const r = await ensureCommentContainerBinding({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      commentId: 'c1',
      executionMode: 'independent',
    })
    expect(r.status).toBe('exists')
  })

  it('ensure binding accepts 200 upsert with synced binding', async () => {
    apiFetch.mockResolvedValue({
      status: 200,
      ok: true,
      json: async () => ({
        status: 'exists',
        binding: { comment_id: 'c1', execution_mode: 'independent' },
      }),
    })
    const r = await ensureCommentContainerBinding({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      commentId: 'c1',
      executionMode: 'independent',
    })
    expect(r.status).toBe('exists')
    expect(r.binding.execution_mode).toBe('independent')
  })

  it('advance bindings POST', async () => {
    apiFetch.mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) })
    await advanceCommentContainerBindings({ tenantId: 't1', workspaceId: 'w1', taskId: 'task1' })
    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringContaining(
        '/api/cloud/compute/comment-container-bindings/tenant_id/t1/workspace_id/w1/advance/',
      ),
      expect.objectContaining({ method: 'POST' }),
    )
  })

  it('cancel waiting_previous binding POST', async () => {
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ status: 'success', binding: { status: 'cancelled' } }),
    })
    await cancelCommentContainerBinding({
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      commentId: 'c2',
    })
    expect(apiFetch).toHaveBeenCalledWith(
      expect.stringContaining(
        '/api/cloud/compute/comment-container-bindings/tenant_id/t1/workspace_id/w1/c2/cancel/',
      ),
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

}
