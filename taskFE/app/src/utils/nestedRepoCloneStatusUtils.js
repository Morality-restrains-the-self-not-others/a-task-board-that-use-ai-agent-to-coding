/**
 * 任务详情「关联项目」：子仓库克隆状态推导（纯函数，无 Vue 依赖）。
 */

import { parseBootstrapCloneLogSections } from './taskDetailContainerCloneProgress.js'

/**
 * @typedef {'waiting_container'|'idle'|'queued'|'running'|'done'|'error'|'unknown'} NestedRepoCloneStatusKind
 * @typedef {{
 *   kind: NestedRepoCloneStatusKind,
 *   label: string,
 *   progress: number,
 *   message: string,
 * }} NestedRepoCloneStatus
 */

/**
 * 从 cloneProgress Map/Object 按仓库 URL 取进度行。
 * @param {Map|Object|null|undefined} progressByMatchKey
 * @param {(url: string) => string} gitCloneRefMatchKey
 * @param {string} repoUrl
 * @returns {object|null}
 */
export function lookupCloneProgressEntry(progressByMatchKey, gitCloneRefMatchKey, repoUrl) {
  const url = String(repoUrl || '').trim()
  if (!url || typeof gitCloneRefMatchKey !== 'function') return null
  const key = gitCloneRefMatchKey(url)
  if (!key) return null
  if (progressByMatchKey instanceof Map) {
    return progressByMatchKey.get(key) || null
  }
  if (progressByMatchKey && typeof progressByMatchKey === 'object') {
    return progressByMatchKey[key] || null
  }
  return null
}

/**
 * 从引导克隆日志（segments 或全文分段）按仓库 URL 取该仓段落正文。
 * @param {string|null|undefined} bootstrapLogFull
 * @param {Array<{ repo_url?: string, url?: string, text?: string }>|null|undefined} bootstrapLogSegments
 * @param {(url: string) => string} gitCloneRefMatchKey
 * @param {string} repoUrl
 * @returns {string}
 */
export function lookupBootstrapLogSectionText(
  bootstrapLogFull,
  bootstrapLogSegments,
  gitCloneRefMatchKey,
  repoUrl,
) {
  const url = String(repoUrl || '').trim()
  if (!url || typeof gitCloneRefMatchKey !== 'function') return ''
  const want = gitCloneRefMatchKey(url)
  if (!want) return ''
  if (Array.isArray(bootstrapLogSegments) && bootstrapLogSegments.length) {
    for (const s of bootstrapLogSegments) {
      const ru = s && (s.repo_url != null ? s.repo_url : s.url)
      if (gitCloneRefMatchKey(String(ru || '')) === want) {
        return String(s.text || '').trim()
      }
    }
  }
  const full = String(bootstrapLogFull || '')
  if (!full.trim()) return ''
  const { sections } = parseBootstrapCloneLogSections(full)
  for (const s of sections) {
    if (gitCloneRefMatchKey(s.url) === want) {
      return String(s.text || '').trim()
    }
  }
  return ''
}

/**
 * 仅根据引导日志段落推断子仓状态（无 SSE 进度行时使用）。
 * nested 子仓成功后会 staging→移入父仓目录，进度 Map 常在全局完成后被清空，
 * 但日志段仍含「已移入」/失败注记，可据此恢复展示。
 * @param {string} sectionText
 * @returns {NestedRepoCloneStatus|null} 无法判断时返回 null
 */
export function resolveNestedRepoCloneStatusFromLogSection(sectionText) {
  const text = String(sectionText || '').trim()
  if (!text) return null
  const failHint =
    /\[bootstrap-clone\]\s*克隆失败|\[bootstrap-clone\]\s*移入失败|父仓目录不存在，跳过移入|fatal:|Authentication failed|denied|拒绝|unauthorized/i.test(
      text,
    )
  if (failHint) {
    return {
      kind: 'error',
      label: '克隆失败',
      progress: 0,
      message: text,
    }
  }
  if (/\[bootstrap-clone\]\s*已移入|已移入\s+\S+|Receiving objects:\s*100%/i.test(text)) {
    return { kind: 'done', label: '已完成', progress: 100, message: '' }
  }
  return null
}

/**
 * @param {object|null|undefined} progressEntry
 * @param {{
 *   containerReady?: boolean,
 *   bootstrapCloneDone?: boolean,
 *   bootstrapLogSectionText?: string,
 * }} [opts]
 * @returns {NestedRepoCloneStatus}
 */
export function resolveNestedRepoCloneStatus(progressEntry, opts = {}) {
  const containerReady = Boolean(opts.containerReady)
  const bootstrapCloneDone = Boolean(opts.bootstrapCloneDone)
  const entry = progressEntry && typeof progressEntry === 'object' ? progressEntry : null
  const message = entry && typeof entry.message === 'string' ? entry.message : ''
  const progressRaw = entry != null ? Number(entry.progress) : NaN
  const progress = Number.isFinite(progressRaw) ? Math.min(100, Math.max(0, progressRaw)) : 0

  if (entry) {
    // 失败文案优先于 progress>=100，避免「失败但仍显示已完成」
    const failHint = /失败|error|failed|fatal|denied|拒绝|unauthorized|未完成/i.test(message)
    if (failHint) {
      return {
        kind: 'error',
        label: '克隆失败',
        progress: progress >= 100 ? 0 : progress,
        message,
      }
    }
    if (progress >= 100) {
      return { kind: 'done', label: '已完成', progress: 100, message: message || '已完成' }
    }
    if (progress <= 0 && !message.trim()) {
      return { kind: 'queued', label: '排队中', progress: 0, message }
    }
    return { kind: 'running', label: '克隆中', progress, message }
  }

  const fromLog = resolveNestedRepoCloneStatusFromLogSection(opts.bootstrapLogSectionText)
  if (fromLog) return fromLog

  if (bootstrapCloneDone && containerReady) {
    return { kind: 'done', label: '已完成', progress: 100, message: '' }
  }
  if (!containerReady) {
    return { kind: 'waiting_container', label: '等待容器', progress: 0, message: '' }
  }
  return { kind: 'idle', label: '未开始', progress: 0, message: '' }
}

/**
 * 从克隆进度 message 抽取可读错误（供失败行展示）。
 * @param {string} message
 * @returns {string}
 */
export function formatNestedRepoCloneErrorDetail(message) {
  const raw = String(message || '').replace(/\s+/g, ' ').trim()
  if (!raw) return ''
  const bootstrapFail = raw.match(/\[bootstrap-clone\]\s*克隆失败[:：]\s*(.+)$/i)
  if (bootstrapFail) return bootstrapFail[1].trim().slice(0, 280)
  const numberedFail = raw.match(/失败\s+[^:]+:\s*(.+)$/i)
  if (numberedFail) return numberedFail[1].trim().slice(0, 280)
  const afterColon = raw.match(/(?:克隆失败|failed|fatal)[:：]\s*(.+)$/i)
  if (afterColon) return afterColon[1].trim().slice(0, 280)
  return raw.slice(0, 280)
}

/**
 * 子仓行是否展示「重新克隆」。
 * @param {NestedRepoCloneStatusKind|string} kind
 * @param {{ containerReady?: boolean, hasUrl?: boolean }} [opts]
 * @returns {boolean}
 */
export function shouldShowNestedRepoRecloneButton(kind, opts = {}) {
  const hasUrl = opts.hasUrl !== false
  if (!hasUrl) return false
  if (kind === 'error') return true
  if (kind === 'idle' && Boolean(opts.containerReady)) return true
  return false
}

/**
 * 状态徽章 Tailwind class（与任务详情关联仓徽章风格一致）。
 * @param {NestedRepoCloneStatusKind} kind
 * @returns {string}
 */
export function nestedRepoCloneStatusBadgeClass(kind) {
  switch (kind) {
    case 'done':
      return 'border-emerald-300 bg-emerald-50 text-emerald-800'
    case 'running':
    case 'queued':
      return 'border-sky-300 bg-sky-50 text-sky-800'
    case 'error':
      return 'border-red-300 bg-red-50 text-red-800'
    case 'waiting_container':
      return 'border-gray-200 bg-gray-50 text-gray-500'
    case 'idle':
    default:
      return 'border-amber-300 bg-amber-50 text-amber-800'
  }
}

/**
 * @param {Array<{ url?: string, path?: string }>} nestedRepos
 * @param {Map|Object|null|undefined} progressByMatchKey
 * @param {(url: string) => string} gitCloneRefMatchKey
 * @param {{
 *   containerReady?: boolean,
 *   bootstrapCloneDone?: boolean,
 *   bootstrapLogFull?: string,
 *   bootstrapLogSegments?: Array<{ repo_url?: string, url?: string, text?: string }>|null,
 * }} [opts]
 * @returns {{ total: number, done: number, running: number, error: number, idle: number }}
 */
export function summarizeNestedRepoCloneStatuses(
  nestedRepos,
  progressByMatchKey,
  gitCloneRefMatchKey,
  opts = {},
) {
  const rows = Array.isArray(nestedRepos) ? nestedRepos : []
  const summary = { total: 0, done: 0, running: 0, error: 0, idle: 0 }
  for (const row of rows) {
    const url = String(row?.url || '').trim()
    if (!url) continue
    summary.total += 1
    const entry = lookupCloneProgressEntry(progressByMatchKey, gitCloneRefMatchKey, url)
    const bootstrapLogSectionText = lookupBootstrapLogSectionText(
      opts.bootstrapLogFull,
      opts.bootstrapLogSegments,
      gitCloneRefMatchKey,
      url,
    )
    const st = resolveNestedRepoCloneStatus(entry, { ...opts, bootstrapLogSectionText })
    if (st.kind === 'done') summary.done += 1
    else if (st.kind === 'error') summary.error += 1
    else if (st.kind === 'running' || st.kind === 'queued') summary.running += 1
    else summary.idle += 1
  }
  return summary
}
