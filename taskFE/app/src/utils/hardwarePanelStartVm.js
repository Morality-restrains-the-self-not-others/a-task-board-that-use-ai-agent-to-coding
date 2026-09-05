import { apiFetch } from './apiUtils.js'
import { queryClientPublicIpForAutoSg } from './publicClientIp.js'
import { resolveHardwareCoresMemoryForRunTemplate } from './projectRunTemplateUtils.js'
import {
  applyStartVmHttpResult,
  buildStartVmErrorStatusUpdate,
} from './startVmHttpResult.js'
import { buildTaskStartVmRequestFromRunTemplate } from './taskStartVmRequest.js'

/**
 * 若请求将自动创建安全组，创建时强制查询并写入 client_public_ip。
 * @param {Record<string, any>} body
 * @returns {Promise<Record<string, any>>}
 */
export async function attachClientPublicIpForAutoSg(body) {
  const next = body && typeof body === 'object' ? { ...body } : {}
  if (!next.auto_create_security_group) {
    return next
  }
  const ip = await queryClientPublicIpForAutoSg()
  if (ip) {
    next.client_public_ip = ip
  }
  return next
}

/**
 * 项目模版路径：构建请求并在需要时附带创建时查询的公网 IP。
 */
export async function buildTemplateStartVmRequestWithClientIp(opts) {
  const built = buildTaskStartVmRequestFromRunTemplate(opts)
  if (!built) return null
  const body = await attachClientPublicIpForAutoSg(built.body)
  return { apiPath: built.apiPath, body }
}

/**
 * POST start-vm / start-vm-auto，统一错误/成功处理。
 * @returns {Promise<'accepted'|'error'|null>}
 */
export async function postHardwarePanelStartVm({
  tenantId,
  workspaceId,
  apiPath,
  body,
  updateServerStatus,
  onAccepted,
}) {
  const response = await apiFetch(
    `/api/cloud/compute/${apiPath}/tenant_id/${tenantId}/workspace_id/${workspaceId}`,
    {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const result = await response.json().catch(() => ({}))
  // 确保启动受理结果带上请求级 traceId（apiFetch 通常注入 _traceId；再兜底 response.traceId）
  if (result && typeof result === 'object' && !Array.isArray(result)) {
    const tid =
      (typeof result._traceId === 'string' && result._traceId.trim()) ||
      (typeof response.traceId === 'string' && response.traceId.trim()) ||
      ''
    if (tid && !result._traceId) result._traceId = tid
  }
  if (!response.ok) {
    console.error('启动服务器失败:', result.message)
    if (updateServerStatus) {
      updateServerStatus(buildStartVmErrorStatusUpdate(result, '启动服务器失败'))
    }
    return 'error'
  }
  const outcome = applyStartVmHttpResult(result, updateServerStatus)
  if (outcome === 'accepted' && typeof onAccepted === 'function') {
    onAccepted(result)
  }
  return outcome
}

/**
 * 手动硬件配置路径：组装 start-vm 请求体（含自动 SG 时的 client_public_ip）。
 */
export async function buildManualHardwareStartVmBody(ctx) {
  const {
    taskId,
    containerImageId,
    hardwareConfig,
    filterOptions,
    selectedInstance,
    selectedBandwidth,
    defaultBandwidth,
    regionId,
    zoneId,
    vpcId,
    vswitchId,
    autoCreateVswitch,
    cloudPlatformId,
    authorizationId,
    securityGroupId,
    autoCreateSecurityGroup,
    bandwidth,
    bandwidthChargingMode,
    autoReleaseEnabled,
    autoReleaseMinutes,
  } = ctx

  // 实例规格是 CPU/内存权威源；禁止沿用 hardwareConfig 里未同步的占位 1核1GB
  const resolvedCoresMemory = resolveHardwareCoresMemoryForRunTemplate({
    hardwareConfig,
    filterOptions,
    selectedInstance,
  })
  const body = {
    task_id: taskId,
    container_image_id: containerImageId,
    hardware_config: {
      ...hardwareConfig,
      cpu_cores: Number(resolvedCoresMemory.cpu_cores),
      memory_gb: Number(resolvedCoresMemory.memory_gb),
      instance_type: selectedInstance?.instance_type,
      spot_strategy: filterOptions?.spot_strategy,
      spot_duration: filterOptions?.spot_duration ?? 0,
      instance_charge_type: filterOptions?.instance_charge_type || 'PostPaid',
      system_disk_category: filterOptions?.system_disk_category,
      internet_max_bandwidth_out: selectedInstance
        ? (selectedBandwidth?.[selectedInstance.instance_type] ?? defaultBandwidth)
        : defaultBandwidth,
    },
    region_id: regionId,
    zone_id: zoneId,
    vpc_id: vpcId,
    vswitch_id: vswitchId,
    auto_create_vswitch: autoCreateVswitch,
    cloud_platform_id: cloudPlatformId,
    authorization_id: authorizationId,
    filter_options: filterOptions,
    selected_instance: selectedInstance?.instance_type || null,
    security_group_id: securityGroupId,
    auto_create_security_group: autoCreateSecurityGroup,
    bandwidth,
    bandwidth_charging_mode: bandwidthChargingMode,
    auto_release_enabled: autoReleaseEnabled,
    auto_release_minutes: autoReleaseEnabled ? Math.floor(Number(autoReleaseMinutes)) : null,
    runtime_source: 'cloud_vm_manual',
  }
  return attachClientPublicIpForAutoSg(body)
}
