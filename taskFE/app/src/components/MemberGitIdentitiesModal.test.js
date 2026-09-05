// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] MemberGitIdentitiesModal.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')

// vi.mock 工厂引用的变量必须经 vi.hoisted() 定义（Vitest 会把工厂 hoist 到模块顶部）。
const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))
vi.mock('../utils/requestErrorDisplay.js', () => ({
  humanizeRequestErrorMessage: (m) => m,
  showRequestError: vi.fn(),
}))

const { default: MemberGitIdentitiesModal } = await import('./MemberGitIdentitiesModal.vue')

function jsonResp(ok, body, status = 200) {
  return {
    ok,
    status,
    traceId: 'tr-test',
    json: async () => body,
  }
}

describe('MemberGitIdentitiesModal', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('loads identities for tenant member', async () => {
    apiFetch.mockResolvedValueOnce(jsonResp(true, {
      identities: [
        { id: 'gi1', git_user_name: 'Bob', git_user_email: 'a.b@daydaymoney.com', label: 'system-auto', is_default: true },
      ],
    }))
    const wrapper = mount(MemberGitIdentitiesModal, {
      props: { tenantId: 't1', memberId: 'm1', memberName: 'Bob' },
    })
    await flushPromises()
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/git-identities/tenant/t1/member/m1/',
      expect.objectContaining({ credentials: 'include' }),
    )
    expect(wrapper.text()).toContain('Bob')
    expect(wrapper.text()).toContain('a.b@daydaymoney.com')
    expect(wrapper.text()).toContain('系统')
  })

  it('shows error with data-traceId on load failure', async () => {
    apiFetch.mockResolvedValueOnce(jsonResp(false, {}, 500))
    const wrapper = mount(MemberGitIdentitiesModal, {
      props: { tenantId: 't1', memberId: 'm1' },
    })
    await flushPromises()
    const err = wrapper.find('[data-testid=member-git-identity-error]')
    expect(err.exists()).toBe(true)
    expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBeTruthy()
  })
})
}
