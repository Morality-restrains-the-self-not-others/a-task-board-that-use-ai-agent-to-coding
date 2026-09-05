/**
 * WorkPanel 首轮初始化时解析 currentWorkspace（OPT-20260816-047）。
 *
 * 规则：URL 显式带 workspace_id（查询参数或路径）时以 URL 为准，禁止用
 * /me/ 的 current_workspace 覆盖后删除 query；仅在 URL 未指定时回退到用户
 * 默认工作空间，最后才是默认占位。
 *
 * @param {{ urlWorkspaceId?: string|number|null, userWorkspace?: {id: string|number, name?: string}|null }} input
 * @returns {{ id: string|number|null, name: string }}
 */
export function resolveWorkPanelInitialWorkspace({ urlWorkspaceId, userWorkspace }) {
  const urlId = urlWorkspaceId == null ? '' : String(urlWorkspaceId)
  if (urlId) {
    // /me/ 已返回同名工作空间 → 复用完整对象（含展示名），否则用 URL 占位对象
    const same = userWorkspace && String(userWorkspace.id) === urlId
    return same ? userWorkspace : { id: urlWorkspaceId, name: '工作空间' }
  }
  if (userWorkspace) return userWorkspace
  return { id: null, name: '默认工作空间' }
}
