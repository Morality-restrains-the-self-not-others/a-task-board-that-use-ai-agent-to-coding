/** 分支预览：需覆盖 gitOauth 内部 5s + GitLab API 10s 的串行上限 */
export const BRANCH_PREVIEW_REQUEST_TIMEOUT_MS = 15000

export const BRANCH_PREVIEW_TIMEOUT_ERROR =
  '分支列表查询超时，可能是 Git OAuth 换票或 GitLab 响应较慢，请稍后重试或检查 Git 网站授权'

export const resolveBranchPreviewFetchError = (err) => {
  if (err?.name === 'AbortError') {
    return BRANCH_PREVIEW_TIMEOUT_ERROR
  }
  return err?.message || '查询失败'
}
