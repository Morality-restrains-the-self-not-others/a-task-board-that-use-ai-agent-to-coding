export function bandwidthShareLabel(shared) {
  return shared ? '带宽共享分区' : '独立带宽'
}

export function bandwidthUsedMbps(region) {
  const total = Number(region?.total_bandwidth_mbps) || 0
  const remaining = Number(region?.remaining_bandwidth_mbps) || 0
  return Math.max(0, total - remaining)
}

export function remainingBandwidthMbps(region) {
  const remaining = Number(region?.remaining_bandwidth_mbps)
  if (!Number.isFinite(remaining)) return 0
  return Math.max(0, remaining)
}

export function bandwidthPercent(region) {
  const total = Number(region?.total_bandwidth_mbps) || 0
  if (!total) return 0
  return Math.min(100, Math.round((bandwidthUsedMbps(region) / total) * 100))
}
