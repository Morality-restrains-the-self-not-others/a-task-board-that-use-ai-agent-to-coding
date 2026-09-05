function truncateAgentStepText(t, max) {
  const u = String(t || '').trim()
  if (!u) return ''
  return u.length <= max ? u : `${u.slice(0, max - 1)}…`
}

const AGENT_STEP_STATE_LABELS = {
  /** 完成态仅保留打勾；文案由 tool command 替代（见 agentStepCardTitle） */
  completed: '✅',
  error: '❌ error',
  calling_tool: '🔧 calling_tool',
  thinking: '🤔 thinking',
  reflecting: '💭 reflecting',
  /** 轨迹里 record_llm_interaction 已写入、整步 record_agent_step 尚未 finalize */
  llm_interaction: '📝 LLM 交互',
}

function agentStepStateLabel(state) {
  if (state == null || state === '') return '—'
  return AGENT_STEP_STATE_LABELS[state] || String(state)
}

function parseToolCallArguments(raw) {
  if (raw == null) return null
  if (typeof raw === 'object' && !Array.isArray(raw)) return raw
  if (typeof raw !== 'string') return null
  const t = raw.trim()
  if (!t) return null
  try {
    const parsed = JSON.parse(t)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) return parsed
  } catch {
    /* ignore */
  }
  return null
}

/**
 * 从 llm_response.tool_calls[].arguments.command 提取命令（优先）；
 * 兼容步骤根级 tool_calls（finalize 后常见）。
 */
function agentStepToolCallList(s) {
  if (!s || typeof s !== 'object') return []
  const fromLlm = Array.isArray(s.llm_response?.tool_calls) ? s.llm_response.tool_calls : []
  const fromStep = Array.isArray(s.tool_calls) ? s.tool_calls : []
  return fromLlm.length ? fromLlm : fromStep
}

/** 主参数候选：command → path/file → query/pattern → url */
function agentStepToolPrimaryArg(args) {
  if (!args || typeof args !== 'object') return ''
  const keys = [
    'command',
    'path',
    'file_path',
    'filepath',
    'filename',
    'query',
    'pattern',
    'url',
    'uri',
  ]
  for (const key of keys) {
    if (args[key] == null) continue
    const v = String(args[key]).trim()
    if (v) return v
  }
  return ''
}

export function agentStepTitleCommands(s) {
  const calls = agentStepToolCallList(s)
  const out = []
  for (const c of calls) {
    if (!c || typeof c !== 'object') continue
    const args = parseToolCallArguments(c.arguments)
    const cmd = args?.command != null ? String(args.command).trim() : ''
    if (cmd) out.push(cmd)
  }
  return out
}

/**
 * 无 command 时：工具名 + 主参数（path/query 等），避免标题仅剩打勾。
 * @param {object} s
 * @returns {string[]}
 */
export function agentStepTitleToolSummaries(s) {
  const calls = agentStepToolCallList(s)
  const out = []
  for (const c of calls) {
    if (!c || typeof c !== 'object') continue
    const name = c.name != null ? String(c.name).trim() : ''
    const args = parseToolCallArguments(c.arguments)
    if (args?.command != null && String(args.command).trim()) continue
    const primary = agentStepToolPrimaryArg(args)
    if (name && primary) out.push(`${name} ${primary}`)
    else if (name) out.push(name)
    else if (primary) out.push(primary)
  }
  return out
}

export function agentStepCardTitle(s) {
  const sn = s?.step_number != null ? s.step_number : '?'
  let statePart = agentStepStateLabel(s?.state)
  const cmds = agentStepTitleCommands(s)
  if (s?.state === 'completed') {
    if (cmds.length) {
      // 打勾后换行再写 command；多条 command 继续换行合并，不截断（标题需 whitespace-pre-wrap）
      statePart = `✅\n${cmds.join('\n')}`
    } else {
      const summaries = agentStepTitleToolSummaries(s)
      if (summaries.length) {
        statePart = `✅\n${summaries.join('\n')}`
      }
    }
  } else if (cmds.length) {
    // OPT-20260902-022：calling_tool / thinking 等非 completed 步骤已有 command 时，
    // 同样把命令展示在折叠区外的 summary（否则 pre 已清空、标题又只写状态，命令不可见）。
    statePart = `${statePart}\n${cmds.join('\n')}`
  }
  const base = `步骤 ${sn} · ${statePart}`
  if (s?.trajectory_provisional) {
    return `${base} · 预览`
  }
  return base
}

export function agentStepCardSubtitle(s) {
  if (!s || typeof s !== 'object') return ''
  const lr = s.llm_response
  if (lr && lr.content && String(lr.content).trim()) {
    return truncateAgentStepText(lr.content, 120000)
  }
  if (s.lakeview_summary && String(s.lakeview_summary).trim()) {
    return truncateAgentStepText(s.lakeview_summary, 280)
  }
  if (s.delivery_summary && String(s.delivery_summary).trim()) {
    return truncateAgentStepText(s.delivery_summary, 280)
  }
  if (s.reflection && String(s.reflection).trim()) {
    return truncateAgentStepText(s.reflection, 280)
  }
  // 不把 tool_calls 拼进正文：折叠/展开 summary 与 indigo pre 均不再展示 bash 命令行
  return ''
}

function looksLikeHtmlFragment(str) {
  const t = String(str || '').trim()
  if (!t.startsWith('<')) return false
  return /^<[a-z][a-z0-9]*(\s|>)/i.test(t)
}

function looksLikeMarkdown(str) {
  const t = String(str || '')
  if (/^#{1,6}\s/m.test(t)) return true
  if (/^[-*]\s/m.test(t)) return true
  if (/```[\s\S]*```/.test(t)) return true
  return false
}

/**
 * @param {object} s
 * @returns {'text'|'html'|'markdown'}
 */
export function agentStepBodyMode(s) {
  if (!s || typeof s !== 'object') return 'text'
  const lr = s.llm_response
  const ct = String(lr?.content_type || lr?.mime_type || s.content_type || s.mime_type || '').toLowerCase()
  if (ct.includes('html')) return 'html'
  if (ct.includes('markdown') || ct === 'text/md' || ct === 'text/x-markdown') return 'markdown'
  if (typeof s.rich_html === 'string' && s.rich_html.trim()) return 'html'
  if (lr && typeof lr.rich_html === 'string' && lr.rich_html.trim()) return 'html'
  if (s.ui && typeof s.ui.rich_html === 'string' && s.ui.rich_html.trim()) return 'html'
  const content = lr?.content != null ? String(lr.content) : ''
  if (content && looksLikeHtmlFragment(content)) return 'html'
  if (content && looksLikeMarkdown(content)) return 'markdown'
  return 'text'
}

export function agentStepRichHtml(s) {
  if (agentStepBodyMode(s) !== 'html') return ''
  const lr = s.llm_response
  if (typeof s.rich_html === 'string' && s.rich_html.trim()) {
    return truncateAgentStepText(s.rich_html, 120000)
  }
  if (lr && typeof lr.rich_html === 'string' && lr.rich_html.trim()) {
    return truncateAgentStepText(lr.rich_html, 120000)
  }
  if (s.ui && typeof s.ui.rich_html === 'string' && s.ui.rich_html.trim()) {
    return truncateAgentStepText(s.ui.rich_html, 120000)
  }
  if (lr && lr.content && String(lr.content).trim()) {
    return truncateAgentStepText(lr.content, 120000)
  }
  return ''
}

export function agentStepMarkdownSource(s) {
  if (agentStepBodyMode(s) !== 'markdown') return ''
  const lr = s.llm_response
  if (lr && lr.content && String(lr.content).trim()) {
    return truncateAgentStepText(lr.content, 120000)
  }
  return ''
}

/** 纯文本步骤正文（非 html/markdown 时在 indigo pre 中展示） */
export function agentStepPlainBodyForPre(s) {
  if (agentStepBodyMode(s) !== 'text') return ''
  // 命令已展示在标题中，indigo pre 不再重复展示 bash/command 类正文
  if (agentStepTitleCommands(s).length) return ''
  return agentStepCardSubtitle(s)
}

function inferModelCategoryFromName(modelName) {
  const s = String(modelName || '').toLowerCase()
  if (!s) return ''
  if (s.includes('claude')) return 'Anthropic'
  if (s.includes('gpt') || /^o[134]/.test(s) || s.includes('chatgpt')) return 'OpenAI'
  if (s.includes('gemini')) return 'Gemini'
  if (s.includes('deepseek')) return 'DeepSeek'
  if (s.includes('qwen')) return 'Qwen'
  if (s.includes('doubao')) return 'Doubao'
  if (s.includes('glm') || s.includes('zhipu')) return 'GLM'
  if (s.includes('llama')) return 'Llama'
  if (s.includes('mistral')) return 'Mistral'
  return modelName
}

export function agentStepModelName(s) {
  if (!s || typeof s !== 'object') return ''
  const modelRaw =
    s.model || s.model_name || s.llm_model || s?.llm_response?.model || ''
  const model = String(modelRaw || '').trim()
  if (model) return model
  return ''
}

export function agentStepModelBadge(s) {
  if (!s || typeof s !== 'object') return ''
  const categoryRaw =
    s.model_category || s.llm_provider || s.provider || s?.llm_response?.provider || ''
  const category = String(categoryRaw || '').trim()
  if (category) return category
  const modelName = agentStepModelName(s)
  if (modelName) return inferModelCategoryFromName(modelName)
  const model = s?.llm_response?.model
  if (model == null || model === '') return ''
  return inferModelCategoryFromName(model)
}

function parseTokenCount(value) {
  const n = Number(value)
  if (!Number.isFinite(n) || n < 0) return null
  return Math.floor(n)
}

export function agentStepTokenTotal(s) {
  const u = s?.llm_response?.usage || s?.usage
  if (!u || typeof u !== 'object') return null
  const direct = [
    u.total_tokens,
    u.total,
    u.tokens,
    u.token_count,
    u.total_token
  ]
  for (const one of direct) {
    const n = parseTokenCount(one)
    if (n != null) return n
  }
  const input = parseTokenCount(u.input_tokens ?? u.prompt_tokens ?? u.input)
  const output = parseTokenCount(u.output_tokens ?? u.completion_tokens ?? u.output)
  if (input == null && output == null) return null
  return (input || 0) + (output || 0)
}

export function agentStepUsageBadge(s, stepIdx, allSteps) {
  const usage = s?.llm_response?.usage || s?.usage
  if (usage && typeof usage === 'object') {
    const input = parseTokenCount(usage.input_tokens ?? usage.prompt_tokens ?? usage.input)
    const output = parseTokenCount(usage.output_tokens ?? usage.completion_tokens ?? usage.output)
    if (input != null || output != null) {
      return `Token I/O ${input || 0}/${output || 0}`
    }
  }
  const delta = agentStepTokenTotal(s)
  if (delta == null) return ''
  const steps = Array.isArray(allSteps) ? allSteps : []
  if (!steps.length || stepIdx == null || stepIdx < 0) {
    return `本步 +${delta} token`
  }
  let cumulative = 0
  for (let i = 0; i <= stepIdx; i++) {
    const val = agentStepTokenTotal(steps[i])
    if (val != null) cumulative += val
  }
  return `本步 +${delta} · 累计 ${cumulative} token`
}

function stringifyAgentStepResultValue(v) {
  if (v == null) return ''
  if (typeof v === 'string') {
    const max = 120000
    return v.length > max ? `${v.slice(0, max)}\n…(已截断)` : v
  }
  try {
    const text = JSON.stringify(v, null, 2)
    const max = 120000
    return text.length > max ? `${text.slice(0, max)}\n…(已截断)` : text
  } catch {
    return String(v)
  }
}

/**
 * 折叠卡片上的 tool_calls 短预览（name + hint）。
 * @param {object|null|undefined} s
 * @param {{ hintMax?: number }} [opts]
 * @returns {Array<{ name: string, hint: string }>}
 */
export function agentStepToolCallsPreview(s, opts = {}) {
  if (!s || typeof s !== 'object') return []
  const calls = Array.isArray(s.tool_calls) ? s.tool_calls : []
  if (!calls.length) return []
  const hintMax = Number.isFinite(opts.hintMax) ? Number(opts.hintMax) : 120
  const truncate = (raw) => {
    const t = String(raw || '').trim()
    if (!t) return ''
    if (t.length <= hintMax) return t
    if (hintMax <= 1) return '…'
    return `${t.slice(0, Math.max(1, hintMax - 1))}…`
  }
  const rows = []
  for (const c of calls) {
    if (!c || typeof c !== 'object') continue
    const name = c.name ? String(c.name) : 'tool'
    const args = c.arguments && typeof c.arguments === 'object' ? c.arguments : {}
    const hint = truncate(
      args.command ?? args.path ?? args.file_path ?? args.filepath ?? '',
    )
    rows.push({ name, hint })
  }
  return rows
}

export function agentStepToolResultDisplay(s) {
  if (!s || typeof s !== 'object') return []
  const lines = []
  const calls = Array.isArray(s.tool_calls) ? s.tool_calls : []
  const results = Array.isArray(s.tool_results) ? s.tool_results : []
  if (calls.length) {
    for (const c of calls) {
      if (!c) continue
      const callId = c.call_id != null ? String(c.call_id) : ''
      const name = c.name ? String(c.name) : 'tool'
      const tr = results.find((r) => r && String(r.call_id || '') === callId)
      if (!tr) continue
      const parts = []
      const val = stringifyAgentStepResultValue(tr.result)
      if (val) parts.push(val)
      if (tr.error != null && String(tr.error).trim()) parts.push(String(tr.error))
      if (parts.length) {
        lines.push(`${name}\n${parts.join('\n')}`)
      }
    }
  }
  if (!lines.length && results.length) {
    for (const tr of results) {
      if (!tr) continue
      const name = tr.call_id ? `call ${tr.call_id}` : 'tool'
      const parts = []
      const val = stringifyAgentStepResultValue(tr.result)
      if (val) parts.push(val)
      if (tr.error != null && String(tr.error).trim()) parts.push(String(tr.error))
      if (parts.length) lines.push(`${name}\n${parts.join('\n')}`)
    }
  }
  return lines
}

export function agentStepJsonPretty(s) {
  if (!s || typeof s !== 'object') return ''
  try {
    const text = JSON.stringify(s, null, 2)
    const max = 120000
    return text.length > max ? `${text.slice(0, max)}\n…(已截断)` : text
  } catch {
    return ''
  }
}

/**
 * 代理步骤列表的稳定 key（兼作复制反馈键）。
 * 流式执行中 step_number 常从缺失变为有值；若 key 含 step_number，Vue 会拆毁重建
 * 整张卡片（含富文本 iframe），执行日志区表现为持续抖动。
 * 优先用后端稳定 id；否则用下标（轨迹步骤以追加为主）。
 */
export function agentStepCopyKey(step, stepIdx) {
  if (step?.id != null && String(step.id).trim() !== '') {
    return `id-${String(step.id).trim()}`
  }
  if (step?.step_id != null && String(step.step_id).trim() !== '') {
    return `sid-${String(step.step_id).trim()}`
  }
  const idx = typeof stepIdx === 'number' && Number.isFinite(stepIdx) ? stepIdx : 0
  return `idx-${idx}`
}
