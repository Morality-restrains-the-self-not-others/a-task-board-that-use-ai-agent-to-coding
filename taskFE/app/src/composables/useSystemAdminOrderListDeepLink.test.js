// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useSystemAdminOrderListDeepLink.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { ref, nextTick } = await import('vue')
  const { useSystemAdminOrderListDeepLink } = await import('./useSystemAdminOrderListDeepLink.js')

  describe('useSystemAdminOrderListDeepLink', () => {
    it('有 tenant_id 时按路由选指定租户而非全部租户', async () => {
      const tenantOptions = ref([
        { id: '877397588196749312', name: 'Acme' },
        { id: 'other', name: 'Other' },
      ])
      const selectTenant = vi.fn()
      const selectAllTenants = vi.fn()
      const fetchTenantOptions = vi.fn(async () => {})
      const statusFilter = ref('paid')
      const route = {
        query: {
          tenant_id: '877397588196749312',
          order_id: '877596007691485184',
        },
      }

      const { applyFromRoute } = useSystemAdminOrderListDeepLink({
        route,
        orders: ref([]),
        loading: ref(false),
        listOffset: ref(0),
        pageSize: 15,
        currentPage: ref(1),
        statusFilter,
        tenantOptions,
        expandOrder: vi.fn(),
        selectTenant,
        selectAllTenants,
        fetchTenantOptions,
      })

      expect(statusFilter.value).toBe('')
      await applyFromRoute()
      expect(fetchTenantOptions).toHaveBeenCalledWith('')
      expect(selectAllTenants).not.toHaveBeenCalled()
      expect(selectTenant).toHaveBeenCalledWith({ id: '877397588196749312', name: 'Acme' })
    })

    it('无 tenant_id 时回退全部租户', async () => {
      const selectTenant = vi.fn()
      const selectAllTenants = vi.fn()
      const { applyFromRoute } = useSystemAdminOrderListDeepLink({
        route: { query: {} },
        orders: ref([]),
        loading: ref(false),
        listOffset: ref(0),
        pageSize: 15,
        currentPage: ref(1),
        statusFilter: ref(''),
        tenantOptions: ref([]),
        expandOrder: vi.fn(),
        selectTenant,
        selectAllTenants,
        fetchTenantOptions: vi.fn(async () => {}),
      })
      await applyFromRoute()
      expect(selectAllTenants).toHaveBeenCalled()
      expect(selectTenant).not.toHaveBeenCalled()
      await nextTick()
    })

    it('?wechat_account= 触发微信关联账号查询且不按租户回退', async () => {
      const onWechatAccountSearch = vi.fn(async () => {})
      const selectAllTenants = vi.fn()
      const { applyFromRoute } = useSystemAdminOrderListDeepLink({
        route: { query: { wechat_account: '微信昵称甲' } },
        orders: ref([]),
        loading: ref(false),
        listOffset: ref(0),
        pageSize: 15,
        currentPage: ref(1),
        statusFilter: ref(''),
        tenantOptions: ref([]),
        expandOrder: vi.fn(),
        selectTenant: vi.fn(),
        selectAllTenants,
        fetchTenantOptions: vi.fn(async () => {}),
        onTradeNoSearch: vi.fn(),
        onWechatAccountSearch,
      })
      await applyFromRoute()
      expect(onWechatAccountSearch).toHaveBeenCalledWith('微信昵称甲')
      expect(selectAllTenants).not.toHaveBeenCalled()
    })
  })
}
