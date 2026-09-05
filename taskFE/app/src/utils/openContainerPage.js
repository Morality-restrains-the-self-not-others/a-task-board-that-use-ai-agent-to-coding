import { apiFetch } from './apiUtils.js'
import { fetchPublicClientIp } from './publicClientIp.js'
import { appendCommentIdPath } from './containerForwardCommentId.js'

/**
 * Best-effort STUN public IP discovery (helps when edge client-ip is a proxy egress
 * but browser DIRECT to the container uses the ISP address).
 * @returns {Promise<string[]>}
 */
export async function discoverStunPublicIps({ timeoutMs = 2500 } = {}) {
  if (typeof RTCPeerConnection === 'undefined') return []
  const ips = new Set()
  const pc = new RTCPeerConnection({
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
  })
  try {
    pc.createDataChannel('sg')
    const done = new Promise((resolve) => {
      const timer = setTimeout(resolve, timeoutMs)
      pc.onicecandidate = (ev) => {
        const cand = ev?.candidate?.candidate || ''
        // candidate:… typ host/srflx … → extract IPv4/IPv6
        const m = cand.match(/ ([0-9a-fA-F:.]+) \d+ typ (srflx|relay)/)
        if (m && m[1]) ips.add(m[1])
        if (!ev.candidate) {
          clearTimeout(timer)
          resolve()
        }
      }
    })
    await pc.setLocalDescription(await pc.createOffer())
    await done
  } catch {
    /* ignore */
  } finally {
    try {
      pc.close()
    } catch {
      /* ignore */
    }
  }
  return [...ips]
}

/**
 * Ensure current browser public IP(s) are on the task VM security-group whitelist,
 * then open container_page_url in a new tab (direct to container; no SaaS reverse proxy).
 *
 * @param {{
 *   containerPageUrl: string,
 *   tenantId: string,
 *   workspaceId: string,
 *   taskId: string,
 *   commentId?: string,
 *   openFn?: (url: string) => void,
 *   settleMs?: number,
 * }} opts
 */
export async function openContainerPageWithIngressEnsure(opts) {
  const url = String(opts?.containerPageUrl || '').trim()
  const tenantId = String(opts?.tenantId || '').trim()
  const workspaceId = String(opts?.workspaceId || '').trim()
  const taskId = String(opts?.taskId || '').trim()
  const commentId = String(opts?.commentId || '').trim()
  const settleMs = Number.isFinite(opts?.settleMs) ? Math.max(0, Number(opts.settleMs)) : 800
  const openFn =
    typeof opts?.openFn === 'function'
      ? opts.openFn
      : (href) => {
          window.open(href, '_blank', 'noopener,noreferrer')
        }
  if (!url) {
    throw new Error('container page url empty')
  }
  if (tenantId && workspaceId && taskId) {
    const ips = []
    try {
      const edgeIp = await fetchPublicClientIp({ force: true })
      if (edgeIp) ips.push(edgeIp)
    } catch {
      /* edge IP optional; server may still parse XFF */
    }
    try {
      const stunIps = await discoverStunPublicIps()
      for (const ip of stunIps) {
        if (ip && !ips.includes(ip)) ips.push(ip)
      }
    } catch {
      /* stun optional */
    }
    const body = {}
    if (ips[0]) body.client_public_ip = ips[0]
    if (ips.length) body.client_public_ips = ips
    try {
      const base = `/api/cloud/compute/ensure-client-ingress/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}?task_id=${encodeURIComponent(taskId)}`
      // 评论级 CSC：comment_id 走 path（/comment_id/{cid}/），后端 commentIDFromComputeRequest 优先解析 path。
      await apiFetch(appendCommentIdPath(base, commentId), {
        method: 'POST',
        credentials: 'include',
        headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      // Aliyun SG rule propagation can lag a moment.
      if (settleMs > 0) {
        await new Promise((r) => setTimeout(r, settleMs))
      }
    } catch (e) {
      console.warn('[openContainerPage] ensure-client-ingress failed; opening anyway', e)
    }
  }
  openFn(url)
}
