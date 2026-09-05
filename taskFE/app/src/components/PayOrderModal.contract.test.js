// @vitest-environment jsdom
// 支付弹窗：canvas 须在挂载后绘制 QR；无模拟支付入口。
if (!process.env.VITEST) {
  console.log('[skip] PayOrderModal.contract.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    renderQrToCanvas: vi.fn(),
  }))

  vi.mock('../composables/useQrCodeCanvas.js', () => ({
    renderQrToCanvas: (...args) => mocks.renderQrToCanvas(...args),
  }))

  const { default: PayOrderModal } = await import('./PayOrderModal.vue')

  const ORDER = { id: '1', order_number: 'ORD-1', total_yuan: '3.00' }

  describe('PayOrderModal QR', () => {
    beforeEach(() => {
      mocks.renderQrToCanvas.mockReset()
      mocks.renderQrToCanvas.mockResolvedValue({ success: true, method: 'qrcode' })
    })

    it('挂载后把 code_url 绘到 canvas（避免弹窗打开时 canvas 尚未入 DOM）', async () => {
      const wrapper = mount(PayOrderModal, {
        props: {
          order: ORDER,
          codeUrl: 'weixin://wxpay/bizpayurl?pr=WXTEST',
          mode: 'live',
        },
        global: { stubs: { PhoneVerificationGate: true } },
      })
      await flushPromises()
      expect(mocks.renderQrToCanvas).toHaveBeenCalled()
      const [canvas, text] = mocks.renderQrToCanvas.mock.calls[0]
      expect(canvas).toBeTruthy()
      expect(canvas.tagName).toBe('CANVAS')
      expect(text).toBe('weixin://wxpay/bizpayurl?pr=WXTEST')
      expect(wrapper.text()).toContain('请使用微信扫描二维码完成支付')
      wrapper.unmount()
    })

    it('不展示模拟支付按钮与测试模式提示', async () => {
      const wrapper = mount(PayOrderModal, {
        props: {
          order: ORDER,
          codeUrl: 'weixin://wxpay/bizpayurl?pr=WXTEST',
          mode: 'mock',
        },
        global: { stubs: { PhoneVerificationGate: true } },
      })
      await flushPromises()
      expect(wrapper.text()).not.toContain('当前为测试模式')
      expect(wrapper.text()).not.toContain('模拟支付成功')
      expect(wrapper.find('[data-testid="wechat-mock-complete"]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}
