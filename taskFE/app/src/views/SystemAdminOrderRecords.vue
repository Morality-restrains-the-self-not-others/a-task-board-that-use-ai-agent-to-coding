<template>
  <div data-alias="view-system-admin-order-records" class="p-6">
    <h1 class="text-xl font-semibold text-gray-900 mb-1">订单与退款</h1>
    <p class="text-sm text-gray-500 mb-4">
      查看租户资源订单、待分账队列，并在同一入口审批退款申请、管理退款策略、知晓并登记租户开票申请（手动开具发票）。需管理员账号权限。
    </p>

    <div class="mb-6 flex flex-wrap gap-2 border-b border-gray-200" role="tablist">
      <button
        type="button"
        role="tab"
        data-testid="order-records-tab-orders"
        :aria-selected="activeTab === 'orders'"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'orders'
          ? 'border-primary text-primary'
          : 'border-transparent text-gray-600 hover:text-gray-900'"
        @click="setTab('orders')"
      >
        订单记录
      </button>
      <button
        type="button"
        role="tab"
        data-testid="order-records-tab-profit-sharing"
        :aria-selected="activeTab === 'profit-sharing'"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'profit-sharing'
          ? 'border-primary text-primary'
          : 'border-transparent text-gray-600 hover:text-gray-900'"
        @click="setTab('profit-sharing')"
      >
        待分账
      </button>
      <button
        type="button"
        role="tab"
        data-testid="order-records-tab-refund"
        :aria-selected="activeTab === 'refund'"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'refund'
          ? 'border-primary text-primary'
          : 'border-transparent text-gray-600 hover:text-gray-900'"
        @click="setTab('refund')"
      >
        退款审批
      </button>
      <button
        type="button"
        role="tab"
        data-testid="order-records-tab-invoice"
        :aria-selected="activeTab === 'invoice'"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'invoice'
          ? 'border-primary text-primary'
          : 'border-transparent text-gray-600 hover:text-gray-900'"
        @click="setTab('invoice')"
      >
        开票申请
      </button>
    </div>

    <SystemAdminOrderListPanel v-if="activeTab === 'orders'" />
    <SystemAdminProfitSharingPanel v-else-if="activeTab === 'profit-sharing'" />
    <SystemAdminRefundPanel v-else-if="activeTab === 'refund'" />
    <SystemAdminInvoicePanel v-else-if="activeTab === 'invoice'" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SystemAdminOrderListPanel from '../components/system-admin/SystemAdminOrderListPanel.vue'
import SystemAdminProfitSharingPanel from '../components/system-admin/SystemAdminProfitSharingPanel.vue'
import SystemAdminRefundPanel from '../components/system-admin/SystemAdminRefundPanel.vue'
import SystemAdminInvoicePanel from '../components/system-admin/SystemAdminInvoicePanel.vue'

const route = useRoute()
const router = useRouter()

const activeTab = computed(() => {
  const raw = String(route.query?.tab ?? 'orders').trim().toLowerCase()
  if (raw === 'refund' || raw === 'profit-sharing' || raw === 'invoice') return raw
  return 'orders'
})

const setTab = (tab) => {
  const next = (tab === 'refund' || tab === 'profit-sharing' || tab === 'invoice') ? tab : 'orders'
  const query = { ...route.query }
  if (next === 'orders') {
    delete query.tab
  } else {
    query.tab = next
  }
  router.replace({ path: '/system-admin/order-records/', query })
}
</script>
