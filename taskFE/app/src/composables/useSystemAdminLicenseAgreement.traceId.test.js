if (!process.env.VITEST) {
  console.log('[skip] useSystemAdminLicenseAgreement.traceId.test.js requires vitest runtime')
} else {
const { describe, it, expect, vi, beforeEach } = await import('vitest')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../utils/apiUtils.js')
const { default: modalService } = await import('../utils/modalService.js')
const { useSystemAdminLicenseAgreement } = await import('./useSystemAdminLicenseAgreement.js')

/**
 * data-traceId 流转：apiFetch 失败路径必须把网关 X-Trace-Id（response.traceId）
 * 带到 showRequestError → modalService.state.traceId → Modal 根节点 data-traceId。
 */
describe('useSystemAdminLicenseAgreement data-traceId', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    modalService.state.show = false
    modalService.state.traceId = ''
  })

  it('refreshAgreements 失败时保留 response.traceId 并展示后端 detail', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 500,
      traceId: 'tid-list-500',
      json: async () => ({ detail: '数据库繁忙', _traceId: 'tid-body' }),
    })

    const { refreshAgreements, agreements } = useSystemAdminLicenseAgreement()
    await refreshAgreements()

    expect(modalService.state.traceId).toBe('tid-list-500')
    expect(modalService.state.message).toContain('数据库繁忙')
    expect(agreements.value).toEqual([])
  })

  it('refreshAgreements 网关 502 非 JSON 响应：兜底文案 + traceId 仍保留', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      traceId: 'tid-gw-502',
      json: async () => {
        throw new Error('invalid json')
      },
    })

    const { refreshAgreements } = useSystemAdminLicenseAgreement()
    await refreshAgreements()

    expect(modalService.state.traceId).toBe('tid-gw-502')
    expect(modalService.state.message).toContain('获取协议列表失败')
  })

  it('refreshAgreements 成功：解析列表且不弹错误', async () => {
    apiFetch.mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'la-1', title: '服务协议' }],
    })

    const { refreshAgreements, agreements } = useSystemAdminLicenseAgreement()
    await refreshAgreements()

    expect(agreements.value).toHaveLength(1)
    expect(agreements.value[0].title).toBe('服务协议')
    expect(modalService.state.show).toBe(false)
  })

  it('handleCreate 失败同样携带 traceId', async () => {
    apiFetch
      .mockResolvedValueOnce({ ok: true, json: async () => [] }) // 初始 refreshAgreements
      .mockResolvedValueOnce({
        ok: false,
        status: 400,
        traceId: 'tid-create-400',
        json: async () => ({ detail: 'title required' }),
      })

    const { handleCreate } = useSystemAdminLicenseAgreement()
    await handleCreate()

    expect(modalService.state.traceId).toBe('tid-create-400')
    expect(modalService.state.message).toContain('title required')
  })
})

}
