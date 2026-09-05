import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('attachClientPublicIpForAutoRun', () => {
  beforeEach(() => {
    vi.resetModules()
  })

  it('auto_run=true 时写入 client_public_ip', async () => {
    vi.doMock('./publicClientIp.js', () => ({
      queryClientPublicIpForAutoSg: vi.fn(async () => '203.0.113.88'),
    }))
    const { attachClientPublicIpForAutoRun } = await import('./workPanelAutoRunClientIp.js')
    const out = await attachClientPublicIpForAutoRun({ title: 't', auto_run: true })
    expect(out.client_public_ip).toBe('203.0.113.88')
    expect(out.auto_run).toBe(true)
  })

  it('auto_run 非 true 时不查询', async () => {
    const query = vi.fn(async () => '203.0.113.88')
    vi.doMock('./publicClientIp.js', () => ({ queryClientPublicIpForAutoSg: query }))
    const { attachClientPublicIpForAutoRun } = await import('./workPanelAutoRunClientIp.js')
    const out = await attachClientPublicIpForAutoRun({ title: 't', auto_run: false })
    expect(query).not.toHaveBeenCalled()
    expect(out.client_public_ip).toBeUndefined()
  })

  it('查询失败不阻断且不写字段', async () => {
    vi.doMock('./publicClientIp.js', () => ({
      queryClientPublicIpForAutoSg: vi.fn(async () => ''),
    }))
    const { attachClientPublicIpForAutoRun } = await import('./workPanelAutoRunClientIp.js')
    const out = await attachClientPublicIpForAutoRun({ title: 't', auto_run: true })
    expect(out.client_public_ip).toBeUndefined()
  })
})
