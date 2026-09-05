<template>
  <div
    v-if="visible"
    class="space-y-2"
    data-testid="order-resource-consumption"
  >
    <h4 class="text-xs font-semibold text-gray-500 uppercase tracking-wide">资源消耗</h4>
    <p
      v-if="resourceRows.length === 0"
      class="text-xs text-gray-400"
    >
      该订单没有已记入的购买资源
    </p>
    <div
      v-for="row in resourceRows"
      :key="row.type"
      class="space-y-1"
      :data-testid="`order-consumption-${row.type}`"
    >
      <p class="text-sm text-gray-800">
        {{ row.label }}：已发放 {{ formatQty(row.granted) }}{{ row.unit }}
        · 已消耗 {{ formatQty(row.consumed) }}{{ row.unit }}
        · 剩余 {{ formatQty(row.remaining) }}{{ row.unit }}
        <span v-if="row.sourceLabel" class="text-xs text-gray-500 ml-1">{{ row.sourceLabel }}</span>
        <span v-if="row.region" class="text-xs text-gray-500 ml-1">{{ row.region }}</span>
      </p>
      <table v-if="row.events.length" class="w-full text-xs">
        <thead>
          <tr class="text-left text-gray-500">
            <th class="pb-1 pr-4 font-medium">时间</th>
            <th class="pb-1 pr-4 font-medium">动作</th>
            <th class="pb-1 pr-4 font-medium">任务</th>
            <th class="pb-1 text-right font-medium">数量</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(ev, idx) in row.events" :key="idx" data-testid="order-consumption-event">
            <td class="py-1 pr-4 text-gray-600">{{ ev.created_at || '—' }}</td>
            <td class="py-1 pr-4 text-gray-800">{{ actionLabel(ev.action) }}</td>
            <td class="py-1 pr-4 font-mono text-gray-700 break-all">{{ ev.task_id || '—' }}</td>
            <td class="py-1 text-right text-gray-800">{{ ev.quantity ?? 1 }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="text-xs text-gray-400">{{ row.emptyEventsHint }}</p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { billingResourceTypeLabel } from '@/utils/billingResourceTypeLabel.js'

const RESOURCE_KEYS = ['task_post', 'gitlab_disk', 'gitlab_traffic']

const props = defineProps({
  consumption: { type: Object, default: null },
  alwaysShow: { type: Boolean, default: false },
})

const resourceRows = computed(() => {
  const c = props.consumption
  if (!c || typeof c !== 'object') return []
  const extras = Object.keys(c).filter((k) => !RESOURCE_KEYS.includes(k) && isResourceBlock(c[k]))
  const keys = [...RESOURCE_KEYS.filter((k) => isResourceBlock(c[k])), ...extras]
  return keys.map((type) => toResourceRow(type, c[type]))
})

const visible = computed(() => props.alwaysShow || resourceRows.value.length > 0)

function isResourceBlock(v) {
  return v && typeof v === 'object' && !Array.isArray(v)
}

function toResourceRow(type, raw) {
  const sourceKind = String(raw.source_kind || '')
  let sourceLabel = ''
  if (sourceKind === 'gift') sourceLabel = '（赠送）'
  else if (sourceKind === 'purchase') sourceLabel = '（购买）'
  const unit = raw.unit ? ` ${raw.unit}` : gitlabUnit(type)
  return {
    type,
    label: billingResourceTypeLabel(type),
    granted: raw.granted ?? 0,
    consumed: raw.consumed ?? 0,
    remaining: raw.remaining ?? 0,
    unit,
    region: raw.region || '',
    sourceLabel,
    events: Array.isArray(raw.events) ? raw.events : [],
    emptyEventsHint: gitlabUnit(type)
      ? '用量记在区域配额上，无按订单逐笔流水'
      : '暂无逐笔消耗记录（历史消耗可能未记入本订单）',
  }
}

function gitlabUnit(type) {
  return type === 'gitlab_disk' || type === 'gitlab_traffic' ? ' GB' : ''
}

function formatQty(n) {
  const x = Number(n)
  if (!Number.isFinite(x)) return '0'
  if (Number.isInteger(x)) return String(x)
  return String(x)
}

const actionLabel = (action) => (action === 'renewal' ? '续存' : '创建任务帖')
</script>
