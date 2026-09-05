// @vitest-environment jsdom
// 创建订单成功后必须进入独立详情页；create 静态段不得被 :orderId 吞掉。
if (!process.env.VITEST) {
  console.log('[skip] router.billingOrderDetail.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { createRouter, createMemoryHistory } = await import('vue-router')
  const { publicRoutes } = await import('./router/publicRoutes.js')
  const { tenantRoutes } = await import('./router/tenantRoutes.js')
  const { adminRoutes } = await import('./router/adminRoutes.js')

  function makeRouter() {
    return createRouter({
      history: createMemoryHistory(),
      routes: [...publicRoutes, ...tenantRoutes, ...adminRoutes],
    })
  }

  describe('billing order detail 路由', () => {
    it(' /billing/orders/create/ 命中 order_create', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/billing/orders/create/')
      expect(router.currentRoute.value.name).toBe('order_create')
    })

    it('/billing/orders/:orderId/ 命中 billing_order_detail', async () => {
      const router = makeRouter()
      await router.push('/tenant/t1/billing/orders/875588283562749952/')
      expect(router.currentRoute.value.name).toBe('billing_order_detail')
      expect(router.currentRoute.value.params.orderId).toBe('875588283562749952')
    })
  })
}
