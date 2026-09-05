/**
 * 判断 URL 的 hostname 是否为环回或未指定地址。
 * 用于避免云实例尚未完成远程注册时，误把本机 127.0.0.1 当作「远程 VS Code」链接。
 * 无法解析的字符串视为环回（不在生产路径展示为远程链接）。
 *
 * @param {string} raw
 * @returns {boolean}
 */
export function isLoopbackHttpUrl(raw) {
  const text = String(raw || '').trim()
  if (!text) {
    return false
  }
  try {
    const withScheme =
      text.startsWith('http://') || text.startsWith('https://') ? text : `http://${text}`
    const u = new URL(withScheme)
    const host = (u.hostname || '').toLowerCase()
    if (!host) {
      return true
    }
    const h = host.startsWith('[') && host.endsWith(']') ? host.slice(1, -1) : host
    return h === 'localhost' || h === '127.0.0.1' || h === '::1' || h === '0.0.0.0'
  } catch {
    return true
  }
}

/**
 * 将 URL 的 hostname 替换为指定公网 IP（端口、路径、查询、hash 不变）。
 * 用于任务详情「打开服务器 VS Code」与 DescribeInstance 返回的公网 IP 一致。
 *
 * @param {string} rawUrl
 * @param {string} publicIp IPv4 文本或 IPv6 文本（无方括号）
 * @returns {string} 失败时返回空字符串
 */
export function replaceHttpUrlHostname(rawUrl, publicIp) {
  const ip = String(publicIp || '').trim()
  if (!ip) {
    return ''
  }
  const text = String(rawUrl || '').trim()
  if (!text) {
    return ''
  }
  try {
    const withScheme =
      text.startsWith('http://') || text.startsWith('https://') ? text : `http://${text}`
    const u = new URL(withScheme)
    u.hostname = ip
    return u.toString()
  } catch {
    return ''
  }
}
