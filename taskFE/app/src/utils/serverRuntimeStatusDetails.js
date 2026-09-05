/** 展示接口/会话时间戳，并附本地时间（与接口 UTC 对照） */
export function formatRuntimeTimestampDisplay(timeRaw) {
  if (!timeRaw) {
    return ''
  }
  const raw = String(timeRaw).trim()
  if (!raw) {
    return ''
  }
  const parsed = Date.parse(raw)
  if (Number.isNaN(parsed)) {
    return raw
  }
  const local = new Date(parsed).toLocaleString('zh-CN', {
    hour12: false,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
  return `${raw}（本地：${local}）`
}

/** 自起始时刻至 nowMs 的运行时长文案（省略为 0 的中间单位） */
export function formatUptimeSinceMs(startedMs, nowMs) {
  const diff = nowMs - startedMs
  if (diff < 0) {
    return '无法计算（启动时间晚于当前时间）'
  }
  const totalSec = Math.floor(diff / 1000)
  const days = Math.floor(totalSec / 86400)
  const hours = Math.floor((totalSec % 86400) / 3600)
  const minutes = Math.floor((totalSec % 3600) / 60)
  const seconds = totalSec % 60
  const parts = []
  if (days > 0) {
    parts.push(`${days} 天`)
  }
  if (hours > 0) {
    parts.push(`${hours} 小时`)
  }
  if (minutes > 0) {
    parts.push(`${minutes} 分钟`)
  }
  if (parts.length === 0) {
    parts.push(`${seconds} 秒`)
  } else if (days === 0 && hours === 0 && seconds > 0) {
    parts.push(`${seconds} 秒`)
  }
  return parts.join(' ')
}

/** 评论 created_at 到云主机 CreationTime 的间隔；不足 1 分钟视为同时刻不展示。 */
export function formatCommentToCreationLag(commentCreatedAt, creationRaw) {
  const from = Date.parse(String(commentCreatedAt || '').trim())
  const to = Date.parse(String(creationRaw || '').trim())
  if (Number.isNaN(from) || Number.isNaN(to)) {
    return ''
  }
  const diff = to - from
  if (diff < 60 * 1000) {
    return ''
  }
  return formatUptimeSinceMs(from, to)
}

export function formatRuntimeStatusValue(value) {
  if (value === null || value === undefined || value === '') {
    return '-'
  }
  if (Array.isArray(value)) {
    return value.length > 0 ? value.join(', ') : '-'
  }
  if (typeof value === 'object') {
    return JSON.stringify(value)
  }
  return String(value)
}

const INTERNET_CHARGE_TYPE_LABELS = Object.freeze({
  PayByTraffic: '按流量计费',
  PayByBandwidth: '按带宽计费',
  PayBy95: '按 95 计费',
})

function firstDefined(...values) {
  for (const value of values) {
    if (value !== null && value !== undefined && value !== '') {
      return value
    }
  }
  return undefined
}

function firstPositiveBandwidth(...values) {
  let zeroFallback
  for (const value of values) {
    if (value === null || value === undefined || value === '') {
      continue
    }
    const n = Number(value)
    if (Number.isFinite(n) && n > 0) {
      return value
    }
    if (zeroFallback === undefined) {
      zeroFallback = value
    }
  }
  return zeroFallback
}

export function formatInternetChargeType(value) {
  const raw = String(value ?? '').trim()
  if (!raw) {
    return '-'
  }
  return INTERNET_CHARGE_TYPE_LABELS[raw] || raw
}

export function formatBandwidthMbps(value) {
  if (value === null || value === undefined || value === '') {
    return '-'
  }
  const n = Number(value)
  if (!Number.isFinite(n)) {
    return formatRuntimeStatusValue(value)
  }
  return `${n} Mbps`
}

function resolveInternetBandwidthFields(instanceBody) {
  const eip = instanceBody?.EipAddress && typeof instanceBody.EipAddress === 'object'
    ? instanceBody.EipAddress
    : {}
  const instanceOut = Number(instanceBody?.InternetMaxBandwidthOut)
  const eipOut = Number(eip.Bandwidth)
  const instanceHasOut = Number.isFinite(instanceOut) && instanceOut > 0
  const eipHasOut = Number.isFinite(eipOut) && eipOut > 0
  if (!instanceHasOut && eipHasOut) {
    return {
      chargeType: firstDefined(eip.InternetChargeType, instanceBody?.InternetChargeType),
      out: eip.Bandwidth,
      inn: firstPositiveBandwidth(instanceBody?.InternetMaxBandwidthIn),
    }
  }
  return {
    chargeType: firstDefined(instanceBody?.InternetChargeType, eip.InternetChargeType),
    out: firstPositiveBandwidth(instanceBody?.InternetMaxBandwidthOut, eip.Bandwidth),
    inn: firstPositiveBandwidth(instanceBody?.InternetMaxBandwidthIn),
  }
}

/**
 * 构建「服务器运行状态」实例详情行。
 * @param {object|null} data server-runtime-status 响应体
 * @param {number} tick 用于驱动已运行时长刷新
 * @param {string} runningContainerImageDisplay 容器镜像展示文案
 * @param {string} commentCreatedAt 评论 created_at，用于对照云主机 CreationTime
 */
export function buildServerRuntimeStatusDetails(data, tick, runningContainerImageDisplay = '', commentCreatedAt = '') {
  if (!data) {
    return []
  }
  const instanceBody = data.instance_attribute?.body || {}
  const publicIps = instanceBody.PublicIpAddress?.IpAddress || []
  const privateIps = instanceBody.VpcAttributes?.PrivateIpAddress?.IpAddress || []
  const securityGroupIds = instanceBody.SecurityGroupIds?.SecurityGroupId || []
  const creationRaw = instanceBody.CreationTime
  const creationLag = formatCommentToCreationLag(commentCreatedAt, creationRaw)
  let creationDisplay = formatRuntimeTimestampDisplay(creationRaw)
  if (creationDisplay && creationLag) {
    creationDisplay = `${creationDisplay}；距评论 ${creationLag}`
  }
  const serverStartRaw = data.server_started_at || ''
  const serverStartDisplay = formatRuntimeTimestampDisplay(serverStartRaw)
  const autoReleaseRaw = data.auto_release_time || instanceBody.AutoReleaseTime || ''
  const autoReleaseDisplay = formatRuntimeTimestampDisplay(autoReleaseRaw)
  const uptimeBaseRaw = serverStartRaw || creationRaw
  let uptimeDisplay = ''
  if (uptimeBaseRaw) {
    const started = Date.parse(String(uptimeBaseRaw))
    if (!Number.isNaN(started)) {
      uptimeDisplay = formatUptimeSinceMs(started, tick || Date.now())
    }
  }
  return [
    { label: '平台', value: formatRuntimeStatusValue(data.platform) },
    { label: '地域', value: formatRuntimeStatusValue(data.region) },
    { label: '可用区', value: formatRuntimeStatusValue(instanceBody.ZoneId) },
    { label: '实例 ID', value: formatRuntimeStatusValue(data.instance_id || instanceBody.InstanceId) },
    { label: '实例名', value: formatRuntimeStatusValue(instanceBody.InstanceName) },
    { label: '运行状态', value: formatRuntimeStatusValue(instanceBody.Status || data.runtime_status) },
    ...(serverStartDisplay ? [{ label: '服务器启动时间', value: serverStartDisplay }] : []),
    ...(autoReleaseDisplay ? [{ label: '服务器预计释放时间', value: autoReleaseDisplay }] : []),
    ...(creationDisplay ? [{ label: '创建时间', value: creationDisplay }] : []),
    ...(uptimeDisplay ? [{ label: '已运行时长', value: uptimeDisplay }] : []),
    { label: '实例规格', value: formatRuntimeStatusValue(instanceBody.InstanceType) },
    { label: 'CPU', value: formatRuntimeStatusValue(instanceBody.Cpu) },
    { label: '内存(MB)', value: formatRuntimeStatusValue(instanceBody.Memory) },
    ...internetBandwidthDetailRows(instanceBody),
    { label: '公网 IP', value: formatRuntimeStatusValue(publicIps) },
    { label: '私网 IP', value: formatRuntimeStatusValue(privateIps) },
    { label: 'VPC ID', value: formatRuntimeStatusValue(instanceBody.VpcAttributes?.VpcId) },
    { label: 'VSwitch ID', value: formatRuntimeStatusValue(instanceBody.VpcAttributes?.VSwitchId) },
    { label: '安全组', value: formatRuntimeStatusValue(securityGroupIds) },
    { label: '镜像 ID', value: formatRuntimeStatusValue(instanceBody.ImageId) },
    ...(runningContainerImageDisplay
      ? [{ label: '运行的容器镜像', value: runningContainerImageDisplay }]
      : []),
  ].filter((item) => item.value !== '-')
}

function internetBandwidthDetailRows(instanceBody) {
  const fields = resolveInternetBandwidthFields(instanceBody)
  return [
    { label: '带宽计费模式', value: formatInternetChargeType(fields.chargeType) },
    { label: '公网出带宽', value: formatBandwidthMbps(fields.out) },
    { label: '公网入带宽', value: formatBandwidthMbps(fields.inn) },
  ]
}

/** 用于 uptime 定时刷新的时间戳来源（优先平台会话启动时间） */
export function resolveRuntimeUptimeSource(data) {
  if (!data) {
    return ''
  }
  return data.server_started_at || data.instance_attribute?.body?.CreationTime || ''
}
