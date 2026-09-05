import { ref } from 'vue'

/** 任务详情「添加评论」依赖草稿；提交后由 submitComment 读取并清空。 */
export const commentDependencyDraft = ref({
  executionMode: 'wait_previous',
  dependsOnCommentIds: [],
  autoCommit: false,
})

export function setCommentDependencyDraft({ executionMode, dependsOnCommentIds, autoCommit } = {}) {
  const mode = executionMode === 'independent' ? 'independent' : 'wait_previous'
  const ids = Array.isArray(dependsOnCommentIds)
    ? dependsOnCommentIds.map((id) => String(id || '').trim()).filter(Boolean)
    : []
  commentDependencyDraft.value = {
    executionMode: mode,
    dependsOnCommentIds: mode === 'independent' ? [] : ids,
    autoCommit: !!autoCommit,
  }
}

export function resetCommentDependencyDraft() {
  commentDependencyDraft.value = {
    executionMode: 'wait_previous',
    dependsOnCommentIds: [],
    autoCommit: false,
  }
}

export function readCommentDependencyDraft() {
  const d = commentDependencyDraft.value || {}
  const mode = d.executionMode === 'independent' ? 'independent' : 'wait_previous'
  const ids = Array.isArray(d.dependsOnCommentIds)
    ? d.dependsOnCommentIds.map((id) => String(id || '').trim()).filter(Boolean)
    : []
  return {
    executionMode: mode,
    dependsOnCommentIds: mode === 'independent' ? [] : ids,
    autoCommit: !!d.autoCommit,
  }
}
