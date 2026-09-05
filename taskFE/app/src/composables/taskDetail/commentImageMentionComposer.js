/**
 * 评论区 $镜像 内联 composer 纯逻辑（无 DOM 依赖，便于单测）。
 * 契约：一条评论最多一个 installed_image mention；正文保留 `$显示名`。
 */

/**
 * 在 caret 前查找进行中的 $query（以空白或行首为起点，query 不含空白）。
 * @param {string} text
 * @param {number} caret
 * @returns {{ start: number, query: string } | null}
 */
export function findActiveAtQuery(text, caret) {
  const s = String(text || '')
  const pos = Math.max(0, Math.min(Number(caret) || 0, s.length))
  const before = s.slice(0, pos)
  const at = before.lastIndexOf('$')
  if (at < 0) return null
  if (at > 0) {
    const prev = before[at - 1]
    if (prev && !/\s/.test(prev)) return null
  }
  const query = before.slice(at + 1)
  if (/\s/.test(query)) return null
  return { start: at, query }
}

/**
 * @param {{ id: string, name: string }[]} images
 * @param {string} query
 */
export function filterInstalledImages(images, query) {
  const q = String(query || '').trim().toLowerCase()
  const list = Array.isArray(images) ? images : []
  if (!q) return list.slice()
  return list.filter((img) => {
    const name = String(img?.name || '').toLowerCase()
    const id = String(img?.id || '').toLowerCase()
    return name.includes(q) || id.includes(q)
  })
}

/**
 * 空格确认：query 与镜像名精确匹配（忽略大小写）时返回该镜像。
 * @param {{ id: string, name: string }[]} images
 * @param {string} query
 */
export function matchImageByExactName(images, query) {
  const q = String(query || '').trim().toLowerCase()
  if (!q) return null
  const list = Array.isArray(images) ? images : []
  return list.find((img) => String(img?.name || '').trim().toLowerCase() === q) || null
}

/**
 * 将 $query 替换为 `$name`（两侧保留原有空白结构）。
 * @param {string} text
 * @param {number} atStart
 * @param {number} caret
 * @param {string} imageName
 * @returns {{ text: string, caret: number }}
 */
export function replaceAtQueryWithMention(text, atStart, caret, imageName) {
  const s = String(text || '')
  const start = Math.max(0, Number(atStart) || 0)
  const end = Math.max(start, Math.min(Number(caret) || 0, s.length))
  const name = String(imageName || '').trim()
  const mention = `$${name}`
  const next = s.slice(0, start) + mention + s.slice(end)
  const nextCaret = start + mention.length
  return { text: next, caret: nextCaret }
}

/**
 * 从纯文本中解析第一个 `$镜像名`（空白分隔），并对照已安装列表。
 * @param {string} text
 * @param {{ id: string, name: string }[]} images
 * @returns {{ id: string, name: string } | null}
 */
export function extractMentionFromPlainText(text, images) {
  const s = String(text || '')
  const list = Array.isArray(images) ? images : []
  if (!list.length) return null
  const re = /(^|\s)\$([^\s$]+)/g
  let m
  while ((m = re.exec(s)) !== null) {
    const token = m[2]
    const hit = matchImageByExactName(list, token)
    if (hit) {
      return { id: String(hit.id), name: String(hit.name || '') }
    }
  }
  return null
}

/**
 * contenteditable 序列化：mention chip → `$name`，其余取 text。
 * @param {ParentNode | null | undefined} root
 */
export function serializeComposerDom(root) {
  if (!root) return ''
  let out = ''
  const walk = (node) => {
    if (!node) return
    if (node.nodeType === 3) {
      out += node.nodeValue || ''
      return
    }
    if (node.nodeType !== 1) return
    const el = /** @type {HTMLElement} */ (node)
    if (el.dataset && el.dataset.mentionId) {
      const name = el.dataset.mentionName || el.textContent || ''
      out += `$${String(name).replace(/^\$/, '')}`
      return
    }
    if (el.tagName === 'BR') {
      out += '\n'
      return
    }
    for (const child of el.childNodes) walk(child)
    if (el.tagName === 'DIV' || el.tagName === 'P') {
      if (out.length && !out.endsWith('\n')) out += '\n'
    }
  }
  for (const child of root.childNodes) walk(child)
  return out.replace(/\n$/, '')
}

/**
 * @param {ParentNode | null | undefined} root
 * @returns {{ id: string, name: string } | null}
 */
export function extractMentionFromDom(root) {
  if (!root || typeof root.querySelector !== 'function') return null
  const el = root.querySelector('[data-mention-id]')
  if (!el) return null
  const id = String(el.getAttribute('data-mention-id') || '').trim()
  if (!id) return null
  const name = String(el.getAttribute('data-mention-name') || el.textContent || '')
    .replace(/^\$/, '')
    .trim()
  return { id, name }
}
