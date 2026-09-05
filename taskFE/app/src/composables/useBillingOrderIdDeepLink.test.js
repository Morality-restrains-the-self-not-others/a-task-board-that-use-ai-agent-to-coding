// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingOrderIdDeepLink.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const { ref, nextTick } = await import('vue')
  const { useBillingOrderIdDeepLink } = await import('./useBillingOrderIdDeepLink.js')

  describe('useBillingOrderIdDeepLink', () => {
    beforeEach(() => {
      document.body.innerHTML = ''
      Element.prototype.scrollIntoView = vi.fn()
    })
    afterEach(() => {
      document.body.innerHTML = ''
    })

    it('orders 就绪后按 order_id 展开并滚动高亮', async () => {
      const expandOrder = vi.fn(async () => {
        const tr = document.createElement('tr')
        tr.setAttribute('data-order-id', '877596007691485184')
        document.body.appendChild(tr)
      })
      const orders = ref([])
      const loading = ref(false)
      const listOffset = ref(0)
      const currentPage = ref(1)
      const route = { query: { order_id: '877596007691485184' } }

      useBillingOrderIdDeepLink({
        route,
        orders,
        loading,
        listOffset,
        pageSize: 15,
        currentPage,
        expandOrder,
      })

      expect(expandOrder).not.toHaveBeenCalled()

      orders.value = [{ id: '877596007691485184', order_number: 'ORD-1' }]
      await nextTick()
      await Promise.resolve()
      await nextTick()

      expect(expandOrder).toHaveBeenCalledWith(
        expect.objectContaining({ id: '877596007691485184' }),
      )
      const el = document.querySelector('[data-order-id="877596007691485184"]')
      expect(el).toBeTruthy()
      expect(el.scrollIntoView).toHaveBeenCalled()
    })

    it('API 返回的 offset 对齐到 currentPage', async () => {
      const expandOrder = vi.fn(async () => {
        const tr = document.createElement('tr')
        tr.setAttribute('data-order-id', '99')
        document.body.appendChild(tr)
      })
      const orders = ref([{ id: '99', order_number: 'ORD-X' }])
      const loading = ref(false)
      const listOffset = ref(30)
      const currentPage = ref(1)
      const route = { query: { order_id: '99' } }

      useBillingOrderIdDeepLink({
        route,
        orders,
        loading,
        listOffset,
        pageSize: 15,
        currentPage,
        expandOrder,
      })
      await nextTick()
      expect(currentPage.value).toBe(3)
    })

    it('列表中无匹配时调用 onFocusMiss（仅一次）', async () => {
      const onFocusMiss = vi.fn()
      const orders = ref([{ id: '1' }])
      const loading = ref(false)
      const listOffset = ref(0)
      const currentPage = ref(1)
      const route = { query: { order_id: 'missing' } }

      useBillingOrderIdDeepLink({
        route,
        orders,
        loading,
        listOffset,
        pageSize: 15,
        currentPage,
        expandOrder: vi.fn(),
        onFocusMiss,
      })
      await nextTick()
      orders.value = [{ id: '1' }, { id: '2' }]
      await nextTick()
      expect(onFocusMiss).toHaveBeenCalledTimes(1)
      expect(onFocusMiss).toHaveBeenCalledWith('missing')
    })
  })
}
