import { parseSupportedModels } from './featureParamsModelValidation.js'

/**
 * 从智能体资源配置中取出可切换模型清单（与层级图「选择模型」同一规则）。
 * @param {Record<string, unknown>|null|undefined} data
 * @returns {{ provider: string, defaultModel: string, options: string[] }}
 */
export function buildAgentModelOptionsFromFeatureParams(data) {
  const providers = Array.isArray(data?.providers) ? data.providers : []
  const selectedProvider = String(data?.agent_model_provider || '').trim()
  const defaultModel = String(data?.agent_model || '').trim()
  const providerItem = providers.find(
    (item) => String(item?.provider || '').trim() === selectedProvider,
  )
  const options = providerItem ? parseSupportedModels(providerItem.supported_models) : []
  if (defaultModel && !options.includes(defaultModel)) {
    options.unshift(defaultModel)
  }
  return {
    provider: selectedProvider,
    defaultModel,
    options: Array.from(new Set(options)),
  }
}

/**
 * 切换智能体资源后的预勾选：默认模型若在清单中则只勾它，否则空。
 * @param {string[]} options
 * @param {string} defaultModel
 * @returns {string[]}
 */
export function preselectDefaultAgentModel(options, defaultModel) {
  const list = Array.isArray(options) ? options : []
  const def = String(defaultModel || '').trim()
  if (def && list.includes(def)) return [def]
  return []
}
