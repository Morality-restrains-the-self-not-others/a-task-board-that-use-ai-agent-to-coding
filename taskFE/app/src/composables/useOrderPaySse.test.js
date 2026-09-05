// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useOrderPaySse.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

const hoisted = vi.hoisted(() => ({
  getApiUrl: (p) => `http://test${p}`,
}))

vi.mock('../utils/config.js', () => ({
  getApiUrl: hoisted.getApiUrl,
}))

// useOrderPaySse 依赖 useBillingRechargeSse 的 openBillingRechargeSse —— 直接用真实现
const { useOrderPaySse } = await import('./useOrderPaySse.js')

class FakeEventSource {
  static instances = []
  constructor(url, opts) {
    this.url = url
    this.opts = opts
    this.onmessage = null
    this.onerror = null
    this.closed = false
    FakeEventSource.instances.push(this)
  }
  close() {
    this.closed = true
  }
  // 模拟服务端推送一条 SSE 消息
  emit(data) {
    this.onmessage?.({ data: JSON.stringify(data) })
  }
}

describe('useOrderPaySse', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    globalThis.EventSource = FakeEventSource
  })
  afterEach(() => {
    delete globalThis.EventSource
  })

  it('订阅 recharge-events 并按 order_id 匹配 order_paid', () => {
    const onPaid = vi.fn()
    const { startOrderSse } = useOrderPaySse({
      tenantId: () => 'tid-1',
      getOrderId: () => 'order-42',
      isActive: () => true,
      onPaid,
    })
    startOrderSse()

    expect(FakeEventSource.instances).toHaveLength(1)
    expect(FakeEventSource.instances[0].url).toBe('http://test/api/sse/recharge-events/tenant_id/tid-1')

    const es = FakeEventSource.instances[0]
    // 不匹配的 order_id：不回调
    es.emit({ event_name: 'order_paid', status: 'completed', order_id: 'order-99' })
    expect(onPaid).not.toHaveBeenCalled()
    // 匹配的 order_id：回调
    es.emit({ event_name: 'order_paid', status: 'completed', order_id: 'order-42' })
    expect(onPaid).toHaveBeenCalledWith('order-42')
  })

  it('重复 start 只保留一条连接（单链）', () => {
    const { startOrderSse } = useOrderPaySse({
      tenantId: 'tid-1',
      getOrderId: () => 'order-1',
      isActive: () => true,
      onPaid: vi.fn(),
    })
    startOrderSse()
    startOrderSse()
    expect(FakeEventSource.instances).toHaveLength(2)
    expect(FakeEventSource.instances[0].closed).toBe(true)
    expect(FakeEventSource.instances[1].closed).toBe(false)
  })

  it('isActive 为 false 时不回调（弹窗关闭即忽略旧单事件）', () => {
    let active = true
    const onPaid = vi.fn()
    const { startOrderSse } = useOrderPaySse({
      tenantId: 'tid-1',
      getOrderId: () => 'order-7',
      isActive: () => active,
      onPaid,
    })
    startOrderSse()
    active = false
    FakeEventSource.instances[0].emit({ event_name: 'order_paid', status: 'completed', order_id: 'order-7' })
    expect(onPaid).not.toHaveBeenCalled()
  })

  it('stopOrderSse 关闭连接（组件卸载即断）', () => {
    const { startOrderSse, stopOrderSse } = useOrderPaySse({
      tenantId: 'tid-1',
      getOrderId: () => 'order-1',
      isActive: () => true,
      onPaid: vi.fn(),
    })
    startOrderSse()
    stopOrderSse()
    expect(FakeEventSource.instances[0].closed).toBe(true)
  })

  it('无 order id / tenant 时不建连', () => {
    const { startOrderSse } = useOrderPaySse({
      tenantId: 'tid-1',
      getOrderId: () => '',
      isActive: () => true,
      onPaid: vi.fn(),
    })
    startOrderSse()
    expect(FakeEventSource.instances).toHaveLength(0)
  })
})
}
