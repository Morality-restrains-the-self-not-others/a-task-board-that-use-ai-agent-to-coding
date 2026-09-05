<template>
  <div
    v-if="show"
    class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50"
    data-testid="task-archive-settings-modal"
    @click.self="$emit('close')"
  >
    <div class="bg-white rounded-xl p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
      <h4 class="text-lg font-bold mb-2">套餐设置 · 任务存档时间</h4>
      <p class="text-sm text-gray-500 mb-4">
        为当前工作空间选择任务数据在此空间内的可存放档位。具体计费以租户所选平台价格套餐为准。
      </p>
      <label class="block text-sm font-medium text-gray-700 mb-1" for="task-archive-tier-select">任务存档时间</label>
      <select
        id="task-archive-tier-select"
        class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary mb-4"
        data-testid="task-archive-tier-select"
        :value="tier"
        @change="$emit('update:tier', $event.target.value)"
      >
        <option v-for="opt in ARCHIVE_TIER_OPTIONS" :key="opt.value" :value="opt.value">
          {{ opt.label }}
        </option>
      </select>
      <div class="flex justify-end space-x-3">
        <button
          class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
          type="button"
          @click="$emit('close')"
        >
          取消
        </button>
        <button
          class="btn-primary"
          type="button"
          data-testid="task-archive-tier-save"
          :disabled="saving"
          :aria-busy="saving ? 'true' : undefined"
          @click="$emit('save')"
        >
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ARCHIVE_TIER_OPTIONS } from '../utils/taskArchiveTiers.js'

defineProps({
  show: { type: Boolean, default: false },
  tier: { type: String, default: '7d' },
  saving: { type: Boolean, default: false },
})
defineEmits(['close', 'save', 'update:tier'])
</script>
