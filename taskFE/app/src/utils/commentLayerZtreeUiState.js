/**
 * 任务详情「可写层串行」加载/释放空态判定（纯函数，供 useTaskDetail 与单测共用）。
 */

/** @type {ReadonlySet<string>} */
export const SERVER_RUNTIME_NOT_SERVING = Object.freeze(
  new Set(['released', 'stopped', 'stopping', 'terminated', 'shutting-down', 'shuttingdown']),
)

/**
 * @param {unknown} runtimeStatus
 * @returns {boolean}
 */
export function isServerRuntimeNotServingStatus(runtimeStatus) {
  const rs = String(runtimeStatus ?? '')
    .trim()
    .toLowerCase()
  return Boolean(rs) && SERVER_RUNTIME_NOT_SERVING.has(rs)
}

/**
 * 服务器对容器层图而言不可服务（未运行且未启动中）。
 * runtime 已释放单独用 `serverRuntimeNotServing` 驱动空态横幅，避免挡住重新启动窗口。
 * @param {{
 *   isServerRunning?: boolean,
 *   isServerStarting?: boolean,
 * }} input
 */
export function isServerNotServingUi({
  isServerRunning = false,
  isServerStarting = false,
} = {}) {
  return !isServerRunning && !isServerStarting
}

/**
 * @param {{
 *   layerGraphNodeCount?: number,
 *   layerGraphHasRealWritableLayer?: boolean,
 *   isServerRunning?: boolean,
 *   isServerStarting?: boolean,
 *   containerHeartbeatStatus?: string,
 *   containerEndpointRegistered?: boolean,
 *   containerHttpUnreachable?: boolean,
 *   layerGraphRefreshing?: boolean,
 *   executionReleased?: boolean,
 *   containerLayerGraphAuthInvalid?: boolean,
 *   containerBootstrapFailureMessage?: string,
 *   bootstrapCloneLogText?: string,
 * }} input
 * @returns {boolean}
 */
export function shouldShowCommentLayerZtreeLoading(input = {}) {
  const loadingError = isCommentLayerZtreeLoadingError({
    containerLayerGraphAuthInvalid: input.containerLayerGraphAuthInvalid,
    containerBootstrapFailureMessage: input.containerBootstrapFailureMessage,
    bootstrapCloneLogText: input.bootstrapCloneLogText,
  })
  const hasRealLayer = input.layerGraphHasRealWritableLayer === true
  const layerGraphNodeCount = Number(input.layerGraphNodeCount) || 0
  // 引导失败且尚无真实可写层：empty/pending 锚点已进树也不能当成就绪。
  // 有树节点时错误画在 comment-layer-ztree-panel 内，不拆掉该面板。
  if (loadingError && !hasRealLayer) {
    return layerGraphNodeCount === 0
  }

  if (input.executionReleased) return false
  if (layerGraphNodeCount > 0) return false

  if (
    isServerNotServingUi({
      isServerRunning: input.isServerRunning,
      isServerStarting: input.isServerStarting,
    })
  ) {
    return false
  }

  // 容器已在服务但树尚未水合（心跳 idle、快照空、GET 未完成）时仍展示加载态，
  // 避免「任务关联」Tab 选中后 comment-layer-association-body 只有空注释节点。
  return true
}

/**
 * 非服务态且没有已保存层图时展示「服务器已释放」空态（从未启动且 idle 时不展示）。
 * 库内 / SaaS 快照节点必须保留：释放后仍要能复查 ztree 与执行日志；写操作由 containerReleased 禁用。
 * @param {{
 *   layerGraphNodeCount?: number,
 *   showLoading?: boolean,
 *   isServerRunning?: boolean,
 *   isServerStarting?: boolean,
 *   serverRuntimeNotServing?: boolean,
 *   containerHeartbeatPaused?: boolean,
 *   containerHeartbeatStatus?: string,
 *   executionReleased?: boolean,
 * }} input
 */
export function shouldShowCommentLayerZtreeReleased(input = {}) {
  const layerGraphNodeCount = Number(input.layerGraphNodeCount) || 0
  if (layerGraphNodeCount > 0) return false
  if (input.executionReleased) return true
  if (input.showLoading) return false
  if (
    !isServerNotServingUi({
      isServerRunning: input.isServerRunning,
      isServerStarting: input.isServerStarting,
      serverRuntimeNotServing: input.serverRuntimeNotServing,
    })
  ) {
    return false
  }

  if (input.containerHeartbeatPaused || input.serverRuntimeNotServing) return true
  const hb = String(input.containerHeartbeatStatus || 'idle')
  return hb === 'connecting' || hb === 'disconnected'
}

/**
 * 将容器 BOOTSTRAP_FAILED 文案收成任务关联区可行动提示。
 * @param {unknown} raw
 * @returns {string}
 */
export function formatContainerBootstrapFailureHint(raw) {
  const msg = String(raw || '').trim()
  if (!msg) return ''
  const missing =
    msg.match(/缺失仓库\s*\(\d+\)\s*[:：]\s*(.+?)(?:\s+detail=|$)/i) ||
    msg.match(/missing_repo_credentials[^[]*\[([^\]]+)\]/i)
  const repoBrief = missing ? String(missing[1] || '').trim().replace(/["']/g, '') : ''
  if (
    /repo-clone-credentials|REPO_CLONE_CREDENTIALS|克隆凭证不完整|Git 授权/i.test(msg)
  ) {
    const repoPart = repoBrief ? `（${repoBrief.slice(0, 120)}${repoBrief.length > 120 ? '…' : ''}）` : ''
    return `引导克隆失败：仓库 Git 授权未齐${repoPart}。请由任务创建者在创建或编辑任务时为全部仓库完成 OAuth 绑定后重试克隆。`
  }
  const clipped = msg.length > 220 ? `${msg.slice(0, 220)}…` : msg
  return `引导克隆失败：${clipped}`
}

/**
 * 从容器 GET bootstrap-clone-log 载荷提取任务关联区失败文案。
 * 凭证 409 时 layer 仍空，但 payload.error_code / text 已有可读摘要。
 * @param {unknown} data
 * @returns {string}
 */
export function bootstrapFailureMessageFromCloneLogPayload(data) {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return ''
  const rec = /** @type {Record<string, unknown>} */ (data)
  const code = String(rec.error_code || '').trim()
  const text = typeof rec.text === 'string' ? rec.text.trim() : ''
  if (code === 'REPO_CLONE_CREDENTIALS_INCOMPLETE') {
    const fromText = formatContainerBootstrapFailureHint(text)
    if (fromText) return fromText
    const missing = Array.isArray(rec.missing_repo_credentials)
      ? rec.missing_repo_credentials.map((u) => String(u || '').trim()).filter(Boolean)
      : []
    const synthetic = missing.length
      ? `REPO_CLONE_CREDENTIALS_INCOMPLETE missing_repo_credentials:[${missing.join(', ')}]`
      : 'REPO_CLONE_CREDENTIALS_INCOMPLETE'
    return formatContainerBootstrapFailureHint(synthetic)
  }
  if (!text) return ''
  if (!/【项目克隆】引导失败|BOOTSTRAP_FAILED|repo-clone-credentials|REPO_CLONE_CREDENTIALS/i.test(text)) {
    return ''
  }
  return formatContainerBootstrapFailureHint(text)
}

/**
 * @param {{
 *   containerLayerGraphAuthInvalid?: boolean,
 *   containerBootstrapFailureMessage?: string,
 *   bootstrapCloneLogText?: string,
 *   containerHttpUnreachable?: boolean,
 *   containerEndpointRegistered?: boolean,
 *   containerHeartbeatStatus?: string,
 *   layerGraphRefreshing?: boolean,
 *   containerHeartbeatProbeOk?: boolean|null,
 * }} input
 * @returns {string}
 */
export function resolveCommentLayerZtreeLoadingHint(input = {}) {
  if (input.containerLayerGraphAuthInvalid) {
    return '容器访问令牌无效，请重新模拟启动或刷新页面后重试。'
  }
  const bootstrapHint = formatContainerBootstrapFailureHint(input.containerBootstrapFailureMessage)
  if (bootstrapHint) {
    return bootstrapHint
  }
  const logHint = bootstrapFailureMessageFromCloneLogPayload({
    text: input.bootstrapCloneLogText,
  })
  if (logHint) {
    return logHint
  }
  const logText = String(input.bootstrapCloneLogText || '')
  if (/正在并行克隆|开始项目克隆|任务详情已就绪，开始项目克隆/i.test(logText)) {
    return '正在克隆任务关联仓库（完成后将自动显示可写层）…'
  }
  if (/开始拉取任务详情|BOOTSTRAP_PHASE=task_detail/i.test(logText)) {
    return '容器已启动，正在拉取任务详情并准备克隆…'
  }
  if (input.containerHttpUnreachable && input.containerEndpointRegistered) {
    return '容器心跳正常，但平台无法访问容器业务地址以拉取层级图（请检查安全组/防火墙是否放行 SaaS 访问容器 BUSINESS_API 端口，或 register-reachability 是否指向平台可达的 host:port）。'
  }
  if (!input.containerEndpointRegistered) {
    if (input.containerHeartbeatStatus === 'connected') {
      return '容器心跳已连通，正在等待业务端点注册（register-reachability / exchange-refresh）…'
    }
    return '服务器已启动，正在等待容器镜像完成初始化并回传连接（安装 Docker / 拉镜像 / register-reachability，通常需数分钟）…'
  }
  if (input.layerGraphRefreshing) {
    return '正在从容器拉取可写层与任务列表…'
  }
  if (input.containerHeartbeatStatus === 'connecting') {
    return '容器连接确认中，即将拉取可写层…'
  }
  if (
    input.containerHeartbeatStatus === 'connected' &&
    input.containerHeartbeatProbeOk === false
  ) {
    return '双向心跳已确认，但平台对容器的 HTTP 探测失败，正在等待可写层经容器推送就绪（若长时间无变化，请检查安全组是否放行 SaaS→容器业务端口）…'
  }
  return '容器已连接，正在等待可写层就绪（引导克隆或 bootstrap 完成后将自动显示）…'
}

/**
 * 任务关联加载区是否应展示为错误态（停转圈，改强调文案）。
 * @param {{ containerBootstrapFailureMessage?: string, containerLayerGraphAuthInvalid?: boolean, bootstrapCloneLogText?: string }} input
 */
export function isCommentLayerZtreeLoadingError(input = {}) {
  if (input.containerLayerGraphAuthInvalid) return true
  if (formatContainerBootstrapFailureHint(input.containerBootstrapFailureMessage)) return true
  return Boolean(bootstrapFailureMessageFromCloneLogPayload({ text: input.bootstrapCloneLogText }))
}

export const COMMENT_LAYER_ZTREE_RELEASED_TITLE = '服务器已释放，暂无已保存的层图'
export const COMMENT_LAYER_ZTREE_RELEASED_BODY =
  '历史评论（含 AI / 容器 Agent）仍可在下方查看。若曾落库的层图快照可用，刷新后会显示在此；否则请重新启动服务器。'
