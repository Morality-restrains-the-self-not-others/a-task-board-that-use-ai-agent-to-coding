import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { parseJsonSafe } from '../utils/workPanelApiUtils.js'
import {
  resolveRepoOAuthProvider,
  resolveRepoOAuthAuthorizeUrl,
} from '../utils/repoOAuthAuthorizeUtils.js'
import {
  createGithubAppReturnKey,
  setGithubAppReturnTarget,
} from '../utils/githubAppReturnStorage.js'

export async function startRepoOAuthAuthorize(repoUrl) {
  const provider = resolveRepoOAuthProvider(repoUrl)
  const startApiUrl = resolveRepoOAuthAuthorizeUrl(repoUrl)
  if (!provider || !startApiUrl) {
    window.alert('当前仓库暂不支持 OAuth 一键授权')
    return
  }
  const nextPath = `${window.location.pathname}${window.location.search || ''}`
  const returnKey = createGithubAppReturnKey()
  setGithubAppReturnTarget(returnKey, nextPath)
  const targetUrl =
    `${startApiUrl}?next=${encodeURIComponent(nextPath)}` +
    `&return_key=${encodeURIComponent(returnKey)}` +
    `&repo_url=${encodeURIComponent(repoUrl)}`
  if (!targetUrl) {
    window.alert('未找到可用的 OAuth 授权地址')
    return
  }
  try {
    const response = await apiFetch(targetUrl, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await parseJsonSafe(response)
    if (!response.ok || !data?.authorize_url) {
      const detail = data?.detail ? String(data.detail) : '无法启动 OAuth 授权'
      showRequestError(detail, data)
      return
    }
    window.location.href = String(data.authorize_url)
  } catch (error) {
    showRequestError(error?.message || '无法启动 OAuth 授权', error)
  }
}
