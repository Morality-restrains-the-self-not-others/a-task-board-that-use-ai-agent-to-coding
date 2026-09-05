// @vitest-environment jsdom
/**
 * 回归：OPT-20260819-005 — 长订单号（ORD-YYYYMMDD-{tenantId}-{snowflake}，约 50 字符）
 * 在支付/取消订单弹窗中须 break-all 换行，避免撑破弹窗。
 */
if (!process.env.VITEST) {
  console.log('[skip] OrderNumberModals.break-all.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    renderQrToCanvas: vi.fn(),
  }))

  vi.mock('../composables/useQrCodeCanvas.js', () => ({
    renderQrToCanvas: (...args) => mocks.renderQrToCanvas(...args),
  }))

  const { default: PayOrderModal } = await import('./PayOrderModal.vue')
  const { default: CancelOrderModal } = await import('./CancelOrderModal.vue')

  const LONG_ORDER_NUMBER = 'ORD-20260820-877397588196749312-876543210987654321'

  const findOrderNumberStrong = (wrapper) =>
    wrapper.findAll('strong').find((el) => el.text() === LONG_ORDER_NUMBER)

  describe('订单弹窗长订单号换行', () => {
    it('PayOrderModal 订单号带 break-all 类', async () => {
      mocks.renderQrToCanvas.mockReset()
      mocks.renderQrToCanvas.mockResolvedValue({ success: true, method: 'qrcode' })
      const wrapper = mount(PayOrderModal, {
        props: {
          order: { id: '1', order_number: LONG_ORDER_NUMBER, total_yuan: '3.00' },
          codeUrl: 'weixin://wxpay/bizpayurl?pr=WXTEST',
          mode: 'live',
        },
        global: { stubs: { PhoneVerificationGate: true } },
      })
      const strong = findOrderNumberStrong(wrapper)
      expect(strong).toBeTruthy()
      expect(strong.classes()).toContain('break-all')
      wrapper.unmount()
    })

    it('CancelOrderModal 订单号带 break-all 类', () => {
      const wrapper = mount(CancelOrderModal, {
        props: {
          order: { id: '1', order_number: LONG_ORDER_NUMBER, total_yuan: '3.00' },
          cancelling: false,
        },
      })
      const strong = findOrderNumberStrong(wrapper)
      expect(strong).toBeTruthy()
      expect(strong.classes()).toContain('break-all')
      wrapper.unmount()
    })
  })
}
