/**
 * 镜像内 /app/imageSkills.yaml 绑定结果的前端读取与高亮。
 * 列表第一项为默认技能。
 */

const SKILL_NAME_RE = /^[a-z0-9][a-z0-9-]{0,62}$/

export function normalizeImageSkillList(raw) {
  const src = raw && typeof raw === 'object' && !Array.isArray(raw) ? raw : {}
  const skillsIn = Array.isArray(src.skills) ? src.skills : []
  const skills = []
  const seen = new Set()
  for (let i = 0; i < skillsIn.length; i += 1) {
    const item = skillsIn[i] || {}
    const name = String(item.name || '').trim()
    if (!SKILL_NAME_RE.test(name) || seen.has(name)) continue
    seen.add(name)
    skills.push({
      // id 为 D1=B 服务端派生的稳定技能 ID（sk_<hash>）；目录老数据可能缺失 → 空串
      id: String(item.id || '').trim(),
      name,
      description: String(item.description || '').trim(),
      isDefault: i === 0 || item.is_default === true,
    })
  }
  if (skills.length) {
    skills.forEach((s, i) => { s.isDefault = i === 0 })
  }
  const defaultSkill = skills.length ? skills[0].name : ''
  return {
    version: Number(src.version) || 1,
    defaultSkill,
    skills,
  }
}

export function imageSkillsFromImage(image) {
  if (!image || typeof image !== 'object') {
    return normalizeImageSkillList(null)
  }
  return normalizeImageSkillList(image.image_skills)
}

export function shouldShowImageSkills(image) {
  const list = imageSkillsFromImage(image)
  if (list.skills.length) return true
  const status = String(image?.image_skills_extract_status || '').trim()
  return status === 'pending' || status === 'failed' || status.startsWith('auth_failed')
}

export function skillToken(name) {
  return `/${String(name || '').trim()}`
}

/**
 * 找出 caret 前进行中的 /query（$mention 确认后、空白分隔）。
 * @returns {{ start: number, query: string } | null}
 */
export function findActiveSlashSkillQuery(text, caret) {
  const s = String(text || '')
  const pos = Math.max(0, Math.min(Number(caret) || 0, s.length))
  const before = s.slice(0, pos)
  const slash = before.lastIndexOf('/')
  if (slash < 0) return null
  if (slash > 0) {
    const prev = before[slash - 1]
    if (prev && !/\s/.test(prev)) return null
  }
  const query = before.slice(slash + 1)
  if (/\s/.test(query)) return null
  if (query && !/^[a-z0-9-]*$/.test(query)) return null
  return { start: slash, query }
}

export function filterImageSkills(skills, query) {
  const q = String(query || '').trim().toLowerCase()
  const list = Array.isArray(skills) ? skills : []
  if (!q) return list.slice()
  return list.filter((s) => {
    const name = String(s?.name || '').toLowerCase()
    const desc = String(s?.description || '').toLowerCase()
    return name.includes(q) || desc.includes(q)
  })
}

/**
 * 将描述中匹配所选镜像技能的 /name 标记为高亮区间。
 * @returns {{ text: string, highlight: boolean }[]}
 */
export function highlightSkillTokens(text, skillNames) {
  const s = String(text || '')
  const names = (Array.isArray(skillNames) ? skillNames : [])
    .map((n) => String(n || '').trim())
    .filter((n) => SKILL_NAME_RE.test(n))
  if (!s || !names.length) {
    return s ? [{ text: s, highlight: false }] : []
  }
  const escaped = names
    .slice()
    .sort((a, b) => b.length - a.length)
    .map((n) => n.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
  const re = new RegExp(`/(${escaped.join('|')})(?![a-z0-9-])`, 'g')
  const parts = []
  let last = 0
  let m
  while ((m = re.exec(s)) !== null) {
    if (m.index > last) {
      parts.push({ text: s.slice(last, m.index), highlight: false })
    }
    parts.push({ text: m[0], highlight: true })
    last = m.index + m[0].length
  }
  if (last < s.length) {
    parts.push({ text: s.slice(last), highlight: false })
  }
  return parts
}

export function insertSkillToken(text, caret, skillName) {
  const s = String(text || '')
  const pos = Math.max(0, Math.min(Number(caret) || 0, s.length))
  const token = skillToken(skillName)
  const active = findActiveSlashSkillQuery(s, pos)
  if (active) {
    const next = s.slice(0, active.start) + token + s.slice(pos)
    return { text: next, caret: active.start + token.length }
  }
  const needsSpace = pos > 0 && !/\s/.test(s[pos - 1])
  const insert = (needsSpace ? ' ' : '') + token
  const next = s.slice(0, pos) + insert + s.slice(pos)
  return { text: next, caret: pos + insert.length }
}

export function extractSkillFromPlainText(text, skills) {
  const names = new Set(
    (Array.isArray(skills) ? skills : [])
      .map((s) => String(s?.name || '').trim())
      .filter((n) => SKILL_NAME_RE.test(n)),
  )
  if (!names.size) return ''
  const re = /(^|\s)\/([a-z0-9][a-z0-9-]{0,62})(?![a-z0-9-])/g
  const s = String(text || '')
  let m
  while ((m = re.exec(s)) !== null) {
    if (names.has(m[2])) return m[2]
  }
  return ''
}
