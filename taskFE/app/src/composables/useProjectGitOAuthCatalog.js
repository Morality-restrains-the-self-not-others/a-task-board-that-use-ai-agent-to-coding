import { ref } from 'vue'
import { loadGitOAuthProviderCatalog } from '../utils/repoOAuthAuthorizeUtils.js'

/** 项目详情页：加载 Git OAuth 目录并驱动授权按钮重算 */
export function useProjectGitOAuthCatalog(apiFetch, getCompanyId) {
  const gitOAuthCatalogVersion = ref(0)
  const touchRepoOAuthButtons = () => {
    void gitOAuthCatalogVersion.value
  }
  const bootstrapGitOAuthCatalog = async () => {
    const cid = typeof getCompanyId === 'function' ? getCompanyId() : String(getCompanyId || '')
    await loadGitOAuthProviderCatalog(apiFetch, cid)
    gitOAuthCatalogVersion.value += 1
  }
  return { gitOAuthCatalogVersion, touchRepoOAuthButtons, bootstrapGitOAuthCatalog }
}
