// @vitest-environment node
/**
 * 失败 Response 必须读 _errorData 的 message/error/detail，不能只拼 HTTP 403。
 */
if (!process.env.VITEST) {
  console.log('[skip] httpError.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { errorFromFailedResponse, messageFromFailedResponse, isTransientHttpStatus } = await import('./httpError.js')

  function failedResp(status, extra = {}) {
    return {
      ok: false,
      status,
      traceId: extra.traceId || '',
      _errorData: extra._errorData,
    }
  }

  describe('messageFromFailedResponse', () => {
    it('reads message when JSON has no detail (APISIX/Cloud 403)', () => {
      const msg = messageFromFailedResponse(
        failedResp(403, { _errorData: { message: 'consumer not found' } }),
        'HTTP 403',
      )
      expect(msg).toBe('consumer not found')
      expect(msg).not.toBe('HTTP 403')
    })

    it('reads error when detail is missing', () => {
      expect(
        messageFromFailedResponse(failedResp(403, { _errorData: { error: 'forbidden' } }), 'HTTP 403'),
      ).toBe('forbidden')
    })

    it('reads detail when present', () => {
      expect(
        messageFromFailedResponse(
          failedResp(403, { _errorData: { detail: 'forbidden scope' } }),
          'HTTP 403',
        ),
      ).toBe('forbidden scope')
    })

    it('strips HTML 403 page instead of leaving HTTP 403 after json() consumes body', () => {
      const html = '<html><head><title>403 Forbidden</title></head><body><h1>403 Forbidden</h1></body></html>'
      const msg = messageFromFailedResponse(
        failedResp(403, { _errorData: { _rawErrorText: html } }),
        'HTTP 403',
      )
      expect(msg.toLowerCase()).toContain('403')
      expect(msg).not.toMatch(/^HTTP 403$/)
    })

    it('falls back to HTTP status when body is empty', () => {
      expect(messageFromFailedResponse(failedResp(403, { _errorData: {} }), 'HTTP 403')).toBe('HTTP 403')
    })

    it('isTransientHttpStatus covers gateway 502/503/504 only', () => {
      expect(isTransientHttpStatus(502)).toBe(true)
      expect(isTransientHttpStatus(503)).toBe(true)
      expect(isTransientHttpStatus(504)).toBe(true)
      expect(isTransientHttpStatus(403)).toBe(false)
      expect(isTransientHttpStatus(200)).toBe(false)
    })

    it('maps APISIX 502 HTML to a retryable Chinese message instead of dumping nginx HTML', () => {
      const html =
        '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>'
      const msg = messageFromFailedResponse(
        failedResp(502, { _errorData: { _rawErrorText: html } }),
        '加载支付记录失败',
      )
      expect(msg).toBe('服务暂时不可用，请稍后重试')
      expect(msg).not.toBe('加载支付记录失败')
      expect(msg.toLowerCase()).not.toContain('openresty')
    })

    it('keeps JSON error body on 502 when upstream returned application/json', () => {
      expect(
        messageFromFailedResponse(
          failedResp(502, { _errorData: { error: 'upstream boom' } }),
          '加载支付记录失败',
        ),
      ).toBe('upstream boom')
    })
  })

  describe('errorFromFailedResponse', () => {
    it('attaches traceId from response', () => {
      const err = errorFromFailedResponse(
        failedResp(403, {
          traceId: 'web-403-trace',
          _errorData: { message: 'nope' },
        }),
        'HTTP 403',
      )
      expect(err.message).toBe('nope')
      expect(err.traceId).toBe('web-403-trace')
    })
  })
}
