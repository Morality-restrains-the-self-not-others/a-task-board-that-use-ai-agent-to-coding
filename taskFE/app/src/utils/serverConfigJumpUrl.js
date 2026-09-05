import { isLoopbackHttpUrl } from './httpUrlHost.js'

/** 跳转链接默认使用的 Web 端口（与下方说明文案一致，修改时请同步） */
export const serverJumpDefaultPort = 8080

/** code-server（VS Code Web）常见宿主机映射端口，与 reachability.mjs / TRAE_HOST_VSCODE_PORT 默认一致 */
export const serverVscodeDefaultPort = 8888

/** 解析 http(s) 或裸 host 的 hostname（不含端口） */
export function parseHttpHostname(value) {
  if (!value) {
    return ''
  }
  const text = String(value).trim()
  if (!text) {
    return ''
  }
  try {
    const withScheme = text.startsWith('http://') || text.startsWith('https://') ? text : `http://${text}`
    const u = new URL(withScheme)
    return u.hostname || ''
  } catch {
    return ''
  }
}

export function isLikelyIpv4(host) {
  return /^\d{1,3}(?:\.\d{1,3}){3}$/.test(host)
}

/**
 * 任务页「跳转到服务器」打开实例 Web 服务（默认端口见 serverJumpDefaultPort）。
 * 优先使用运行状态接口返回的公网 IP，否则从 SSE/后端的 server_url 解析主机名。
 */
export function buildJumpToServerUrl(serverUrl, publicIp, port) {
  const ipTrim = String(publicIp || '').trim()
  if (ipTrim) {
    const fromIp =
      ipTrim.includes(':') && !isLikelyIpv4(ipTrim)
        ? `http://[${ipTrim}]:${port}`
        : `http://${ipTrim}:${port}`
    if (isLoopbackHttpUrl(fromIp)) {
      return ''
    }
    return fromIp
  }
  const text = String(serverUrl || '').trim()
  if (!text) {
    return ''
  }
  const host = parseHttpHostname(text)
  if (!host) {
    return ''
  }
  if (isLoopbackHttpUrl(host.includes(':') && !isLikelyIpv4(host) ? `http://[${host}]/` : `http://${host}/`)) {
    return ''
  }
  let candidate = ''
  if (host.includes('aliyuncs.com')) {
    candidate = text.startsWith('http://') || text.startsWith('https://') ? text : `https://${text}`
  } else if (isLikelyIpv4(host)) {
    candidate = `http://${host}:${port}`
  } else if (host.includes(':')) {
    candidate = `http://[${host}]:${port}`
  } else {
    candidate = `http://${host}:${port}`
  }
  if (isLoopbackHttpUrl(candidate)) {
    return ''
  }
  return candidate
}
