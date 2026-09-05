// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useSystemAdminUsers.edit-phone-unbind.test.js requires vitest runtime')
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

  describe('useSystemAdminUsers edit phone unbind', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation((url, opts) => {
        if (opts?.method === 'PUT') {
          return Promise.resolve({
            ok: true,
            status: 200,
            json: async () => ({ id: '877397583960502272', phone: '' }),
            headers: { get: () => null },
          })
        }
        return Promise.resolve({
          ok: true,
          json: async () => ({ users: [], total: 0 }),
          headers: { get: () => null },
        })
      })
    })

    it('sends empty phone and closes the edit modal on 200', async () => {
      const api = useSystemAdminUsers()
      api.openEditUserModal({
        id: '877397583960502272',
        username: '软刀',
        email: 'contact@daydaymoney.com',
        phone: '18959264502',
        is_superuser: false,
        is_staff: false,
        is_tenant: true,
        is_tester: false,
      })
      api.editUserForm.value.phone = ''
      await api.handleEditUser()
      await flushPromises()

      const putCall = hoisted.apiFetch.mock.calls.find((c) => c[1]?.method === 'PUT')
      expect(putCall).toBeTruthy()
      const payload = JSON.parse(putCall[1].body)
      expect(payload).toHaveProperty('phone')
      expect(payload.phone).toBe('')
      expect(api.editUserModalVisible.value).toBe(false)
    })
  })
}
