import {
  buildAutoRunStartVmRequest,
  resolveRunTemplateInstanceType,
} from './projectRunTemplateUtils.js'
import { isServerRuntimeNotServingStatus } from './commentLayerZtreeUiState.js'

/**
 * 从项目 server_run_template 构建任务详情「启动服务器」请求（含自动释放与 filter_options）。
 * @returns {{ apiPath: string, body: object } | null}
 */
export function buildTaskStartVmRequestFromRunTemplate({
  taskId,
  containerImageId,
  runTemplate,
  autoReleaseEnabled = true,
  autoReleaseMinutes = 30,
} = {}) {
  const base = buildAutoRunStartVmRequest({ taskId, containerImageId, runTemplate })
  if (!base) return null

  const filterOptions =
    runTemplate?.filter_options && typeof runTemplate.filter_options === 'object'
      ? runTemplate.filter_options
      : {}
  const instanceType = resolveRunTemplateInstanceType(runTemplate)
  const bandwidth = Number(runTemplate?.bandwidth ?? 5)

  return {
    apiPath: base.apiPath,
    body: {
      ...base.body,
      hardware_config: {
        ...base.body.hardware_config,
        spot_strategy: filterOptions.spot_strategy,
        spot_duration: filterOptions.spot_duration ?? 0,
        instance_charge_type: filterOptions.instance_charge_type || 'PostPaid',
        system_disk_category: filterOptions.system_disk_category,
        internet_max_bandwidth_out: bandwidth,
      },
      filter_options: filterOptions,
      bandwidth,
      bandwidth_charging_mode: runTemplate?.bandwidth_charging_mode || 'PayByTraffic',
      selected_instance: instanceType || base.body.selected_instance || null,
      auto_release_enabled: Boolean(autoReleaseEnabled),
      auto_release_minutes: autoReleaseEnabled ? Math.floor(Number(autoReleaseMinutes)) : null,
      runtime_source: 'cloud_vm_template',
    },
  }
}

/**
 * 项目模版模式下的启动禁用原因。
 * @returns {string} 空字符串表示可点击
 */
export function resolveProjectTemplateStartDisabledReason(ctx = {}) {
  if (ctx.isServerStarting) return '服务器正在启动中，请等待完成'

  const runtimeStatus = String(ctx.serverRuntimeStatus || '').trim()
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

  const tpl = ctx.runTemplate
  if (!tpl || typeof tpl !== 'object' || !Object.keys(tpl).length) {
    return '关联项目未配置运行硬件模版'
  }

  const instanceType = resolveRunTemplateInstanceType(tpl)
  if (!instanceType) return '项目运行硬件模版未配置实例规格'

  const req = buildTaskStartVmRequestFromRunTemplate({
    taskId: ctx.taskId || 'task',
    containerImageId: ctx.selectedImageId,
    runTemplate: tpl,
    autoReleaseEnabled: ctx.autoReleaseEnabled,
    autoReleaseMinutes: ctx.autoReleaseMinutes,
  })
  if (!req) return '项目运行硬件模版配置不完整，请检查地域或云平台'

  return ''
}
