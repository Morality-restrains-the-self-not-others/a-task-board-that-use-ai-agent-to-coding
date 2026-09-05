import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils'
import { useBillingDashboardStatistics } from './useBillingDashboardStatistics.js'

export function useBillingDashboard() {
  const route = useRoute()
  const tenantId = computed(() => route.params.tenant)
  const tenantPath = computed(() => `/tenant/${tenantId.value}`)

  const cents = ref(null)
  const frozenBalance = ref(0)
  const refundEnabled = ref(true)
  const recentTransactions = ref([])

  const { monthlyConsumption, totalConsumption, totalRecharge, fetchStatistics } =
    useBillingDashboardStatistics(tenantId)

  const balanceDisplay = computed(() => cents.value ?? '0')

  const formatDate = (dateString) => {
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const getTransactionTypeText = (type) => {
    const typeMap = {
      recharge: '入账',
      consumption: '消耗',
      refund: '退款'
    }
    return typeMap[type] || type
  }

  const fetchBalance = async () => {
    try {
      const response = await apiFetch(`/api/tenant/${tenantId.value}/billing/accounts/balance/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' }
      })
      if (response.ok) {
        const data = await response.json()
        cents.value = data.cents != null ? String(data.cents) : '0'
        frozenBalance.value = Number(data.frozen_balance) || 0
        refundEnabled.value = data.refund_enabled !== false
      }
    } catch (error) {
      console.error('获取配额失败:', error)
    }
  }

  const fetchRecentTransactions = async () => {
    try {
      const response = await apiFetch(
        `/api/tenant/${tenantId.value}/billing/transactions/?page=1&page_size=5`,
        {
          credentials: 'include',
          headers: { Accept: 'application/json' }
        }
      )
      if (response.ok) {
        const data = await response.json()
        const list = Array.isArray(data)
          ? data
          : Array.isArray(data?.results)
            ? data.results
            : []
        recentTransactions.value = list.slice(0, 5)
      }
    } catch (error) {
      console.error('获取交易记录失败:', error)
    }
  }

  onMounted(() => {
    fetchBalance()
    fetchRecentTransactions()
    fetchStatistics()
  })

  return {
    tenantPath,
    balanceDisplay,
    frozenBalance,
    refundEnabled,
    monthlyConsumption,
    totalConsumption,
    totalRecharge,
    recentTransactions,
    formatDate,
    getTransactionTypeText,
    fetchBalance,
  }
}
