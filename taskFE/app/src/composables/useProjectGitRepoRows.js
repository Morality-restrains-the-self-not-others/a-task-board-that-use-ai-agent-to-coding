import { ref } from 'vue'
import {
  GIT_REPO_URL_EXAMPLES,
  GIT_REPO_URL_FORMAT_HINT,
  INVALID_REPO_URL_MSG,
  gitRepoRowsFormatError,
  isValidGitRepoUrl,
} from '../utils/gitRepoUrlUtils.js'

export {
  GIT_REPO_URL_EXAMPLES,
  GIT_REPO_URL_FORMAT_HINT,
  INVALID_REPO_URL_MSG,
}

/** 编辑项目页：多 Git 仓库行与格式校验（与创建页共用 isValidGitRepoUrl）。 */
export function useProjectGitRepoRows() {
  let gitRepoRowIdSeq = 1
  const gitRepoRows = ref([{ id: gitRepoRowIdSeq, url: '', cloneAlias: '' }])
  const gitRepoRowFormatErrors = ref({})

  const trimmedGitRepoPayload = () => {
    const out = []
    for (const r of gitRepoRows.value) {
      const url = String(r.url || '').trim()
      if (!url) continue
      const cloneAlias = String(r.cloneAlias || '').trim()
      if (cloneAlias) {
        out.push({ url, clone_alias: cloneAlias })
      } else {
        out.push(url)
      }
    }
    return out
  }

  const duplicateCloneAliasError = () => {
    const seen = new Map()
    for (const r of gitRepoRows.value) {
      const alias = String(r.cloneAlias || '').trim().toLowerCase()
      if (!alias) continue
      if (seen.has(alias)) return '同一项目内仓库别名不能重复'
      seen.set(alias, r.id)
    }
    return ''
  }

  const refreshGitRepoFormatErrors = () => {
    const next = {}
    for (const r of gitRepoRows.value) {
      const url = String(r.url || '').trim()
      if (url && !isValidGitRepoUrl(url)) next[r.id] = INVALID_REPO_URL_MSG
    }
    gitRepoRowFormatErrors.value = next
    return gitRepoRowsFormatError(gitRepoRows.value)
  }

  const addGitRepoRow = () => {
    gitRepoRowIdSeq += 1
    gitRepoRows.value = [...gitRepoRows.value, { id: gitRepoRowIdSeq, url: '', cloneAlias: '' }]
  }

  const removeGitRepoRow = (rowId) => {
    if (gitRepoRows.value.length <= 1) return
    gitRepoRows.value = gitRepoRows.value.filter((r) => r.id !== rowId)
    if (rowId in gitRepoRowFormatErrors.value) {
      const next = { ...gitRepoRowFormatErrors.value }
      delete next[rowId]
      gitRepoRowFormatErrors.value = next
    }
  }

  const setGitRepoRowsFromApi = (data) => {
    let entries = []
    if (Array.isArray(data.git_repo_entries) && data.git_repo_entries.length) {
      entries = data.git_repo_entries
    } else if (Array.isArray(data.git_repos) && data.git_repos.length) {
      entries = data.git_repos.filter(Boolean)
    } else {
      entries = ['']
    }
    gitRepoRowIdSeq = 0
    gitRepoRows.value = entries.map((entry) => {
      if (entry && typeof entry === 'object') {
        return {
          id: ++gitRepoRowIdSeq,
          url: String(entry.url || entry.git_repo || '').trim(),
          cloneAlias: String(entry.clone_alias || entry.cloneAlias || '').trim(),
        }
      }
      return { id: ++gitRepoRowIdSeq, url: String(entry || '').trim(), cloneAlias: '' }
    })
    if (!gitRepoRows.value.length) {
      gitRepoRows.value = [{ id: ++gitRepoRowIdSeq, url: '', cloneAlias: '' }]
    }
    refreshGitRepoFormatErrors()
  }

  return {
    gitRepoRows,
    gitRepoRowFormatErrors,
    trimmedGitRepoPayload,
    duplicateCloneAliasError,
    refreshGitRepoFormatErrors,
    addGitRepoRow,
    removeGitRepoRow,
    setGitRepoRowsFromApi,
    GIT_REPO_URL_EXAMPLES,
    GIT_REPO_URL_FORMAT_HINT,
    INVALID_REPO_URL_MSG,
  }
}
