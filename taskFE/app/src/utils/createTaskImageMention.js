/**
 * 创建任务：任务描述中的 镜像 $/ 技能选择 往返纯逻辑（无 DOM 依赖，便于单测）。
 *
 * 契约（与评论区 $镜像 /技能 一致）：
 * - mention 文本形如 `$trae-agent /dev`，其中技能可省略；
 * - 描述 textarea（TaskDescriptionSkillField）直接写入 mention 文本，
 *   提交链路 strip 旧 mention 后在描述「自由段」（首个 `##` 结构化段落之前）归一化，
 *   编辑回填时描述即展示 mention 原文，保证 round-trip；
 * - 镜像的 id 单独落到 editingTask.container_image.id（提交 payload 的 container_image_id 契约不变）。
 */
import { extractSkillFromPlainText, imageSkillsFromImage } from './imageSkills.js'
import { matchImageByExactName } from '../composables/taskDetail/commentImageMentionComposer.js'

/** @typedef {{ id: string, name: string, skill: string }} ImageMentionDraft */

const MENTION_TOKEN_RE = /^\$([^\s$]+)(?:\s+\/([a-z0-9][a-z0-9-]{0,62}))?/

/**
 * 从任意文本解析镜像 mention（对照已安装列表）与技能标记。
 * @param {string} text
 * @param {{ id: string, name: string }[]} images
 * @returns {ImageMentionDraft|null}
 */
export function parseImageMentionFromText(text, images) {
  const list = Array.isArray(images) ? images : []
  const s = String(text || '')
  const re = /(^|\s)\$([^\s$]+)/g
  let m
  while ((m = re.exec(s)) !== null) {
    const hit = matchImageByExactName(list, m[2])
    if (hit) {
      const img = list.find((item) => String(item.id) === String(hit.id)) || null
      const skills = imageSkillsFromImage(img).skills
      const skill = extractSkillFromPlainText(s, skills)
      return { id: String(hit.id), name: String(hit.name || ''), skill }
    }
  }
  return null
}

/**
 * 从描述自由段剥离镜像 mention token。
 * 只剥离「行首/空白前导的 $name」且满足下列之一：
 * - name 命中已安装镜像（正常 round-trip）；
 * - token 位于自由段开头（历史任务/镜像已卸载时兜底，避免重复累加）。
 * 不触碰正文中段的用户自写 $ 文本。
 * @param {string} description
 * @param {{ id: string, name: string }[]} [images]
 * @returns {string}
 */
export function stripImageMentionFromDescription(description, images = []) {
  const raw = String(description || '')
  const free = raw.split(/^##\s+/m)[0]
  const at = free.indexOf('$')
  if (at < 0) return raw
  if (at > 0) {
    const prev = free[at - 1]
    if (prev !== '\n' && !/\s/.test(prev)) return raw
  }
  const m = free.slice(at).match(MENTION_TOKEN_RE)
  if (!m) return raw
  const isInstalled = matchImageByExactName(images, m[1]) !== null
  if (!isInstalled && at > 0) return raw
  const stripped = free.slice(0, at) + free.slice(at + m[0].length)
  const cleaned = stripped.replace(/^\n{1,2}/, '').replace(/\s*\n{2,}$/, '\n').trimEnd()
  const tail = raw.slice(free.length).replace(/^\n+/, '')
  if (!cleaned) return tail
  return tail ? `${cleaned}\n\n${tail}` : cleaned
}

/**
 * 提交前把字段 mention 文本合并进任务描述自由段：
 * - 先剥离旧 mention token，再在自由段开头置入新文本（有正文时以空行分隔）。
 * @param {string} description
 * @param {string} mentionText
 * @param {{ id: string, name: string }[]} [images]
 * @returns {string}
 */
export function applyImageMentionToDescription(description, mentionText, images = []) {
  const raw = String(description || '')
  const base = stripImageMentionFromDescription(raw, images)
  const token = String(mentionText || '').trim()
  if (!token) return base
  if (!base.trim()) return token
  return `${token}\n\n${base.trim()}`
}

/**
 * 提交前把 task 草稿上的镜像 mention（container_image.mentionText）合并进描述。
 * @param {Record<string, unknown>|null|undefined} task
 * @param {{ id: string, name: string }[]} [images]
 * @returns {Record<string, unknown>}
 */
export function applyImageMentionToTaskDescription(task, images = []) {
  if (!task || typeof task !== 'object') return task
  const mentionText = String(task.container_image?.mentionText || '').trim()
  task.description = applyImageMentionToDescription(task.description, mentionText, images)
  return task
}

/**
 * 把 mention 文本同步到 task 草稿的 container_image（提交绑定契约）：
 * - mention 命中已安装镜像 → 写 id / mentionText / skill；
 * - 同名多变体时显式选中 id 优先（$name 文本反解会拿错变体）；
 * - 未命中 → 不改动 task，返回 mention: null，是否清空由调用方决定。
 * 供「已安装镜像」字段与描述 textarea 两处输入源共用，保证单一绑定来源。
 * @param {Record<string, unknown>|null|undefined} task
 * @param {string} mentionText
 * @param {{ id: string, name: string }[]} [images]
 * @param {string|number} [explicitId] 用户显式选中的镜像 id
 * @returns {{ mention: ImageMentionDraft|null, resolved: object|null }}
 */
export function syncTaskContainerImageFromMention(task, mentionText, images = [], explicitId = '') {
  if (!task || typeof task !== 'object') return { mention: null, resolved: null }
  if (!task.container_image || typeof task.container_image !== 'object') {
    task.container_image = { id: null }
  }
  const list = Array.isArray(images) ? images : []
  const mention = parseImageMentionFromText(mentionText, list)
  if (!mention?.id) return { mention: null, resolved: null }
  const explicit =
    explicitId != null && String(explicitId).trim() !== ''
      ? list.find((img) => String(img.id) === String(explicitId)) || null
      : null
  const resolved = explicit || list.find((img) => String(img.id) === String(mention.id)) || null
  if (!resolved) return { mention: null, resolved: null }
  task.container_image.id = String(resolved.id)
  task.container_image.mentionText = String(mentionText || '').trim()
  task.container_image.skill = extractSkillFromPlainText(
    String(mentionText || ''),
    imageSkillsFromImage(resolved).skills,
  )
  return { mention, resolved }
}

/**
 * 提交前把任务草稿上的技能名解析为稳定 ID（D1=B），写入 container_image.skillId。
 * 输入源（mention 文本 / chips 点击）只产生技能名；ID 由所选镜像目录 image_skills 反查。
 * 目录无 id（老数据）或技能名不在列表时置空 —— 服务端按描述 /token 反解兜底，不阻断提交。
 * @param {Record<string, unknown>|null|undefined} task
 * @param {{ id: string, name: string }[]} [images]
 * @returns {boolean} whether a container_image object was present to bind
 */
export function bindTaskContainerImageSkillId(task, images = []) {
  if (!task || typeof task !== 'object') return false
  const ci = task.container_image
  if (!ci || typeof ci !== 'object') return false
  const imageId = String(ci.id || '').trim()
  const skillName = String(ci.skill || '').trim().replace(/^\//, '')
  if (!imageId || !skillName) {
    ci.skillId = ''
    return true
  }
  const img = (Array.isArray(images) ? images : []).find((it) => String(it.id) === imageId) || null
  const hit = (imageSkillsFromImage(img).skills).find((s) => String(s.name) === skillName) || null
  ci.skillId = hit?.id ? String(hit.id) : ''
  return true
}
