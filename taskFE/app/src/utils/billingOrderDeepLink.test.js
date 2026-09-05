// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] billingOrderDeepLink.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const {
    parseOrderIdQuery,
    parseTenantIdQuery,
    buildSystemAdminOrderRecordsHref,
    SYSTEM_ADMIN_ORDER_RECORDS_PATH,
    findOrderInList,
    pageFromListOffset,
    findOrderRowEl,
    scrollAndHighlightOrderRow,
    BILLING_ORDER_HIGHLIGHT_CLASS,
  } = await import('./billingOrderDeepLink.js')

  describe('billingOrderDeepLink', () => {
    it('parseOrderIdQuery 读取并 trim order_id', () => {
      expect(parseOrderIdQuery({ order_id: ' 877596007691485184 ' })).toBe('877596007691485184')
      expect(parseOrderIdQuery({})).toBe('')
      expect(parseOrderIdQuery(null)).toBe('')
    })

    it('parseTenantIdQuery 读取并 trim tenant_id', () => {
      expect(parseTenantIdQuery({ tenant_id: ' 877397588196749312 ' })).toBe('877397588196749312')
      expect(parseTenantIdQuery({})).toBe('')
    })

    it('buildSystemAdminOrderRecordsHref 生成管理端订单 Tab 深链', () => {
      expect(
        buildSystemAdminOrderRecordsHref({
          tenantId: '877397588196749312',
          orderId: '877596007691485184',
        }),
      ).toBe(
        `${SYSTEM_ADMIN_ORDER_RECORDS_PATH}?tenant_id=877397588196749312&order_id=877596007691485184`,
      )
      expect(buildSystemAdminOrderRecordsHref({})).toBe(SYSTEM_ADMIN_ORDER_RECORDS_PATH)
    })

    it('findOrderInList 按 id 或 order_number 匹配', () => {
      const orders = [
        { id: '1', order_number: 'ORD-A' },
        { id: '877596007691485184', order_number: 'ORD-B' },
      ]
      expect(findOrderInList(orders, '877596007691485184')?.order_number).toBe('ORD-B')
      expect(findOrderInList(orders, 'ORD-A')?.id).toBe('1')
      expect(findOrderInList(orders, 'missing')).toBeNull()
    })

    it('pageFromListOffset 对齐页码', () => {
      expect(pageFromListOffset(0, 15)).toBe(1)
      expect(pageFromListOffset(15, 15)).toBe(2)
      expect(pageFromListOffset(30, 15)).toBe(3)
    })

    describe('scrollAndHighlightOrderRow', () => {
      beforeEach(() => {
        document.body.innerHTML = ''
        Element.prototype.scrollIntoView = vi.fn()
      })
      afterEach(() => {
        document.body.innerHTML = ''
      })

      it('找到 data-order-id 行后 scrollIntoView 并加高亮 class', () => {
        const tr = document.createElement('tr')
        tr.setAttribute('data-order-id', '877596007691485184')
        document.body.appendChild(tr)

        expect(scrollAndHighlightOrderRow('877596007691485184')).toBe(true)
        expect(tr.scrollIntoView).toHaveBeenCalled()
        expect(tr.classList.contains(BILLING_ORDER_HIGHLIGHT_CLASS)).toBe(true)
        expect(findOrderRowEl('877596007691485184')).toBe(tr)
      })

      it('找不到行返回 false', () => {
        expect(scrollAndHighlightOrderRow('nope')).toBe(false)
      })
    })
  })
}
