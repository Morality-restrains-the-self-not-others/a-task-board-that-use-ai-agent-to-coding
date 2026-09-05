import { computed } from 'vue'
import { resolveCommentCloneProgressWithCatalog, omitSkippedNestedCloneProgressRows } from '../../utils/commentCloneProgressRepoCatalog.js'
import { taskProjectsAllowNestedClone } from '../../utils/commentRepoIdentity.js'

/** 评论区克隆进度行解析（从 TaskDetailCommentsSection 抽出以控行数）。 */
export function useCommentSectionCloneProgressRows(props, {
  bindingStatusFor,
  buildPerBindingServerStatusProps,
}) {
  const soleActiveCloneCommentId = computed(() => {
    const live = (props.displayComments || []).filter((c) => {
      const st = bindingStatusFor(c?.id)
      return st === 'starting' || st === 'running'
    })
    if (live.length !== 1) return ''
    return String(live[0]?.id || '').trim()
  })

  function commentCloneProgressRows(commentId) {
    const cid = String(commentId || '').trim()
    const logs = cid
      ? (buildPerBindingServerStatusProps(cid).statusLogs || [])
      : (Array.isArray(props.statusLogs) ? props.statusLogs : [])
    const rows = resolveCommentCloneProgressWithCatalog({
      commentId: cid,
      statusLogs: logs,
      byCommentId: props.cloneProgressByCommentId,
      taskLevelEntries: props.containerCloneProgressEntries,
      soleActiveCommentId: soleActiveCloneCommentId.value,
    }, {
      projects: props.taskProjectsWithDetails,
      taskRepoRows: props.taskRepoRows,
      extraEntries: props.containerCloneProgressEntries,
      bootstrapLog: props.bootstrapCloneLogText || props.layerCloneLogText,
      statusLogs: logs,
    })
    return omitSkippedNestedCloneProgressRows(rows, {
      autoCloneNestedRepos: taskProjectsAllowNestedClone(props.taskProjectsWithDetails),
    })
  }

  return { commentCloneProgressRows }
}
