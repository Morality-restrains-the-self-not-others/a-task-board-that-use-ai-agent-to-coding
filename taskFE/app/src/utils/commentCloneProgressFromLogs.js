/**
 * 评论启动日志 / live SSE map → 评论级项目克隆进度行。
 * 无 Vue 依赖，供执行细节进度条与单测复用。
 */
import {
  gitCloneRefMatchKey,
  isCloneProgressFailureMessage,
  shouldKeepCloneProgress,
  shortCloneRepoLabel,
} from './taskDetailContainerCloneProgress.js'

const CLONE_HINT_RE = /【项目克隆】|项目克隆\s*\(/
const PCT_RE = /(?:【项目克隆】|项目克隆)\s*(?:\((\d+)\/(\d+)\)\s+)?(.+?)\s*(?:…|\.{2,})\s*(\d+)\s*%/
const DONE_REPO_RE = /项目克隆\s*\((\d+)\/(\d+)\)\s*完成\s+(\S+)/
const FAIL_REPO_RE = /【项目克隆】\s*(?:\((\d+)\/(\d+)\)\s+)?失败\s+(\S+?)(?::|\s|$)/
const GLOBAL_DONE_RE = /【项目克隆】克隆完成|仓库克隆已完成|【项目克隆】仓库克隆已完成/
const RETRY_RE = /准备第\s*(\d+)\s*\/\s*(\d+)\s*次重试/
const FRAC_RE = /(?:【项目克隆】|项目克隆)\s*\((\d+)\/(\d+)\)/

/**
 * @typedef {{
 *   key: string,
 *   label: string,
 *   progress: number,
 *   message: string,
 *   recvProgress: number|null,
 *   unpackProgress: number|null,
 *   failed: boolean,
 *   repoUrl: string,
 *   repoIndex: number,
 *   repoTotal: number,
 *   retryAttempt: number,
 *   retryMax: number,
 *   retrying: boolean,
 * }} CommentCloneProgressRow
 */

export function stripStartupLogDecorations(line) {
  return String(line || '')
    .replace(/^\[[^\]]+\]\s*/, '')
    .replace(/\s+trace_id=\S+\s*$/i, '')
    .trim()
}

export function looksLikeGitRepoRef(value) {
  const s = String(value || '').trim()
  if (!s) return false
  return /^(https?:\/\/|ssh:\/\/|git@)/i.test(s)
}

const GIT_REF_IN_TEXT_RE = /((?:https?:\/\/[^\s,;]+)|(?:ssh:\/\/[^\s,;]+)|(?:git@[^\s,;]+))/i
const REPO_URL_EQ_RE = /\brepo_url=([^\s]+)/i

/**
 * 从克隆进度文案中抽出 git URL（失败行可能把 URL 写在仓库名后，或带 repo_url=）。
 * @param {string} text
 * @returns {string}
 */
export function extractGitRepoRefFromText(text) {
  const raw = String(text || '')
  const tagged = raw.match(REPO_URL_EQ_RE)
  if (tagged && looksLikeGitRepoRef(tagged[1])) {
    return String(tagged[1]).replace(/[.,;]+$/g, '')
  }
  const m = raw.match(GIT_REF_IN_TEXT_RE)
  if (!m) return ''
  let u = String(m[1] || '').trim()
  u = u.replace(/[.,;]+$/g, '')
  if (/^https?:\/\//i.test(u)) u = u.replace(/:+$/g, '')
  return looksLikeGitRepoRef(u) ? u : ''
}

export function parseCloneRetryHint(message) {
  const m = String(message || '').match(RETRY_RE)
  if (!m) return { retrying: false, retryAttempt: 0, retryMax: 0 }
  return {
    retrying: true,
    retryAttempt: Number(m[1]) || 0,
    retryMax: Number(m[2]) || 0,
  }
}

export function parseCloneRepoFraction(message) {
  const m = String(message || '').match(FRAC_RE)
  if (!m) return { repoIndex: 0, repoTotal: 0 }
  return { repoIndex: Number(m[1]) || 0, repoTotal: Number(m[2]) || 0 }
}

function clampPct(n) {
  const p = Number(n)
  if (!Number.isFinite(p)) return 0
  return Math.min(100, Math.max(0, p))
}

function metaFromMessage(message) {
  const retry = parseCloneRetryHint(message)
  const frac = parseCloneRepoFraction(message)
  return { ...retry, ...frac }
}

function emptyRow(key, label, message) {
  const meta = metaFromMessage(message)
  return {
    key,
    label,
    progress: 0,
    message,
    recvProgress: null,
    unpackProgress: null,
    failed: false,
    repoUrl: looksLikeGitRepoRef(key) ? key : extractGitRepoRefFromText(message),
    repoIndex: meta.repoIndex,
    repoTotal: meta.repoTotal,
    retryAttempt: meta.retryAttempt,
    retryMax: meta.retryMax,
    retrying: meta.retrying,
  }
}

function enrichRow(prev, patch) {
  const message = patch.message != null ? patch.message : prev.message
  const meta = metaFromMessage(message)
  const failed = Boolean(patch.failed) && !meta.retrying
  return {
    ...prev,
    ...patch,
    message,
    failed,
    repoUrl: patch.repoUrl
      || prev.repoUrl
      || (looksLikeGitRepoRef(prev.key) ? prev.key : '')
      || extractGitRepoRefFromText(message),
    repoIndex: meta.repoIndex || prev.repoIndex || 0,
    repoTotal: meta.repoTotal || prev.repoTotal || 0,
    retryAttempt: meta.retryAttempt,
    retryMax: meta.retryMax,
    retrying: meta.retrying,
  }
}

function findRowKey(byKey, { key, repoIndex }) {
  if (key && byKey.has(key)) return key
  if (repoIndex) {
    for (const [k, row] of byKey) {
      if (Number(row.repoIndex) === Number(repoIndex)) return k
    }
  }
  return key
}

/**
 * 从评论 per-binding 启动日志解析克隆进度（冷打开 / 无 SSE）。
 * @param {string[]} logs
 * @returns {CommentCloneProgressRow[]}
 */
export function parseCommentCloneProgressFromLogs(logs) {
  const list = Array.isArray(logs) ? logs : []
  /** @type {Map<string, CommentCloneProgressRow>} */
  const byKey = new Map()
  let sawClone = false
  let globalDone = false
  let globalFailed = false
  let lastGlobalMessage = ''

  for (const raw of list) {
    const line = stripStartupLogDecorations(raw)
    if (!CLONE_HINT_RE.test(line) && !/项目克隆/.test(line)) continue
    sawClone = true
    lastGlobalMessage = line

    const retryHint = parseCloneRetryHint(line)
    if (retryHint.retrying) {
      const frac = parseCloneRepoFraction(line)
      const existing = findRowKey(byKey, { key: '', repoIndex: frac.repoIndex })
      const key = existing || (frac.repoIndex ? `__idx_${frac.repoIndex}` : '__global__')
      const prev = byKey.get(key) || emptyRow(key, '项目克隆', line)
      byKey.set(key, enrichRow(prev, {
        message: line,
        failed: false,
        progress: prev.progress,
      }))
      continue
    }

    const pct = line.match(PCT_RE)
    if (pct) {
      const label = String(pct[3] || '').trim()
      const key = label || '__global__'
      const prev = byKey.get(key) || emptyRow(key, label || '项目克隆', line)
      const nextPct = clampPct(pct[4])
      const keep = shouldKeepCloneProgress(prev.progress, nextPct, line)
      byKey.set(key, enrichRow(prev, {
        progress: keep ? prev.progress : nextPct,
        message: keep ? prev.message : line,
        failed: isCloneProgressFailureMessage(line),
        label: label || prev.label,
      }))
      continue
    }

    const doneRepo = line.match(DONE_REPO_RE)
    if (doneRepo) {
      const label = String(doneRepo[3] || '').trim()
      const key = label || '__global__'
      const prev = byKey.get(key) || emptyRow(key, label || '项目克隆', line)
      byKey.set(key, enrichRow(prev, { progress: 100, message: line, failed: false, label: label || prev.label }))
      continue
    }

    const failRepo = line.match(FAIL_REPO_RE)
    if (failRepo) {
      const label = String(failRepo[3] || '').replace(/:$/, '').trim()
      const frac = parseCloneRepoFraction(line)
      const existing = findRowKey(byKey, { key: label, repoIndex: frac.repoIndex })
      const key = existing || label || '__global__'
      const prev = byKey.get(key) || emptyRow(key, label || '项目克隆', line)
      byKey.set(key, enrichRow(prev, {
        message: line,
        failed: true,
        label: label || prev.label,
        key: looksLikeGitRepoRef(prev.key) ? prev.key : key,
        repoUrl: extractGitRepoRefFromText(line) || prev.repoUrl || '',
      }))
      continue
    }

    if (GLOBAL_DONE_RE.test(line) && !/失败|未完成/.test(line)) {
      globalDone = true
    }
    if (isCloneProgressFailureMessage(line)) {
      globalFailed = true
    }
  }

  if (!sawClone) return []
  if (byKey.size === 0) {
    return [emptyRow('__global__', '项目克隆', lastGlobalMessage)].map((row) => enrichRow(row, {
      progress: globalDone ? 100 : 0,
      failed: globalFailed && !globalDone,
      message: lastGlobalMessage,
    }))
  }
  if (globalDone) {
    for (const [k, row] of byKey) {
      if (!row.failed && row.progress < 100) {
        byKey.set(k, { ...row, progress: 100 })
      }
    }
  }
  return [...byKey.values()]
}

function rowFromLiveOrEntry(key, v, repoUrlHint) {
  const message = typeof v?.message === 'string' ? v.message : ''
  const repoUrl = looksLikeGitRepoRef(repoUrlHint) ? repoUrlHint : (looksLikeGitRepoRef(key) ? key : '')
  const repoLike = key && key !== '__global__' ? key : ''
  const retry = parseCloneRetryHint(message)
  return enrichRow(emptyRow(key, repoLike ? shortCloneRepoLabel(repoLike) : '项目克隆', message), {
    progress: retry.retrying && clampPct(v?.progress) === 0 ? 0 : clampPct(v?.progress),
    message,
    recvProgress:
      typeof v?.recvProgress === 'number' && Number.isFinite(v.recvProgress) ? v.recvProgress : null,
    unpackProgress:
      typeof v?.unpackProgress === 'number' && Number.isFinite(v.unpackProgress) ? v.unpackProgress : null,
    failed: isCloneProgressFailureMessage(message) && !retry.retrying,
    repoUrl,
    label: repoLike ? shortCloneRepoLabel(repoLike) : '项目克隆',
  })
}

/**
 * @param {Record<string, { progress?: number, message?: string, recvProgress?: number|null, unpackProgress?: number|null }>|null|undefined} map
 * @returns {CommentCloneProgressRow[]}
 */
export function cloneProgressRowsFromLiveMap(map) {
  if (!map || typeof map !== 'object') return []
  return Object.entries(map).map(([key, v]) => rowFromLiveOrEntry(key, v, key))
}

/**
 * @param {Array<{ key?: string, repoUrl?: string, label?: string, progress?: number, message?: string, recvProgress?: number|null, unpackProgress?: number|null }>|null|undefined} entries
 * @returns {CommentCloneProgressRow[]}
 */
export function cloneProgressRowsFromEntries(entries) {
  if (!Array.isArray(entries) || !entries.length) return []
  return entries.map((e) => {
    const repoUrl = typeof e?.repoUrl === 'string' ? e.repoUrl : ''
    const key = e?.key || repoUrl || '__global__'
    const row = rowFromLiveOrEntry(key, e, repoUrl)
    if (!repoUrl && e?.label) return { ...row, label: e.label }
    return row
  })
}

export function cloneRowIdentity(row) {
  const urlLike = String(row?.repoUrl || '').trim()
    || (looksLikeGitRepoRef(row?.key) ? String(row.key) : '')
  if (urlLike) {
    const k = gitCloneRefMatchKey(urlLike)
    if (k) return k
  }
  return String(row?.label || row?.key || '').trim().toLowerCase()
}

function identitiesMatch(a, b) {
  if (!a || !b) return false
  if (a === b) return true
  const aBase = a.split('/').pop()
  const bBase = b.split('/').pop()
  return Boolean(aBase) && aBase === bBase
}

/**
 * @param {CommentCloneProgressRow[]} baseRows
 * @param {CommentCloneProgressRow[]} overlayRows
 * @returns {CommentCloneProgressRow[]}
 */
export function mergeCommentCloneProgressRows(baseRows, overlayRows) {
  const result = (Array.isArray(baseRows) ? baseRows : []).map((r) => ({ ...r }))
  for (const over of Array.isArray(overlayRows) ? overlayRows : []) {
    const oid = cloneRowIdentity(over)
    const idx = result.findIndex((r) => identitiesMatch(cloneRowIdentity(r), oid))
    if (idx >= 0) {
      const prev = result[idx]
      const overlayPct = clampPct(over.progress)
      const prevPct = clampPct(prev.progress)
      const keepPrev = !over.retrying && !over.failed
        && shouldKeepCloneProgress(prevPct, overlayPct, over.message)
      const progress = over.retrying && overlayPct === 0 && prevPct > 0
        ? prevPct
        : (keepPrev ? prevPct : overlayPct)
      result[idx] = enrichRow(prev, {
        ...over,
        progress,
        message: keepPrev ? prev.message : over.message,
        repoUrl: over.repoUrl || prev.repoUrl || '',
        label: over.label && over.label !== '项目克隆' ? over.label : prev.label,
        key: looksLikeGitRepoRef(over.key) ? over.key : (prev.key || over.key),
      })
    } else {
      result.push({ ...over })
    }
  }
  return result
}

export function expectedCloneRepoTotal(rows) {
  const list = Array.isArray(rows) ? rows : []
  let maxN = list.length
  for (const row of list) {
    const n = Number(row?.repoTotal)
    if (Number.isFinite(n) && n > maxN) maxN = n
  }
  return maxN
}

export function overallCloneProgressPct(rows) {
  const list = Array.isArray(rows) ? rows : []
  if (!list.length) return 0
  const n = expectedCloneRepoTotal(list)
  if (n <= 0) return 0
  const sum = list.reduce((acc, row) => acc + clampPct(row?.progress), 0)
  return Math.round(sum / n)
}

export function shouldShowCloneManualRetry(row) {
  if (!row || typeof row !== 'object') return false
  if (row.retrying) return false
  if (!row.failed) return false
  return looksLikeGitRepoRef(row.repoUrl || row.key)
}

export function canEmitCloneManualRetry(row) {
  return shouldShowCloneManualRetry(row)
}

/**
 * 引导结束的全局摘要行与分仓失败行同时存在时，只保留可操作的分仓行。
 * @param {CommentCloneProgressRow[]} rows
 * @returns {CommentCloneProgressRow[]}
 */
export function omitRedundantGlobalCloneRow(rows) {
  const list = Array.isArray(rows) ? rows : []
  if (list.length <= 1) return list
  const perRepo = list.filter((row) => {
    const key = String(row?.key || '')
    const label = String(row?.label || '')
    if (key === '__global__') return false
    if (label === '项目克隆' && !looksLikeGitRepoRef(row?.repoUrl || key)) return false
    return true
  })
  return perRepo.length ? perRepo : list
}

/**
 * live SSE 与启动日志按仓库身份合并；live 覆盖同仓字段。
 * 无 live/日志时回退任务级 entries（无 comment 或 sole-active）。
 * @param {{
 *   commentId?: string,
 *   statusLogs?: string[],
 *   byCommentId?: Record<string, object>,
 *   taskLevelEntries?: object[],
 *   soleActiveCommentId?: string,
 * }} opts
 * @returns {CommentCloneProgressRow[]}
 */
export function resolveCommentCloneProgress(opts = {}) {
  const cid = String(opts.commentId || '').trim()
  const byCommentId = opts.byCommentId && typeof opts.byCommentId === 'object' ? opts.byCommentId : {}
  const liveMap = cid && byCommentId[cid] && typeof byCommentId[cid] === 'object' ? byCommentId[cid] : null
  const liveRows = liveMap && Object.keys(liveMap).length ? cloneProgressRowsFromLiveMap(liveMap) : []
  const parsed = parseCommentCloneProgressFromLogs(opts.statusLogs)
  if (liveRows.length && parsed.length) return omitRedundantGlobalCloneRow(mergeCommentCloneProgressRows(parsed, liveRows))
  if (liveRows.length) return omitRedundantGlobalCloneRow(liveRows)
  if (parsed.length) return omitRedundantGlobalCloneRow(parsed)
  const taskRows = cloneProgressRowsFromEntries(opts.taskLevelEntries)
  if (!taskRows.length) return []
  const sole = String(opts.soleActiveCommentId || '').trim()
  if (!cid || (sole && cid === sole)) return omitRedundantGlobalCloneRow(taskRows)
  return []
}

/**
 * 是否存在至少一个评论的 live map 已能驱动评论级克隆进度条。
 * 用于在评论级进度条生效时收起任务级「容器项目克隆进度」横幅（OPT-20260815-019）。
 * @param {Record<string, object|null|undefined>|null|undefined} byCommentId
 * @returns {boolean}
 */
export function hasActiveCommentCloneProgress(byCommentId) {
  if (!byCommentId || typeof byCommentId !== 'object') return false
  return Object.values(byCommentId).some(
    (m) => m && typeof m === 'object' && Object.keys(m).length > 0,
  )
}

const BOOTSTRAP_DONE_RE = /【项目克隆】克隆完成|仓库克隆已完成|【项目克隆】仓库克隆已完成/
const BOOTSTRAP_FAIL_SUMMARY_RE = /【项目克隆】未完成|部分失败|均失败/

/**
 * 容器引导日志已「克隆完成」时，评论条不得仍显示 SSE/启动日志里的中间百分比。
 * @param {CommentCloneProgressRow[]} rows
 * @param {string|null|undefined} bootstrapLog
 * @returns {CommentCloneProgressRow[]}
 */
export function applyBootstrapCloneDoneToRows(rows, bootstrapLog) {
  const text = String(bootstrapLog || '')
  if (!BOOTSTRAP_DONE_RE.test(text) || BOOTSTRAP_FAIL_SUMMARY_RE.test(text)) {
    return Array.isArray(rows) ? rows : []
  }
  const list = Array.isArray(rows) ? rows : []
  if (!list.length) {
    return [enrichRow(emptyRow('__global__', '项目克隆', '【项目克隆】克隆完成。'), {
      progress: 100,
      recvProgress: 100,
      unpackProgress: 100,
      failed: false,
    })]
  }
  return list.map((row) => {
    if (!row || row.failed || row.retrying) return row
    if (clampPct(row.progress) >= 100) return row
    return enrichRow(row, {
      progress: 100,
      recvProgress: 100,
      unpackProgress: 100,
      message: /完成/.test(String(row.message || '')) ? row.message : '【项目克隆】克隆完成。',
    })
  })
}
