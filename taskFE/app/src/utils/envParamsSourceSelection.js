/**
 * 任务详情评论区「智能体资源配置」来源是否已就绪；未就绪时返回用户可见提示。
 * @param {{ featureParamsSource?: string, selectedPersonalConfigId?: string }} ctx
 * @returns {string} 空字符串表示已选择且可用
 */
export function resolveEnvParamsSourceRequiredHint(ctx = {}) {
  const source = String(ctx.featureParamsSource || '').trim()
  if (!source) return '请先选择智能体资源配置'
  if (source === 'personal' && !String(ctx.selectedPersonalConfigId || '').trim()) {
    return '请先选择智能体资源配置'
  }
  return ''
}

/**
 * 创建/编辑任务提交门禁：未选智能体资源配置时返回按钮旁说明文案。
 * @param {{
 *   featureParamsSource?: string,
 *   selectedPersonalConfigId?: string,
 *   isEdit?: boolean,
 * }} ctx
 * @returns {string}
 */
export function resolveCreateTaskFeatureParamsBlockedReason(ctx = {}) {
  if (!resolveEnvParamsSourceRequiredHint(ctx)) return ''
  return ctx.isEdit
    ? '请先选择智能体资源配置后再保存'
    : '请先选择智能体资源配置后再创建'
}

/**
 * 读取 feature-params GET 的 env_var_sources_available 标志。
 * 真实响应位于 data 内；兼容旧/测试形状的顶层 flag。缺标志时返回 null（调用方 fail-open）。
 * @param {unknown} body
 * @returns {{ company?: boolean, workspace?: boolean } | null}
 */
export function readEnvVarSourcesAvailableFlag(body) {
  if (!body || typeof body !== 'object') return null
  const nested = body.data && typeof body.data === 'object'
    ? body.data.env_var_sources_available
    : undefined
  const flag = (nested && typeof nested === 'object')
    ? nested
    : body.env_var_sources_available
  if (!flag || typeof flag !== 'object') return null
  return flag
}
export const FEATURE_PARAMS_SOURCE_REQUIRED_API_MESSAGE =
  '智能体资源配置为必填项，请选择公司默认、工作空间默认或个人配置'

/**
 * 智能体资源配置空状态引导链接：公司 / 工作空间 / 个人。
 * 无 tenantId 时公司与工作空间 href 为空（调用方应渲染为纯文本）。
 * @param {string|number|null|undefined} tenantId
 * @returns {{ company: string, workspace: string, personal: string }}
 */
export function featureParamsSettingsHrefs(tenantId) {
  const tid = String(tenantId ?? '').trim()
  return {
    company: tid ? `/tenant/${encodeURIComponent(tid)}/settings/feature-params/` : '',
    workspace: tid ? `/tenant/${encodeURIComponent(tid)}/settings/task-panel/` : '',
    personal: '/profile/feature-params/',
  }
}

const FEATURE_PARAMS_SOURCES = new Set(['company', 'workspace', 'personal'])

/**
 * 将任务上的 feature_params_source 规范为选择器可用值；none/空/未知 → ''。
 * @param {unknown} raw
 * @returns {string}
 */
export function normalizeFeatureParamsSourceForSelect(raw) {
  const source = String(raw ?? '').trim()
  if (!source || source === 'none') return ''
  return FEATURE_PARAMS_SOURCES.has(source) ? source : ''
}

/**
 * 组装创建/更新任务时的智能体资源配置字段；未选则返回空对象（不写入 payload）。
 * @param {{ feature_params_source?: unknown, personal_feature_params_config_id?: unknown }} task
 * @returns {{ feature_params_source?: string, personal_feature_params_config_id?: string }}
 */
export function buildFeatureParamsPayloadFields(task = {}) {
  const source = normalizeFeatureParamsSourceForSelect(task.feature_params_source)
  if (!source) return {}
  if (source === 'personal') {
    const configId = String(task.personal_feature_params_config_id || '').trim()
    if (!configId) return {}
    return {
      feature_params_source: source,
      personal_feature_params_config_id: configId,
    }
  }
  return { feature_params_source: source }
}

/**
 * 选择变更后即时 PATCH `/api/tasks/{id}/feature-params/`。
 * 未就绪或缺 taskId 时跳过，不发请求。
 *
 * @param {{
 *   apiFetch: typeof fetch,
 *   taskId?: unknown,
 *   featureParamsSource?: string,
 *   selectedPersonalConfigId?: string,
 * }} opts
 * @returns {Promise<{
 *   ok: boolean,
 *   skipped?: boolean,
 *   reason?: string,
 *   status?: number,
 *   data?: Record<string, unknown>,
 *   message?: string,
 * }>}
 */
export async function persistTaskFeatureParamsBinding(opts = {}) {
  const taskId = String(opts.taskId || '').trim()
  if (!taskId) {
    return { ok: false, skipped: true, reason: 'missing_task_id' }
  }

  const body = buildFeatureParamsPayloadFields({
    feature_params_source: opts.featureParamsSource,
    personal_feature_params_config_id: opts.selectedPersonalConfigId,
  })
  if (!body.feature_params_source) {
    return { ok: false, skipped: true, reason: 'incomplete_selection' }
  }

  const apiFetch = opts.apiFetch
  if (typeof apiFetch !== 'function') {
    throw new Error('persistTaskFeatureParamsBinding: apiFetch is required')
  }

  const resp = await apiFetch(`/api/tasks/${encodeURIComponent(taskId)}/feature-params/`, {
    method: 'PATCH',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
    },
    body: JSON.stringify({
      feature_params_source: body.feature_params_source,
      personal_feature_params_config_id: body.personal_feature_params_config_id || '',
    }),
  })

  const data = await resp.json().catch(() => ({}))
  if (!resp.ok) {
    const message = typeof data?.message === 'string' && data.message
      ? data.message
      : `持久化智能体资源配置失败（HTTP ${resp.status}）`
    return { ok: false, status: resp.status, data, message }
  }

  return {
    ok: true,
    status: resp.status,
    data: {
      feature_params_source: data.feature_params_source || body.feature_params_source,
      personal_feature_params_config_id:
        data.personal_feature_params_config_id
        ?? (body.feature_params_source === 'personal' ? body.personal_feature_params_config_id : ''),
    },
  }
}
