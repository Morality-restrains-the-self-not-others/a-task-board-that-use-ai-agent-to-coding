import { ref } from 'vue'

/** 提交并运行时的仓库身份草稿；submitComment 读取后清空。 */
export const commentRepoIdentityDraft = ref([])

export function setCommentRepoIdentityDraft(rows) {
  commentRepoIdentityDraft.value = Array.isArray(rows) ? rows.map((row) => {
    const item = {
      repo_url: String(row?.repo_url || '').trim(),
      git_identity_id: String(row?.git_identity_id || '').trim(),
      github_user_id: String(row?.github_user_id || '').trim(),
    }
    const oauthGitsite = String(row?.oauth_gitsite || '').trim().toLowerCase()
    if (oauthGitsite) item.oauth_gitsite = oauthGitsite
    const remote = String(row?.oauth_remote_user_id || '').trim()
    if (remote) item.oauth_remote_user_id = remote
    return item
  }).filter((row) => row.repo_url) : []
}

export function resetCommentRepoIdentityDraft() {
  commentRepoIdentityDraft.value = []
}

export function readCommentRepoIdentityDraft() {
  return Array.isArray(commentRepoIdentityDraft.value) ? commentRepoIdentityDraft.value.slice() : []
}
