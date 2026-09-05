// @vitest-environment jsdom
/**
 * OPT-20260807-031 回归测试：CreateVswitchModal 的云资源请求全部使用 kv 形式
 * /api/cloud/{family}/{sub}/tenant_id/{tid}/，不再调用旧式 /api/tenant/{tid}/cloud/*（404）。
 * 覆盖：挂载时 zones / 已占用网段列表，提交时 create-vswitch / update-vswitch。
 */
if (!process.env.VITEST) {
  console.log('[skip] CreateVswitchModal.endpoint.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, it, expect, vi, beforeEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    alertMock: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoisted.apiFetchMock,
  }))

  vi.mock('../../utils/modalService.js', () => ({
    __esModule: true,
    default: { alert: hoisted.alertMock },
  }))

  const { default: CreateVswitchModal } = await import('./CreateVswitchModal.vue')

  const TENANT_ID = '873472655125147648'

  function mountModal(initialData = {}) {
    return mount(CreateVswitchModal, {
      props: {
        visible: true,
        initialData: {
          authorization_id: 'auth-123',
          platform_type: 'aliyun',
          vpc_id: 'vpc-1',
          vpc_cidr_block: '192.168.0.0/16',
          region: 'cn-chengdu',
          ...initialData,
        },
      },
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, 'location', {
      value: {
        pathname: `/tenant/${TENANT_ID}/projects/proj_-3243404192695446749/`,
      },
      writable: true,
      configurable: true,
    })
    hoisted.apiFetchMock.mockResolvedValue({
      ok: true,
      json: async () => [],
    })
  })

  it('挂载时 zones 使用 cloud-platform kv 形式', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const urls = hoisted.apiFetchMock.mock.calls.map((c) => String(c[0]))
    expect(urls).toContain(
      `/api/cloud/cloud-platform/auth-123/zones/tenant_id/${TENANT_ID}/?region_id=cn-chengdu`
    )
    expect(urls.some((u) => u.includes('/api/tenant/'))).toBe(false)
  })

  it('挂载时已占用网段使用 server-images vswitches kv 形式', async () => {
    const wrapper = mountModal()
    await flushPromises()

    const urls = hoisted.apiFetchMock.mock.calls.map((c) => String(c[0]))
    expect(urls).toContain(
      `/api/cloud/server-images/vswitches/tenant_id/${TENANT_ID}/?region_id=cn-chengdu&vpc_id=vpc-1&authorization_id=auth-123`
    )
  })

  it('创建交换机：POST create-vswitch 端点（非 /api/tenant/）', async () => {
    // 可用区列表返回一个 zone，使 select 可选中
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/zones/')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ status: 'success', zones: [{ zone_id: 'cn-chengdu-h', name: '可用区H' }] }),
        })
      }
      return Promise.resolve({ ok: true, json: async () => [] })
    })

    const wrapper = mountModal()
    await flushPromises()

    const inputs = wrapper.findAll('input[type="text"]')
    await inputs[0].setValue('my-vsw') // name
    await wrapper.find('select').setValue('cn-chengdu-h')
    await inputs[1].setValue('192.168.1.0/24') // cidr_block
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    const postCall = hoisted.apiFetchMock.mock.calls.find(([, o]) => o?.method === 'POST')
    expect(postCall).toBeTruthy()
    expect(String(postCall[0])).toBe(`/api/cloud/server-images/create-vswitch/tenant_id/${TENANT_ID}/`)
    expect(String(postCall[0])).not.toContain('/api/tenant/')
  })

  it('编辑交换机：PUT update-vswitch 端点（非 /api/tenant/）', async () => {
    hoisted.apiFetchMock.mockImplementation((url) => {
      if (String(url).includes('/zones/')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ status: 'success', zones: [{ zone_id: 'cn-chengdu-h', name: '可用区H' }] }),
        })
      }
      return Promise.resolve({ ok: true, json: async () => [] })
    })

    const wrapper = mountModal({
      vswitch_id: 'vsw-1',
      vswitch_name: 'old-vsw',
      zone_id: 'cn-chengdu-h',
      cidr_block: '192.168.1.0/24',
    })
    await flushPromises()

    // 编辑模式表单已回填；只需重新触发提交
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    const putCall = hoisted.apiFetchMock.mock.calls.find(([, o]) => o?.method === 'PUT')
    expect(putCall).toBeTruthy()
    expect(String(putCall[0])).toBe(`/api/cloud/server-images/update-vswitch/tenant_id/${TENANT_ID}/`)
    expect(String(putCall[0])).not.toContain('/api/tenant/')
  })
}
