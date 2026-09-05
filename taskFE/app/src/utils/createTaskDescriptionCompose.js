/**
 * 创建任务：结构化可选说明 ↔ description Markdown 组装/回解析。
 */

/** @typedef {{ key: string, heading: string, kind?: 'text'|'deps' }} StructuredFieldDef */

/** @type {StructuredFieldDef[]} */
export const CREATE_TASK_STRUCTURED_FIELDS = [
  { key: 'taskBackground', heading: '任务背景', kind: 'text' },
  { key: 'currentProblem', heading: '当前问题是什么', kind: 'text' },
  { key: 'requestedChanges', heading: '需要模型完成什么修改', kind: 'text' },
  { key: 'targetLocation', heading: '目标文件、模块或命令在哪里', kind: 'text' },
  { key: 'preserveBehavior', heading: '需要保持哪些原有行为不变', kind: 'text' },
  { key: 'forbidNewDeps', heading: '是否禁止新增依赖', kind: 'deps' },
  { key: 'suggestedVerification', heading: '建议验证方式', kind: 'text' },
]

const HEADING_TO_FIELD = new Map(
  CREATE_TASK_STRUCTURED_FIELDS.map((f) => [f.heading, f]),
)

/** @returns {Record<string, string>} */
export function emptyCreateTaskStructuredFields() {
  const out = {}
  for (const f of CREATE_TASK_STRUCTURED_FIELDS) {
    out[f.key] = ''
  }
  return out
}

/**
 * @param {unknown} value
 * @returns {''|'yes'|'no'}
 */
export function normalizeForbidNewDeps(value) {
  if (value === 'yes' || value === 'no') return value
  if (typeof value === 'string') {
    const t = value.trim()
    if (t === '是' || t.toLowerCase() === 'yes' || t === 'true') return 'yes'
    if (t === '否' || t.toLowerCase() === 'no' || t === 'false') return 'no'
  }
  return ''
}

/**
 * @param {''|'yes'|'no'|string} value
 * @returns {string}
 */
function formatForbidNewDepsBody(value) {
  const n = normalizeForbidNewDeps(value)
  if (n === 'yes') return '是'
  if (n === 'no') return '否'
  return ''
}

/**
 * @param {{
 *   description?: unknown,
 *   taskBackground?: unknown,
 *   currentProblem?: unknown,
 *   requestedChanges?: unknown,
 *   targetLocation?: unknown,
 *   preserveBehavior?: unknown,
 *   forbidNewDeps?: unknown,
 *   suggestedVerification?: unknown,
 * }} fields
 * @returns {string}
 */
export function composeCreateTaskDescription(fields = {}) {
  const free = fields.description == null ? '' : String(fields.description).trim()
  const sections = []
  for (const def of CREATE_TASK_STRUCTURED_FIELDS) {
    let body = ''
    if (def.kind === 'deps') {
      body = formatForbidNewDepsBody(fields[def.key])
    } else {
      const raw = fields[def.key]
      body = raw == null ? '' : String(raw).trim()
    }
    if (!body) continue
    sections.push(`## ${def.heading}\n${body}`)
  }
  if (!sections.length) return free
  if (!free) return sections.join('\n\n')
  return `${free}\n\n${sections.join('\n\n')}`
}

/**
 * @param {unknown} description
 * @returns {{
 *   description: string,
 *   taskBackground: string,
 *   currentProblem: string,
 *   requestedChanges: string,
 *   targetLocation: string,
 *   preserveBehavior: string,
 *   forbidNewDeps: ''|'yes'|'no',
 *   suggestedVerification: string,
 * }}
 */
export function parseCreateTaskDescription(description) {
  const raw = description == null ? '' : String(description)
  const base = emptyCreateTaskStructuredFields()
  base.forbidNewDeps = ''
  if (!raw.trim()) {
    return { description: '', ...base }
  }

  const headingPattern = /^##\s+(.+?)\s*$/gm
  /** @type {{ heading: string, start: number, headingEnd: number }[]} */
  const matches = []
  let m
  while ((m = headingPattern.exec(raw)) !== null) {
    matches.push({
      heading: m[1].trim(),
      start: m.index,
      headingEnd: m.index + m[0].length,
    })
  }

  const knownIndexes = matches
    .map((item, idx) => ({ item, idx }))
    .filter(({ item }) => HEADING_TO_FIELD.has(item.heading))

  if (!knownIndexes.length) {
    return { description: raw.trim(), ...base }
  }

  const firstKnown = knownIndexes[0].item
  const freePart = raw.slice(0, firstKnown.start).trim()
  const out = { description: freePart, ...base }

  for (let i = 0; i < knownIndexes.length; i += 1) {
    const { item } = knownIndexes[i]
    const def = HEADING_TO_FIELD.get(item.heading)
    if (!def) continue
    const nextStart = i + 1 < knownIndexes.length
      ? knownIndexes[i + 1].item.start
      : raw.length
    // If there are unknown ## headings between known ones, stop body at next ## of any kind
    let bodyEnd = nextStart
    const afterHeading = raw.slice(item.headingEnd, nextStart)
    const nested = afterHeading.search(/\n##\s+/)
    if (nested >= 0) {
      bodyEnd = item.headingEnd + nested
    }
    const body = raw.slice(item.headingEnd, bodyEnd).replace(/^\n+/, '').trim()
    if (def.kind === 'deps') {
      out[def.key] = normalizeForbidNewDeps(body)
    } else {
      out[def.key] = body
    }
  }

  return out
}

/**
 * 将 compose 结果写回 task；结构化字段保留在 draft 上便于继续编辑。
 * 提交前会先剥离 description 中已有的已知段落，避免失败重试时重复追加。
 * @param {Record<string, unknown>} task
 * @returns {Record<string, unknown>}
 */
export function applyComposedDescriptionToTask(task) {
  if (!task || typeof task !== 'object') return task
  const free = parseCreateTaskDescription(task.description).description
  task.description = composeCreateTaskDescription({
    ...task,
    description: free,
  })
  return task
}

/**
 * 从 description 回填结构化字段（编辑场景）。
 * @param {Record<string, unknown>} task
 * @returns {Record<string, unknown>}
 */
export function hydrateStructuredFieldsFromDescription(task) {
  if (!task || typeof task !== 'object') return task
  const parsed = parseCreateTaskDescription(task.description)
  task.description = parsed.description
  for (const def of CREATE_TASK_STRUCTURED_FIELDS) {
    task[def.key] = parsed[def.key]
  }
  return task
}
