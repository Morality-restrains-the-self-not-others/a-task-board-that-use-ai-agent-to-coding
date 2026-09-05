// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailEditing.forkCopyCount.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')
  const { forkTask } = await import('./taskDetailEditing.js')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
    humanizeRequestErrorMessage: (m) => String(m ?? ''),
  }))
  vi.mock('../../utils/publicClientIp.js', () => ({
    queryClientPublicIpForAutoSg: vi.fn(async () => ''),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')

  describe('forkTask copy count', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      vi.stubGlobal('window', {
        alert: vi.fn(),
        location: { origin: 'http://127.0.0.1:4000' },
        open: vi.fn(() => ({ closed: false })),
      })
    })

    function baseForkDeps(overrides = {}) {
      return {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('w1'),
        effectiveTaskId: ref('source-task'),
        localTask: ref({
          title: 'fork me',
          owner: '827923618472235008',
          assignees: [],
          progress_column_id: 'col-wip',
          projects: [],
        }),
        progressStatusOptions: ref([{ id: 'col-todo', name: '待处理' }]),
        isForking: ref(false),
        router: {
          resolve: vi.fn((loc) => ({
            href: `/tenant/t1/workspace/w1/task-detail/${loc.params.taskId}/`,
          })),
          push: vi.fn(),
        },
        detailQuery: ref({}),
        ...overrides,
      }
    }

    it('T10: auto-run two models posts twice with different agent_models and no fork_count', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: 'task_a' }),
      })
      apiFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ id: 'task_a' }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ id: 'task_b' }) })
      const onForked = vi.fn()
      const onForkProgress = vi.fn()
      const ok = await forkTask(baseForkDeps({
        autoRun: true,
        batchIdempotencyKey: 'batch-xyz',
        featureParamsSource: 'company',
        agentModelProvider: 'openai',
        agents: [{ model: 'gpt-4.1' }, { model: 'gpt-4.1-mini' }],
        onForked,
        onForkProgress,
      }))
      expect(ok).toBe(true)
      expect(apiFetch).toHaveBeenCalledTimes(2)
      const bodies = apiFetch.mock.calls.map(([, opts]) => JSON.parse(opts.body))
      expect(bodies[0].fork_count).toBeUndefined()
      expect(bodies[1].fork_count).toBeUndefined()
      expect(bodies[0].agent_models).toEqual([{ provider: 'openai', model: 'gpt-4.1' }])
      expect(bodies[1].agent_models).toEqual([{ provider: 'openai', model: 'gpt-4.1-mini' }])
      expect(apiFetch.mock.calls[0][1].headers['Idempotency-Key']).toBe('batch-xyz:1')
      expect(apiFetch.mock.calls[1][1].headers['Idempotency-Key']).toBe('batch-xyz:2')
      expect(onForkProgress).toHaveBeenCalledWith(1, 2)
      expect(window.open).toHaveBeenCalledTimes(1)
      expect(String(window.open.mock.calls[0][0])).toContain('task_a')
      expect(onForked).toHaveBeenCalledTimes(1)
    })

    it('T11: fork-only ignores copyCount 100 — single POST without fork_count', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ id: 'task_x' }),
      })
      await forkTask(baseForkDeps({ autoRun: false, copyCount: 100, batchIdempotencyKey: 'b' }))
      expect(apiFetch).toHaveBeenCalledTimes(1)
      const [, opts] = apiFetch.mock.calls[0]
      const body = JSON.parse(opts.body)
      expect(body.fork_count).toBeUndefined()
      expect(body.agent_models).toBeUndefined()
      expect(body.auto_run).toBe(false)
    })

    it('T12: first copy failure returns false and does not open tab', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 402,
        text: async () => JSON.stringify({ detail: '配额不足' }),
      })
      const ok = await forkTask(baseForkDeps({
        autoRun: true,
        batchIdempotencyKey: 'b',
        agentModelProvider: 'openai',
        agents: [{ model: 'a' }, { model: 'b' }],
      }))
      expect(ok).toBe(false)
      expect(window.open).not.toHaveBeenCalled()
      expect(showRequestError).toHaveBeenCalled()
    })

    it('T13: second copy failure reports created count and still opens first tab', async () => {
      apiFetch
        .mockResolvedValueOnce({ ok: true, json: async () => ({ id: 'task_ok' }) })
        .mockResolvedValueOnce({
          ok: false,
          status: 402,
          text: async () => JSON.stringify({ detail: '配额不足' }),
        })
      const ok = await forkTask(baseForkDeps({
        autoRun: true,
        batchIdempotencyKey: 'b',
        agentModelProvider: 'openai',
        agents: [{ model: 'a' }, { model: 'b' }],
      }))
      expect(ok).toBe(true)
      expect(window.open).toHaveBeenCalledTimes(1)
      expect(String(window.open.mock.calls[0][0])).toContain('task_ok')
      expect(showRequestError).toHaveBeenCalled()
      const msg = String(showRequestError.mock.calls[0][0])
      expect(msg).toContain('1/2')
    })
  })
}
