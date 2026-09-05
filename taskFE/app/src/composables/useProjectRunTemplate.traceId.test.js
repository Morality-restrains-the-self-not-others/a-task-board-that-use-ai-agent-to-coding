// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useProjectRunTemplate.traceId.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

vi.mock('../utils/workspaceCloudPlatformsApi.js', () => ({
  fetchWorkspaceCloudPlatforms: async () => [],
}))

const { useProjectRunTemplate } = await import('./useProjectRunTemplate.js')

function jsonResponse(body, { ok = true, status = 200, headers = {} } = {}) {
  return {
    ok,
    status,
    headers: { get: (k) => headers[k] || headers[String(k).toLowerCase()] || null },
    json: async () => body,
  }
}

describe('useProjectRunTemplate data-traceId', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('saveTemplate surfaces X-Trace-Id from failed PATCH response', async () => {
    apiFetch.mockImplementation(async () =>
      jsonResponse(
        { detail: 'method not allowed' },
        { ok: false, status: 405, headers: { 'X-Trace-Id': 'tr-save-405' } },
      ),
    )
    const c = useProjectRunTemplate({
      tenantId: () => 't1',
      projectId: () => 'p1',
      workspaceId: () => '',
    })
    await expect(c.saveTemplate()).rejects.toThrow()
    expect(c.errorTraceId.value).toBe('tr-save-405')
  })

  it('saveTemplate surfaces trace_id from error body when header missing', async () => {
    apiFetch.mockImplementation(async () =>
      jsonResponse({ detail: 'internal error', trace_id: 'tr-save-body' }, { ok: false, status: 500 }),
    )
    const c = useProjectRunTemplate({
      tenantId: () => 't1',
      projectId: () => 'p1',
      workspaceId: () => '',
    })
    await expect(c.saveTemplate()).rejects.toThrow()
    expect(c.errorTraceId.value).toBe('tr-save-body')
  })

  it('loadTemplateOptions failure sets errorTraceId from thrown error', async () => {
    apiFetch.mockImplementation(async () => {
      const err = new Error('加载服务器模版失败')
      err.traceId = 'tr-load-1'
      throw err
    })
    const c = useProjectRunTemplate({
      tenantId: () => 't1',
      projectId: () => 'p1',
      workspaceId: () => '',
    })
    await c.loadTemplateOptions()
    expect(c.error.value).toContain('加载服务器模版失败')
    expect(c.errorTraceId.value).toBe('tr-load-1')
  })

  it('successful save clears errorTraceId', async () => {
    apiFetch.mockImplementation(async () =>
      jsonResponse({ server_run_template: { platform_type: 'aliyun' } }),
    )
    const c = useProjectRunTemplate({
      tenantId: () => 't1',
      projectId: () => 'p1',
      workspaceId: () => '',
    })
    c.errorTraceId.value = 'stale'
    await c.saveTemplate()
    expect(c.errorTraceId.value).toBe('')
  })

  // 真实网关格式（2026-08-07 真机复验）：APISIX 网关与上游服务各自注入一行
  // X-Trace-Id，浏览器合并为 "id1, id2"；errorTraceId 必须取首段而非原样合并串。
  it('saveTemplate errorTraceId takes first segment of comma-joined X-Trace-Id', async () => {
    apiFetch.mockImplementation(async () =>
      jsonResponse(
        { detail: 'method not allowed' },
        {
          ok: false,
          status: 405,
          headers: { 'X-Trace-Id': 'ef4fc3d811c51cc9351f8275f927afc1, ef4fc3d811c51cc9351f8275f927afc1' },
        },
      ),
    )
    const c = useProjectRunTemplate({
      tenantId: () => 't1',
      projectId: () => 'p1',
      workspaceId: () => '',
    })
    await expect(c.saveTemplate()).rejects.toThrow()
    expect(c.errorTraceId.value).toBe('ef4fc3d811c51cc9351f8275f927afc1')
  })
})
}
