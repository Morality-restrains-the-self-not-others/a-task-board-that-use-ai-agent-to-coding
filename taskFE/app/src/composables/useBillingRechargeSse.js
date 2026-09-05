import { getCurrentInstance, onUnmounted } from 'vue'
import { getApiUrl } from '../utils/config.js'

/**
 * 打开支付完成 SSE（taskSSE：/api/sse/recharge-events/tenant_id/{tid}）。
 * 用户身份由网关 forward-auth 注入 X-User-Id；浏览器只带 Cookie（EventSource withCredentials）。
 * 禁止 HTTP 长轮询；关闭时须调用返回的 close()。
 *
 * @param {{
 *   tenantId: string,
 *   onCompleted: (data: object) => void,
 *   match?: (data: object) => boolean,
 * }} opts
 */
export function openBillingRechargeSse({ tenantId, onCompleted, match }) {
  const tid = String(tenantId || '').trim()
  if (!tid || typeof EventSource === 'undefined') {
    return { close: () => {} }
  }
  const url = getApiUrl(`/api/sse/recharge-events/tenant_id/${tid}`)
  const es = new EventSource(url, { withCredentials: true })
  let closed = false

  const close = () => {
    if (closed) return
    closed = true
    try {
      es.close()
    } catch {
      /* ignore */
    }
  }

  es.onmessage = (event) => {
    let data
    try {
      data = JSON.parse(event.data)
    } catch {
      return
    }
    if (!data || typeof data !== 'object') return
    if (data.type === 'heartbeat' || data.event_name === 'recharge_sse_heartbeat') return
    if (data.event_name === 'recharge_sse_connected') return
    if (data.event_name !== 'recharge_completed' && data.status !== 'completed') return
    if (typeof match === 'function' && !match(data)) return
    onCompleted?.(data)
  }

  es.onerror = () => {
    close()
  }

  if (getCurrentInstance()) {
    onUnmounted(close)
  }

  return { close }
}
