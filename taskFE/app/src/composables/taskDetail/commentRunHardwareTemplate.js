/**
 * 评论 $镜像 启动所用硬件：临时配置完整时覆盖项目模版。
 * 硬件卡是 UI 真源；submitComment 读取当前面板快照。
 */

/**
 * $param {object | null | undefined} panel ServerConfigHardwarePanel expose
 * $returns {object | null} server_run_template 或 null（用项目模版 / 临时规格不齐则不覆盖）
 */
export function resolveCommentServerRunTemplate(panel) {
  if (!panel) return null
  const source = typeof panel.getHardwareConfigSource === 'function'
    ? panel.getHardwareConfigSource()
    : panel.hardwareConfigSource
  if (source === 'project_template') return null
  const payload = typeof panel.buildRunTemplatePayload === 'function'
    ? panel.buildRunTemplatePayload()
    : null
  if (!payload || typeof payload !== 'object') return null
  const platformId = String(payload.cloud_platform_id || '').trim()
  const region = String(payload.region || '').trim()
  if (!platformId || !region) return null
  return payload
}

let commentHardwarePanelReader = () => null

export function registerCommentHardwarePanelReader(fn) {
  commentHardwarePanelReader = typeof fn === 'function' ? fn : () => null
}

export function readRegisteredCommentHardwarePanel() {
  return commentHardwarePanelReader()
}

/**
 * 临时配置已展开但规格不完整时，提示无法运行（不拦截发评）。
 * Advisory only: submitComment must not use this as a send blocker.
 * $param {object | null | undefined} panel
 * $returns {string} 空字符串表示无提示
 */
export function resolveCommentHardwareRunHint(panel) {
  if (!panel) return ''
  const source = typeof panel.getHardwareConfigSource === 'function'
    ? panel.getHardwareConfigSource()
    : panel.hardwareConfigSource
  if (source !== 'temporary') return ''
  if (resolveCommentServerRunTemplate(panel)) return ''
  return '临时硬件配置不齐，无法启动运行；仍可发送评论'
}
