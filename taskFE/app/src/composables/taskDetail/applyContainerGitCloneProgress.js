/**
 * SSE `container_git_clone_progress` → 任务级进度 Map + 可选评论级镜像。
 * 从 updateServerStatus 抽出以控制行数，并按 comment_id 隔离并行评论。
 */
import {
  mergeCloneProgressSubPhases,
  isCloneProgressFailureMessage,
  shouldKeepCloneProgress,
} from '../../utils/taskDetailContainerCloneProgress.js'

function commentIdFromStatus(statusData) {
  return String(statusData?.comment_id || statusData?.commentId || '').trim()
}

function writeCommentCloneMap(byCommentIdRef, commentId, nextMap) {
  const cid = String(commentId || '').trim()
  if (!cid || !byCommentIdRef) return
  const prevAll =
    byCommentIdRef.value && typeof byCommentIdRef.value === 'object' ? byCommentIdRef.value : {}
  byCommentIdRef.value = { ...prevAll, [cid]: nextMap }
}

function anyCloneEntryComplete(map) {
  if (!map || typeof map !== 'object') return false
  return Object.values(map).some((v) => v && Number.isFinite(Number(v.progress)) && Number(v.progress) >= 100)
}

/** 全局「克隆完成」须保留已有分仓 100%，不得只留下 __global__ 以致迟到 3% 当成新仓写入。 */
function markCommentMapCloneDone(prevMap, message, globalKey) {
  const prev = prevMap && typeof prevMap === 'object' ? prevMap : {}
  const next = {}
  let keptPerRepo = false
  for (const [k, v] of Object.entries(prev)) {
    if (!k || k === globalKey) continue
    const row = v && typeof v === 'object' ? v : {}
    next[k] = {
      ...row,
      progress: 100,
      recvProgress: 100,
      unpackProgress: 100,
      message: /完成/.test(String(row.message || '')) ? row.message : message,
    }
    keptPerRepo = true
  }
  if (!keptPerRepo) {
    next[globalKey] = {
      progress: 100,
      message,
      recvProgress: 100,
      unpackProgress: 100,
    }
  }
  return next
}

/**
 * @param {object} statusData
 * @param {object} deps 与 updateServerStatus 同源
 * @returns {boolean} 已处理该 status 时 true
 */
export function applyContainerGitCloneProgress(statusData, deps) {
  if (!statusData || statusData.status !== 'container_git_clone_progress') return false
  const {
    containerBootstrapCloneLogFull,
    containerCloneProgressByKey,
    containerCloneProgressByCommentId,
    markContainerTransportOk,
    onBootstrapCloneLogUpdate,
    refreshLayerGraphFromServer,
    bumpProjectFileTreeRefresh,
    rescheduleContainerCloneProgressClearTimer,
    CONTAINER_CLONE_PROGRESS_GLOBAL_KEY,
  } = deps

  if (typeof markContainerTransportOk === 'function') markContainerTransportOk()

  const hasBlText = typeof statusData.bootstrap_log_text === 'string'
  const hasBlSeg =
    Array.isArray(statusData.bootstrap_log_segments) && statusData.bootstrap_log_segments.length
  if (hasBlText || hasBlSeg) {
    const text = hasBlText
      ? statusData.bootstrap_log_text
      : containerBootstrapCloneLogFull?.value
    const segments = hasBlSeg ? statusData.bootstrap_log_segments : null
    if (typeof onBootstrapCloneLogUpdate === 'function') {
      onBootstrapCloneLogUpdate({
        text: typeof text === 'string' ? text : '',
        segments: Array.isArray(segments) && segments.length ? segments : null,
      })
    }
  }

  const p = Number(statusData.progress)
  const progress = Number.isFinite(p) ? Math.min(100, Math.max(0, p)) : 0
  const message = typeof statusData.message === 'string' ? statusData.message : ''
  const seg =
    statusData.segment && typeof statusData.segment === 'object' ? statusData.segment : null
  const rawRepoFromSeg = seg && typeof seg.repo_url === 'string' ? seg.repo_url.trim() : ''
  const rawRepo =
    (typeof statusData.repo_url === 'string' ? statusData.repo_url.trim() : '') || rawRepoFromSeg
  const commentId = commentIdFromStatus(statusData)
  const globalKey = CONTAINER_CLONE_PROGRESS_GLOBAL_KEY || '__global__'

  const isBootstrapAllDone =
    progress >= 100 &&
    !rawRepo &&
    (message.includes('仓库克隆已完成') || message.includes('【项目克隆】仓库克隆已完成'))

  if (isBootstrapAllDone) {
    if (deps.containerCloneProgressTimer !== null && deps.containerCloneProgressTimer !== undefined) {
      clearTimeout(deps.containerCloneProgressTimer)
      deps.containerCloneProgressTimer = null
    }
    containerCloneProgressByKey.value = {}
    const prevComment =
      (containerCloneProgressByCommentId?.value && commentId
        ? containerCloneProgressByCommentId.value[commentId]
        : null) || {}
    writeCommentCloneMap(
      containerCloneProgressByCommentId,
      commentId,
      markCommentMapCloneDone(prevComment, message, globalKey),
    )
    if (typeof refreshLayerGraphFromServer === 'function') void refreshLayerGraphFromServer(true)
    if (typeof bumpProjectFileTreeRefresh === 'function') bumpProjectFileTreeRefresh()
    return true
  }

  if (progress >= 100 && !rawRepo) {
    if (deps.containerCloneProgressTimer !== null && deps.containerCloneProgressTimer !== undefined) {
      clearTimeout(deps.containerCloneProgressTimer)
      deps.containerCloneProgressTimer = null
    }
    const prevG = containerCloneProgressByKey.value[globalKey]
    const subG = mergeCloneProgressSubPhases(prevG, seg)
    const nextMap = {
      [globalKey]: {
        progress,
        message,
        recvProgress: subG.recv,
        unpackProgress: subG.unpack,
      },
    }
    containerCloneProgressByKey.value = nextMap
    const prevComment =
      (containerCloneProgressByCommentId?.value && commentId
        ? containerCloneProgressByCommentId.value[commentId]
        : null) || {}
    writeCommentCloneMap(
      containerCloneProgressByCommentId,
      commentId,
      markCommentMapCloneDone(prevComment, message, globalKey),
    )
    if (typeof refreshLayerGraphFromServer === 'function') void refreshLayerGraphFromServer(true)
    if (typeof bumpProjectFileTreeRefresh === 'function') bumpProjectFileTreeRefresh()
    return true
  }

  if (!rawRepo && isCloneProgressFailureMessage(message)) {
    const prevTask = containerCloneProgressByKey.value || {}
    const existingComment =
      (containerCloneProgressByCommentId?.value && commentId
        ? containerCloneProgressByCommentId.value[commentId]
        : null) || {}
    const hasPerRepo = (m) => Object.keys(m || {}).some((k) => k && k !== globalKey)
    if (hasPerRepo(prevTask) || hasPerRepo(existingComment)) {
      if (typeof rescheduleContainerCloneProgressClearTimer === 'function') {
        rescheduleContainerCloneProgressClearTimer()
      }
      return true
    }
  }

  const key = rawRepo || globalKey
  const prev = containerCloneProgressByKey.value || {}
  const prevEntry = prev[key]
  const existingComment =
    (containerCloneProgressByCommentId?.value && commentId
      ? containerCloneProgressByCommentId.value[commentId]
      : null) || {}
  const prevCommentEntry = commentId ? existingComment[key] : undefined
  const completedHint =
    Number(prevEntry?.progress) >= 100
    || Number(prevCommentEntry?.progress) >= 100
    || anyCloneEntryComplete(existingComment)
    || anyCloneEntryComplete(prev)
  if (
    shouldKeepCloneProgress(prevEntry?.progress, progress, message)
    || shouldKeepCloneProgress(prevCommentEntry?.progress, progress, message)
    || (completedHint && shouldKeepCloneProgress(100, progress, message))
  ) {
    if (typeof rescheduleContainerCloneProgressClearTimer === 'function') {
      rescheduleContainerCloneProgressClearTimer()
    }
    return true
  }
  const sub = mergeCloneProgressSubPhases(prevEntry, seg)
  const entry = { progress, message, recvProgress: sub.recv, unpackProgress: sub.unpack }
  const next = {
    ...prev,
    [key]: entry,
  }
  if (rawRepo) {
    delete next[globalKey]
  }
  containerCloneProgressByKey.value = next

  const prevComment =
    (containerCloneProgressByCommentId?.value && commentId
      ? containerCloneProgressByCommentId.value[commentId]
      : null) || {}
  const commentSub = mergeCloneProgressSubPhases(prevComment[key], seg)
  const nextComment = {
    ...prevComment,
    [key]: {
      progress,
      message,
      recvProgress: commentSub.recv,
      unpackProgress: commentSub.unpack,
    },
  }
  if (rawRepo) {
    delete nextComment[globalKey]
  }
  writeCommentCloneMap(containerCloneProgressByCommentId, commentId, nextComment)

  if (typeof rescheduleContainerCloneProgressClearTimer === 'function') {
    rescheduleContainerCloneProgressClearTimer()
  }

  if (progress >= 100) {
    if (typeof refreshLayerGraphFromServer === 'function') void refreshLayerGraphFromServer(true)
    if (typeof bumpProjectFileTreeRefresh === 'function') bumpProjectFileTreeRefresh()
  }
  return true
}
