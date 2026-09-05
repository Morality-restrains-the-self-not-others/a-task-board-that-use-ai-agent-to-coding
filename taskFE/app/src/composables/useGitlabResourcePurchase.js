import { computed, reactive, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'

/** 磁盘购买时长选项（月）；末尾 "自定义" 由前端 Number input 覆盖 */
export const DISK_MONTH_OPTIONS = [1, 3, 6]

/**
 * 系统内建 GitLab 磁盘/流量预购：加载配额与按锁价整单新购。
 * 磁盘费用 = GB × 锁价(元/GB/月) × 购买月数。
 * @param {import('vue').ComputedRef<string>|import('vue').Ref<string>} tenantIdRef
 */
export function useGitlabResourcePurchase(tenantIdRef) {
  const loading = ref(false)
  const errorMessage = ref('')
  const errorTraceId = ref('')
  const successMessage = ref('')
  const diskUnitPrice = ref(0)
  const trafficUnitPrice = ref(0)
  const currentDiskGb = ref(0)
  const currentTrafficGb = ref(0)
  const currentDiskMonths = ref(0)
  const currentDiskExpiresAt = ref('')
  const diskUsedGb = ref(0)
  const trafficUsedGb = ref(0)
  const trafficDownloadAllowed = ref(true)
  const region = ref('')
  const regionName = ref('')
  const availableRegions = ref([])
  const gitlabWebUrl = ref('')
  const provisioningStatus = ref('not_purchased')

  const form = reactive({
    disk_gb: 0,
    disk_months: 1,
    traffic_prepaid_gb: 0,
    custom_months: 0,   // 自定义月数：仅当 disk_months 为 0 时有效
  })

  const apiBase = computed(() => {
    const tid = String(tenantIdRef.value || '').trim()
    return `/api/billing/gitlab-resources/tenant_id/${encodeURIComponent(tid)}`
  })

  const regionsApi = computed(() => {
    const tid = String(tenantIdRef.value || '').trim()
    return `/api/billing/gitlab-regions/tenant_id/${encodeURIComponent(tid)}/`
  })

  /** 返回实际生效的月数：优先自定义月数（custom_months），否则取下拉选项值 */
  function effectiveMonths() {
    const custom = Number(form.custom_months) || 0
    if (custom >= 1 && custom <= 36) return custom
    const dropdown = Number(form.disk_months) || 0
    if (dropdown >= 1 && dropdown <= 36) return dropdown
    return 0
  }

  const estimatedCost = computed(() => {
    const disk = Math.max(0, Number(form.disk_gb) || 0)
    const months = disk > 0 ? effectiveMonths() : 0
    const traffic = Math.max(0, Number(form.traffic_prepaid_gb) || 0)
    return (
      disk * Number(diskUnitPrice.value || 0) * months +
      traffic * Number(trafficUnitPrice.value || 0)
    )
  })

  /**
   * 将后端 GET 配额响应映射到本地状态。
   */
  const applyView = (data = {}) => {
    currentDiskGb.value = Number(data.disk_gb) || 0
    currentTrafficGb.value = Number(data.traffic_prepaid_gb) || 0
    currentDiskMonths.value = Number(data.disk_months) || 0
    currentDiskExpiresAt.value = String(data.disk_expires_at || '')
    diskUsedGb.value = Number(data.disk_used_gb) || 0
    trafficUsedGb.value = Number(data.traffic_used_gb) || 0
    trafficDownloadAllowed.value =
      data.traffic_download_allowed == null ? currentTrafficGb.value > 0 && trafficUsedGb.value < currentTrafficGb.value : !!data.traffic_download_allowed
    form.disk_gb = currentDiskGb.value
    form.traffic_prepaid_gb = currentTrafficGb.value
    const months = Number(data.disk_months) || 0
    form.disk_months = months > 0 ? months : 1
    // 磁盘单价（分/GB/月）: GET/purchase 均返回 gitlab_disk_unit_price_cents
    diskUnitPrice.value =
      Number(data.locked_gitlab_disk_cents_per_gb_per_month ?? data.gitlab_disk_unit_price_cents) || 0
    // 流量单价（分/GB）: GET/purchase 均返回 gitlab_traffic_unit_price_cents
    trafficUnitPrice.value =
      Number(data.locked_gitlab_traffic_cents_per_gb ?? data.gitlab_traffic_unit_price_cents) || 0
    region.value = String(data.region || '')
    regionName.value = String(data.region_name || '')
    gitlabWebUrl.value = String(data.gitlab_web_url || '')
    provisioningStatus.value = String(data.provisioning_status || 'not_purchased')
  }

  const loadRegions = async () => {
    const tid = String(tenantIdRef.value || '').trim()
    if (!tid) return
    const response = await apiFetch(regionsApi.value, { headers: { Accept: 'application/json' } })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const err = new Error(typeof data.error === 'string' ? data.error : '无法加载 GitLab 区域')
      err.traceId = response.traceId
      throw err
    }
    availableRegions.value = Array.isArray(data.regions) ? data.regions : []
  }

  const load = async () => {
    loading.value = true
    errorMessage.value = ''
    errorTraceId.value = ''
    try {
      const tid = String(tenantIdRef.value || '').trim()
      if (!tid) throw new Error('缺少租户 ID')
      await loadRegions()
      const fetchResources = async (regionSlug) => {
        const resourceURL = regionSlug
          ? `${apiBase.value}/?region=${encodeURIComponent(regionSlug)}`
          : `${apiBase.value}/`
        const response = await apiFetch(resourceURL, { headers: { Accept: 'application/json' } })
        const data = await response.json().catch(() => ({}))
        if (!response.ok) {
          const err = new Error(typeof data.error === 'string' ? data.error : '无法加载 GitLab 资源配额')
          err.traceId = response.traceId
          throw err
        }
        return data
      }
      let data = await fetchResources(region.value)
      // 无预选区域时 GET 返回 { resources[] }（Navbar 契约）。设置页需要单区详情。
      if (!String(region.value || '').trim() && Array.isArray(data.resources) && data.resources.length) {
        const picked =
          data.resources.find((r) => {
            const disk = Number(r?.disk_gb) || 0
            const traffic = Number(r?.traffic_prepaid_gb) || 0
            const status = String(r?.provisioning_status || '')
            return (disk > 0 || traffic > 0) && (status === 'active' || status === 'pending_admin')
          }) || data.resources[0]
        const slug = String(picked?.region || '').trim()
        if (slug) {
          region.value = slug
          data = await fetchResources(slug)
        }
      }
      applyView(data)
    } catch (e) {
      errorMessage.value = e?.message || '无法加载 GitLab 资源配额'
      errorTraceId.value = e?.traceId || ''
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    errorMessage,
    errorTraceId,
    successMessage,
    diskUnitPrice,
    trafficUnitPrice,
    currentDiskGb,
    currentTrafficGb,
    currentDiskMonths,
    currentDiskExpiresAt,
    diskUsedGb,
    trafficUsedGb,
    trafficDownloadAllowed,
    region,
    regionName,
    availableRegions,
    gitlabWebUrl,
    provisioningStatus,
    form,
    estimatedCost,
    diskMonthOptions: DISK_MONTH_OPTIONS,
    customMonths: form.custom_months,
    diskMonthsCustomLabel: '自定义',
    applyView,
    load,
    loadRegions,
  }
}
