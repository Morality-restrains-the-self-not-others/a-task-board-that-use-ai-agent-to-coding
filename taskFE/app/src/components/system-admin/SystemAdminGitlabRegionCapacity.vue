<template>
  <div class="space-y-3" data-testid="gitlab-region-capacity">
    <div class="flex items-center gap-2 text-xs">
      <span data-testid="gitlab-region-bandwidth-share"
            :class="region.bandwidth_shared ? 'text-primary' : 'text-text-light'">
        {{ shareLabel }}
      </span>
    </div>
    <div class="grid grid-cols-3 gap-4 text-sm">
      <div>
        <div class="flex justify-between text-text-light mb-1">
          <span>磁盘</span>
          <span>{{ region.allocated_disk_gb }} / {{ region.total_disk_gb }} GB</span>
        </div>
        <div class="w-full bg-gray-200 rounded-full h-3">
          <div class="bg-primary h-3 rounded-full transition-all"
               :style="{ width: diskPct + '%' }"></div>
        </div>
        <div class="text-xs text-text-light mt-1">
          剩余 <span class="font-medium text-text">{{ remainDisk }}</span> GB
        </div>
      </div>
      <div>
        <div class="flex justify-between text-text-light mb-1">
          <span>流量</span>
          <span>{{ region.allocated_traffic_gb }} / {{ region.total_traffic_gb }} GB</span>
        </div>
        <div class="w-full bg-gray-200 rounded-full h-3">
          <div class="bg-green-500 h-3 rounded-full transition-all"
               :style="{ width: trafficPct + '%' }"></div>
        </div>
        <div class="text-xs text-text-light mt-1">
          剩余 <span class="font-medium text-text">{{ remainTraffic }}</span> GB
        </div>
      </div>
      <div>
        <div class="flex justify-between text-text-light mb-1">
          <span>带宽</span>
          <span data-testid="gitlab-region-bandwidth-total">
            {{ bandwidthUsed }} / {{ region.total_bandwidth_mbps || 0 }} Mbps
          </span>
        </div>
        <div class="w-full bg-gray-200 rounded-full h-3">
          <div class="bg-amber-500 h-3 rounded-full transition-all"
               :style="{ width: bwPct + '%' }"></div>
        </div>
        <div class="text-xs text-text-light mt-1">
          剩余 <span class="font-medium text-text" data-testid="gitlab-region-bandwidth-remaining">{{ remainBw }}</span> Mbps
        </div>
      </div>
    </div>
    <div class="flex gap-3 items-end pt-2 border-t border-gray-100 flex-wrap">
      <div>
        <label class="block text-xs text-text-light mb-1">磁盘总量 (GB)</label>
        <input v-model.number="disk" type="number" min="0"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">流量总量 (GB)</label>
        <input v-model.number="traffic" type="number" min="0"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">总带宽 (Mbps)</label>
        <input v-model.number="totalBw" type="number" min="0"
               data-testid="gitlab-region-bandwidth-total-input"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <div>
        <label class="block text-xs text-text-light mb-1">剩余带宽 (Mbps)</label>
        <input v-model.number="remainBwInput" type="number" min="0"
               data-testid="gitlab-region-bandwidth-remaining-input"
               class="w-24 px-2 py-1.5 border border-border rounded text-sm" />
      </div>
      <label class="text-xs text-text-light select-none cursor-pointer flex items-center gap-1 pb-2">
        <input v-model="shared" type="checkbox" data-testid="gitlab-region-bandwidth-shared-input" class="rounded" />
        带宽共享分区
      </label>
      <button class="px-3 py-1.5 bg-primary text-white rounded text-sm hover:bg-primary/90 disabled:opacity-60"
              :disabled="saving || bwInvalid"
              :aria-busy="saving ? 'true' : 'false'"
              data-testid="gitlab-region-capacity-save"
              @click="onSave">
        {{ saving ? '保存中...' : '更新容量' }}
      </button>
    </div>
    <p v-if="bwInvalid" class="text-xs text-red-600"
       data-testid="gitlab-region-bandwidth-invalid-hint">
      剩余带宽不能大于总带宽，服务端会按总量截断，请先调整再保存
    </p>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import {
  bandwidthPercent,
  bandwidthShareLabel,
  bandwidthUsedMbps,
  remainingBandwidthMbps,
} from './gitlabRegionCapacity.js'

const props = defineProps({
  region: { type: Object, required: true },
  saving: { type: Boolean, default: false },
})
const emit = defineEmits(['save'])

const disk = ref(0)
const traffic = ref(0)
const totalBw = ref(0)
const remainBwInput = ref(0)
const shared = ref(false)

function syncFromRegion(r) {
  disk.value = r?.total_disk_gb || 0
  traffic.value = r?.total_traffic_gb || 0
  totalBw.value = r?.total_bandwidth_mbps || 0
  remainBwInput.value = r?.remaining_bandwidth_mbps || 0
  shared.value = !!r?.bandwidth_shared
}

watch(() => props.region, syncFromRegion, { immediate: true, deep: true })

const shareLabel = computed(() => bandwidthShareLabel(!!props.region?.bandwidth_shared))
const bandwidthUsed = computed(() => bandwidthUsedMbps(props.region))
const remainBw = computed(() => remainingBandwidthMbps(props.region))
const bwPct = computed(() => bandwidthPercent(props.region))
const diskPct = computed(() => {
  const total = Number(props.region?.total_disk_gb) || 0
  if (!total) return 0
  return Math.min(100, Math.round((Number(props.region?.allocated_disk_gb) / total) * 100))
})
const trafficPct = computed(() => {
  const total = Number(props.region?.total_traffic_gb) || 0
  if (!total) return 0
  return Math.min(100, Math.round((Number(props.region?.allocated_traffic_gb) / total) * 100))
})
const remainDisk = computed(() => Math.max(0, (props.region?.total_disk_gb || 0) - (props.region?.allocated_disk_gb || 0)))
const remainTraffic = computed(() => Math.max(0, (props.region?.total_traffic_gb || 0) - (props.region?.allocated_traffic_gb || 0)))

/** 剩余带宽 > 总带宽：服务端会按总量截断，提交前即时提示并禁用保存 */
const bwInvalid = computed(() => {
  const total = Number(totalBw.value) || 0
  const remain = Number(remainBwInput.value) || 0
  return remain > total
})

function onSave() {
  if (bwInvalid.value) return
  emit('save', {
    slug: props.region.slug,
    total_disk_gb: Number(disk.value) || 0,
    total_traffic_gb: Number(traffic.value) || 0,
    total_bandwidth_mbps: Number(totalBw.value) || 0,
    remaining_bandwidth_mbps: Number(remainBwInput.value) || 0,
    bandwidth_shared: !!shared.value,
  })
}
</script>
