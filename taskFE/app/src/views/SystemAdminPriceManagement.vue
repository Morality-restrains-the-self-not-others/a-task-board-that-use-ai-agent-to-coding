<template>
  <div class="p-6 max-w-6xl" data-alias="view-system-admin-price-management">
    <h2 class="text-xl font-semibold text-gray-900 mb-4">价格管理</h2>

    <nav class="flex gap-1 border-b border-gray-200 mb-6" aria-label="价格管理标签页">
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'pricing' ? 'border-primary text-primary' : 'border-transparent text-gray-600 hover:text-gray-900'"
        @click="setTab('pricing')"
      >
        资源定价
      </button>
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
        :class="activeTab === 'consumption' ? 'border-primary text-primary' : 'border-transparent text-gray-600 hover:text-gray-900'"
        @click="setTab('consumption')"
      >
        消费情况
      </button>
    </nav>

    <SystemAdminResourcePricingPanel v-if="activeTab === 'pricing'" />
    <SystemAdminRechargeConsumptionPanel v-else-if="activeTab === 'consumption'" />
  </div>
</template>

<script setup>
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SystemAdminResourcePricingPanel from './SystemAdminResourcePricingPanel.vue'
import SystemAdminRechargeConsumptionPanel from './SystemAdminRechargeConsumptionPanel.vue'

/* @alias:view-system-admin-price-management */

const route = useRoute()
const router = useRouter()

const activeTab = computed(() => {
  const tab = String(route.query.tab || '')
  return tab === 'consumption' ? 'consumption' : 'pricing'
})

const setTab = (tab) => {
  const nextQuery = tab === 'consumption' ? { tab: 'consumption' } : {}
  router.replace({ path: route.path, query: nextQuery })
}

watch(
  () => route.query.tab,
  (tab) => {
    if (tab && tab !== 'consumption' && tab !== 'pricing') {
      router.replace({ path: route.path, query: {} })
    }
  }
)
</script>
