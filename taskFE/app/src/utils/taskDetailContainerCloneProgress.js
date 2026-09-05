/**
 * 任务详情页：容器克隆进度 / 引导日志相关的纯函数（无 Vue 依赖，便于单测与复用）。
 */

/**
 * 将 HTTPS、`git@host:path` 与 `ssh://` 规约为可比较 key（host + 路径，无 .git 后缀）
 * @param {string} u
 * @returns {string}
 */
export function gitCloneRefMatchKey(u) {
  const raw = String(u || '').trim()
  if (!raw) return ''
  if (/^git@/i.test(raw)) {
    const m = raw.match(/^git@([^:]+):(.+?)(?:\.git)?\/?$/i)
    if (m) {
      const host = String(m[1]).toLowerCase()
      let p = String(m[2] || '')
        .replace(/\\/g, '/')
        .replace(/\/+$/, '')
        .replace(/\.git$/i, '')
      return `${host}/${p}`.toLowerCase()
    }
  }
  try {
    const x = new URL(raw)
    let path = (x.pathname || '/').replace(/\/+$/, '').replace(/\.git$/i, '')
    if (path.startsWith('/')) path = path.slice(1)
    const host = x.protocol === 'ssh:' ? x.hostname : x.host
    return `${host.toLowerCase()}/${path}`.toLowerCase()
  } catch {
    return raw
      .toLowerCase()
      .replace(/\/+$/, '')
      .replace(/\.git$/i, '')
  }
}

export function shortCloneRepoLabel(repoUrl) {
  const u = String(repoUrl || '').trim()
  if (!u) return ''
  if (/^git@/i.test(u)) {
    const m = u.match(/^git@[^:]+:(.+?)(?:\.git)?\/?$/i)
    if (m) {
      const pathPart = String(m[1] || '').replace(/\\/g, '/').replace(/\/+$/, '')
      if (pathPart) return pathPart
    }
  }
  try {
    const x = new URL(u)
    const pathPart = (x.pathname || '/').replace(/^\/+/, '').replace(/\.git$/i, '')
    return pathPart || u
  } catch {
    return u.length > 56 ? `${u.slice(0, 53)}…` : u
  }
}

/**
 * 与 onlineServiceJS bootstrap 写入格式一致：每仓以「━━ (i/n) <url>」起一段。
 * @param {string} fullText
 * @returns {{ preamble: string, sections: { url: string, text: string }[] }}
 */
const BOOTSTRAP_CLONE_SECTION_FAIL_RE =
  /\[bootstrap-clone\]\s*克隆失败|\[bootstrap-clone\]\s*移入失败|父仓目录不存在，跳过移入|fatal:|Authentication failed|denied|拒绝|unauthorized/i

/** 引导日志页脚「失败仓库」行：`- name — https://…（原因）` */
const BOOTSTRAP_CLONE_FOOTER_FAIL_LINE_RE =
  /^-\s+.+\s+[—–-]\s+(https?:\/\/[^\s（(]+)/i

/**
 * 从引导克隆全文提取失败仓库 URL（段内失败注记 + 页脚汇总行）。
 * @param {string|null|undefined} fullText
 * @returns {string[]}
 */
export function extractBootstrapCloneFailedRepoUrls(fullText) {
  const text = typeof fullText === 'string' ? fullText : ''
  if (!text.trim()) return []

  /** @type {Set<string>} */
  const urls = new Set()

  const addUrl = (raw) => {
    const u = String(raw || '').trim().replace(/\/+$/, '')
    if (u) urls.add(u)
  }

  const { sections } = parseBootstrapCloneLogSections(text)
  for (const section of sections) {
    const body = String(section?.text || '')
    if (BOOTSTRAP_CLONE_SECTION_FAIL_RE.test(body)) {
      addUrl(section.url)
    }
  }

  for (const line of text.split(/\r?\n/)) {
    const m = String(line).trim().match(BOOTSTRAP_CLONE_FOOTER_FAIL_LINE_RE)
    if (m) addUrl(m[1])
  }

  return [...urls]
}

/**
 * SSE / 进度 message 是否表示克隆失败（区别于「克隆完成」）。
 * @param {string|null|undefined} message
 * @returns {boolean}
 */
export function isCloneProgressFailureMessage(message) {
  const text = String(message || '')
  if (!text.trim()) return false
  if (/【项目克隆】克隆完成|仓库克隆已完成|【项目克隆】仓库克隆已完成/.test(text)) {
    return false
  }
  if (/准备第\s*\d+\s*\/\s*\d+\s*次重试/.test(text)) {
    return false
  }
  return /失败|未完成|fatal/i.test(text)
}

/** 重试/重新克隆允许进度从高值回落；普通 SSE 乱序不允许。 */
export function isCloneProgressResetMessage(message) {
  const text = String(message || '')
  if (!text.trim()) return false
  if (/准备第\s*\d+\s*\/\s*\d+\s*次重试/.test(text)) return true
  if (/【重新克隆】/.test(text) && /开始/.test(text)) return true
  return false
}

/**
 * 迟到的低百分比不得覆盖已完成/更高进度（git 进度 POST 无序、SSE 乱序）。
 * @returns {boolean} true 表示应保留 prevProgress
 */
export function shouldKeepCloneProgress(prevProgress, nextProgress, message) {
  const prev = Number(prevProgress)
  const next = Number(nextProgress)
  if (!Number.isFinite(prev) || !Number.isFinite(next)) return false
  if (next >= prev) return false
  if (isCloneProgressFailureMessage(message)) return false
  if (isCloneProgressResetMessage(message)) return false
  return true
}

export function parseBootstrapCloneLogSections(fullText) {
  const text = typeof fullText === 'string' ? fullText : ''
  const lines = text.split(/\r?\n/).map((line) => String(line).replace(/\r$/, ''))
  /** @type {{ url: string, text: string }[]} */
  const sections = []
  /** @type {string[]} */
  const preamble = []
  let currentUrl = /** @type {string | null} */ (null)
  /** @type {string[]} */
  let currentChunk = []
  const flush = () => {
    if (currentUrl) {
      sections.push({ url: currentUrl, text: currentChunk.join('\n').trimEnd() })
      currentUrl = null
      currentChunk = []
    }
  }
  for (const line of lines) {
    const m = line.match(/^\s*━━\s*\(\d+\/\d+\)\s+(.+)$/)
    if (m) {
      flush()
      currentUrl = String(m[1] || '').trim()
      currentChunk.push(line)
    } else if (currentUrl) {
      currentChunk.push(line)
    } else {
      preamble.push(line)
    }
  }
  flush()
  return { preamble: preamble.join('\n').trimEnd(), sections }
}

/** SSE `segment.recv_progress` / `unpack_progress` 与上一状态合并（同相只升不降，避免迟到 9% 覆盖 100%） */
export function mergeCloneProgressSubPhases(prevEntry, seg) {
  const p = prevEntry && typeof prevEntry === 'object' ? prevEntry : {}
  let recv = typeof p.recvProgress === 'number' && Number.isFinite(p.recvProgress) ? p.recvProgress : null
  let unpack = typeof p.unpackProgress === 'number' && Number.isFinite(p.unpackProgress) ? p.unpackProgress : null
  if (seg && typeof seg === 'object') {
    if ('recv_progress' in seg && typeof seg.recv_progress === 'number' && Number.isFinite(seg.recv_progress)) {
      const v = Math.min(100, Math.max(0, seg.recv_progress))
      recv = recv == null ? v : Math.max(recv, v)
    }
    if ('unpack_progress' in seg && typeof seg.unpack_progress === 'number' && Number.isFinite(seg.unpack_progress)) {
      const v = Math.min(100, Math.max(0, seg.unpack_progress))
      unpack = unpack == null ? v : Math.max(unpack, v)
    }
  }
  return { recv, unpack }
}

export function cloneProgressRowHasSubPhases(row) {
  if (!row || typeof row !== 'object') return false
  return row.recvProgress != null || row.unpackProgress != null
}

export function cloneProgressRecvPct(row) {
  if (!row) return 0
  return row.recvProgress != null ? Math.min(100, Math.max(0, row.recvProgress)) : 0
}

export function cloneProgressUnpackPct(row) {
  if (!row) return 0
  return row.unpackProgress != null ? Math.min(100, Math.max(0, row.unpackProgress)) : 0
}
