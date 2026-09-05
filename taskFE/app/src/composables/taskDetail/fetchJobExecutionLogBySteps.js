/**
 * 按 step 分页拉取 container-job-execution-log，避免一次拉全量 job.output / 全量 steps。
 *
 * OPT-20260718-027: 新增 fetchJobFullOutput 支持按需拉取完整 output（"复制日志"按钮用）。
 * 列表 API 默认 output_omitted + meta=0；此函数按需请求 /api/jobs/:id/output 全文。
 */
import { apiFetch } from '../../utils/apiUtils.js'
import { appendCommentIdPath } from '../../utils/containerForwardCommentId.js'
import { normalizeContainerJobExecutionLogPayload } from '../../utils/normalizeContainerJobExecutionLogPayload.js'
import { extractTraceId } from '../../utils/traceId.js'
import { resolveApiErrorMessage } from '../../utils/workPanelFormat.js'

export const JOB_EXEC_LOG_STEP_PAGE_SIZE = 20
export const JOB_EXEC_LOG_MAX_PAGES = 100

function stepNumberOf(step) {
  const n = Number(step?.step_number)
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
}

/**
 * @param {object[]} prev
 * @param {object[]} page
 */
export function mergeAgentStepsByNumber(prev, page) {
  const map = new Map()
  for (const s of prev || []) {
    const n = stepNumberOf(s)
    if (n > 0) map.set(n, s)
    else map.set(`i:${map.size}`, s)
  }
  for (const s of page || []) {
    const n = stepNumberOf(s)
    if (n > 0) map.set(n, s)
    else map.set(`i:${map.size}`, s)
  }
  return [...map.entries()]
    .sort((a, b) => {
      const na = typeof a[0] === 'number' ? a[0] : 1e9
      const nb = typeof b[0] === 'number' ? b[0] : 1e9
      if (na !== nb) return na - nb
      return String(a[0]).localeCompare(String(b[0]))
    })
    .map(([, v]) => v)
}

const JOB_STATUS_TERMINAL = new Set(['completed', 'failed', 'interrupted', 'done'])

function isStepActiveState(step) {
  const st = step?.state
  return st != null && st !== '' && st !== 'completed' && st !== 'error'
}

/**
 * meta=0 时网关 stub 常带 status:""；不得用空串盖掉 jobHint / 已推断的终态。
 * @param {object} base
 * @param {object} remote
 */
export function mergeJobMetaPreserveHint(base, remote) {
  if (!remote || typeof remote !== 'object') return { ...base }
  const out = { ...base, ...remote, output_omitted: true }
  delete out.output
  const remoteSt = remote.status != null ? String(remote.status).trim() : ''
  const baseSt = base?.status != null ? String(base.status).trim() : ''
  if (!remoteSt && baseSt) {
    out.status = baseSt
  }
  const remoteCmd = typeof remote.command === 'string' ? remote.command.trim() : ''
  const baseCmd = typeof base?.command === 'string' ? base.command.trim() : ''
  if (!remoteCmd && baseCmd) {
    out.command = base.command
  }
  return out
}

/**
 * 步骤已全部拉完且存在明确终态、无一活跃时，推断 job 终态。
 * 用于修正：层图仍 running，而代理步骤区因 meta=0 空 status + 步骤已完成显示「已结束」。
 * @param {object[]} steps
 * @param {string} currentStatus
 * @param {{ hasMore?: boolean, totalSteps?: number }} [opts]
 * @returns {string}
 */
export function inferJobStatusFromAgentSteps(steps, currentStatus, opts = {}) {
  const cur = String(currentStatus || '')
    .trim()
    .toLowerCase()
  if (JOB_STATUS_TERMINAL.has(cur)) {
    return cur === 'done' ? 'completed' : cur
  }
  const hasMore = Boolean(opts.hasMore)
  if (hasMore) return currentStatus != null ? String(currentStatus) : ''
  const list = Array.isArray(steps) ? steps : []
  if (!list.length) return currentStatus != null ? String(currentStatus) : ''
  const totalSteps = Number(opts.totalSteps)
  if (Number.isFinite(totalSteps) && totalSteps > list.length) {
    return currentStatus != null ? String(currentStatus) : ''
  }
  let hasExplicitState = false
  let hasError = false
  for (const s of list) {
    if (isStepActiveState(s)) {
      return cur || 'running'
    }
    const st = s?.state
    if (st != null && st !== '') {
      hasExplicitState = true
      if (st === 'error') hasError = true
    }
  }
  // 无任何 state 时不臆造终态（旧轨迹可能缺字段）
  if (!hasExplicitState) {
    return currentStatus != null ? String(currentStatus) : ''
  }
  if (hasError) return 'failed'
  return 'completed'
}

/**
 * @param {object} opts
 * @param {string} opts.apiBase 以 / 结尾的 cloud/compute/ 前缀
 * @param {(qs: URLSearchParams) => string} [opts.buildJobLogUrl] 覆盖默认 `${apiBase}container-job-execution-log/?…`
 * @param {string} opts.jobId
 * @param {string} [opts.layerId]
 * @param {object|null} [opts.jobHint] 层图快照中的 job 元数据（status/command），配合 meta=0
 * @param {number} [opts.afterStep] 仅拉取 step_number > afterStep；默认从 0 拉全部分页
 * @param {AbortSignal} [opts.signal]
 * @param {( partial: object ) => void} [opts.onPartial] 每页合并后回调
 * @returns {Promise<{ ok: true, body: object } | { ok: false, err: unknown, httpStatus?: number, traceId?: string }>}
 */
export async function fetchJobExecutionLogBySteps(opts) {
  const {
    apiBase,
    jobId,
    layerId = '',
    jobHint = null,
    afterStep: startAfter = 0,
    signal,
    onPartial,
    pageSize = JOB_EXEC_LOG_STEP_PAGE_SIZE,
    maxPages = JOB_EXEC_LOG_MAX_PAGES,
    commentId = '',
  } = opts
  const jid = String(jobId || '').trim()
  if (!jid) {
    return { ok: false, err: 'job_id 必填' }
  }

  const lid0 = String(layerId || '').trim()
  let after = Math.max(0, Math.floor(Number(startAfter) || 0))
  let job = {
    id: jid,
    layer_id: lid0 || (jobHint?.layer_id != null ? String(jobHint.layer_id) : null),
    status: jobHint?.status != null ? String(jobHint.status) : '',
    command: typeof jobHint?.command === 'string' ? jobHint.command : '',
    command_kind: jobHint?.command_kind,
    created_at: jobHint?.created_at,
    output_omitted: true,
  }
  let layerChanges = null
  let stepsMeta = {
    note: null,
    trajectory_file: null,
    task: null,
    total_steps: 0,
    has_more: false,
    next_after_step: null,
    after_step: after,
  }
  let mergedSteps = []
  let pages = 0

  while (pages < maxPages) {
    if (signal?.aborted) {
      return { ok: false, err: 'aborted' }
    }
    const qs = new URLSearchParams({
      job_id: jid,
      after_step: String(after),
      limit: String(pageSize),
      // 避免网关再拉可能含十余 MB output 的 GET /jobs/:id；步骤才是展示主数据
      meta: '0',
    })
    if (lid0) qs.set('layer_id', lid0)

    const url = appendCommentIdPath(
      typeof opts.buildJobLogUrl === 'function'
        ? opts.buildJobLogUrl(qs)
        : `${apiBase}container-job-execution-log/?${qs.toString()}`,
      commentId,
    )
    const response = await apiFetch(url, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
      signal,
    })
    if (!response.ok) {
      const j = await response.json().catch(() => ({}))
      return {
        ok: false,
        err: resolveApiErrorMessage(j && typeof j === 'object' ? j : {}, {
          fallback: '拉取任务日志失败',
          httpStatus: response.status,
        }),
        httpStatus: response.status,
        traceId: extractTraceId(response) || extractTraceId(j),
      }
    }
    const raw = await response.json()
    const normalized = normalizeContainerJobExecutionLogPayload(raw) || {}
    if (normalized.job && typeof normalized.job === 'object') {
      job = mergeJobMetaPreserveHint(job, normalized.job)
    }
    if (normalized.layer_changes && typeof normalized.layer_changes === 'object') {
      layerChanges = normalized.layer_changes
    }
    const stepsObj =
      normalized.steps && typeof normalized.steps === 'object' && !Array.isArray(normalized.steps)
        ? normalized.steps
        : { steps: [] }
    const pageSteps = Array.isArray(stepsObj.steps) ? stepsObj.steps : []
    mergedSteps = mergeAgentStepsByNumber(mergedSteps, pageSteps)
    stepsMeta = {
      note: stepsObj.note ?? stepsMeta.note,
      trajectory_file: stepsObj.trajectory_file ?? stepsMeta.trajectory_file,
      task: stepsObj.task ?? stepsMeta.task,
      total_steps:
        typeof stepsObj.total_steps === 'number' ? stepsObj.total_steps : mergedSteps.length,
      has_more: Boolean(stepsObj.has_more),
      next_after_step:
        stepsObj.next_after_step != null ? Number(stepsObj.next_after_step) : null,
      after_step: typeof stepsObj.after_step === 'number' ? stepsObj.after_step : after,
    }

    const inferred = inferJobStatusFromAgentSteps(mergedSteps, job.status, {
      hasMore: Boolean(stepsMeta.has_more),
      totalSteps: stepsMeta.total_steps,
    })
    if (inferred) {
      job = { ...job, status: inferred }
    }

    const pageTraceId = extractTraceId(response) || extractTraceId(raw) || ''
    const body = {
      job,
      steps: {
        ...stepsMeta,
        steps: mergedSteps,
        has_more: stepsMeta.has_more,
        next_after_step: stepsMeta.next_after_step,
      },
      layer_changes: layerChanges,
    }
    if (typeof onPartial === 'function') {
      onPartial(body, pageTraceId)
    }

    if (!stepsMeta.has_more) {
      return { ok: true, body, traceId: pageTraceId }
    }
    const next = stepsMeta.next_after_step
    if (next == null || !Number.isFinite(next) || next <= after) {
      return { ok: true, body, traceId: pageTraceId }
    }
    after = next
    pages += 1
  }

  return {
    ok: true,
    body: {
      job,
      steps: {
        ...stepsMeta,
        steps: mergedSteps,
        note: stepsMeta.note || '已达分页上限，后续步骤请刷新继续拉取',
        has_more: true,
      },
      layer_changes: layerChanges,
    },
    traceId: '',
  }
}

/**
 * 按需拉取 job 完整 output 文本（"复制日志"/"原始控制台"按钮使用）。
 * 列表 API 默认 output_omitted + meta=0；此函数直接请求 /api/jobs/:id/output。
 *
 * @param {{ apiBase: string, jobId: string, signal?: AbortSignal }} opts
 * @returns {Promise<{ ok: boolean, text?: string, err?: string, httpStatus?: number, traceId?: string }>}
 */
export async function fetchJobFullOutput({ apiBase, jobId, signal } = {}) {
  if (!apiBase || !jobId) {
    return { ok: false, err: 'apiBase 和 jobId 必填' }
  }
  try {
    const response = await apiFetch(
      `${apiBase}/jobs/${encodeURIComponent(jobId)}/output`,
      {
        credentials: 'include',
        headers: { Accept: 'text/plain, application/json' },
        signal,
      },
    )
    if (!response.ok) {
      const traceId = extractTraceId(response)
      return { ok: false, err: `HTTP ${response.status}`, httpStatus: response.status, traceId }
    }
    const text = await response.text()
    return { ok: true, text }
  } catch (e) {
    if (e && typeof e === 'object' && 'name' in e && e.name === 'AbortError') {
      return { ok: false, err: 'aborted' }
    }
    return { ok: false, err: String(e?.message ?? e), traceId: extractTraceId(e) }
  }
}
