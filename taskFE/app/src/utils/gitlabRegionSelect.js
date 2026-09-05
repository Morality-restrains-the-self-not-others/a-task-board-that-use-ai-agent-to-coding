/**
 * Group sellable GitLab regions by cloud_provider for OrderCreate <optgroup>.
 * Aliyun pending_node rows stay selectable; the group label tells buyers
 * the node is provisioned manually.
 */

export function isGitlabRegionPendingNode(region) {
  return String(region?.infra_status || '').trim().toLowerCase() === 'pending_node'
}

function providerKey(region) {
  const raw = String(region?.cloud_provider || '').trim().toLowerCase()
  return raw || 'other'
}

export function gitlabProviderGroupLabel(provider, regions) {
  const p = String(provider || '').trim().toLowerCase() || 'other'
  if (p === 'tencent') return '腾讯云'
  if (p === 'aliyun') {
    const pending = Array.isArray(regions) && regions.some(isGitlabRegionPendingNode)
    return pending ? '阿里云（人工开通节点）' : '阿里云'
  }
  return p
}

const PROVIDER_ORDER = ['tencent', 'aliyun']

export function groupGitlabRegionsByProvider(regions) {
  const list = Array.isArray(regions) ? regions.filter((r) => r && r.slug) : []
  const buckets = new Map()
  for (const r of list) {
    const key = providerKey(r)
    if (!buckets.has(key)) buckets.set(key, [])
    buckets.get(key).push(r)
  }
  const groups = []
  for (const p of PROVIDER_ORDER) {
    if (!buckets.has(p)) continue
    const regs = buckets.get(p)
    groups.push({ provider: p, label: gitlabProviderGroupLabel(p, regs), regions: regs })
    buckets.delete(p)
  }
  for (const [p, regs] of buckets) {
    groups.push({ provider: p, label: gitlabProviderGroupLabel(p, regs), regions: regs })
  }
  return groups
}
