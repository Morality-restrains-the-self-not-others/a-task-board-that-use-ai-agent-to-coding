/**
 * 功能参数：智能体/摘要模型名称须在对应供应商 supported_models 内。
 * 空模型名跳过；返回错误文案或 null。
 */

/** 支持模型列表分隔符：换行、英文/中文逗号、分号、空白 */
const SUPPORTED_MODELS_SPLIT_RE = /[\n\r,，;； \t]+/

/**
 * 将 supported_models / supported_models_text 规范为模型名数组。
 * 兼容中文逗号「，」等常见分隔符，避免整段被当成单一模型名。
 * @param {unknown} value
 * @returns {string[]}
 */
export function parseSupportedModels(value) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item || '').trim()).filter(Boolean)
  }
  return String(value || '')
    .split(SUPPORTED_MODELS_SPLIT_RE)
    .map((item) => item.trim())
    .filter(Boolean)
}

/**
 * 将 supported_models_text 规范化为「每行一个模型」（保存/失焦回写）。
 * @param {unknown} value
 * @returns {string}
 */
export function normalizeSupportedModelsText(value) {
  return parseSupportedModels(value).join('\n')
}

function providerSupportedModelsMap(providers) {
  const result = {}
  if (!Array.isArray(providers)) return result
  providers.forEach((item) => {
    const provider = String(item?.provider || '').trim()
    if (!provider) return
    const models = parseSupportedModels(item.supported_models_text || item.supported_models)
    if (!result[provider]) result[provider] = new Set()
    models.forEach((m) => result[provider].add(m))
  })
  return result
}

function validateOneModel(roleLabel, model, provider, supportedByProvider) {
  const modelName = String(model || '').trim()
  if (!modelName) return null
  const providerName = String(provider || '').trim()
  if (!providerName) {
    return `${roleLabel}「${modelName}」已填写，但未选择供应商`
  }
  if (!Object.prototype.hasOwnProperty.call(supportedByProvider, providerName)) {
    return `${roleLabel}供应商「${providerName}」不在当前供应商列表中`
  }
  const allowed = supportedByProvider[providerName]
  if (!allowed.has(modelName)) {
    return `${roleLabel}「${modelName}」不在供应商「${providerName}」的支持模型列表中，禁止保存`
  }
  return null
}

/**
 * @param {Array} providers
 * @param {{ agentModel?: string, agentModelProvider?: string, summaryModel?: string, summaryModelProvider?: string }} fields
 * @returns {string|null}
 */
export function validateModelsAgainstProviders(providers, fields = {}) {
  const supportedByProvider = providerSupportedModelsMap(providers)
  const checks = [
    ['智能体模型', fields.agentModel, fields.agentModelProvider],
    ['摘要模型', fields.summaryModel, fields.summaryModelProvider],
  ]
  for (const [roleLabel, model, provider] of checks) {
    const err = validateOneModel(roleLabel, model, provider, supportedByProvider)
    if (err) return err
  }
  return null
}
