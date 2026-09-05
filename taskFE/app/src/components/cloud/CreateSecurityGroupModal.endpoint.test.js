// @vitest-environment jsdom
/**
 * 回归测试：OPT-20260807-031 — CreateSecurityGroupModal 创建/编辑端点曾为字面量
 * /api/cloud/server-images/$1/tenant_id/...（两个分支相同），后端必然 404。
 * 修复后：创建 → create-security-group（POST），编辑 → update-security-group（PUT）。
 */
if (!process.env.VITEST) {
  console.log('[skip] CreateSecurityGroupModal.endpoint.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
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

  const { default: CreateSecurityGroupModal } = await import('./CreateSecurityGroupModal.vue')

  const TENANT_ID = '873472655125147648'

  function mountModal(initialData = {}) {
    return mount(CreateSecurityGroupModal, {
      props: {
        visible: true,
        initialData: {
          authorization_id: 'auth-123',
          platform_type: 'aliyun',
          vpc_id: 'vpc-1',
          region: 'cn-chengdu',
          ...initialData,
        },
      },
    })
  }

  async function fillNameAndSubmit(wrapper, name) {
    await wrapper.find('input[type="text"]').setValue(name)
    await nextTick()
    await wrapper.find('form').trigger('submit')
    await nextTick()
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
      json: async () => ({ status: 'success' }),
    })
  })

  it('创建安全组：POST create-security-group 端点（非 $1）', async () => {
    const wrapper = mountModal()
    await fillNameAndSubmit(wrapper, 'my-sg')

    expect(hoisted.apiFetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = hoisted.apiFetchMock.mock.calls[0]
    expect(url).toBe(`/api/cloud/server-images/create-security-group/tenant_id/${TENANT_ID}/`)
    expect(url).not.toContain('$1')
    expect(options.method).toBe('POST')
    expect(wrapper.emitted('created')).toBeTruthy()
  })

  it('编辑安全组：PUT update-security-group 端点（非 $1）', async () => {
    const wrapper = mountModal({ security_group_id: 'sg-1', security_group_name: 'old-sg' })
    await fillNameAndSubmit(wrapper, 'renamed-sg')

    expect(hoisted.apiFetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = hoisted.apiFetchMock.mock.calls[0]
    expect(url).toBe(`/api/cloud/server-images/update-security-group/tenant_id/${TENANT_ID}/`)
    expect(url).not.toContain('$1')
    expect(options.method).toBe('PUT')
  })
}
