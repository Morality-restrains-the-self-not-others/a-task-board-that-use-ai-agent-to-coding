<template>
  <div class="mt-3 space-y-4" data-testid="gitlab-region-quota-rows">
    <p class="text-xs font-medium text-gray-500">按区域配额</p>
    <div
      v-for="row in rows"
      :key="row.regionSlug || row.region"
      class="space-y-2"
      :data-region-slug="row.regionSlug"
    >
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
        <h3 class="text-sm font-semibold text-gray-800">{{ row.region }}</h3>
        <a
          v-if="row.webUrl"
          :href="row.webUrl"
          target="_blank"
          rel="noopener"
          class="text-xs text-primary hover:underline"
        >进入仓库</a>
        <!-- Anti-Replay-OK: real outbound GitLab href; no write POST -->
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div
          class="bg-green-50 rounded-lg p-4"
          data-testid="gitlab-region-disk-card"
        >
          <p class="text-sm text-gray-500 mb-1">GitLab 磁盘</p>
          <p class="text-2xl font-bold text-green-700">
            {{ formatUsedGb(row.diskUsedGb) }} / {{ row.diskGb }}
            <span class="text-sm font-normal">GB</span>
          </p>
          <div
            class="mt-2 h-1.5 rounded-full bg-green-100 overflow-hidden"
            role="progressbar"
            :aria-valuenow="usagePercent(row.diskUsedGb, row.diskGb)"
            aria-valuemin="0"
            aria-valuemax="100"
            :aria-label="`${row.region} 磁盘已用`"
          >
            <div
              class="h-full bg-green-600 rounded-full"
              :style="{ width: usagePercent(row.diskUsedGb, row.diskGb) + '%' }"
            />
          </div>
          <p v-if="row.diskExpiresAt" class="text-xs text-gray-500 mt-1">到期：{{ row.diskExpiresAt }}</p>
          <p v-if="row.diskGiftedGb != null" class="text-xs text-gray-400 mt-1">
            赠送 {{ row.diskGiftedGb }} · 购买 {{ row.diskPurchasedGb }}
          </p>
        </div>
        <div
          class="bg-amber-50 rounded-lg p-4"
          data-testid="gitlab-region-traffic-card"
        >
          <p class="text-sm text-gray-500 mb-1">GitLab 流量</p>
          <p class="text-2xl font-bold text-amber-700">
            {{ formatUsedGb(row.trafficUsedGb) }} / {{ row.trafficPrepaidGb }}
            <span class="text-sm font-normal">GB</span>
          </p>
          <div
            class="mt-2 h-1.5 rounded-full bg-amber-100 overflow-hidden"
            role="progressbar"
            :aria-valuenow="usagePercent(row.trafficUsedGb, row.trafficPrepaidGb)"
            aria-valuemin="0"
            aria-valuemax="100"
            :aria-label="`${row.region} 流量已用`"
          >
            <div
              class="h-full bg-amber-600 rounded-full"
              :style="{ width: usagePercent(row.trafficUsedGb, row.trafficPrepaidGb) + '%' }"
            />
          </div>
          <p v-if="row.trafficGiftedGb != null" class="text-xs text-gray-400 mt-1">
            赠送 {{ row.trafficGiftedGb }} · 购买 {{ row.trafficPurchasedGb }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { formatUsedGb, usagePercent } from '../utils/formatUsedGb.js'

defineProps({
  rows: {
    type: Array,
    required: true,
  },
})
</script>
