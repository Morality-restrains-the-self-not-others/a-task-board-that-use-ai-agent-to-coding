import { normalizeProjectTags, PROJECT_TAG_MAX_LENGTH, PROJECT_TAGS_MAX_COUNT } from './projectTagsUtils.js'

const FORBIDDEN_TOP_KEYS = new Set([
  'workspace_id',
  'workspace_ids',
  'project_id',
  'project_ids',
  'company_id',
  'tenant_id',
])

const SERVICE_ID_RE = /^[A-Za-z][A-Za-z0-9_-]{1,63}$/

function stripYamlComment(line) {
  let inSingle = false
  let inDouble = false
  for (let i = 0; i < line.length; i += 1) {
    const c = line[i]
    if (c === "'" && !inDouble) inSingle = !inSingle
    if (c === '"' && !inSingle) inDouble = !inDouble
    if (c === '#' && !inSingle && !inDouble) return line.slice(0, i).trimEnd()
  }
  return line
}

function parseYamlScalar(raw) {
  const s = String(raw ?? '').trim()
  if (!s) return ''
  if ((s.startsWith('"') && s.endsWith('"')) || (s.startsWith("'") && s.endsWith("'"))) {
    return s.slice(1, -1)
  }
  if (s === 'true') return true
  if (s === 'false') return false
  if (/^-?\d+$/.test(s)) return Number(s)
  return s
}

function parseSimpleYaml(text) {
  const lines = String(text ?? '').split(/\r?\n/)
  const root = {}
  let currentKey = null
  let listItems = null

  for (const rawLine of lines) {
    const line = stripYamlComment(rawLine)
    if (!line.trim()) continue

    const listMatch = line.match(/^\s+-\s+(.*)$/)
    if (listMatch && currentKey && listItems) {
      listItems.push(parseYamlScalar(listMatch[1]))
      continue
    }

    const kvMatch = line.match(/^([A-Za-z_][A-Za-z0-9_-]*)\s*:\s*(.*)$/)
    if (!kvMatch) {
      throw new Error(`无法解析 YAML 行: ${line.trim()}`)
    }

    const key = kvMatch[1]
    const rest = kvMatch[2]

    if (rest === '') {
      currentKey = key
      listItems = []
      root[key] = listItems
    } else {
      currentKey = null
      listItems = null
      root[key] = parseYamlScalar(rest)
    }
  }

  return root
}

function parseTagsList(value) {
  if (!value) return []
  if (Array.isArray(value)) {
    return value.map((t) => String(t ?? '').trim()).filter(Boolean)
  }
  if (typeof value === 'string') {
    return value.split(',').map((t) => t.trim()).filter(Boolean)
  }
  return []
}

function assertTagConstraints(tags) {
  if (!tags.length) {
    throw new Error('daydaymoney.yaml tags 不能为空')
  }
  if (tags.length > PROJECT_TAGS_MAX_COUNT) {
    throw new Error(`daydaymoney.yaml tags 最多 ${PROJECT_TAGS_MAX_COUNT} 个`)
  }
  for (const tag of tags) {
    if (tag.length > PROJECT_TAG_MAX_LENGTH) {
      throw new Error(`daydaymoney.yaml 标签过长（>${PROJECT_TAG_MAX_LENGTH}）: ${tag}`)
    }
  }
}

/**
 * 解析 daydaymoney.yaml 文本。
 * @returns {{ service_id: string, tags: string[], display_name: string }}
 */
export function parseDaydaymoneyYaml(text) {
  const root = parseSimpleYaml(text)

  for (const key of Object.keys(root)) {
    if (FORBIDDEN_TOP_KEYS.has(key)) {
      throw new Error(`daydaymoney.yaml 禁止顶层键: ${key}`)
    }
  }

  const version = root.version
  if (version !== undefined && version !== 1 && version !== '1') {
    throw new Error('daydaymoney.yaml version 必须为 1')
  }

  const service_id = String(root.service_id ?? '').trim()
  if (!SERVICE_ID_RE.test(service_id)) {
    throw new Error('daydaymoney.yaml service_id 格式无效')
  }

  let tags = normalizeProjectTags(parseTagsList(root.tags))
  const svcTag = `svc:${service_id}`
  if (!tags.some((t) => t.toLowerCase() === svcTag.toLowerCase())) {
    tags = normalizeProjectTags([svcTag, ...tags])
  }
  assertTagConstraints(tags)

  const display_name = root.display_name != null ? String(root.display_name).trim() : ''

  return { service_id, tags, display_name }
}

/** 将 YAML 元信息 tags 合并进现有项目 tags（去重、规范化） */
export function mergeDaydaymoneyTags(existing, metaTags, serviceId) {
  const base = normalizeProjectTags(Array.isArray(existing) ? existing : [])
  const incoming = normalizeProjectTags(parseTagsList(metaTags))
  const svcTag = `svc:${String(serviceId ?? '').trim()}`
  const combined = [...base]

  for (const tag of incoming) {
    if (!combined.some((x) => x.toLowerCase() === tag.toLowerCase())) {
      combined.push(tag)
    }
  }
  if (svcTag.length > 4 && !combined.some((x) => x.toLowerCase() === svcTag.toLowerCase())) {
    combined.unshift(svcTag)
  }

  return normalizeProjectTags(combined)
}

/** 将 tags 数组转为 HTML meta content（逗号分隔） */
export function metaTagsToHeaderContent(tags) {
  return normalizeProjectTags(Array.isArray(tags) ? tags : []).join(',')
}
