import { ref, computed, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { COUNTRY_DIAL_OPTIONS } from '../../utils/countryDialCodes.js'
import { DEFAULT_PHONE_COUNTRY_PREFIX } from '../../utils/phoneInput.js'

const PUBLIC_FEATURE_POLICY_API = '/api/public/system-feature-policy/'

// Module-level cache: fetch once per page load, shared across all component instances.
// Stores the phone_country_options array returned by the backend.
// - null  = not yet loaded → use COUNTRY_DIAL_OPTIONS fallback
// - []    = API returned empty options → use COUNTRY_DIAL_OPTIONS fallback
// - [...] = filtered options from the backend policy → use directly as data source
const _cachedCountryOptions = ref(null)
let _fetchPromise = null

/**
 * 从后端 API 获取「手机号区号下拉选项」，后端已根据策略完成过滤。
 *
 * - 后端 phone_country_options 非空 → 直接作为下拉框数据源（后端为权威）
 * - 后端 phone_country_options 为空/缺失 → fallback 到前端 COUNTRY_DIAL_OPTIONS
 * - API 请求失败 → fallback 到前端 COUNTRY_DIAL_OPTIONS
 *
 * 使用模块级缓存，同一页面内多次调用共享一次 API 请求。
 */
export function useAllowedCountryCodes() {
  const loading = ref(_cachedCountryOptions.value === null)
  const error = ref('')

  /**
   * 下拉框选项：优先使用后端动态返回的选项，失败时回退到前端静态列表。
   * MUST be a computed that reads _cachedCountryOptions (a Vue ref) so
   * the UI re-renders when the fetch completes.
   */
  const filteredCountryOptions = computed(() => {
    if (_cachedCountryOptions.value !== null && _cachedCountryOptions.value.length > 0) {
      // 后端返回了过滤后的选项 → 直接使用
      return _cachedCountryOptions.value
    }
    // 未加载、空列表（允许所有）、或 API 失败 → 使用前端全量列表
    return COUNTRY_DIAL_OPTIONS
  })

  /**
   * 发起 API 请求获取区号下拉选项。同一页面内多次调用共享缓存。
   */
  const fetchAllowedCodes = async () => {
    // 已有缓存：直接返回
    if (_cachedCountryOptions.value !== null) {
      loading.value = false
      return _cachedCountryOptions.value
    }

    // 正在请求中：等待完成
    if (_fetchPromise) {
      await _fetchPromise
      loading.value = false
      return _cachedCountryOptions.value
    }

    loading.value = true
    error.value = ''

    _fetchPromise = (async () => {
      try {
        const response = await apiFetch(PUBLIC_FEATURE_POLICY_API, {
          method: 'GET',
          headers: { Accept: 'application/json' },
        })
        const payload = await response.json().catch(() => ({}))
        if (response.ok) {
          const data = payload?.data ?? payload
          const options = data?.phone_country_options
          if (Array.isArray(options) && options.length > 0) {
            // 后端返回了过滤后的选项 → 作为数据源
            _cachedCountryOptions.value = options
          } else {
            // 空列表表示允许所有地区 → 使用 fallback
            _cachedCountryOptions.value = []
          }
        } else {
          // API 错误 → 使用 fallback
          _cachedCountryOptions.value = []
        }
      } catch {
        // 网络错误 → 使用 fallback
        _cachedCountryOptions.value = []
      } finally {
        loading.value = false
        _fetchPromise = null
      }
    })()

    await _fetchPromise
    return _cachedCountryOptions.value
  }

  /**
   * 默认区号前缀（OPT-20260806-027）：
   * - 选项含 +86 → '+86'（保持既有默认行为）
   * - 白名单不含 +86（如仅允许 +852）→ 取选项首个（否则下拉无该选项时 select 显示空）
   * - 选项为空/未加载 → 回退 '+86'（与 DEFAULT_PHONE_COUNTRY_PREFIX 一致）
   */
  const defaultCountryPrefix = computed(() => {
    const options = filteredCountryOptions.value
    if (Array.isArray(options) && options.length > 0) {
      if (options.some((o) => o?.code === DEFAULT_PHONE_COUNTRY_PREFIX)) {
        return DEFAULT_PHONE_COUNTRY_PREFIX
      }
      return options[0]?.code || DEFAULT_PHONE_COUNTRY_PREFIX
    }
    return DEFAULT_PHONE_COUNTRY_PREFIX
  })

  /**
   * 将消费端 prefix ref 与后端白名单联动：options 加载完成后校正一次
   * （当前 prefix 不在选项内 → 切换为 defaultCountryPrefix），避免白名单
   * 不含 +86 时下拉框显示空。options 未加载/为空时不干预（保持初始值）。
   */
  function syncPrefixWithAllowedOptions(prefixRef) {
    watch(
      filteredCountryOptions,
      (options) => {
        if (!Array.isArray(options) || options.length === 0) return
        const current = prefixRef.value
        if (!options.some((o) => o?.code === current)) {
          prefixRef.value = defaultCountryPrefix.value
        }
      },
      { immediate: true },
    )
  }

  return {
    filteredCountryOptions,
    defaultCountryPrefix,
    syncPrefixWithAllowedOptions,
    loading,
    error,
    fetchAllowedCodes,
  }
}
