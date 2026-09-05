// @vitest-environment node
/**
 * 契约：网关 502 HTML（APISIX 上游 connection-refused）不得作为 API 成功体解析。
 * OPT-20260825-018 后网关对 502/503/504 输出 JSON；本测试锁定 safeResponseJson
 * 在 HTML 502 上返回 error（data=fallback）而非抛错或把 HTML 当数据。
 */
if (!process.env.VITEST) {
  console.log('[skip] safeResponseJson.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { safeResponseJson } = await import('./safeResponseJson.js')

  function htmlResponse(status, html, traceId) {
    const response = { ok: status < 400, status, headers: {} }
    // 模拟 fetch Response.json()：HTML body 解析即抛错
    response.json = () => Promise.reject(new SyntaxError('Unexpected token < in JSON'))
    if (traceId) response.traceId = traceId
    response.text = async () => html
    return response
  }

  function jsonResponse(status, body, traceId) {
    const response = { ok: status < 400, status, headers: {} }
    response.json = () => Promise.resolve(body)
    if (traceId) response.traceId = traceId
    return response
  }

  describe('safeResponseJson — HTML 502 契约', () => {
    it('HTML 502 返回 error 且 data=fallback，不把 HTML 当成功体', async () => {
      const html = '<html><head><title>502 Bad Gateway</title></head><body><h1>502</h1></body></html>'
      const { data, error, traceId } = await safeResponseJson(
        htmlResponse(502, html, 'web-502-trace'),
        { fallback: { items: [] } },
      )
      expect(data).toEqual({ items: [] })
      expect(error).toBeTruthy()
      expect(traceId).toBe('web-502-trace')
    })

    it('网关 JSON 502（error + trace_id）作为数据返回，不抛错', async () => {
      const body = { error: '服务暂时不可用，请稍后重试', trace_id: 'gw-502-trace' }
      const { data, error } = await safeResponseJson(jsonResponse(502, body, 'web-502-trace'))
      expect(data).toEqual(body)
      expect(error).toBe('')
    })

    it('200 HTML（异常上游）同样返回 error 而非脏数据', async () => {
      const { data, error } = await safeResponseJson(
        htmlResponse(200, '<html>not json</html>'),
        { fallback: null },
      )
      expect(data).toBeNull()
      expect(error).toBeTruthy()
    })

    it('无 json() 的对象返回 error', async () => {
      const { data, error } = await safeResponseJson({}, { fallback: 0 })
      expect(data).toBe(0)
      expect(error).toBeTruthy()
    })
  })
}
