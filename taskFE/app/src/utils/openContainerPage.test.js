import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('openContainerPageWithIngressEnsure', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
  })

  it('calls ensure-client-ingress then opens url', async () => {
    const apiFetch = vi.fn(async () => ({ ok: true, json: async () => ({ status: 'success' }) }))
    const fetchPublicClientIp = vi.fn(async () => '203.0.113.9')
    vi.doMock('./apiUtils.js', () => ({ apiFetch }))
    vi.doMock('./publicClientIp.js', () => ({ fetchPublicClientIp }))
    // Avoid real RTCPeerConnection
    vi.stubGlobal('RTCPeerConnection', undefined)

    const { openContainerPageWithIngressEnsure } = await import('./openContainerPage.js')
    const openFn = vi.fn()
    await openContainerPageWithIngressEnsure({
      containerPageUrl: 'http://203.0.113.50:8765/ui/tok_x',
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      openFn,
      settleMs: 0,
    })
    expect(apiFetch).toHaveBeenCalled()
    const [path, init] = apiFetch.mock.calls[0]
    expect(String(path)).toContain('/cloud/compute/ensure-client-ingress/')
    expect(String(path)).toContain('task_id=task1')
    expect(init.method).toBe('POST')
    const body = JSON.parse(init.body)
    expect(body.client_public_ip).toBe('203.0.113.9')
    expect(openFn).toHaveBeenCalledWith('http://203.0.113.50:8765/ui/tok_x')
  })

  it('passes comment_id in path (not query) when commentId provided', async () => {
    const apiFetch = vi.fn(async () => ({ ok: true, json: async () => ({ status: 'success' }) }))
    vi.doMock('./apiUtils.js', () => ({ apiFetch }))
    vi.doMock('./publicClientIp.js', () => ({
      fetchPublicClientIp: vi.fn(async () => '203.0.113.9'),
    }))
    vi.stubGlobal('RTCPeerConnection', undefined)

    const { openContainerPageWithIngressEnsure } = await import('./openContainerPage.js')
    const openFn = vi.fn()
    await openContainerPageWithIngressEnsure({
      containerPageUrl: 'http://203.0.113.50:8765/ui/tok_x',
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      commentId: 'cmt_a',
      openFn,
      settleMs: 0,
    })
    const [path, init] = apiFetch.mock.calls[0]
    expect(String(path)).toContain('/comment_id/cmt_a/')
    expect(String(path)).toContain('task_id=task1')
    expect(String(path)).not.toMatch(/[?&]comment_id=/)
    expect(init.method).toBe('POST')
    expect(openFn).toHaveBeenCalledWith('http://203.0.113.50:8765/ui/tok_x')
  })

  it('opens even when ensure fails', async () => {
    vi.doMock('./apiUtils.js', () => ({
      apiFetch: vi.fn(async () => {
        throw new Error('network')
      }),
    }))
    vi.doMock('./publicClientIp.js', () => ({
      fetchPublicClientIp: vi.fn(async () => '203.0.113.9'),
    }))
    vi.stubGlobal('RTCPeerConnection', undefined)
    const { openContainerPageWithIngressEnsure } = await import('./openContainerPage.js')
    const openFn = vi.fn()
    await openContainerPageWithIngressEnsure({
      containerPageUrl: 'http://203.0.113.50:8765/ui/tok_x',
      tenantId: 't1',
      workspaceId: 'w1',
      taskId: 'task1',
      openFn,
      settleMs: 0,
    })
    expect(openFn).toHaveBeenCalledWith('http://203.0.113.50:8765/ui/tok_x')
  })
})
