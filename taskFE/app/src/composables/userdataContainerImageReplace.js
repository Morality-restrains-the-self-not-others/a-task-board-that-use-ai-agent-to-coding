/**
 * UserData 运行时占位符：须与后端
 * `Saas_project/cloud/userdata_container_image_replace.py` 保持一致。
 *
 * 「模拟替换」仅遍历 USERDATA_RUNTIME_SIMULATE_SLOT_DEFS：当合并预览正文中出现某 slot 的任一 token 时，
 * 界面展示对应输入框；新增占位符时只需在本文件与后端同步增加 slot 定义即可。
 */

export const TASK2APP_CONTAINER_IMAGE_PLACEHOLDER = '__TASK2APP_CONTAINER_IMAGE__'

/** 与生成脚本、后端 replace 使用同一字面值 */
export const TASK2APP_ACCESS_TOKEN_PLACEHOLDER = '__TASK2APP_ACCESS_TOKEN__'

export const TASK2APP_TASK_API_ENDPOINT_PLACEHOLDER = '__TASK2APP_TASK_API_ENDPOINT__'

/** 由 API 根与 tenant/workspace/task[/comment] 派生，与 SaaS machine-container skill 任务云前缀一致 */
export const TASK2APP_TASK_CLOUD_PREFIX_PLACEHOLDER = '__TASK2APP_TASK_CLOUD_PREFIX__'

export const TASK2APP_SSH_PUBLIC_KEY_PLACEHOLDER = '__TASK2APP_SSH_PUBLIC_KEY__'

export const TASK2APP_SSH_MATCH_ADDRESS_PLACEHOLDER = '__TASK2APP_SSH_MATCH_ADDRESS__'

export const TASK2APP_TENANT_ID_PLACEHOLDER = '__TASK2APP_TENANT_ID__'

export const TASK2APP_WORKSPACE_ID_PLACEHOLDER = '__TASK2APP_WORKSPACE_ID__'

export const TASK2APP_TASK_ID_PLACEHOLDER = '__TASK2APP_TASK_ID__'

/** OPT-20260806-053: Django 退役，注释更新 — Vite define 缺省回退本地 API 端点 */
const viteEnv =
  typeof import.meta !== 'undefined' && import.meta.env ? import.meta.env : {}
const DEFAULT_TASK_API_ENDPOINT =
  viteEnv.VITE_USERDATA_VERIFY_BASE_URL != null &&
  String(viteEnv.VITE_USERDATA_VERIFY_BASE_URL).trim() !== ''
    ? String(viteEnv.VITE_USERDATA_VERIFY_BASE_URL).trim()
    : 'http://localhost:8000'

/**
 * 与后端 `normalize_container_image_url_for_docker` 对齐：Docker Hub 网页 URL → docker 镜像引用。
 * @param {string|null|undefined} raw
 * @returns {string}
 */
export function normalizeContainerImageUrlForDocker (raw) {
  const s = raw == null ? '' : String(raw).trim()
  if (!s || !/hub\.docker\.com/i.test(s) || !s.includes('://')) {
    return s
  }
  let parsed
  try {
    parsed = new URL(s)
  } catch {
    return s
  }
  const host = (parsed.hostname || '').toLowerCase()
  if (host !== 'hub.docker.com' && host !== 'www.hub.docker.com') {
    return s
  }
  const segments = decodeURIComponent(parsed.pathname || '')
    .split('/')
    .filter(Boolean)
  if (!segments.length) {
    return s
  }
  const withTagAfterUnderscore = () => {
    if (
      segments.length >= 4 &&
      segments[segments.length - 2] === 'tags' &&
      segments[segments.length - 1]
    ) {
      return `${segments[1]}:${segments[segments.length - 1]}`
    }
    return segments[1]
  }
  if (segments[0] === '_') {
    if (segments.length < 2) return s
    return withTagAfterUnderscore()
  }
  if (segments[0] === 'r' && segments.length >= 3) {
    const ns = segments[1]
    const repo = segments[2]
    const base = ns === 'library' ? repo : `${ns}/${repo}`
    const ti = segments.indexOf('tags')
    if (ti !== -1 && segments[ti + 1]) {
      return `${base}:${segments[ti + 1]}`
    }
    return base
  }
  return s
}

/** @type {readonly string[]} */
export const CONTAINER_IMAGE_PLACEHOLDER_TOKEN_ORDER = Object.freeze([
  '__TASK2APP_CONTAINER_IMAGE__',
  '__TASK2APP_CONTAINER_IMAGE_URL__',
  '{{CONTAINER_IMAGE}}',
  '{{CONTAINER_IMAGE_URL}}',
  '${CONTAINER_IMAGE}',
  '${CONTAINER_IMAGE_URL}',
  '__CONTAINER_IMAGE__',
  '__CONTAINER_IMAGE_URL__',
  'TASK2APP_CONTAINER_IMAGE',
  'TASK2APP_CONTAINER_IMAGE_URL'
])

/**
 * 模拟替换 UI 元数据 + token 列表（顺序即表单项展示顺序）
 * @type {readonly { id: string, label: string, hint: string, defaultValue: string, tokens: readonly string[] }[]}
 */
export const USERDATA_RUNTIME_SIMULATE_SLOT_DEFS = Object.freeze([
  {
    id: 'container_image_url',
    label: '已安装镜像地址',
    hint: '例如 nginx 或 registry.example.com/namespace/image:tag',
    defaultValue: 'nginx',
    tokens: CONTAINER_IMAGE_PLACEHOLDER_TOKEN_ORDER
  },
  {
    id: 'access_token',
    label: 'ACCESS_TOKEN',
    hint: '模拟注入容器的环境变量 ACCESS_TOKEN',
    defaultValue: 'demo-access-token',
    tokens: Object.freeze([TASK2APP_ACCESS_TOKEN_PLACEHOLDER])
  },
  {
    id: 'task_api_endpoint',
    label: 'TaskApiEndPoint（任务 API 基址）',
    hint: 'SaaS API 根（无 /api/tenant 路径），例如 https://api.daydaymoney.com；与 cloud_prefix 不同。预览时若正文误粘浏览器任务详情页 URL，会在有四段上下文时纠错为 cloud_prefix',
    defaultValue: DEFAULT_TASK_API_ENDPOINT,
    tokens: Object.freeze([TASK2APP_TASK_API_ENDPOINT_PLACEHOLDER])
  },
  {
    id: 'tenant_id',
    label: 'tenantId（租户 ID）',
    hint: 'RunInstances 前替换为当前租户（公司）ID，占位符 __TASK2APP_TENANT_ID__',
    defaultValue: '1',
    tokens: Object.freeze([TASK2APP_TENANT_ID_PLACEHOLDER])
  },
  {
    id: 'workspace_id',
    label: 'workspaceId（工作区 ID）',
    hint: '替换为当前工作区 ID，占位符 __TASK2APP_WORKSPACE_ID__',
    defaultValue: 'demo-workspace',
    tokens: Object.freeze([TASK2APP_WORKSPACE_ID_PLACEHOLDER])
  },
  {
    id: 'task_id',
    label: 'taskId（任务 ID）',
    hint: '替换为当前云主机任务 ID，占位符 __TASK2APP_TASK_ID__',
    defaultValue: 'demo-task-id',
    tokens: Object.freeze([TASK2APP_TASK_ID_PLACEHOLDER])
  },
  {
    id: 'ssh_public_key',
    label: 'SSH 公钥（启动时由平台注入）',
    hint: '占位符仅在 RunInstances 前由后端替换；预览可填 ssh-ed25519 AAAA…',
    defaultValue: 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDEMOKEYPLACEHOLDER',
    tokens: Object.freeze([TASK2APP_SSH_PUBLIC_KEY_PLACEHOLDER])
  },
  {
    id: 'ssh_match_address',
    label: 'SSH Match Address 白名单',
    hint: '逗号分隔 IP/CIDR，与 port_config 及请求客户端 IP 合并',
    defaultValue: '203.0.113.1/32',
    tokens: Object.freeze([TASK2APP_SSH_MATCH_ADDRESS_PLACEHOLDER])
  }
])

/**
 * @param {string} plain
 * @returns {string[]}
 */
export function matchedContainerImagePlaceholders (plain) {
  if (plain == null || typeof plain !== 'string') {
    return []
  }
  const found = CONTAINER_IMAGE_PLACEHOLDER_TOKEN_ORDER.filter((t) => plain.includes(t))
  found.sort((a, b) => b.length - a.length)
  return found
}

/**
 * @param {string} plain
 * @returns {typeof USERDATA_RUNTIME_SIMULATE_SLOT_DEFS}
 */
export function activeRuntimeSimulateSlotDefs (plain) {
  const p = plain == null ? '' : String(plain)
  let defs = USERDATA_RUNTIME_SIMULATE_SLOT_DEFS.filter((d) => d.tokens.some((t) => p.includes(t)))
  if (p.includes(TASK2APP_TASK_CLOUD_PREFIX_PLACEHOLDER)) {
    const extraIds = ['task_api_endpoint', 'tenant_id', 'workspace_id', 'task_id']
    const have = new Set(defs.map((d) => d.id))
    for (const id of extraIds) {
      if (!have.has(id)) {
        const def = USERDATA_RUNTIME_SIMULATE_SLOT_DEFS.find((d) => d.id === id)
        if (def) {
          defs.push(def)
          have.add(id)
        }
      }
    }
  }
  return defs
}

/**
 * 与后端 `rewrite_browser_task_detail_url_to_cloud_prefix_in_text` 对齐：
 * 将误粘贴的浏览器任务详情 URL 整块替换为规范 cloud_prefix。
 *
 * @param {string} text
 * @param {string} cloudPrefix
 * @param {string} tenantId
 * @param {string} workspaceId
 * @param {string} taskId
 * @returns {string}
 */
export function rewriteBrowserTaskDetailUrlToCloudPrefixInText (
  text,
  cloudPrefix,
  tenantId,
  workspaceId,
  taskId
) {
  const s = text == null ? '' : String(text)
  const cp = String(cloudPrefix || '').replace(/\/$/, '')
  const tid = String(tenantId || '').trim()
  const wid = String(workspaceId || '').trim()
  const tk = String(taskId || '').trim()
  if (!s || !cp || !tid || !wid || !tk) {
    return s
  }
  const esc = (x) => x.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const pat = new RegExp(
    `https?://[^\\s'"<>]+/tenant/${esc(tid)}/workspace/${esc(wid)}/task-detail/${esc(tk)}/?`,
    'gi'
  )
  return s.replace(pat, cp)
}

function buildReplacementEntries () {
  const entries = []
  for (const def of USERDATA_RUNTIME_SIMULATE_SLOT_DEFS) {
    for (const token of def.tokens) {
      entries.push({ token, slotId: def.id, len: token.length })
    }
  }
  entries.sort((a, b) => b.len - a.len)
  return entries
}

/**
 * @param {Record<string, string>} [seed]
 * @returns {Record<string, string>}
 */
export function buildDefaultUserdataSimulateValues (seed = {}) {
  const out = {}
  for (const def of USERDATA_RUNTIME_SIMULATE_SLOT_DEFS) {
    const s = seed[def.id]
    out[def.id] = s != null && String(s).trim() !== '' ? String(s) : def.defaultValue
  }
  return out
}

/**
 * 与后端 replace_userdata_runtime_placeholders 预览一致：命中占位符且任一所填值为空时不替换。
 * 在已填写 API 根与 tenant/workspace/task 时，对正文中的浏览器任务详情 URL 做与后端一致的整块替换（便于控制台预览与真实 RunInstances 对齐）。
 *
 * @param {string} userDataPlain
 * @param {Record<string, string|undefined|null>} valuesBySlotId
 * @returns {{ text: string, missingSlots: string[], activeSlotIds: string[], matchedLabels: string[] }}
 */
export function replaceUserdataRuntimePlaceholdersForPreview (userDataPlain, valuesBySlotId) {
  const plain = userDataPlain == null ? '' : String(userDataPlain)
  const active = activeRuntimeSimulateSlotDefs(plain)
  const matchedLabels = active.map((d) => d.label)
  const activeSlotIds = active.map((d) => d.id)
  const missingSlots = []
  for (const def of active) {
    const v = valuesBySlotId[def.id]
    if (v == null || String(v).trim() === '') {
      missingSlots.push(def.id)
    }
  }
  if (missingSlots.length) {
    return {
      text: plain,
      missingSlots,
      activeSlotIds,
      matchedLabels
    }
  }
  let replaced = plain
  for (const { token, slotId } of buildReplacementEntries()) {
    let val = String(valuesBySlotId[slotId] ?? '').trim()
    if (slotId === 'container_image_url') {
      val = normalizeContainerImageUrlForDocker(val)
    }
    if (val && replaced.includes(token)) {
      replaced = replaced.split(token).join(val)
    }
  }
  const origin = String(valuesBySlotId.task_api_endpoint ?? '').trim().replace(/\/$/, '')
  const tid = String(valuesBySlotId.tenant_id ?? '').trim()
  const wid = String(valuesBySlotId.workspace_id ?? '').trim()
  const tk = String(valuesBySlotId.task_id ?? '').trim()
  const cid = String(valuesBySlotId.comment_id ?? '').trim()
  const cloudPrefix =
    origin && tid && wid && tk && cid && cid !== '-'
      ? `${origin}/api/tenant/${tid}/workspace/${wid}/task/${tk}/comment/${cid}/cloud`
      : null
  if (cloudPrefix && replaced.includes(TASK2APP_TASK_CLOUD_PREFIX_PLACEHOLDER)) {
    replaced = replaced.split(TASK2APP_TASK_CLOUD_PREFIX_PLACEHOLDER).join(cloudPrefix)
  }
  if (cloudPrefix) {
    replaced = rewriteBrowserTaskDetailUrlToCloudPrefixInText(replaced, cloudPrefix, tid, wid, tk)
  }
  return {
    text: replaced,
    missingSlots: [],
    activeSlotIds,
    matchedLabels
  }
}

/**
 * 与后端 replace_container_image_placeholders 一致：只替换已安装镜像类占位符，其它字面值原样保留。
 *
 * @param {string|null|undefined} userDataPlain
 * @param {string|null|undefined} containerImageUrl
 * @returns {string|null|undefined}
 */
export function replaceContainerImagePlaceholders (userDataPlain, containerImageUrl) {
  if (userDataPlain == null || userDataPlain === '') {
    return userDataPlain
  }
  const matched = matchedContainerImagePlaceholders(userDataPlain)
  if (matched.length === 0) {
    return userDataPlain
  }
  let url = containerImageUrl != null ? String(containerImageUrl).trim() : ''
  if (!url) {
    throw new Error(
      `UserData 包含已安装镜像占位符 ${JSON.stringify(matched)}，但未提供 container_image_url`
    )
  }
  url = normalizeContainerImageUrlForDocker(url)
  let replaced = userDataPlain
  for (const token of matched) {
    replaced = replaced.split(token).join(url)
  }
  return replaced
}

/**
 * @param {string} userDataPlain
 * @param {string} containerImageUrl
 * @returns {{ text: string, missingUrl: boolean, matched: string[] }}
 */
export function replaceContainerImagePlaceholdersForPreview (userDataPlain, containerImageUrl) {
  const plain = userDataPlain == null ? '' : String(userDataPlain)
  const matched = matchedContainerImagePlaceholders(plain)
  if (matched.length === 0) {
    return { text: plain, missingUrl: false, matched: [] }
  }
  let url = containerImageUrl != null ? String(containerImageUrl).trim() : ''
  if (!url) {
    return { text: plain, missingUrl: true, matched }
  }
  url = normalizeContainerImageUrlForDocker(url)
  let replaced = plain
  for (const token of matched) {
    replaced = replaced.split(token).join(url)
  }
  return { text: replaced, missingUrl: false, matched }
}
