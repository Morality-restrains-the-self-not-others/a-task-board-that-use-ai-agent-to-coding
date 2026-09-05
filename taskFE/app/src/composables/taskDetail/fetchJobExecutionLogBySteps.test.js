// @vitest-environment node
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] fetchJobExecutionLogBySteps.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { mergeAgentStepsByNumber, mergeJobMetaPreserveHint, inferJobStatusFromAgentSteps, fetchJobExecutionLogBySteps, } = await import('./fetchJobExecutionLogBySteps.js')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))


describe('mergeAgentStepsByNumber', () => {
  it('merges and sorts by step_number', () => {
    const out = mergeAgentStepsByNumber(
      [{ step_number: 1, delivery_summary: 'a' }],
      [
        { step_number: 3, delivery_summary: 'c' },
        { step_number: 2, delivery_summary: 'b' },
      ],
    )
    expect(out.map((s) => s.step_number)).toEqual([1, 2, 3])
  })

  it('replaces same step_number with newer page', () => {
    const out = mergeAgentStepsByNumber(
      [{ step_number: 1, delivery_summary: 'old' }],
      [{ step_number: 1, delivery_summary: 'new' }],
    )
    expect(out).toHaveLength(1)
    expect(out[0].delivery_summary).toBe('new')
  })
})

describe('mergeJobMetaPreserveHint', () => {
  it('does not let empty meta=0 stub status wipe jobHint running', () => {
    const out = mergeJobMetaPreserveHint(
      { id: 'J1', status: 'running', command: 'do work' },
      { id: 'J1', status: '', output_omitted: true },
    )
    expect(out.status).toBe('running')
    expect(out.command).toBe('do work')
  })

  it('accepts non-empty remote status', () => {
    const out = mergeJobMetaPreserveHint(
      { id: 'J1', status: 'running' },
      { id: 'J1', status: 'completed' },
    )
    expect(out.status).toBe('completed')
  })
})

describe('inferJobStatusFromAgentSteps', () => {
  it('infers completed when all steps terminal and no has_more', () => {
    expect(
      inferJobStatusFromAgentSteps(
        [
          { step_number: 1, state: 'completed' },
          { step_number: 2, state: 'completed' },
        ],
        'running',
        { hasMore: false, totalSteps: 2 },
      ),
    ).toBe('completed')
  })

  it('keeps running when a step is still active', () => {
    expect(
      inferJobStatusFromAgentSteps(
        [
          { step_number: 1, state: 'completed' },
          { step_number: 2, state: 'running' },
        ],
        'running',
        { hasMore: false },
      ),
    ).toBe('running')
  })

  it('does not invent terminal when steps lack state', () => {
    expect(
      inferJobStatusFromAgentSteps([{ step_number: 1 }], 'running', { hasMore: false }),
    ).toBe('running')
  })
})

describe('fetchJobExecutionLogBySteps', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('pages until has_more false and strips job.output', async () => {
    apiFetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          job: { id: 'J1', layer_id: 'L1', status: 'running', output: 'HUGE' },
          steps: {
            steps: [{ step_number: 1 }, { step_number: 2 }],
            has_more: true,
            next_after_step: 2,
            total_steps: 3,
          },
          layer_changes: { layer_id: 'L1', changes: [], change_count: 0 },
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          job: { id: 'J1', layer_id: 'L1', status: 'completed' },
          steps: {
            steps: [{ step_number: 3 }],
            has_more: false,
            next_after_step: null,
            total_steps: 3,
          },
          layer_changes: { layer_id: 'L1', changes: [{ path: 'a' }], change_count: 1 },
        }),
      })

    const partials = []
    const result = await fetchJobExecutionLogBySteps({
      apiBase: '/api/t/w/task/T/cloud/compute/',
      jobId: 'J1',
      layerId: 'L1',
      onPartial: (b) => partials.push(b),
    })

    expect(result.ok).toBe(true)
    expect(result.body.job.output).toBeUndefined()
    expect(result.body.job.output_omitted).toBe(true)
    expect(result.body.steps.steps.map((s) => s.step_number)).toEqual([1, 2, 3])
    expect(result.body.layer_changes.change_count).toBe(1)
    expect(partials.length).toBe(2)
    expect(apiFetch).toHaveBeenCalledTimes(2)
    const firstUrl = String(apiFetch.mock.calls[0][0])
    expect(firstUrl).toContain('after_step=0')
    expect(firstUrl).toContain('limit=20')
    expect(firstUrl).toContain('layer_id=L1')
    expect(firstUrl).toContain('meta=0')
    expect(String(apiFetch.mock.calls[1][0])).toContain('after_step=2')
  })

  it('preserves jobHint status when meta=0 stub returns empty status, then infers completed from steps', async () => {
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        job: { id: 'J9', layer_id: 'L9', status: '', output_omitted: true },
        steps: {
          steps: [
            { step_number: 1, state: 'completed' },
            { step_number: 2, state: 'completed' },
          ],
          has_more: false,
          total_steps: 2,
        },
      }),
    })
    const result = await fetchJobExecutionLogBySteps({
      apiBase: '/api/t/w/task/T/cloud/compute/',
      jobId: 'J9',
      layerId: 'L9',
      jobHint: { id: 'J9', layer_id: 'L9', status: 'running', command: 'fix bug' },
    })
    expect(result.ok).toBe(true)
    expect(result.body.job.status).toBe('completed')
    expect(result.body.job.command).toBe('fix bug')
  })

  it('does not surface a bare HTTP status when JSON detail is missing', async () => {
    apiFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      json: async () => ({}),
    })
    const result = await fetchJobExecutionLogBySteps({
      apiBase: '/api/cloud/compute/tenant_id/t/workspace_id/w/task_id/T/',
      jobId: 'J1',
    })
    expect(result.ok).toBe(false)
    expect(result.err).not.toBe(404)
    expect(result.err).not.toBe('404')
    expect(String(result.err)).toMatch(/HTTP 404/)
    expect(String(result.err)).toMatch(/任务日志/)
  })

  it('keeps JSON detail when present on HTTP error', async () => {
    apiFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({ detail: 'Invalid or missing access token' }),
    })
    const result = await fetchJobExecutionLogBySteps({
      apiBase: '/api/cloud/compute/tenant_id/t/workspace_id/w/task_id/T/',
      jobId: 'J1',
    })
    expect(result.ok).toBe(false)
    expect(result.err).toBe('Invalid or missing access token')
  })

  it('uses buildJobLogUrl when provided', async () => {
    apiFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        job: { id: 'J1', status: 'completed' },
        steps: { steps: [], has_more: false },
      }),
    })
    await fetchJobExecutionLogBySteps({
      jobId: 'J1',
      commentId: 'cmt-exec',
      buildJobLogUrl: (qs) => `/api/cloud/compute/container-job-execution-log/tenant_id/t/?${qs.toString()}`,
    })
    const url = String(apiFetch.mock.calls[0][0])
    expect(url).toContain(
      '/api/cloud/compute/container-job-execution-log/tenant_id/t/',
    )
    expect(url).toContain('/comment_id/cmt-exec/')
    expect(url).not.toMatch(/[?&]comment_id=/)
  })
})
}
