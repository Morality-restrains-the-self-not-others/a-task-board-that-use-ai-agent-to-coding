// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useBillingRechargeSse.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

vi.mock('../utils/config.js', () => ({
  getApiUrl: (p) => `http://test${p}`,
}))

const { openBillingRechargeSse } = await import('./useBillingRechargeSse.js')

class FakeEventSource {
  static instances = []
  constructor(url, opts) {
    this.url = url
    this.opts = opts
    this.onmessage = null
    this.onerror = null
    FakeEventSource.instances.push(this)
  }
  close() {
    this.closed = true
  }
}

describe('useBillingRechargeSse', () => {
  beforeEach(() => {
    FakeEventSource.instances = []
    global.EventSource = FakeEventSource
  })

  afterEach(() => {
    delete global.EventSource
  })

  it('opens EventSource without query user_id (gateway injects X-User-Id)', () => {
    const onCompleted = vi.fn()
    const { close } = openBillingRechargeSse({
      tenantId: 't1',
      match: (d) => d.out_trade_no === 'WX1',
      onCompleted,
    })
    expect(FakeEventSource.instances).toHaveLength(1)
    const url = FakeEventSource.instances[0].url
    expect(url).toContain('/api/sse/recharge-events/tenant_id/t1')
    expect(url).not.toContain('user_id=')
    expect(FakeEventSource.instances[0].opts).toEqual({ withCredentials: true })
    FakeEventSource.instances[0].onmessage({
      data: JSON.stringify({
        event_name: 'recharge_completed',
        status: 'completed',
        out_trade_no: 'WX1',
      }),
    })
    expect(onCompleted).toHaveBeenCalledTimes(1)
    close()
  })
})

}
