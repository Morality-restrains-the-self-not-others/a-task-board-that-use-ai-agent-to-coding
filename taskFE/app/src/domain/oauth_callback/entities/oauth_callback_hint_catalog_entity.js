/** 静态 OAuth 回调码 → 用户文案目录（与 gitOauth 错误码契约对齐） */

export const GITHUB_CALLBACK_HINTS = Object.freeze({
  ok: 'GitHub 授权成功',
  bad_state: '授权失败：状态校验未通过，请重试',
  exchange_failed: '授权失败：无法连接 GitHub 服务器，请检查网络后重试',
  exchange_rejected: '授权失败：GitHub 拒绝了授权请求（授权码可能已过期），请重新发起授权',
  bind_failed: '授权失败：后端绑定未通过，请联系管理员',
  no_refresh:
    '授权失败：未获得 refresh_token。请在 GitHub App 中启用「User authorization token rotation / 刷新用户访问令牌」，并重新安装或重新授权。',
  no_access: '授权失败：未获取到 access_token',
  profile_failed: '授权失败：无法读取 GitHub 用户资料',
  no_gh_id: '授权失败：未获取到 GitHub 用户 ID',
  need_login: '授权未完成：请先登录后再试',
  return_expired: '回跳信息已过期（超过 30 秒），请重新发起授权',
})

export const GITLAB_CALLBACK_HINTS = Object.freeze({
  ok: 'GitLab 授权成功',
  bad_state: '授权失败：状态校验未通过，请重试',
  exchange_failed: '授权失败：无法连接 GitLab 服务器，请检查网络后重试',
  exchange_rejected: '授权失败：GitLab 拒绝了授权请求（授权码可能已过期），请重新发起授权',
  bind_failed: '授权失败：后端绑定未通过，请联系管理员',
  no_refresh: '授权失败：未获得 refresh_token，请检查 GitLab OAuth 应用配置',
  no_access: '授权失败：未获取到 access_token',
  profile_failed: '授权失败：无法读取 GitLab 用户资料',
  no_gitlab_id: '授权失败：未获取到 GitLab 用户 ID',
  return_expired: '回跳信息已过期（超过 30 秒），请重新发起授权',
})

export class OAuthCallbackHintCatalog {
  constructor({ githubHints = GITHUB_CALLBACK_HINTS, gitlabHints = GITLAB_CALLBACK_HINTS } = {}) {
    this._githubHints = githubHints
    this._gitlabHints = gitlabHints
  }

  hintsFor(providerValue) {
    return providerValue === 'gitlab' ? this._gitlabHints : this._githubHints
  }

  resolveRawMessage(providerValue, codeValue) {
    const hints = this.hintsFor(providerValue)
    const key = String(codeValue ?? '').trim()
    if (key === 'ok') {
      return { severity: 'success', message: hints.ok }
    }
    if (hints[key]) {
      return { severity: 'error', message: hints[key] }
    }
    return { severity: 'error', message: `授权未完成（${key}）` }
  }
}
