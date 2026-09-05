// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useWechatRechargePoll.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref, nextTick } = await import('vue')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
  getApiUrl: (p) => `http://test${p}`,
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: hoisted.apiFetch,
}))

vi.mock('../utils/config.js', () => ({
  getApiUrl: hoisted.getApiUrl,
}))

const { useWechatRechargePoll } = await import('./useWechatRechargePoll.js')

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
}

describe('useWechatRechargePoll SSE', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    global.EventSource = FakeEventSource
    hoisted.apiFetch.mockReset()
  })

  afterEach(() => {
    delete global.EventSource
  })

  it('opens EventSource on modal open and marks success on matching recharge_completed', async () => {
    const router = { push: vi.fn() }
    const route = { params: { tenant: 't1' } }
    const {
      openWechatModalFromCreate,
      wechatModalVisible,
      closeWechatModal
    } = useWechatRechargePoll({
      tenantId: ref('t1'),
      router,
      route,
      finalAmount: ref(10),
      expectedYuanCents: ref(1000)
    })
    const rechargeSuccess = ref(false)
    const lastRechargeSummary = ref(null)
    openWechatModalFromCreate(
      { code_url: 'weixin://x', out_trade_no: 'WX99', mode: 'mock' },
      { rechargeSuccess, lastRechargeSummary }
    )
    expect(wechatModalVisible.value).toBe(true)
    expect(FakeEventSource.instances).toHaveLength(1)
    expect(FakeEventSource.instances[0].url).toContain('/api/sse/recharge-events/tenant_id/t1')

    FakeEventSource.instances[0].onmessage({
      data: JSON.stringify({
        event_name: 'recharge_completed',
        status: 'completed',
        out_trade_no: 'WX99',
        recharge_cents: 1000,
        recharge_yuan: '10'
      })
    })
    await nextTick()
    expect(rechargeSuccess.value).toBe(true)
    expect(wechatModalVisible.value).toBe(false)
    closeWechatModal()
  })
})

}
