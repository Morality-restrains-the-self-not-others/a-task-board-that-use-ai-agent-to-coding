<template>
        <div class="flex items-center gap-2">
          <button
            v-if="!panel.runTemplateMode && panel.hasConfiguredProjectRunTemplate() && !panel.temporaryConfigExpanded"
            type="button"
            data-testid="open-temporary-hardware-config-btn"
            class="px-3 py-1 border border-primary text-primary rounded-md text-xs font-medium hover:bg-primary hover:text-white focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
            @click="panel.openTemporaryConfig"
          >
            临时配置
          </button>
          <button
            v-if="!panel.runTemplateMode && panel.hasConfiguredProjectRunTemplate() && panel.temporaryConfigExpanded"
            type="button"
            data-testid="close-temporary-hardware-config-btn"
            class="px-3 py-1 border border-gray-300 text-gray-700 rounded-md text-xs font-medium hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
            @click="panel.closeTemporaryConfig"
          >
            收起临时配置
          </button>
          <!-- 应用上一次运行配置按钮 -->
          <button
            v-if="!panel.runTemplateMode && panel.temporaryConfigExpanded && panel.previousServerConfig"
            type="button"
            @click="panel.applyPreviousConfig"
            class="px-3 py-1 border border-primary text-primary rounded-md text-xs font-medium hover:bg-primary hover:text-white focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary"
          >
            应用上一次运行配置
          </button>
          <button
            v-if="panel.runTemplateMode"
            type="button"
            data-testid="hardware-panel-collapse-toggle"
            class="inline-flex items-center gap-1 px-2 py-1 border border-gray-300 rounded-md text-xs text-gray-600 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-primary"
            :aria-expanded="panel.panelExpanded ? 'true' : 'false'"
            @click="panel.panelExpanded = !panel.panelExpanded"
          >
            <span>{{ panel.panelExpanded ? '收起' : '展开' }}</span>
            <svg
              class="w-3.5 h-3.5 transition-transform duration-200"
              :class="{ 'rotate-180': panel.panelExpanded }"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
        </div>
</template>

<script setup>
defineProps({
  panel: { type: Object, required: true },
})
</script>
