/**
 * Detect cloud-vendor VPC / intranet container registries.
 * Host rules live in shareLib/registryhost (registryhost.go). Both this helper and
 * the Go package are driven by the shared case table
 * shareLib/registryhost/testdata/registry_cases.json to prevent drift.
 */

export const PRIVATE_REGISTRY_HINT =
  '该地址为云厂商内网/VPC 私有地址，本平台无法触及。请改用公网 Registry 地址。'

const PRIVATE_DNS_LABELS = new Set([
  'vpc',
  'vpce',
  'internal',
  'intranet',
  'inner',
  'private',
])

export function hostnameOfRegistry(raw) {
  let host = String(raw || '').trim().toLowerCase()
  host = host.replace(/^https?:\/\//, '')
  const slash = host.search(/[/\\]/)
  if (slash >= 0) host = host.slice(0, slash)
  if (host.startsWith('[')) {
    const end = host.indexOf(']')
    if (end > 0) return host.slice(1, end)
  }
  if (/^\d{1,3}(?:\.\d{1,3}){3}:\d+$/.test(host)) {
    return host.split(':')[0]
  }
  const colon = host.lastIndexOf(':')
  if (colon > 0 && host.indexOf(':') === colon && /^\d+$/.test(host.slice(colon + 1))) {
    return host.slice(0, colon)
  }
  return host
}

function isPrivateIPv4(host) {
  const m = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(host)
  if (!m) return false
  const a = Number(m[1])
  const b = Number(m[2])
  const c = Number(m[3])
  const d = Number(m[4])
  if ([a, b, c, d].some((n) => n > 255)) return false
  if (a === 10) return true
  if (a === 192 && b === 168) return true
  if (a === 172 && b >= 16 && b <= 31) return true
  return false
}

export function isPrivateRegistryHost(rawHost) {
  const host = hostnameOfRegistry(rawHost)
  if (!host) return false
  if (isPrivateIPv4(host)) return true
  for (const lab of host.split('.')) {
    if (PRIVATE_DNS_LABELS.has(lab)) return true
    if (lab.startsWith('registry-vpc') || lab.startsWith('registry-internal')) return true
  }
  return false
}

export function isCloudVendorPrivateRegistryText(text) {
  const s = String(text || '')
  if (!s.trim()) return false
  if (isPrivateRegistryHost(s)) return true
  const urlRe = /https?:\/\/([^/\s"']+)/gi
  let m
  while ((m = urlRe.exec(s))) {
    if (isPrivateRegistryHost(m[1])) return true
  }
  const firstToken = s.trim().split(/[\s]/)[0]
  if (firstToken && isPrivateRegistryHost(firstToken)) return true
  return false
}

export function suggestPublicAliyunAcr(raw) {
  const host = hostnameOfRegistry(raw)
  if (!host.endsWith('.aliyuncs.com')) return ''
  if (host.startsWith('registry-vpc.')) {
    return `registry.${host.slice('registry-vpc.'.length)}`
  }
  if (host.startsWith('registry-internal.')) {
    return `registry.${host.slice('registry-internal.'.length)}`
  }
  return ''
}

function requestImageURL(requestBody) {
  if (!requestBody) return ''
  if (typeof requestBody === 'string') {
    try {
      return String(JSON.parse(requestBody).image_url || '')
    } catch {
      return ''
    }
  }
  if (typeof requestBody === 'object') {
    return String(requestBody.image_url || '')
  }
  return ''
}

export function enrichResolveArchitectureError(message, requestBody) {
  const text = message instanceof Error ? message.message : String(message ?? '')
  const imageURL = requestImageURL(requestBody)
  const haystack = `${text} ${imageURL}`
  if (!isCloudVendorPrivateRegistryText(haystack)) {
    return text
  }
  if (text.includes('无法触及')) {
    return text
  }
  let hint = PRIVATE_REGISTRY_HINT
  const vpcHost =
    (imageURL.match(/registry-vpc\.[^\s/"']+/i) ||
      text.match(/registry-vpc\.[^\s/"']+/i) ||
      [])[0] || imageURL
  const pub = suggestPublicAliyunAcr(vpcHost) || suggestPublicAliyunAcr(imageURL)
  if (pub) {
    const privateHost = hostnameOfRegistry(vpcHost)
    hint += `（可将 ${privateHost} 改为 ${pub}）`
  }
  if (!text.trim()) return hint
  return `${text} ${hint}`
}
