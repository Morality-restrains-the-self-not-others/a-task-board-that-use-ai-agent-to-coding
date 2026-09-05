// @vitest-environment node
/**
 * 支付渠道 SDK dump 不得原样展示；NOT_ENOUGH 抽成可读中文。
 */
if (!process.env.VITEST) {
  console.log('[skip] humanizePaymentProviderError.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { humanizePaymentProviderError, looksLikePaymentProviderDump } = await import(
    './humanizePaymentProviderError.js'
  )

  const WECHAT_DUMP = `error http response:[StatusCode: 403 Code: "NOT_ENOUGH"
Message: 基本账户余额不足，请充值后重新发起
Header:
 - Wechatpay-Serial=[PUB_KEY_ID_0116947288792026071402191500000000]
 - Wechatpay-Signature=[NVkk5cJ3rdfE9Yl2OJ0s1RpF0cRBqqXxO8y6SrL3ESVtkFULLrg68e68jWJwDLMykeIhCbCqIgTXubZgfG+lGzRLCvgWosXUiNUl6mziU4LQ4G56qHwgfcBKVOt20CbFwMVaDHkgxVX3T+YRecFpApL/RtCbObtUn618D2oCOybQSO/ZmDRBzfz4bro9BgXXYlqIKD5S1fPvTFKVxahKvOy9pG2imaTYy05LVXGYNd/bvcQEHaRJFW2rwkf+DlssF2SpdNIYa+4ke8+XXPrbrbp+uvqlVd63+dHr0paBlikuzF25ZO9CSHGmCgrJ9+s5cwuZ88vDBLZQC4R+eQSAGw==]
]`

  describe('humanizePaymentProviderError', () => {
    it('识别微信 HTTP dump', () => {
      expect(looksLikePaymentProviderDump(WECHAT_DUMP)).toBe(true)
      expect(looksLikePaymentProviderDump('frozen balance insufficient')).toBe(false)
    })

    it('NOT_ENOUGH dump 抽成余额不足文案且不含签名', () => {
      const out = humanizePaymentProviderError(WECHAT_DUMP)
      expect(out).toContain('微信退款失败')
      expect(out).toContain('余额不足')
      expect(out).not.toContain('Wechatpay-Signature')
      expect(out).not.toContain('NVkk5cJ3rdfE')
    })

    it('普通业务错误原样返回', () => {
      expect(humanizePaymentProviderError('refund application is not pending')).toBe(
        'refund application is not pending',
      )
    })

    it('SDK dump 格式（error http response + StatusCode）同样净化', () => {
      const sdkDump = `error http response:[StatusCode: 403 Code: "NOT_ENOUGH"
Message: 基本账户余额不足，请充值后重新发起
Header:
 - Wechatpay-Signature=[sig-secret]
]`
      const out = humanizePaymentProviderError(sdkDump)
      expect(out).toContain('微信退款失败')
      expect(out).toContain('余额不足')
      expect(out).not.toContain('Wechatpay-Signature')
      expect(out).not.toContain('sig-secret')
    })
  })
}
