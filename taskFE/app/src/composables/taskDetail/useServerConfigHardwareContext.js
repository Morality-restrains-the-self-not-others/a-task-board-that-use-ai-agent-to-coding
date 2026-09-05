import { readRegisteredCommentHardwarePanel } from './commentRunHardwareTemplate.js'

/**
 * 展开硬件配置面板（供评论区「临时调节」按钮调用）。
 * 硬件已并入镜像卡：滚动至面板并打开临时配置，不再切换服务器信息 Tab。
 */
export function expandHardwareForComment(expandServerSectionBody, _activeServerSection, serverHardwarePanelRef) {
  expandServerSectionBody()
  const panel = serverHardwarePanelRef?.value || readRegisteredCommentHardwarePanel()
  if (panel && typeof panel.openTemporaryConfig === 'function') {
    panel.openTemporaryConfig()
  }
  if (typeof document !== 'undefined') {
    const el = document.getElementById('hardware-config-section')
    if (el && typeof el.scrollIntoView === 'function') {
      el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    }
  }
}

/** 恢复硬件配置为项目模版默认值 */
export function restoreHardwareToProjectDefaults(serverHardwarePanelRef) {
  const panel = serverHardwarePanelRef?.value || readRegisteredCommentHardwarePanel()
  if (panel && typeof panel.restoreProjectTemplateDefaults === 'function') {
    panel.restoreProjectTemplateDefaults()
  }
}
