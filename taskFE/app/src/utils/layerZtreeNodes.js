import { layerPushErrorFields } from './layerZtreePushError.js'
import {
  bootstrapAnchorDisplayCommand,
  bootstrapAnchorZtStyle,
  layerZtreeBootstrapAnchorFields,
} from './layerZtreeBootstrapAnchor.js'

export const LAYER_GRAPH_ROOT_ID = '__layer_graph_root__'
// 500-line rule exception: layerZtreeNodes is 819 lines. See taskDetailCloneProgress.js for split progress.

export const LAYER_TREE_NODE_PREFIX = '__layer__:'


function layerNodeId(layerId) {
  return LAYER_TREE_NODE_PREFIX + layerId
}


function createdAtMs(iso) {
  const m = Date.parse(iso || '')
  return Number.isFinite(m) ? m : 0
}

/** 与容器任务卡门控一致：pending / running 视为进行中（大小写不敏感） */

function jobStatusIsActive(raw) {
  const v = String(raw || '').trim().toLowerCase()
  return v === 'pending' || v === 'running'
}


function dedupeLayersById(layers) {
  const by = new Map()
  for (const l of layers || []) {
    const id = l && l.layer_id
    if (!id) continue
    if (!by.has(id)) {
      by.set(id, Object.assign({}, l))
    } else {
      const cur = by.get(id)
      for (const k of Object.keys(l)) {
        const v = l[k]
        if (v !== undefined && v !== null && v !== '') cur[k] = v
      }
    }
  }
  return Array.from(by.values())
}


function normalizeParentLayerId(layer) {
  if (!layer || typeof layer !== 'object') return ''
  const candidates = [
    layer.parent_layer_id,
    layer.parentLayerId,
    layer.parent_id,
    layer.parentId,
    layer.parent,
  ]
  for (const raw of candidates) {
    if (raw == null) continue
    const v = String(raw).trim()
    if (v) return v
  }
  return ''
}


function sortZTreeLayerSiblings(siblings, bootstrapLayerId) {
  const bs =
    bootstrapLayerId && siblings.some((r) => r.layer_id === bootstrapLayerId)
      ? bootstrapLayerId
      : ''
  siblings.sort((a, b) => {
    const sa = bs && a.layer_id === bs ? 1 : 0
    const sb = bs && b.layer_id === bs ? 1 : 0
    if (sa !== sb) return sb - sa
    const da = createdAtMs(a.created_at)
    const db = createdAtMs(b.created_at)
    if (da !== db) return da - db
    return String(a.layer_id || '').localeCompare(String(b.layer_id || ''))
  })
}

/** 串行视图：全部可写层按 created_at 旧→新（同秒内 bootstrap 层优先）。与 onlineService static/index.html 一致。 */

function sortLayersSerialChronological(layerList, bootstrapLayerId) {
  const bs =
    bootstrapLayerId && layerList.some((r) => r.layer_id === bootstrapLayerId)
      ? bootstrapLayerId
      : ''
  return [...layerList].sort((a, b) => {
    const da = createdAtMs(a.created_at)
    const db = createdAtMs(b.created_at)
    if (da !== db) return da - db
    const sa = bs && a.layer_id === bs ? 1 : 0
    const sb = bs && b.layer_id === bs ? 1 : 0
    if (sa !== sb) return sb - sa
    return String(a.layer_id || '').localeCompare(String(b.layer_id || ''))
  })
}


function buildJobGraphMaps(jobs, layerById) {
  const byId = new Map(jobs.map((j) => [j.id, j]))
  const childMap = new Map()
  for (const j of jobs) {
    const p = j.parent_job_id
    if (p && byId.has(p)) {
      if (!childMap.has(p)) childMap.set(p, [])
      childMap.get(p).push(j)
    }
  }
  for (const arr of childMap.values()) {
    arr.sort((a, b) => (a.created_at || '').localeCompare(b.created_at || ''))
  }
  function effectiveClean(job) {
    const L = layerById.get(job.layer_id)
    if (!L) return false
    if (L.git_worktree_dirty === false) return true
    if (L.git_worktree_dirty === null) return false
    const kids = childMap.get(job.id) || []
    for (const k of kids) {
      if (effectiveClean(k)) return true
    }
    return false
  }
  return { byId, childMap, effectiveClean }
}


function normalizeLayerMindApiFields(layers, jobs) {
  if (!layers || !layers.length) return
  const layerIdHasActiveJob = (layerId) =>
    (jobs || []).some((j) => j.layer_id === layerId && jobStatusIsActive(j.status))
  for (let i = 0; i < layers.length; i++) {
    const l = layers[i]
    if (!l.mind_state) {
      l.mind_state = layerIdHasActiveJob(l.layer_id) ? 'running' : 'idle_done'
    }
    if (typeof l.queue_depth !== 'number') l.queue_depth = 0
  }
}

/**
 * 当该层已有非 clone 任务且无一处于 pending/running，却仍携带进行中的 layer.job_status / mind_state
 * 时，与任务列表对齐，避免 zTree 层级行滞后于执行日志（常见于 SSE 与 GET 快照字段不同步）。
 * @param {object[]} layerList
 * @param {Map<string, object[]>} jobsByLayer
 */

function reconcileLayerTerminalDisplayWithJobs(layerList, jobsByLayer) {
  for (const layer of layerList || []) {
    const lid = layer?.layer_id
    if (!lid) continue
    const raw = jobsByLayer.get(lid) || []
    const visible = raw.filter((j) => j.command_kind !== 'clone')
    if (!visible.length) continue
    if (visible.some((j) => jobStatusIsActive(j.status))) continue

    const layerJobSaysActive = jobStatusIsActive(layer.job_status)
    const mind = String(layer.mind_state || '').trim().toLowerCase()
    const layerMindSaysActive = jobStatusIsActive(layer.mind_state) || mind === 'running'

    if (!layerJobSaysActive && !layerMindSaysActive) continue

    const sorted = [...visible].sort((a, b) => {
      const da = createdAtMs(a.created_at)
      const db = createdAtMs(b.created_at)
      if (da !== db) return da - db
      return String(a.id || '').localeCompare(String(b.id || ''))
    })
    const last = sorted[sorted.length - 1]
    const fallback = String(last?.status || '').trim() || 'completed'

    if (layerJobSaysActive) {
      layer.job_status = fallback
    }
    if (layerMindSaysActive) {
      layer.mind_state = 'idle_done'
    }
  }
}

/** zTree 层级编号展示：取完整 id 字符串前 4 位（与 snowflake 等大整数 id 一致） */

function ztreeIdPrefix4(id) {
  const s = String(id ?? '')
  return s.length >= 4 ? s.slice(0, 4) : s
}

/**
 * 层级标题单行：${运行状态} ${时间} ${层级编号前4位} ${指令内容}
 * @param {string} status
 * @param {string} timeIso
 * @param {string|number} hierarchyId
 * @param {string} [command]
 */

function ztreeHierarchyDisplayName(status, timeIso, hierarchyId, command) {
  const st = status || '—'
  const t = timeIso || ''
  const id4 = ztreeIdPrefix4(hierarchyId)
  let cmd = typeof command === 'string' ? command : ''
  if (cmd.length > 36) cmd = cmd.slice(0, 36) + '…'
  return `${st} ${t} ${id4} ${cmd}`.replace(/\s+/g, ' ').trim()
}


function ztreeHierarchyTooltip(status, timeIso, fullHierarchyId, command) {
  const lines = [
    `状态: ${status || '—'}`,
    `时间: ${timeIso || '—'}`,
    `编号: ${fullHierarchyId ?? '—'}`,
  ]
  if (command) lines.push(`指令: ${command}`)
  return lines.join('\n')
}


function parseTokenCount(value) {
  const n = Number(value)
  if (!Number.isFinite(n) || n < 0) return null
  return Math.floor(n)
}


function parseJobModelList(job) {
  if (!job || typeof job !== 'object') return []
  const arr = []
  const pushOne = (v) => {
    const s = String(v ?? '').trim()
    if (!s || arr.includes(s)) return
    arr.push(s)
  }
  if (Array.isArray(job.llm_models)) {
    for (const one of job.llm_models) pushOne(one)
  }
  pushOne(job.llm_model)
  pushOne(job.model)
  pushOne(job.model_name)
  return arr
}


function parseJobTotalTokens(job) {
  if (!job || typeof job !== 'object') return null
  const direct = [
    job.llm_total_tokens,
    job.total_tokens,
    job.token_count,
    job.tokens,
  ]
  for (const one of direct) {
    const n = parseTokenCount(one)
    if (n != null) return n
  }
  return null
}


function jobInlineMeta(job) {
  const parts = []
  const models = parseJobModelList(job)
  const totalTokens = parseJobTotalTokens(job)
  if (models.length > 0) {
    parts.push(`模型:${models.join('/')}`)
  }
  if (totalTokens != null) {
    parts.push(`token:${totalTokens}`)
  }
  return parts.join(' ')
}


export function normalizeLayerGitDirty(rawDirty) {
  if (rawDirty === true || rawDirty === false || rawDirty === null) {
    return rawDirty
  }
  if (typeof rawDirty === 'string') {
    const v = rawDirty.trim().toLowerCase()
    if (!v || v === 'null' || v === 'none') return null
    if (v === 'true' || v === '1' || v === 'yes') return true
    if (v === 'false' || v === '0' || v === 'no') return false
    return null
  }
  if (typeof rawDirty === 'number') {
    if (rawDirty === 1) return true
    if (rawDirty === 0) return false
  }
  return null
}

/** 与 onlineService 层快照 ``git_remote``（``git rev-list --count @{u}..HEAD`` 聚合）一致 */

function normalizeLayerGitRemote(layer) {
  if (!layer || typeof layer !== 'object') return null
  const r = layer.git_remote
  if (!r || typeof r !== 'object') return null
  return r
}

/** 层当前检出分支（优先 ``git_remote.current_branch``） */
function resolveLayerCurrentBranch(layer) {
  const gr = normalizeLayerGitRemote(layer)
  if (gr && typeof gr.current_branch === 'string') {
    const b = gr.current_branch.trim()
    if (b) return b
  }
  if (layer && typeof layer.current_branch === 'string') {
    const b = layer.current_branch.trim()
    if (b) return b
  }
  return ''
}

/**
 * 当前 HEAD 是否已在合并目标分支（无需再点「合并到目标分支」）。
 * 未知当前分支或 detached HEAD 时返回 false（保持展示按钮）。
 */
function isLayerOnMergeTarget(layer, mergeTargetBranch) {
  const target = typeof mergeTargetBranch === 'string' ? mergeTargetBranch.trim() : ''
  if (!target) return false
  const cur = resolveLayerCurrentBranch(layer)
  if (!cur || cur === 'HEAD') return false
  return cur === target
}

/** @returns {number|null} */

function coerceGitAhead(raw) {
  if (typeof raw === 'number' && Number.isFinite(raw) && raw >= 0) return Math.floor(raw)
  if (typeof raw === 'string' && raw.trim() !== '') {
    const n = parseInt(raw, 10)
    if (Number.isFinite(n) && n >= 0) return n
  }
  return null
}

/** 推送按钮前展示：相对上游可推送的提交数（与层快照 git_remote.ahead 一致） */

function formatLayerPushAheadLabel(layer) {
  const gr = normalizeLayerGitRemote(layer)
  if (!gr || gr.is_git === false) return ''
  if (gr.no_upstream === true) return '无上游'
  const ahead = coerceGitAhead(gr.ahead)
  if (ahead === null) return ''
  if (ahead > 0) return `${ahead} 个提交可推送`
  const pushed = coerceGitAhead(gr.last_pushed_count)
  if (pushed !== null && pushed > 0) return `${pushed} 个提交已推送`
  return ''
}

/** @returns {boolean} 仅当存在可推送的未推送提交时为 true */

function layerHasUnpushedCommits(layer) {
  const gr = normalizeLayerGitRemote(layer)
  if (!gr || gr.is_git === false || gr.no_upstream === true) return false
  const ahead = coerceGitAhead(gr.ahead)
  return ahead !== null && ahead > 0
}

/** @returns {string} 层快照中的 PR 审查页 URL（推送并创建 PR 成功后写入） */
function layerPrHtmlUrl(layer) {
  const gr = normalizeLayerGitRemote(layer)
  if (!gr) return ''
  return typeof gr.pr_html_url === 'string' ? gr.pr_html_url.trim() : ''
}

/**
 * @param {object[]} layers
 * @param {object[]} jobs
 * @param {{ bootstrapLayerId?: string, mergeTargetBranch?: string }} [options]
 */

function createLayerGraphNodeContext(layers, jobs, options = {}) {
  const bootstrapLayerId = options.bootstrapLayerId || ''
  const mergeTargetBranch =
    typeof options.mergeTargetBranch === 'string' ? options.mergeTargetBranch.trim() : ''
  // OPT-20260822-001(3)：per-comment 容器已释放时，git 动作按钮一律禁用并给出明确文案。
  const containerReleased = options.containerReleased === true
  const layerList = dedupeLayersById(layers)
  normalizeLayerMindApiFields(layerList, jobs)
  const layerById = new Map(layerList.map((l) => [l.layer_id, l]))
  const maps = buildJobGraphMaps(jobs || [], layerById)

  const jobsByLayer = new Map()
  const jobApiOrder = new Map()
  for (let i = 0; i < (jobs || []).length; i++) {
    const j = jobs[i]
    if (j && j.id && !jobApiOrder.has(j.id)) jobApiOrder.set(j.id, i)
  }
  for (const j of jobs || []) {
    if (!layerById.has(j.layer_id)) continue
    if (!jobsByLayer.has(j.layer_id)) jobsByLayer.set(j.layer_id, [])
    jobsByLayer.get(j.layer_id).push(j)
  }
  for (const arr of jobsByLayer.values()) {
    arr.sort((a, b) => {
      const da = createdAtMs(a.created_at)
      const db = createdAtMs(b.created_at)
      if (da !== db) return da - db
      return (jobApiOrder.get(a.id) ?? 0) - (jobApiOrder.get(b.id) ?? 0)
    })
  }

  reconcileLayerTerminalDisplayWithJobs(layerList, jobsByLayer)

  function topicForLayer(layer, omitLayerCommand) {
    const lid = layer.layer_id || ''
    const st = layer.job_status || layer.mind_state || '—'
    const time = layer.created_at || ''
    let cmd = omitLayerCommand ? '' : layer.command || ''
    const bootCmd = bootstrapAnchorDisplayCommand(layer)
    if (bootCmd) cmd = cmd || bootCmd
    return {
      name: ztreeHierarchyDisplayName(st, time, lid, cmd),
      title: ztreeHierarchyTooltip(st, time, lid, cmd || layer.command || ''),
    }
  }

  function topicForLayerMergedWithJob(layer, job) {
    const lid = layer.layer_id || ''
    const st = job.status || '—'
    const time = job.created_at || ''
    const cmd = job.command || ''
    const tip = ztreeHierarchyTooltip(st, time, lid, cmd)
    const jobLine = job.id != null && job.id !== '' ? `\n任务: ${job.id}` : ''
    const inlineMeta = jobInlineMeta(job)
    const usageLine = inlineMeta ? `\n用量: ${inlineMeta}` : ''
    return {
      name: `${ztreeHierarchyDisplayName(st, time, lid, cmd)}${inlineMeta ? ` · ${inlineMeta}` : ''}`,
      title: tip + jobLine + usageLine,
    }
  }

  /** 串行列表中「任务」行标题（与 dev-local-token 页 buildSerialRowsFromLayers 规则一致） */
  function topicForJobUnderLayer(job) {
    const lid = job.layer_id || ''
    const st = job.status || '—'
    const time = job.created_at || ''
    const cmd = job.command || ''
    const tip = ztreeHierarchyTooltip(st, time, lid, cmd)
    const jobLine = job.id != null && job.id !== '' ? `\n任务: ${job.id}` : ''
    const inlineMeta = jobInlineMeta(job)
    const usageLine = inlineMeta ? `\n用量: ${inlineMeta}` : ''
    const jtid = job.id != null ? String(job.id) : ''
    const shortId = jtid.length > 14 ? jtid.slice(0, 8) + '…' : jtid
    const base = ztreeHierarchyDisplayName(st, time, lid, cmd)
    const name = shortId ? `任务 ${shortId} · ${base}${inlineMeta ? ` · ${inlineMeta}` : ''}` : base
    return { name, title: tip + jobLine + usageLine }
  }

  function ztStyleForLayer(layer) {
    const bootStyle = bootstrapAnchorZtStyle(layer)
    if (bootStyle) return bootStyle
    if (jobStatusIsActive(layer.job_status)) return 'active'
    const dirty = normalizeLayerGitDirty(layer.git_worktree_dirty)
    if (dirty === null) return 'no_git'
    if (dirty === false) return 'clean'
    return 'dirty'
  }

  function ztStyleForJob(job) {
    const L = layerById.get(job.layer_id)
    const dirty = L ? normalizeLayerGitDirty(L.git_worktree_dirty) : null
    const showSubmitted = maps.effectiveClean(job)
    const active = jobStatusIsActive(job.status)
    if (active) return 'active'
    if (dirty === null) return 'no_git'
    if (dirty === false || showSubmitted) return 'clean'
    return 'dirty'
  }

  /** 与 onlineService 任务卡操作门控一致 */
  function applyReleasedGitGate(base) {
    if (!containerReleased) return base
    // 保持各 can* 的 git 态可见性（有 git 才展示按钮），仅把容器内 git 动作禁用并给出明确文案；
    // 前端 LayerGraphZtreeNode 用 containerReleased 让禁用态按钮仍渲染。
    return {
      ...base,
      containerReleased: true,
      submitAndPushDisabled: true,
      submitAndPushTitle: '服务器已释放，无法提交并创建 PR',
      pushDisabled: true,
      pushTitle: '服务器已释放，无法推送',
      submitDisabled: true,
      submitTitle: '服务器已释放，无法提交',
      mergeDisabled: true,
      mergeTitle: '服务器已释放，无法合并',
      submitAndMergeDisabled: true,
      submitAndMergeTitle: '服务器已释放，无法提交并合并',
    }
  }

  function actionUiForJob(job) {
    if (!job) {
      return applyReleasedGitGate({
        jobId: null,
        layerId: '',
        command: '',
        commandKind: 'trae',
        canRedo: false,
        redoDisabled: true,
        redoTitle: '',
        canInterrupt: false,
        interruptDisabled: true,
        interruptTitle: '',
        canContinue: false,
        continueDisabled: true,
        continueTitle: '',
        canEditRun: false,
        editRunDisabled: true,
        editRunTitle: '',
        canDelete: false,
        deleteDisabled: true,
        deleteTitle: '',
        canSubmit: false,
        submitDisabled: true,
        submitTitle: '',
        canPush: false,
        pushDisabled: true,
        pushTitle: '',
        pushAheadLabel: '',
        canSubmitAndPush: false,
        submitAndPushDisabled: true,
        submitAndPushTitle: '',
        canOpenPr: false,
        prHtmlUrl: '',
        prTitle: '',
        canMerge: false,
        mergeDisabled: true,
        mergeTitle: '',
        pushErrorLabel: '',
        pushErrorTitle: '',
        pushErrorTraceId: '',
        pushErrorKind: '',
      })
    }
    const layerId = job.layer_id != null ? String(job.layer_id) : ''
    const L = layerById.get(job.layer_id)
    const layerDirty = L ? normalizeLayerGitDirty(L.git_worktree_dirty) : null
    const hasLayer = !!layerId
    const gr = normalizeLayerGitRemote(L)
    const noUpstreamBlock = !!(gr && gr.is_git !== false && gr.no_upstream === true)
    const hasUnpushed = layerHasUnpushedCommits(L)
    const canPushNow = hasLayer && layerDirty === false && hasUnpushed
    const pushAheadLabel = hasLayer ? formatLayerPushAheadLabel(L) : ''
    const prHtmlUrl = hasLayer ? layerPrHtmlUrl(L) : ''
    const canOpenPr = Boolean(prHtmlUrl)
    const cloneTask = job.command_kind === 'clone'
    const gitLock = job.git_destructive_locked === true
    const dis = cloneTask || gitLock
    const isInterrupted = job.status === 'interrupted'
    const isRunning = String(job.status || '').trim().toLowerCase() === 'running'
    const canContinue = isInterrupted && !cloneTask && job.command_kind !== 'shell'
    let title = ''
    if (cloneTask) {
      title = '克隆层任务不支持重新执行，请使用「克隆仓库」'
    } else if (gitLock) {
      title =
        '该可写层在任务开始后已有 git 提交（HEAD 已变化），不可中断、重新执行或删除'
    }
    /** 有 git 即展示「提交」（干净时禁用）；避免文件变动列表有条目时层级行完全看不到提交入口 */
    const hasGit = hasLayer && layerDirty !== null
    const showSubmit = hasGit
    const showPush = canPushNow
    const onMergeTarget = isLayerOnMergeTarget(L, mergeTargetBranch)
    const showMerge = hasGit && !onMergeTarget
    const canMergeNow = showMerge && layerDirty === false
    return applyReleasedGitGate({
      jobId: job.id != null ? String(job.id) : null,
      layerId,
      command: typeof job.command === 'string' ? job.command : '',
      commandKind: job.command_kind === 'shell' ? 'shell' : 'trae',
      canRedo: true,
      redoDisabled: dis,
      redoTitle: title,
      canInterrupt: true,
      interruptDisabled: !isRunning,
      interruptTitle: isRunning ? '' : '仅运行中任务可中断',
      canContinue: canContinue || (isInterrupted && (cloneTask || job.command_kind === 'shell')),
      continueDisabled: !canContinue,
      continueTitle: canContinue ? '' : '仅 interrupted 且非 shell/clone 任务可继续',
      canEditRun: true,
      editRunDisabled: cloneTask,
      editRunTitle: cloneTask ? '克隆任务不支持修改后执行' : '',
      canDelete: hasLayer,
      deleteDisabled: !hasLayer,
      deleteTitle: hasLayer ? '' : '缺少可写层信息',
      canSubmit: showSubmit,
      submitDisabled: layerDirty !== true,
      submitTitle: !hasLayer ? '缺少可写层信息' : (layerDirty === null ? '该可写层无 git 仓库' : (layerDirty === false ? '该可写层暂无未提交变更' : '')),
      canPush: showPush,
      pushDisabled: !canPushNow,
      pushTitle: !hasLayer
        ? '缺少可写层信息'
        : layerDirty === null
          ? '该可写层无 git 仓库'
          : noUpstreamBlock
            ? '当前分支未设置上游跟踪分支，无法推送'
            : layerDirty !== false
              ? '请先提交'
              : (canPushNow ? '' : '暂无未推送提交'),
      pushAheadLabel,
      canSubmitAndPush: hasGit && (layerDirty !== false || hasUnpushed),
      submitAndPushDisabled: dis || !hasGit || (layerDirty === false && !hasUnpushed),
      submitAndPushTitle: dis
        ? title
        : !hasLayer
          ? '缺少可写层信息'
          : layerDirty === null
            ? '该可写层无 git 仓库'
            : noUpstreamBlock
              ? '当前分支未设置上游跟踪分支'
              : (layerDirty === false && !hasUnpushed)
                ? '暂无需要提交或推送的变更'
                : '',
      canOpenPr,
      prHtmlUrl,
      prTitle: canOpenPr ? '打开 PR 审查页' : '',
      canMerge: showMerge,
      mergeDisabled: !canMergeNow,
      mergeTitle: !hasLayer
        ? '缺少可写层信息'
        : layerDirty === null
          ? '该可写层无 git 仓库'
          : layerDirty !== false
            ? '请先提交后再合并到目标分支'
            : '将当前分支合并进合并目标分支',
      ...layerPushErrorFields(L),
    })
  }

  function actionUiForLayer(layer) {
    const layerId = layer && layer.layer_id != null ? String(layer.layer_id) : ''
    const dirty = layer ? normalizeLayerGitDirty(layer.git_worktree_dirty) : null
    const hasLayer = !!layerId
    const gr = normalizeLayerGitRemote(layer)
    const noUpstreamBlock = !!(gr && gr.is_git !== false && gr.no_upstream === true)
    const hasUnpushed = layerHasUnpushedCommits(layer)
    const canPushNow = hasLayer && dirty === false && hasUnpushed
    const pushAheadLabel = hasLayer ? formatLayerPushAheadLabel(layer) : ''
    const prHtmlUrl = hasLayer ? layerPrHtmlUrl(layer) : ''
    const canOpenPr = Boolean(prHtmlUrl)
    const hasGit = hasLayer && dirty !== null
    const showSubmit = hasGit
    const showPush = canPushNow
    const onMergeTarget = isLayerOnMergeTarget(layer, mergeTargetBranch)
    const showMerge = hasGit && !onMergeTarget
    const canMergeNow = showMerge && dirty === false
    return applyReleasedGitGate({
      jobId: null,
      layerId,
      command: '',
      commandKind: 'trae',
      canRedo: false,
      redoDisabled: true,
      redoTitle: '',
      canInterrupt: false,
      interruptDisabled: true,
      interruptTitle: '',
      canContinue: false,
      continueDisabled: true,
      continueTitle: '',
      canEditRun: false,
      editRunDisabled: true,
      editRunTitle: '',
      canDelete: hasLayer,
      deleteDisabled: !hasLayer,
      deleteTitle: hasLayer ? '' : '缺少可写层信息',
      canSubmit: showSubmit,
      submitDisabled: dirty !== true,
      submitTitle: !hasLayer ? '缺少可写层信息' : (dirty === null ? '该可写层无 git 仓库' : (dirty === false ? '该可写层暂无未提交变更' : '')),
      canPush: showPush,
      pushDisabled: !canPushNow,
      pushTitle: !hasLayer
        ? '缺少可写层信息'
        : dirty === null
          ? '该可写层无 git 仓库'
          : noUpstreamBlock
            ? '当前分支未设置上游跟踪分支，无法推送'
            : dirty !== false
              ? '请先提交'
              : (canPushNow ? '' : '暂无未推送提交'),
      pushAheadLabel,
      canSubmitAndPush: hasGit && (dirty !== false || hasUnpushed),
      submitAndPushDisabled: !hasGit || (dirty === false && !hasUnpushed),
      submitAndPushTitle: !hasLayer
        ? '缺少可写层信息'
        : dirty === null
          ? '该可写层无 git 仓库'
          : noUpstreamBlock
            ? '当前分支未设置上游跟踪分支'
            : (dirty === false && !hasUnpushed)
              ? '暂无需要提交或推送的变更'
              : '',
      canOpenPr,
      prHtmlUrl,
      prTitle: canOpenPr ? '打开 PR 审查页' : '',
      canMerge: showMerge,
      mergeDisabled: !canMergeNow,
      mergeTitle: !hasLayer
        ? '缺少可写层信息'
        : dirty === null
          ? '该可写层无 git 仓库'
          : dirty !== false
            ? '请先提交后再合并到目标分支'
            : '将当前分支合并进合并目标分支',
      ...layerPushErrorFields(layer),
    })
  }

  return {
    bootstrapLayerId,
    layerList,
    layerById,
    jobsByLayer,
    jobApiOrder,
    topicForLayer,
    topicForLayerMergedWithJob,
    topicForJobUnderLayer,
    ztStyleForLayer,
    ztStyleForJob,
    actionUiForJob,
    actionUiForLayer,
  }
}

/**
 * 按父子可写层关系建树（旧版评论区展示）。
 * @param {object[]} layers
 * @param {object[]} jobs
 * @param {{ bootstrapLayerId?: string, mergeTargetBranch?: string }} [options]
 * @returns {object[]}
 */

export function buildZTreeNodesFromLayers(layers, jobs, options = {}) {
  const ZROOT = 0
  const ctx = createLayerGraphNodeContext(layers, jobs, options)
  const {
    bootstrapLayerId,
    layerList,
    layerById,
    jobsByLayer,
    jobApiOrder,
    topicForLayer,
    topicForLayerMergedWithJob,
    ztStyleForLayer,
    ztStyleForJob,
    actionUiForJob,
    actionUiForLayer,
  } = ctx

  const nodes = []
  const nRoot = layerList.length
  nodes.push({
    id: LAYER_GRAPH_ROOT_ID,
    pId: ZROOT,
    name: nRoot <= 1 ? '可写层' : '可写层（' + nRoot + ' 节点）',
    title: nRoot <= 1 ? '可写层' : '可写层（按父子关系展示）',
    nodeKind: 'virtual',
    ztStyle: 'virtual',
    open: true,
  })

  const layerChildren = new Map()
  const layerParent = new Map()
  const rootLayers = []
  for (const layer of layerList) {
    const lid = layer && layer.layer_id ? String(layer.layer_id) : ''
    if (!lid) continue
    const parent = normalizeParentLayerId(layer)
    if (parent && parent !== lid && layerById.has(parent)) {
      layerParent.set(lid, parent)
      if (!layerChildren.has(parent)) layerChildren.set(parent, [])
      layerChildren.get(parent).push(layer)
      continue
    }
    rootLayers.push(layer)
  }
  sortZTreeLayerSiblings(rootLayers, bootstrapLayerId)
  for (const arr of layerChildren.values()) {
    sortZTreeLayerSiblings(arr, bootstrapLayerId)
  }

  const visited = new Set()
  const appendLayerNode = (layer, parentNodeId) => {
    const lid = layer && layer.layer_id ? String(layer.layer_id) : ''
    if (!lid || visited.has(lid)) return
    visited.add(lid)
    const zid = layerNodeId(layer.layer_id)
    const jlistAll = jobsByLayer.get(layer.layer_id) || []
    const jlist = jlistAll.filter((j) => j.command_kind !== 'clone')
    const cloneList = jlistAll.filter((j) => j && j.command_kind === 'clone')
    jlist.sort((a, b) => {
      const da = createdAtMs(a.created_at)
      const db = createdAtMs(b.created_at)
      if (da !== db) return da - db
      return (jobApiOrder.get(a.id) ?? 0) - (jobApiOrder.get(b.id) ?? 0)
    })
    let latest = jlist.length ? jlist[jlist.length - 1] : null
    if (!latest && cloneList.length > 0) {
      latest = cloneList[cloneList.length - 1]
    }
    const ltopic = latest ? topicForLayerMergedWithJob(layer, latest) : topicForLayer(layer, false)
    const layerActions = latest ? actionUiForJob(latest) : actionUiForLayer(layer)
    const parentLayerId = layerParent.get(lid) || ''
    nodes.push({
      id: zid,
      pId: parentNodeId,
      name: ltopic.name,
      title: ltopic.title,
      nodeKind: 'layer',
      ztStyle: latest ? ztStyleForJob(latest) : ztStyleForLayer(layer),
      open: true,
      logicalParentLayerId: parentLayerId,
      ...layerActions,
      ...layerZtreeBootstrapAnchorFields(layer),
    })
    const children = layerChildren.get(lid) || []
    for (const ch of children) {
      appendLayerNode(ch, zid)
    }
  }

  for (const layer of rootLayers) {
    appendLayerNode(layer, LAYER_GRAPH_ROOT_ID)
  }
  for (const layer of layerList) {
    const lid = layer && layer.layer_id ? String(layer.layer_id) : ''
    if (!lid || visited.has(lid)) continue
    appendLayerNode(layer, LAYER_GRAPH_ROOT_ID)
  }
  return nodes
}

/**
 * 与可写层时间轴一致：按创建时间旧→新排序；
 * 同层仅一条可见任务时合并到层行；多条时层行与逐条任务行（跳过 clone）均挂在同一虚拟根下，按各自创建时间与层创建时间混排为单一串行列表。
 * 若该层尚无非 clone 任务、仅有克隆任务：将**最新一条 clone** 合并到可写层行，展示克隆指令与 running→completed，避免克隆阶段层行无指令/状态滞后。
 * 首节点为虚拟根（id=`LAYER_GRAPH_ROOT_ID`），其余层/任务行 pId 均指向该根；UI 侧展开虚拟根后仅渲染其子列表，故视觉上为单分支串行、无父子层缩进嵌套。
 * @param {object[]} layers
 * @param {object[]} jobs
 * @param {{ bootstrapLayerId?: string, mergeTargetBranch?: string }} [options]
 * @returns {object[]}
 */

export function buildZTreeNodesSerialFromLayers(layers, jobs, options = {}) {
  const VROOT = LAYER_GRAPH_ROOT_ID
  const ctx = createLayerGraphNodeContext(layers, jobs, options)
  const {
    bootstrapLayerId,
    layerList,
    jobsByLayer,
    topicForLayer,
    topicForLayerMergedWithJob,
    topicForJobUnderLayer,
    ztStyleForLayer,
    ztStyleForJob,
    actionUiForJob,
    actionUiForLayer,
  } = ctx

  const nodes = []
  const sortedLayers = sortLayersSerialChronological(layerList, bootstrapLayerId)

  for (const layer of sortedLayers) {
    const lid = layer && layer.layer_id ? String(layer.layer_id) : ''
    if (!lid) continue
    const zid = layerNodeId(layer.layer_id)
    const jlistRaw = jobsByLayer.get(layer.layer_id) || []
    const visibleJobs = jlistRaw.filter((j) => j.command_kind !== 'clone')
    const cloneJobs = jlistRaw.filter((j) => j && j.command_kind === 'clone')
    let soleVisible = visibleJobs.length === 1 ? visibleJobs[0] : null
    if (soleVisible == null && visibleJobs.length === 0 && cloneJobs.length > 0) {
      soleVisible = cloneJobs[cloneJobs.length - 1]
    }

    const ltopic = soleVisible
      ? topicForLayerMergedWithJob(layer, soleVisible)
      : topicForLayer(layer, visibleJobs.length > 1)
    const layerRowActions = soleVisible ? actionUiForJob(soleVisible) : actionUiForLayer(layer)
    const logicalParent = normalizeParentLayerId(layer)
    const layerSerialMs = createdAtMs(layer.created_at)
    nodes.push({
      id: zid,
      pId: VROOT,
      name: ltopic.name,
      title: ltopic.title,
      nodeKind: 'layer',
      ztStyle: soleVisible ? ztStyleForJob(soleVisible) : ztStyleForLayer(layer),
      open: true,
      logicalParentLayerId: logicalParent,
      /** 与「可写层串行」一致：建树后按此字段对同级子节点排序（子层行与任务行可按时序交错） */
      ztreeSerialMs: layerSerialMs,
      ...layerRowActions,
      ...layerZtreeBootstrapAnchorFields(layer),
    })
    if (!soleVisible) {
      for (const j of jlistRaw) {
        if (j.command_kind === 'clone') continue
        const jid = j.id != null ? String(j.id) : ''
        if (!jid) continue
        const jtopic = topicForJobUnderLayer(j)
        nodes.push({
          id: jid,
          pId: VROOT,
          name: jtopic.name,
          title: jtopic.title,
          nodeKind: 'job',
          ztStyle: ztStyleForJob(j),
          open: true,
          logicalParentLayerId: '',
          ztreeSerialMs: createdAtMs(j.created_at),
          ...actionUiForJob(j),
        })
      }
    }
  }

  if (nodes.length > 0) {
    const nRoot = layerList.length
    nodes.unshift({
      id: VROOT,
      pId: 0,
      name: nRoot <= 1 ? '可写层' : `可写层（${nRoot} 节点）`,
      title: '按创建时间串行展示',
      nodeKind: 'virtual',
      ztStyle: 'virtual',
      open: true,
    })
  }
  return nodes
}


function sortZtreeSiblingsBySerial(a, b) {
  const da = a.ztreeSerialMs ?? 0
  const db = b.ztreeSerialMs ?? 0
  if (da !== db) return da - db
  return String(a.id).localeCompare(String(b.id))
}


function sortZtreeChildrenRec(node) {
  if (!node?.children || !node.children.length) return
  node.children.sort(sortZtreeSiblingsBySerial)
  for (const ch of node.children) {
    sortZtreeChildrenRec(ch)
  }
}

/**
 * 将 simpleData 转为嵌套根列表；对每层子节点按 `ztreeSerialMs` 串行排序（与 buildZTreeNodesSerialFromLayers 的创建时间轴一致）。
 */

export function simpleDataToTreeRoots(nodes) {
  if (!nodes || !nodes.length) return []
  const map = new Map()
  for (const n of nodes) {
    map.set(n.id, { ...n, children: [] })
  }
  const roots = []
  for (const n of nodes) {
    const node = map.get(n.id)
    const pid = n.pId
    if (pid === 0 || pid === '0') {
      roots.push(node)
    } else {
      const p = map.get(pid)
      if (p) p.children.push(node)
      else roots.push(node)
    }
  }
  roots.sort(sortZtreeSiblingsBySerial)
  for (const r of roots) {
    sortZtreeChildrenRec(r)
  }
  return roots
}

