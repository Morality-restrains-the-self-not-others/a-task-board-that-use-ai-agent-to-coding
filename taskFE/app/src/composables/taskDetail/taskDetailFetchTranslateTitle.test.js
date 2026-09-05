// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchTranslateTitle.test.js requires vitest runtime')
} else {
const { describe, it, expect, vi, beforeEach } = await import('vitest')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const { fetchTranslatedTaskTitleSegment } = await import('./taskDetailFetchTranslateTitle.js')

describe('fetchTranslatedTaskTitleSegment', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('throws a user-safe message instead of fanyi JSON parse internals', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      json: async () => ({
        error: '任务标题翻译失败: fanyi_agent 响应无效: unexpected end of JSON input',
      }),
    })

    await expect(fetchTranslatedTaskTitleSegment('中文标题', {
      effectiveTenantId: { value: 't1' },
      sanitizeBranchSegment: (v, fallback) => v || fallback,
    })).rejects.toThrow(/本地规则/)

    try {
      await fetchTranslatedTaskTitleSegment('中文标题', {
        effectiveTenantId: { value: 't1' },
        sanitizeBranchSegment: (v, fallback) => v || fallback,
      })
    } catch (err) {
      expect(String(err.message)).not.toContain('unexpected end of JSON')
      expect(String(err.message)).not.toContain('fanyi_agent')
    }
  })

  it('forwards AbortSignal to apiFetch so overlapping title translates can cancel', async () => {
    const ac = new AbortController()
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ translated_title: 'fix-login' }),
    })
    await fetchTranslatedTaskTitleSegment('中文标题', {
      effectiveTenantId: { value: 't1' },
      sanitizeBranchSegment: (v, fallback) => v || fallback,
      signal: ac.signal,
    })
    expect(apiFetch).toHaveBeenCalled()
    expect(apiFetch.mock.calls[0][1].signal).toBe(ac.signal)
  })
})
}
