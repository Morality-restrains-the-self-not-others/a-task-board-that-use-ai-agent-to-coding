<template>
      <!-- 自动释放服务器（项目模版摘要模式与临时配置模式均可见） -->
      <div v-if="!panel.runTemplateMode" class="mt-4 p-3 border border-dashed border-gray-200 rounded-lg bg-gray-50/80">
        <div class="flex flex-wrap items-center gap-3">
          <label class="inline-flex items-center cursor-pointer">
            <input
              id="auto-release-enabled"
              type="checkbox"
              v-model="panel.autoReleaseEnabled"
              class="rounded border-gray-300 text-primary focus:ring-primary"
            />
            <span class="ml-1.5 text-xs text-gray-700">自动释放服务器</span>
          </label>
          <div v-if="panel.autoReleaseEnabled" class="flex items-center gap-1.5 flex-wrap">
            <label for="auto-release-minutes" class="text-xs text-gray-600">在</label>
            <input
              id="auto-release-minutes"
              type="number"
              v-model.number="panel.autoReleaseMinutes"
              min="30"
              max="1440"
              step="1"
              class="w-20 px-2 py-0.5 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary"
            />
            <span class="text-xs text-gray-600">分钟后自动释放</span>
          </div>
        </div>
        <p v-if="panel.autoReleaseEnabled" class="mt-1.5 text-[10px] text-gray-500">
          阿里云要求自动释放时间不早于当前时间 30 分钟（UTC ISO8601，秒为 00）；未勾选则不会自动释放。
        </p>
      </div>

</template>

<script setup>
defineProps({
  panel: { type: Object, required: true },
})
</script>
