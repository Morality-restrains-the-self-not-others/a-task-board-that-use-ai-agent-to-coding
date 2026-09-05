/**
 * 管理端删除确认展示用：优先 Web 地址，其次 API 基址。
 * @param {{ gitlab_web_url?: string, gitlab_api_base?: string } | null | undefined} region
 * @returns {string}
 */
export function resolveGitlabRegionRepoAddress(region) {
  const web = String(region?.gitlab_web_url || '').trim()
  if (web) return web
  const api = String(region?.gitlab_api_base || '').trim()
  if (api) return api
  return '（未配置仓库地址）'
}
