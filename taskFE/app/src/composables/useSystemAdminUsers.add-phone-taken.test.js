// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useSystemAdminUsers.add-phone-taken.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ query: {} }),
    useRouter: () => ({ replace: vi.fn() }),
  }))

  const { useSystemAdminUsers } = await import('./useSystemAdminUsers.js')
  const { showRequestError } = await import('../utils/requestErrorDisplay.js')

  const TRACE = 'a1b2c3d4-98e4-4eee-b954-64f0e049c38c'

  describe('useSystemAdminUsers add phone taken', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      showRequestError.mockReset()
      hoisted.apiFetch.mockImplementation((url, opts) => {
        if (opts?.method === 'POST' && String(url).includes('/users/create/')) {
          return Promise.resolve({
            ok: false,
            status: 409,
            json: async () => ({
              detail: '该手机号已被其他用户使用',
              error: '该手机号已被其他用户使用',
              code: 'phone_taken',
              trace_id: TRACE,
            }),
            headers: {
              get: (name) => (String(name).toLowerCase() === 'x-trace-id' ? TRACE : null),
            },
          })
        }
        return Promise.resolve({
          ok: true,
          json: async () => ({ users: [], total: 0 }),
          headers: { get: () => null },
        })
      })
    })

    it('sets phone field error and keeps the add modal open on 409', async () => {
      const api = useSystemAdminUsers()
      api.openAddUserModal()
      api.addUserForm.value = {
        username: '新人',
        email: 'new-user@test.com',
        phone: '13900001111',
        password: 'secret12',
        is_superuser: false,
        is_staff: false,
        is_tenant: false,
        is_tester: false,
      }
      await api.handleAddUser()
      await flushPromises()

      expect(api.addUserModalVisible.value).toBe(true)
      expect(api.addUserError.value).toBe('该手机号已被其他用户使用')
      expect(api.addUserErrorField.value).toBe('phone')
      expect(api.addUserErrorTraceId.value).toBe(TRACE)
      expect(showRequestError).not.toHaveBeenCalled()
    })
  })
}
