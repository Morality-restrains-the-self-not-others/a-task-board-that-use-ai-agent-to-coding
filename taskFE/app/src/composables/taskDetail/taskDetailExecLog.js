/**
 * Pure function: refreshZTreeExecutionLog for TaskDetail.
 * 任务日志按 step 分页拉取，避免整包 job.output 过大导致超时。
 */
import { apiFetch } from '../../utils/apiUtils.js'
import { extractTraceId } from '../../utils/traceId.js'
import { resolveApiErrorMessage } from '../../utils/workPanelFormat.js'
import { userFacingContainerComputeFailure } from '../../utils/containerComputeWaiting.js'
import { containerComputeFuncFirstUrl } from './containerComputeRequest.js'
import {
  fetchJobExecutionLogBySteps,
  mergeAgentStepsByNumber,
} from './fetchJobExecutionLogBySteps.js'
import { patchLayerGraphJobStatus } from './updateServerStatus.js'
import { resolveContainerUiContextCommentId } from './resolveContainerUiContextCommentId.js'
import { isCommentExecutionReleased } from './useCommentExecutionContext.js'

function readFlag(v) {
  if (v && typeof v === 'object' && 'value' in v) return Boolean(v.value)
  return Boolean(v)
}

/** 服务器已释放 / 非服务：跳过打容器的 clone-log 与层变动，但仍拉 SaaS job 日志（COS step_full）。 */
export function shouldSkipZTreeExecLogFetch(deps) {
  if (!deps) return false
  if (readFlag(deps.containerReleased) || readFlag(deps.serverRuntimeNotServing)) return true
  const commentId = String(
    (deps.commentId && typeof deps.commentId === 'object' && 'value' in deps.commentId
      ? deps.commentId.value
      : deps.commentId) ||
    (deps.effectiveCommentId && typeof deps.effectiveCommentId === 'object' && 'value' in deps.effectiveCommentId
      ? deps.effectiveCommentId.value
      : deps.effectiveCommentId) ||
    '',
  ).trim()
  const binding = typeof deps.bindingStatusFor === 'function' ? deps.bindingStatusFor(commentId) : ''
  const runtime = typeof deps.runtimeStatusFor === 'function' ? deps.runtimeStatusFor(commentId) : ''
  return isCommentExecutionReleased(binding, runtime)
}

export async function refreshZTreeExecutionLog(opts, deps) {
  const {
    selectedLayerGraphNode, layerGraphSnapshot,
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    layerExecLogLoading, layerExecLogTopError, layerCloneLogText,
    layerCloneLogFetchError, layerCloneLogFetchErrorTraceId,
    layerJobLogFetchError, layerJobLogFetchErrorTraceId, layerJobExecutionPayload,
    layerChangesByLayerId, ingestLayerChangesFromExecutionPayload,
    markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
    resolveZTreeLogTargets,
  } = deps
  const reset = Boolean(opts && opts.reset)
  if (deps.layerLogAbortController) { deps.layerLogAbortController.abort() }
  const ac = new AbortController()
  deps.layerLogAbortController = ac
  const skipContainerCloneLog = shouldSkipZTreeExecLogFetch(deps)
  if (skipContainerCloneLog) {
    // 容器已释放：clone-log 打容器会 409；SaaS job 日志（COS step_full）仍可复查，不清空。
    layerCloneLogText.value = ''
    layerCloneLogFetchError.value = ''
    if (layerCloneLogFetchErrorTraceId) layerCloneLogFetchErrorTraceId.value = ''
  }
  const node = selectedLayerGraphNode.value
  if (!node || node.nodeKind === 'virtual' || node.nodeKind === 'cycle') {
    layerExecLogTopError.value = ''; layerCloneLogText.value = ''
    layerCloneLogFetchError.value = ''; layerJobLogFetchError.value = ''
    if (layerCloneLogFetchErrorTraceId) layerCloneLogFetchErrorTraceId.value = ''
    if (layerJobLogFetchErrorTraceId) layerJobLogFetchErrorTraceId.value = ''
    layerJobExecutionPayload.value = null; layerExecLogLoading.value = false
    deps.layerLogAbortController = null; return
  }
  const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
  const taskId = effectiveTaskId.value
  if (!tenantId || !workspaceId || !taskId) { layerExecLogLoading.value = false; deps.layerLogAbortController = null; return }
  if (reset) {
    layerExecLogTopError.value = ''
    layerCloneLogText.value = ''
    layerCloneLogFetchError.value = ''
    layerJobLogFetchError.value = ''
    if (layerCloneLogFetchErrorTraceId) layerCloneLogFetchErrorTraceId.value = ''
    if (layerJobLogFetchErrorTraceId) layerJobLogFetchErrorTraceId.value = ''
    layerJobExecutionPayload.value = null
  }
  const commentId = resolveContainerUiContextCommentId(deps)
  if (!commentId) {
    layerExecLogLoading.value = false
    layerExecLogTopError.value = '缺少评论ID'
    deps.layerLogAbortController = null
    return
  }
  const jobs = layerGraphSnapshot.value?.jobs
  const { layerId, jobId } = resolveZTreeLogTargets(node, jobs)
  if (!layerId && !jobId) { layerExecLogLoading.value = false; deps.layerLogAbortController = null; return }
  if (reset) { layerExecLogLoading.value = true }
  const fmtErr = (e) => (typeof e === 'string' ? e : JSON.stringify(e))

  /** 增量刷新：已有 steps 时从末步继续拉 */
  let afterStep = 0
  const prevPayload =
    !reset && jobId && layerJobExecutionPayload.value?.job?.id === jobId
      ? layerJobExecutionPayload.value
      : null
  if (prevPayload) {
    const existing = prevPayload?.steps?.steps
    if (Array.isArray(existing) && existing.length) {
      let max = 0
      for (const s of existing) {
        const n = Number(s?.step_number)
        if (Number.isFinite(n) && n > max) max = Math.floor(n)
      }
      afterStep = max
    }
  }

  const applyJobBody = (partial, fetchTraceId = '') => {
    if (ac.signal.aborted) return
    const mergedBody = { ...partial }
    const cachedLayerChanges = layerId ? layerChangesByLayerId.value[layerId] : null
    if (cachedLayerChanges && !mergedBody.layer_changes) {
      mergedBody.layer_changes = cachedLayerChanges
    }
    if (prevPayload?.steps?.steps?.length) {
      mergedBody.steps = {
        ...mergedBody.steps,
        steps: mergeAgentStepsByNumber(prevPayload.steps.steps, mergedBody.steps?.steps || []),
      }
    }
    ingestLayerChangesFromExecutionPayload(mergedBody, fetchTraceId)
    layerJobExecutionPayload.value = mergedBody
    layerJobLogFetchError.value = ''
    if (layerJobLogFetchErrorTraceId) layerJobLogFetchErrorTraceId.value = ''
    // 执行日志已确认终态时回写层图，避免 zTree 标题仍显示 running 而代理步骤显示「已结束」
    const st = String(mergedBody?.job?.status || '')
      .trim()
      .toLowerCase()
    const jid = mergedBody?.job?.id != null ? String(mergedBody.job.id) : jobId
    if (
      jid &&
      (st === 'completed' || st === 'failed' || st === 'interrupted') &&
      layerGraphSnapshot
    ) {
      patchLayerGraphJobStatus(layerGraphSnapshot, jid, st)
    }
  }

  try {
    const fetchOpts = { credentials: 'include', headers: { Accept: 'application/json' }, signal: ac.signal }
    const tasks = []
    if (layerId && !skipContainerCloneLog) {
      tasks.push(apiFetch(
        containerComputeFuncFirstUrl(
          tenantId,
          workspaceId,
          taskId,
          'container-clone-log',
          commentId,
          `layer_id=${encodeURIComponent(layerId)}`,
        ),
        fetchOpts,
      )
        .then(async (r) => {
          if (!r.ok) {
            const j = await r.json().catch(() => ({}))
            return {
              kind: 'clone',
              ok: false,
              err: resolveApiErrorMessage(j && typeof j === 'object' ? j : {}, {
                fallback: '拉取克隆日志失败',
                httpStatus: r.status,
              }),
              httpStatus: r.status,
              traceId: extractTraceId(r) || extractTraceId(j),
            }
          }
          const j = await r.json()
          return { kind: 'clone', ok: true, text: typeof j.text === 'string' ? j.text : '' }
        }))
    }
    if (jobId) {
      const jobHint = Array.isArray(jobs)
        ? jobs.find((j) => j && String(j.id) === String(jobId)) || null
        : null
      tasks.push(
        fetchJobExecutionLogBySteps({
          buildJobLogUrl: (qs) =>
            containerComputeFuncFirstUrl(
              tenantId,
              workspaceId,
              taskId,
              'container-job-execution-log',
              commentId,
              qs.toString(),
            ),
          jobId,
          layerId: layerId || '',
          jobHint,
          afterStep,
          signal: ac.signal,
          onPartial: applyJobBody,
          commentId,
        }).then((r) => ({ kind: 'job', ...r })),
      )
    }
    const results = await Promise.all(tasks)
    if (ac.signal.aborted) return
    for (const res of results) {
      if (res.kind === 'clone') {
        if (res.ok) {
          layerCloneLogText.value = res.text || ''
          layerCloneLogFetchError.value = ''
          if (layerCloneLogFetchErrorTraceId) layerCloneLogFetchErrorTraceId.value = ''
        } else {
          const mapped = userFacingContainerComputeFailure(res.err, res.httpStatus)
          layerCloneLogFetchError.value = mapped.message || fmtErr(res.err)
          if (layerCloneLogFetchErrorTraceId) {
            layerCloneLogFetchErrorTraceId.value = extractTraceId(res.traceId) || ''
          }
          if (mapped.tone !== 'waiting') {
            markContainerTransportUnreachableIfForwardingFailed(res.httpStatus, fmtErr(res.err))
          }
        }
      } else if (res.kind === 'job') {
        if (res.ok) {
          applyJobBody(res.body, res.traceId || '')
        } else if (res.err !== 'aborted') {
          const mapped = userFacingContainerComputeFailure(res.err, res.httpStatus)
          layerJobLogFetchError.value = mapped.message || fmtErr(res.err)
          if (layerJobLogFetchErrorTraceId) {
            layerJobLogFetchErrorTraceId.value = extractTraceId(res.traceId) || ''
          }
          if (mapped.tone !== 'waiting') {
            markContainerTransportUnreachableIfForwardingFailed(res.httpStatus, fmtErr(res.err))
          }
        }
      }
    }
    if (results.length > 0 && results.every((r) => r.ok)) { markContainerTransportOk() }
  } catch (e) {
    if (e?.name === 'AbortError') return
    console.error('拉取 zTree 执行日志失败', e)
    layerExecLogTopError.value = '拉取执行日志失败'
  }
  finally { if (!ac.signal.aborted) { layerExecLogLoading.value = false }; if (deps.layerLogAbortController === ac) { deps.layerLogAbortController = null } }
}
