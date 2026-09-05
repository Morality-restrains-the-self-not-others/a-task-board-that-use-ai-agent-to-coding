import { resolveEnvParamsSourceRequiredHint } from './envParamsSourceSelection.js'
import { resolveProjectTemplateStartDisabledReason } from './taskStartVmRequest.js'
import { isServerRuntimeNotServingStatus } from './commentLayerZtreeUiState.js'

/**
 * 任务详情「启动服务器」按钮禁用原因（临时硬件配置模式）。
 * @param {object} ctx
 * @returns {string} 空字符串表示可点击
 */
export function resolveTemporaryHardwareStartDisabledReason(ctx = {}) {
  if (ctx.isServerStarting) return '服务器正在启动中，请等待完成'

  const runtimeStatus = String(ctx.serverRuntimeStatus || '').trim()
  // 云态已明确非服务时，不以可能陈旧的 isServerRunning 阻断重启
  const cloudNotServing = isServerRuntimeNotServingStatus(runtimeStatus)
  if (!cloudNotServing && (ctx.isServerRunning || ctx.isServerRuntimeRunning)) {
    return '服务器已在运行，如需重新启动请先停止'
  }

  if (runtimeStatus === 'Starting' || runtimeStatus === 'Pending' || runtimeStatus === 'Initializing') {
    return '云实例正在创建或初始化中，请稍后再试'
  }
  if (runtimeStatus === 'Stopping' || runtimeStatus === 'Rebooting') {
    return '云实例正在停止或重启中，请稍后再试'
  }

  if (!ctx.hasCloudContext) return '工作空间上下文未就绪，请刷新页面后重试'
  if (!ctx.selectedImageId) return '请先在评论区选择镜像'
  if (ctx.isLoadingPlatforms) return '正在加载云平台列表…'
  if (!ctx.selectedCloudPlatform) return '请先选择云平台'
  if (ctx.isLoadingRegions) return '正在加载地域列表…'
  if (!ctx.selectedRegion) return '请选择地域'
  if (!ctx.selectedVpc) return '请选择 VPC'
  if (!ctx.selectedZone) return '请选择可用区'
  if (ctx.isLoadingInstances) return '正在加载可用实例，请稍候'
  if (!ctx.selectedInstance) return '请从可用实例列表中选择一台实例'

  return ''
}

/**
 * 统一入口：按硬件配置来源解析禁用原因。
 * @param {object} ctx
 * @param {'project_template'|'temporary'} ctx.hardwareConfigSource
 */
export function resolveServerStartDisabledReason(ctx = {}) {
  if (ctx.requireEnvParamsSource) {
    const envHint = resolveEnvParamsSourceRequiredHint(ctx)
    if (envHint) return envHint
  }
  if (ctx.hardwareConfigSource === 'project_template') {
    return resolveProjectTemplateStartDisabledReason(ctx)
  }
  return resolveTemporaryHardwareStartDisabledReason(ctx)
}
