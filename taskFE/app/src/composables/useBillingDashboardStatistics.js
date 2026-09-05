import { ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'

/**
 * 账单首页消耗/支付统计（本月消耗、累计消耗、累计支付）。
 * 经 taskBill `/billing/statistics/` 服务端聚合。
 *
 * 口径：
 * - 累计消耗 = consumption 合计（用量扣费 + 资源订单 resource_purchase）
 * - 累计支付 = 用户实付（钱包充值渠道 + 资源订单实付），不含 admin_grant
 *
 * API 契约字段为 `*_points`（人民币分）；`*_cents` 为同值别名。
 */

export function localMonthStartISODate(now = new Date()) {
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  return `${y}-${m}-01`
}

/**
 * @param {Record<string, unknown>} data
 * @param {string} stem 如 total_consumption / monthly_consumption / user_recharge
 * @returns {number} 分；合法缺失时为 0
 */
export function readStatisticsCents(data, stem) {
  const points = data?.[`${stem}_points`]
  if (points != null && points !== '') {
    const n = Number(points)
    if (Number.isFinite(n)) return n
  }
  const cents = data?.[`${stem}_cents`]
  if (cents != null && cents !== '') {
    const n = Number(cents)
    if (Number.isFinite(n)) return n
  }
  return 0
}

export function useBillingDashboardStatistics(tenantId) {
  const monthlyConsumption = ref(null)
  const totalConsumption = ref(null)
  const totalRecharge = ref(null)

  const fetchStatistics = async () => {
    try {
      const monthStartStr = localMonthStartISODate()
      const response = await apiFetch(
        `/api/tenant/${tenantId.value}/billing/statistics/?month_start=${encodeURIComponent(monthStartStr)}`,
        {
          credentials: 'include',
          headers: { Accept: 'application/json' }
        }
      )
      if (!response.ok) {
        console.error('获取统计数据失败')
        return
      }
      const data = await response.json()
      totalConsumption.value = readStatisticsCents(data, 'total_consumption')
      monthlyConsumption.value = readStatisticsCents(data, 'monthly_consumption')
      totalRecharge.value = readStatisticsCents(data, 'user_recharge')
    } catch (error) {
      console.error('获取统计数据失败:', error)
    }
  }

  return {
    monthlyConsumption,
    totalConsumption,
    totalRecharge,
    fetchStatistics
  }
}
