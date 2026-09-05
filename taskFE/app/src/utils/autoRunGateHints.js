/**
 * 创建/编辑任务「是否自动运行」门禁提示（与 taskTaskService validateAutoRunPrerequisites 对齐）。
 */

export function resolveAutoRunGateState({
  hasLinkedProject = false,
  hasConfiguredRunTemplate = false,
  hasInstalledImage = false,
  projectAllowsAutoRun = false,
} = {}) {
  const blockers = []
  if (!hasLinkedProject) {
    blockers.push({
      code: 'AUTO_RUN_PROJECT_REQUIRED',
      message: '请先选择关联项目',
    })
  } else if (!hasConfiguredRunTemplate) {
    blockers.push({
      code: 'AUTO_RUN_RUN_TEMPLATE_REQUIRED',
      message: '当前项目未配置运行硬件模版，请在项目编辑页设置后再启用',
    })
  } else if (!projectAllowsAutoRun) {
    blockers.push({
      code: 'AUTO_RUN_PROJECT_NOT_ALLOWED',
      message: '当前项目未允许自动运行，请在项目设置中开启「是否允许自动运行」',
    })
  }
  if (!hasInstalledImage) {
    blockers.push({
      code: 'AUTO_RUN_IMAGE_REQUIRED',
      message: '请先选择已安装镜像',
    })
  }
  return {
    canEnable: blockers.length === 0,
    blockers,
  }
}

/** 禁用时多条原因文案（逐条展示） */
export function resolveAutoRunDisabledReasons(ctx = {}) {
  return resolveAutoRunGateState(ctx).blockers.map((b) => b.message)
}

/** 可启用时的说明（含服务端门禁提示） */
export function resolveAutoRunEnabledHint({ templateSummary = '已配置' } = {}) {
  const name = String(templateSummary || '').trim() || '已配置'
  return `创建/保存后按项目运行模版「${name}」自动启动服务器；须事先为全部 GitHub/GitLab 仓库完成 OAuth 绑定；若云启动服务未配置，服务端将拒绝开启自动运行`
}

/** 编辑态且已开启自动运行时的强制重跑说明 */
export function resolveForceAutoRunHint() {
  return '仅在已开启自动运行时生效：保存后再次触发 start-vm（默认重复保存不会二次启动）'
}

/**
 * 服务端软跳过自动启服时的提示（auto_run 仍为 true）。
 * @param {{ auto_run_start_skipped?: boolean, auto_run_start_skip_reason?: string }|null|undefined} body
 */
export function resolveAutoRunStartSkippedMessage(body) {
  if (!body || body.auto_run_start_skipped !== true) return ''
  const reason = String(body.auto_run_start_skip_reason || '').trim()
  if (reason) {
    return `已保存自动运行，但未自动启动服务器：${reason}`
  }
  return '已保存自动运行，但因无法获取 Git / 子 Git 仓库而未自动启动服务器。请先完成 Git 网站绑定后重试。'
}

/** 创建任务保存后若服务端软跳过启服则弹窗提示 */
export function alertAutoRunStartSkippedIfNeeded(body, modalService) {
  const msg = resolveAutoRunStartSkippedMessage(body)
  if (msg) modalService.alert(msg, '自动运行未启服', { autoClose: 6000 })
}
