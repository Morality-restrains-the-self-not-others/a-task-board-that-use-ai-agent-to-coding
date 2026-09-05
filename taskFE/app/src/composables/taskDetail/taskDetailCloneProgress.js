import { ref, computed } from 'vue'
import {
  gitCloneRefMatchKey,
  parseBootstrapCloneLogSections,
} from '../../utils/taskDetailContainerCloneProgress.js'

export const CONTAINER_CLONE_PROGRESS_GLOBAL_KEY = '__global__'

export const BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE =
  /【项目克隆】克隆完成|仓库克隆已完成|【项目克隆】仓库克隆已完成/

export const CONTAINER_CLONE_PROGRESS_TIMEOUT_MS = 10 * 60 * 1000

const CLONE_LOG_PLACEHOLDER = '（暂无本仓库引导日志；等待轮询或确认引导日志中 ━━ 行与仓库地址一致。）'

/**
 * 引导克隆日志是否表明批量克隆已成功结束（全量完成，非「部分失败」）。
 * @param {string|null|undefined} fullText
 * @returns {boolean}
 */
export function isBootstrapCloneLogDone(fullText) {
  const full = String(fullText || '')
  if (!full.trim()) return false
  return BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test(full)
}

export function createCloneProgressState() {
  const containerCloneProgressByKey = ref(
    /** @type {Record<string, { progress: number, message: string, recvProgress?: number | null, unpackProgress?: number | null }>} */ ({})
  )
  /** 按 comment_id 隔离的克隆进度（并行评论各自独立条） */
  const containerCloneProgressByCommentId = ref(
    /** @type {Record<string, Record<string, { progress: number, message: string, recvProgress?: number | null, unpackProgress?: number | null }>>} */ ({})
  )
  const containerBootstrapCloneLogFull = ref('')
  const containerBootstrapCloneLogSegments = ref(/** @type {{ repo_url: string, text: string }[] | null} */ (null))
  const containerPageLinkPendingReveal = ref(false)
  let containerCloneProgressTimer = null

  return {
    containerCloneProgressByKey,
    containerCloneProgressByCommentId,
    containerBootstrapCloneLogFull,
    containerBootstrapCloneLogSegments,
    containerPageLinkPendingReveal,
    get containerCloneProgressTimer() { return containerCloneProgressTimer },
    set containerCloneProgressTimer(v) { containerCloneProgressTimer = v },
  }
}

export function onBootstrapCloneLogUpdate(payload, state) {
  if (payload && typeof payload === 'object' && !Array.isArray(payload) && 'text' in payload) {
    const nextText = typeof payload.text === 'string' ? payload.text : ''
    // SSE 偶发带空 text：勿覆盖已有完整引导日志（否则 bootstrapCloneDone 回落为 false → 子仓全显「未开始」）
    if (nextText.trim() || !String(state.containerBootstrapCloneLogFull.value || '').trim()) {
      state.containerBootstrapCloneLogFull.value = nextText
    }
    const s = payload.segments
    if (Array.isArray(s) && s.length) {
      state.containerBootstrapCloneLogSegments.value = s
    }
    return
  }
  if (typeof payload === 'string') {
    if (payload.trim() || !String(state.containerBootstrapCloneLogFull.value || '').trim()) {
      state.containerBootstrapCloneLogFull.value = payload
    }
    return
  }
  state.containerBootstrapCloneLogFull.value = ''
  state.containerBootstrapCloneLogSegments.value = null
}

export function createCloneProgressComputeds(state, { taskRepoRows }) {
  const bootstrapCloneDone = computed(() =>
    isBootstrapCloneLogDone(state.containerBootstrapCloneLogFull.value),
  )

  const cloneProgressBarWidthTransitionClass = computed(() => {
    if (bootstrapCloneDone.value) {
      return ''
    }
    return 'transition-[width] duration-300 ease-out'
  })

  function cloneProgressRowLogText(row, rowIndex) {
    const liveMsg = typeof row?.message === 'string' ? row.message.trim() : ''
    const liveProgress = Number.isFinite(Number(row?.progress)) ? Number(row.progress) : null

    const full = state.containerBootstrapCloneLogFull.value
    if (!full || !String(full).trim()) {
      return liveMsg || ''
    }
    const bootstrapDone = isBootstrapCloneLogDone(full)
    const { preamble, sections } = parseBootstrapCloneLogSections(full)
    if (!row.repoUrl) {
      let out = String(full).trim()
      if (!bootstrapDone && liveMsg && liveProgress != null && liveProgress < 100) {
        out = `${out}\n\n━━ 实时进度（SSE）━━\n${liveMsg}`
      }
      return out
    }
    const want = gitCloneRefMatchKey(row.repoUrl)
    const apiSegs = state.containerBootstrapCloneLogSegments.value
    if (apiSegs && apiSegs.length) {
      for (const s of apiSegs) {
        const ru = s && s.repo_url != null ? s.repo_url : ''
        if (gitCloneRefMatchKey(ru) === want) {
          let body = String(s.text || '').trim()
          if (rowIndex === 0 && preamble) {
            body = preamble + (body ? `\n\n${body}` : '')
          }
          body = String(body || '').trim()
          if (!body && liveMsg) {
            return liveMsg
          }
          if (!bootstrapDone && body && liveMsg && liveProgress != null && liveProgress < 100) {
            return `${body}\n\n━━ 实时进度（SSE）━━\n${liveMsg}`
          }
          return body
        }
      }
    }
    let body = ''
    for (const s of sections) {
      if (gitCloneRefMatchKey(s.url) === want) {
        body = s.text
        break
      }
    }
    if (rowIndex === 0 && preamble) {
      body = preamble + (body ? `\n\n${body}` : '')
    }
    body = String(body || '').trim()
    if (!body && liveMsg) {
      return liveMsg
    }
    if (!bootstrapDone && body && liveMsg && liveProgress != null && liveProgress < 100) {
      return `${body}\n\n━━ 实时进度（SSE）━━\n${liveMsg}`
    }
    return body
  }

  function cloneProgressRowLogDisplayText(row, rowIndex) {
    const t = cloneProgressRowLogText(row, rowIndex)
    return t && String(t).trim() ? t : CLONE_LOG_PLACEHOLDER
  }

  function cloneProgressRowLogIsPlaceholder(row, rowIndex) {
    return !cloneProgressRowLogText(row, rowIndex).trim()
  }

  const containerCloneProgressEntries = computed(() => {
    const m = state.containerCloneProgressByKey.value
    if (!m || typeof m !== 'object') return []
    const entries = Object.entries(m).map(([key, v]) => ({
      key,
      repoUrl: key === CONTAINER_CLONE_PROGRESS_GLOBAL_KEY ? '' : key,
      progress: Number.isFinite(Number(v?.progress)) ? Math.min(100, Math.max(0, Number(v.progress))) : 0,
      message: typeof v?.message === 'string' ? v.message : '',
      recvProgress:
        typeof v?.recvProgress === 'number' && Number.isFinite(v.recvProgress)
          ? Math.min(100, Math.max(0, v.recvProgress))
          : null,
      unpackProgress:
        typeof v?.unpackProgress === 'number' && Number.isFinite(v.unpackProgress)
          ? Math.min(100, Math.max(0, v.unpackProgress))
          : null,
    }))
    const hasRepoRow = entries.some((e) => e.key !== CONTAINER_CLONE_PROGRESS_GLOBAL_KEY)
    const list = (hasRepoRow
      ? entries.filter((e) => e.key !== CONTAINER_CLONE_PROGRESS_GLOBAL_KEY)
      : entries
    ).slice()
    const rowsOrder = taskRepoRows.value
    const orderOf = (repoUrl) => {
      const u = String(repoUrl || '')
      const k = gitCloneRefMatchKey(u)
      const i = rowsOrder.findIndex((r) => gitCloneRefMatchKey(r.url) === k)
      return i >= 0 ? i : Number.MAX_SAFE_INTEGER
    }
    list.sort((a, b) => {
      if (a.key === CONTAINER_CLONE_PROGRESS_GLOBAL_KEY) return -1
      if (b.key === CONTAINER_CLONE_PROGRESS_GLOBAL_KEY) return 1
      const oa = orderOf(a.repoUrl)
      const ob = orderOf(b.repoUrl)
      if (oa !== ob) return oa - ob
      return a.repoUrl.localeCompare(b.repoUrl, 'zh-CN')
    })
    return list
  })

  const isGlobalOnlyCloneProgress = computed(
    () =>
      containerCloneProgressEntries.value.length === 1 &&
      containerCloneProgressEntries.value[0].key === CONTAINER_CLONE_PROGRESS_GLOBAL_KEY,
  )

  const cloneProgressEntryByRepoMatchKey = computed(() => {
    const m = new Map()
    for (const e of containerCloneProgressEntries.value) {
      if (e.repoUrl) {
        m.set(gitCloneRefMatchKey(e.repoUrl), e)
      }
    }
    return m
  })

  function cloneProgressEntryIndexInList(entry) {
    if (!entry) return 0
    const i = containerCloneProgressEntries.value.findIndex((e) => e.key === entry.key)
    return i >= 0 ? i : 0
  }

  return {
    bootstrapCloneDone,
    cloneProgressBarWidthTransitionClass,
    cloneProgressRowLogText,
    cloneProgressRowLogDisplayText,
    cloneProgressRowLogIsPlaceholder,
    containerCloneProgressEntries,
    isGlobalOnlyCloneProgress,
    cloneProgressEntryByRepoMatchKey,
    cloneProgressEntryIndexInList,
  }
}

export function rescheduleContainerCloneProgressClearTimer(state) {
  if (state.containerCloneProgressTimer !== null) {
    clearTimeout(state.containerCloneProgressTimer)
    state.containerCloneProgressTimer = null
  }
  const m = state.containerCloneProgressByKey.value
  const anyIncomplete = Object.values(m).some(
    (x) => x && Number.isFinite(Number(x.progress)) && Number(x.progress) < 100,
  )
  if (Object.keys(m).length > 0 && anyIncomplete) {
    state.containerCloneProgressTimer = setTimeout(() => {
      state.containerCloneProgressByKey.value = {}
      state.containerCloneProgressTimer = null
    }, CONTAINER_CLONE_PROGRESS_TIMEOUT_MS)
  }
}

function markCloneMapEntriesDone(map, doneMsg) {
  if (!map || typeof map !== 'object') return { next: map, changed: false }
  const next = { ...map }
  let changed = false
  for (const k of Object.keys(next)) {
    const v = next[k]
    if (!v || typeof v !== 'object') continue
    const p = Number(v.progress)
    const msg = String(v.message || '').trim()
    if (p >= 100 && (msg === doneMsg || msg.includes('完成') || msg.includes('仓库克隆已'))) {
      continue
    }
    next[k] = {
      ...v,
      progress: 100,
      message: doneMsg,
      recvProgress: 100,
      unpackProgress: 100,
    }
    changed = true
  }
  return { next, changed }
}

export function applyBootstrapCloneLogDoneToCloneProgress(state, { refreshLayerGraphFromServer, bumpProjectFileTreeRefresh }) {
  const full = state.containerBootstrapCloneLogFull.value
  if (!full || !String(full).trim() || !BOOTSTRAP_CLONE_LOG_INDICATES_DONE_RE.test(String(full))) {
    return
  }
  const doneMsg = '【项目克隆】克隆完成'
  const globalResult = markCloneMapEntriesDone(state.containerCloneProgressByKey.value, doneMsg)
  let changed = globalResult.changed
  if (changed) {
    state.containerCloneProgressByKey.value = globalResult.next
  }
  const byC = state.containerCloneProgressByCommentId?.value
  if (byC && typeof byC === 'object') {
    const nextByC = { ...byC }
    let cChanged = false
    for (const cid of Object.keys(nextByC)) {
      const inner = markCloneMapEntriesDone(nextByC[cid], doneMsg)
      if (inner.changed) {
        nextByC[cid] = inner.next
        cChanged = true
      }
    }
    if (cChanged) {
      state.containerCloneProgressByCommentId.value = nextByC
      changed = true
    }
  }
  if (!changed) {
    return
  }
  rescheduleContainerCloneProgressClearTimer(state)
  void refreshLayerGraphFromServer(true)
  bumpProjectFileTreeRefresh()
}
