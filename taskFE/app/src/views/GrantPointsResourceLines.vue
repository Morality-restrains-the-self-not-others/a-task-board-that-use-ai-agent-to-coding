<template>
  <div>
    <div class="flex items-center justify-between mb-2">
      <label class="block text-sm font-medium text-gray-700">赠送资源</label>
      <button
        type="button"
        class="text-xs text-primary hover:text-primary/80 font-medium"
        @click="$emit('add')"
      >
        + 添加资源类型
      </button>
    </div>

    <div
      v-for="(res, idx) in resourceLines"
      :key="idx"
      class="border border-gray-200 rounded-lg p-4 mb-3 bg-gray-50/50"
    >
      <div class="flex items-center justify-between mb-3">
        <span class="text-xs font-medium text-gray-500">资源 #{{ idx + 1 }}</span>
        <button
          v-if="resourceLines.length > 1 || allowRemoveLast"
          type="button"
          class="text-xs text-red-500 hover:text-red-700"
          @click="$emit('remove', idx)"
        >
          移除
        </button>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label class="block text-xs text-gray-600 mb-1">资源类型</label>
          <select
            v-model="res.resource_type"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
          >
            <option v-for="rt in resourceTypes" :key="rt.value" :value="rt.value">
              {{ rt.label }}
            </option>
          </select>
        </div>

        <div v-if="needsGitlabRegion(res.resource_type)">
          <label class="block text-xs text-gray-600 mb-1">GitLab 区域</label>
          <select
            v-model="res.region"
            data-testid="grant-gitlab-region"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
          >
            <option value="">请选择区域</option>
            <option v-for="rg in regions" :key="rg.slug" :value="rg.slug">
              {{ rg.name || rg.slug }}
            </option>
          </select>
        </div>

        <div>
          <label class="block text-xs text-gray-600 mb-1">数量</label>
          <input
            v-model.number="res.quantity"
            type="number"
            min="1"
            step="1"
            data-testid="grant-quantity-input"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            placeholder="须填写，默认不赠送"
          >
        </div>

        <div>
          <label class="block text-xs text-gray-600 mb-1">
            过期时间
            <span class="text-gray-400">（可选，留空为永不过期）</span>
          </label>
          <input
            v-model="res.expires_at"
            type="date"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
          >
        </div>

        <div>
          <label class="block text-xs text-gray-600 mb-1">
            赠送原因
            <span class="text-gray-400">（可选）</span>
          </label>
          <input
            v-model.trim="res.reason"
            type="text"
            class="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            placeholder="例如：活动补偿、新用户礼包、人工调账"
          >
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  resourceLines: { type: Array, required: true },
  regions: { type: Array, default: () => [] },
  allowRemoveLast: { type: Boolean, default: false },
})
defineEmits(['add', 'remove'])

const resourceTypes = [
  { value: 'task_post', label: '任务帖' },
  { value: 'gitlab_disk', label: 'GitLab 磁盘 (GB)' },
  { value: 'gitlab_traffic', label: 'GitLab 流量 (GB)' },
]

function needsGitlabRegion(rt) {
  return rt === 'gitlab_disk' || rt === 'gitlab_traffic'
}
</script>
