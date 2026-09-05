/**
 * 评论执行面板策略：每条评论独立实例。
 * isActive 只用于「当前执行」徽章，不得作为完整面板的单例门控。
 */

export function shouldEnableCommentRuntimeTab(serverRuntimeStatusPanel) {
  return Boolean(serverRuntimeStatusPanel)
}

export function shouldMountCommentRuntimeStatusSection(serverRuntimeStatusPanel) {
  return Boolean(serverRuntimeStatusPanel)
}

export function shouldMountCommentConnectionPanel(ownsSharedContainer) {
  return Boolean(ownsSharedContainer)
}
